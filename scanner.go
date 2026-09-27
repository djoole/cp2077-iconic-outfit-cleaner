package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	patchRelativePath       = `r6\scripts\IconicOutfitCleaner\generated.reds`
	legacyPatchRelativePath = `r6\tweaks\zz_IconicOutfitCleaner\generated.yaml`
)

var (
	topLevelRecordRE = regexp.MustCompile(`^([^\s#][^:]*):\s*(?:#.*)?$`)
	baseRE           = regexp.MustCompile(`(?m)^\s+\$base:\s*([^\s#]+)`)
	qualityRE        = regexp.MustCompile(`(?m)^\s+quality:\s*(Quality\.[A-Za-z0-9_]+)`)
	iconicAppendRE   = regexp.MustCompile(`(?m)^\s*-\s*!(?:append|append-once)\s+Quality\.IconicItem\s*(?:#.*)?$`)
	iconicPlainRE    = regexp.MustCompile(`(?m)^\s*-\s*Quality\.IconicItem\s*(?:#.*)?$`)
	instancesRE      = regexp.MustCompile(`(?m)^\s+\$instances:\s*(?:&[^\s#]+\s*)?(?:#.*)?$`)
	nexusSuffixRE    = regexp.MustCompile(`-\d{3,}(?:-|\s).*$`)
)

type deploymentManifest struct {
	StagingPath string           `json:"stagingPath"`
	TargetPath  string           `json:"targetPath"`
	Files       []deploymentFile `json:"files"`
}

type deploymentFile struct {
	RelPath string `json:"relPath"`
	Source  string `json:"source"`
	Target  string `json:"target"`
}

type record struct {
	id              string
	body            string
	base            string
	quality         string
	file            string
	relFile         string
	modID           string
	modName         string
	instancesBlock  string
	concreteIDs     []string
	isTemplate      bool
	directClothing  bool
	hasIconicMarker bool
}

type modResult struct {
	id    string
	name  string
	items []*record
	files map[string]struct{}
}

type scanResult struct {
	scanRoot      string
	outputRoot    string
	patchPath     string
	mods          []*modResult
	totalItems    int
	scannedFiles  int
	tweaksFolders int
	usingVortex   bool
	warnings      []string
}

type Scanner struct{}

func NewScanner() *Scanner { return &Scanner{} }

func (s *Scanner) Scan(scanRoot, outputRoot string) (*scanResult, error) {
	abs, err := filepath.Abs(scanRoot)
	if err != nil {
		return nil, fmt.Errorf("invalid scan path: %w", err)
	}
	info, err := os.Stat(abs)
	if err != nil || !info.IsDir() {
		return nil, fmt.Errorf("scan folder not found: %s", abs)
	}
	outputAbs, err := filepath.Abs(outputRoot)
	if err != nil {
		return nil, fmt.Errorf("invalid patch destination: %w", err)
	}

	manifest, sourceByRel := readDeploymentManifest(abs)
	result := &scanResult{
		scanRoot:    abs,
		outputRoot:  outputAbs,
		patchPath:   filepath.Join(outputAbs, patchRelativePath),
		usingVortex: manifest != nil,
	}
	tweaksDirs, discoveryWarnings, err := findTweaksFolders(abs)
	result.warnings = append(result.warnings, discoveryWarnings...)
	if err != nil {
		return nil, err
	}
	if len(tweaksDirs) == 0 {
		return nil, fmt.Errorf("no r6\\tweaks folders found under: %s", abs)
	}
	result.tweaksFolders = len(tweaksDirs)

	var records []*record
	for _, tweaksDir := range tweaksDirs {
		folderID, folderName := folderSource(abs, tweaksDir)
		err = filepath.WalkDir(tweaksDir, func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				result.warnings = append(result.warnings, fmt.Sprintf("%s: %v", path, walkErr))
				if entry != nil && entry.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
			if entry.IsDir() {
				return nil
			}
			ext := strings.ToLower(filepath.Ext(entry.Name()))
			if ext != ".yaml" && ext != ".yml" {
				return nil
			}
			result.scannedFiles++
			rel, relErr := filepath.Rel(abs, path)
			if relErr != nil {
				return relErr
			}
			modID, modName := sourceForFile(rel, tweaksDir, path, sourceByRel, folderID, folderName)
			parsed, parseErr := parseTweakFile(path, filepath.ToSlash(rel), modID, modName)
			if parseErr != nil {
				result.warnings = append(result.warnings, fmt.Sprintf("%s: %v", rel, parseErr))
				return nil
			}
			records = append(records, parsed...)
			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("failed to scan %s: %w", tweaksDir, err)
		}
	}

	byID := make(map[string]*record, len(records))
	for _, rec := range records {
		byID[rec.id] = rec
	}
	clothingMemo := make(map[string]bool, len(records))
	var isClothing func(*record, map[string]bool) bool
	isClothing = func(rec *record, visiting map[string]bool) bool {
		if value, ok := clothingMemo[rec.id]; ok {
			return value
		}
		if rec.directClothing {
			clothingMemo[rec.id] = true
			return true
		}
		if rec.base == "" || visiting[rec.id] {
			return false
		}
		visiting[rec.id] = true
		parent := byID[rec.base]
		value := parent != nil && isClothing(parent, visiting)
		delete(visiting, rec.id)
		clothingMemo[rec.id] = value
		return value
	}

	mods := make(map[string]*modResult)
	for _, rec := range records {
		if !rec.hasIconicMarker || !isClothing(rec, make(map[string]bool)) {
			continue
		}
		mod := mods[rec.modID]
		if mod == nil {
			mod = &modResult{id: rec.modID, name: rec.modName, files: make(map[string]struct{})}
			mods[rec.modID] = mod
		}
		mod.items = append(mod.items, rec)
		mod.files[rec.relFile] = struct{}{}
		result.totalItems++
	}

	for _, mod := range mods {
		sort.Slice(mod.items, func(i, j int) bool { return mod.items[i].id < mod.items[j].id })
		result.mods = append(result.mods, mod)
	}
	sort.Slice(result.mods, func(i, j int) bool {
		return strings.ToLower(result.mods[i].name) < strings.ToLower(result.mods[j].name)
	})
	return result, nil
}

func (s *Scanner) Generate(scan *scanResult, selectedIDs []string) (GenerateResultDTO, error) {
	selected := make(map[string]bool, len(selectedIDs))
	for _, id := range selectedIDs {
		selected[id] = true
	}

	var mods []*modResult
	itemCount := 0
	for _, mod := range scan.mods {
		if selected[mod.id] {
			mods = append(mods, mod)
			itemCount += len(mod.items)
		}
	}
	if len(mods) == 0 {
		return GenerateResultDTO{}, fmt.Errorf("none of the selected mods belong to the latest scan")
	}

	var concreteIDs []string
	for _, mod := range mods {
		for _, rec := range mod.items {
			concreteIDs = append(concreteIDs, rec.concreteIDs...)
		}
	}
	concreteIDs = uniqueSorted(concreteIDs)
	if len(concreteIDs) == 0 {
		return GenerateResultDTO{}, fmt.Errorf("the selected templates could not be expanded into concrete item IDs")
	}

	var out strings.Builder
	out.WriteString("// Generated by CP2077 Iconic Outfit Cleaner.\n")
	out.WriteString("// Runs after declarative TweakXL imports so source !append operations are already applied.\n")
	out.WriteString("// Source mods are not modified. Regenerate after adding or updating outfit mods.\n\n")
	out.WriteString("module IconicOutfitCleaner\n\n")
	out.WriteString("public class IconicOutfitCleaner_Generated extends ScriptableTweak {\n")
	out.WriteString("  protected cb func OnApply() -> Void {\n")
	const chunkSize = 200
	for start := 0; start < len(concreteIDs); start += chunkSize {
		fmt.Fprintf(&out, "    this.Apply%d();\n", start/chunkSize)
	}
	out.WriteString("  }\n\n")
	for start := 0; start < len(concreteIDs); start += chunkSize {
		end := start + chunkSize
		if end > len(concreteIDs) {
			end = len(concreteIDs)
		}
		fmt.Fprintf(&out, "  private func Apply%d() -> Void {\n", start/chunkSize)
		for _, id := range concreteIDs[start:end] {
			fmt.Fprintf(&out, "    this.RemoveIconic(t\"%s\");\n", escapeRedscriptString(id))
		}
		out.WriteString("  }\n\n")
	}
	out.WriteString("  private func RemoveIconic(item: TweakDBID) -> Void {\n")
	out.WriteString("    let flat: Variant = TweakDBInterface.GetFlat(item + t\".statModifiers\");\n")
	out.WriteString("    let changed: Bool = false;\n")
	out.WriteString("    if IsDefined(flat) {\n")
	out.WriteString("      let modifiers: array<TweakDBID> = FromVariant<array<TweakDBID>>(flat);\n")
	out.WriteString("      while ArrayContains(modifiers, t\"Quality.IconicItem\") {\n")
	out.WriteString("        ArrayRemove(modifiers, t\"Quality.IconicItem\");\n")
	out.WriteString("        changed = true;\n")
	out.WriteString("      };\n")
	out.WriteString("      if changed {\n")
	out.WriteString("        TweakDBManager.SetFlat(item + t\".statModifiers\", ToVariant(modifiers));\n")
	out.WriteString("      };\n")
	out.WriteString("    };\n")
	out.WriteString("    let quality: Variant = TweakDBInterface.GetFlat(item + t\".quality\");\n")
	out.WriteString("    if IsDefined(quality) && Equals(FromVariant<TweakDBID>(quality), t\"Quality.Iconic\") {\n")
	out.WriteString("      TweakDBManager.SetFlat(item + t\".quality\", t\"Quality.Legendary\");\n")
	out.WriteString("      changed = true;\n")
	out.WriteString("    };\n")
	out.WriteString("    if changed { TweakDBManager.UpdateRecord(item); };\n")
	out.WriteString("  }\n")
	out.WriteString("}\n")

	patchDir := filepath.Dir(scan.patchPath)
	if err := os.MkdirAll(patchDir, 0o755); err != nil {
		return GenerateResultDTO{}, fmt.Errorf("failed to create the patch folder: %w", err)
	}
	backupPath := ""
	if old, err := os.ReadFile(scan.patchPath); err == nil {
		backupPath = filepath.Join(patchDir, "generated."+time.Now().Format("20060102-150405")+".reds.bak")
		if err := os.WriteFile(backupPath, old, 0o644); err != nil {
			return GenerateResultDTO{}, fmt.Errorf("failed to back up the previous patch: %w", err)
		}
	}
	tempPath := scan.patchPath + ".tmp"
	if err := os.WriteFile(tempPath, []byte(out.String()), 0o644); err != nil {
		return GenerateResultDTO{}, fmt.Errorf("failed to write the temporary patch: %w", err)
	}
	if err := os.Rename(tempPath, scan.patchPath); err != nil {
		_ = os.Remove(tempPath)
		return GenerateResultDTO{}, fmt.Errorf("failed to activate the patch: %w", err)
	}
	if err := deactivateLegacyYAMLPatches(scan.outputRoot); err != nil {
		return GenerateResultDTO{}, err
	}

	return GenerateResultDTO{
		PatchPath:    scan.patchPath,
		ModCount:     len(mods),
		ItemCount:    itemCount,
		BackupPath:   backupPath,
		PatchSummary: fmt.Sprintf("%d concrete outfit items from %d mods", len(concreteIDs), len(mods)),
	}, nil
}

func (r *scanResult) DTO() ScanResultDTO {
	dto := ScanResultDTO{
		ScanRoot:      r.scanRoot,
		OutputRoot:    r.outputRoot,
		TotalItems:    r.totalItems,
		ScannedFiles:  r.scannedFiles,
		TweaksFolders: r.tweaksFolders,
		UsingVortex:   r.usingVortex,
		PatchPath:     r.patchPath,
		Warnings:      append([]string(nil), r.warnings...),
	}
	for _, mod := range r.mods {
		m := ModDTO{ID: mod.id, Name: mod.name, SourceLabel: mod.id, ItemCount: len(mod.items), FileCount: len(mod.files)}
		for _, item := range mod.items {
			m.Items = append(m.Items, ItemDTO{
				RecordID: item.id, SourceFile: item.relFile,
				OriginalQuality: item.quality, Template: item.isTemplate,
			})
		}
		dto.Mods = append(dto.Mods, m)
	}
	return dto
}

func parseTweakFile(path, relFile, modID, modName string) ([]*record, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	data = bytes.ReplaceAll(data, []byte("\r\n"), []byte("\n"))
	lines := strings.Split(string(data), "\n")
	type section struct {
		start int
		id    string
	}
	var sections []section
	for i, line := range lines {
		match := topLevelRecordRE.FindStringSubmatch(line)
		if match == nil {
			continue
		}
		id := strings.TrimSpace(match[1])
		sections = append(sections, section{start: i, id: id})
	}
	var out []*record
	for index, current := range sections {
		if !strings.HasPrefix(current.id, "Items.") {
			continue
		}
		start := current.start
		end := len(lines)
		if index+1 < len(sections) {
			end = sections[index+1].start
		}
		body := strings.Join(lines[start+1:end], "\n")
		rec := &record{id: current.id, body: body, file: path, relFile: relFile, modID: modID, modName: modName}
		if match := baseRE.FindStringSubmatch(body); match != nil {
			rec.base = strings.TrimSpace(match[1])
		}
		if match := qualityRE.FindStringSubmatch(body); match != nil {
			rec.quality = match[1]
		}
		rec.hasIconicMarker = iconicAppendRE.MatchString(body) || iconicPlainRE.MatchString(body) || rec.quality == "Quality.Iconic"
		rec.directClothing = looksLikeClothing(rec.base, body)
		rec.instancesBlock = extractInstancesBlock(lines[start+1 : end])
		rec.isTemplate = strings.Contains(rec.id, "$(") || strings.Contains(rec.id, "${")
		if rec.isTemplate {
			rec.concreteIDs = expandTemplateIDs(rec.id, rec.instancesBlock)
		} else {
			rec.concreteIDs = []string{rec.id}
		}
		out = append(out, rec)
	}
	return out, nil
}

func extractInstancesBlock(lines []string) string {
	start := -1
	baseIndent := 0
	for i, line := range lines {
		if instancesRE.MatchString(line) {
			start = i
			baseIndent = leadingSpaces(line)
			break
		}
	}
	if start < 0 {
		return ""
	}
	end := len(lines)
	for i := start + 1; i < len(lines); i++ {
		trimmed := strings.TrimSpace(lines[i])
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if leadingSpaces(lines[i]) <= baseIndent {
			end = i
			break
		}
	}
	block := strings.Join(lines[start:end], "\n")
	if block != "" && !strings.HasSuffix(block, "\n") {
		block += "\n"
	}
	return block
}

func looksLikeClothing(base, body string) bool {
	value := strings.ToLower(base + "\n" + body)
	markers := []string{
		"genericinnerchestclothing", "genericouterchestclothing", "genericlegclothing",
		"genericfootclothing", "genericheadclothing", "genericfaceclothing", "items.genericclothing",
		"equipmentarea.head", "equipmentarea.face", "equipmentarea.outerchest", "equipmentarea.innerchest",
		"equipmentarea.legs", "equipmentarea.feet", "equipmentarea.outfit", "outfitslots.",
	}
	for _, marker := range markers {
		if strings.Contains(value, marker) {
			return true
		}
	}
	return base == "Items.Outfit" || strings.Contains(value, "itemtype.clo_")
}

func readDeploymentManifest(gameDir string) (*deploymentManifest, map[string]string) {
	path := filepath.Join(gameDir, "vortex.deployment.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, nil
	}
	var manifest deploymentManifest
	if json.Unmarshal(data, &manifest) != nil || !strings.EqualFold(filepath.Clean(manifest.TargetPath), filepath.Clean(gameDir)) {
		return nil, nil
	}
	mapping := make(map[string]string)
	for _, file := range manifest.Files {
		rel := normalizeRelPath(file.RelPath)
		if (strings.HasSuffix(strings.ToLower(rel), ".yaml") || strings.HasSuffix(strings.ToLower(rel), ".yml")) && strings.HasPrefix(strings.ToLower(rel), "r6/tweaks/") {
			mapping[rel] = file.Source
		}
	}
	return &manifest, mapping
}

func sourceForFile(rel string, tweaksDir string, path string, sourceByRel map[string]string, folderID, folderName string) (string, string) {
	normalized := normalizeRelPath(rel)
	if source := sourceByRel[normalized]; source != "" {
		return source, prettyModName(source)
	}
	if folderID != "" {
		return folderID, folderName
	}
	relTweak, err := filepath.Rel(tweaksDir, path)
	if err != nil {
		relTweak = filepath.Base(path)
	}
	parts := strings.Split(filepath.ToSlash(relTweak), "/")
	id := parts[0]
	if len(parts) == 1 {
		id = strings.TrimSuffix(parts[0], filepath.Ext(parts[0]))
	}
	return "fallback:" + id, prettyModName(id)
}

func findTweaksFolders(root string) ([]string, []string, error) {
	var folders []string
	var warnings []string
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			warnings = append(warnings, fmt.Sprintf("%s: %v", path, walkErr))
			if entry != nil && entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if !entry.IsDir() || !strings.EqualFold(entry.Name(), "tweaks") {
			return nil
		}
		if strings.EqualFold(filepath.Base(filepath.Dir(path)), "r6") {
			folders = append(folders, filepath.Clean(path))
			return filepath.SkipDir
		}
		return nil
	})
	if err != nil {
		return nil, warnings, fmt.Errorf("failed to search for r6\\tweaks folders: %w", err)
	}
	sort.Slice(folders, func(i, j int) bool { return strings.ToLower(folders[i]) < strings.ToLower(folders[j]) })
	return folders, warnings, nil
}

func folderSource(scanRoot, tweaksDir string) (string, string) {
	modRoot := filepath.Dir(filepath.Dir(tweaksDir))
	if strings.EqualFold(filepath.Clean(modRoot), filepath.Clean(scanRoot)) && IsGameDirectory(scanRoot) {
		return "", ""
	}
	rel, err := filepath.Rel(scanRoot, modRoot)
	if err != nil || rel == "" {
		rel = filepath.Base(modRoot)
	}
	if rel == "." {
		rel = filepath.Base(modRoot)
	}
	name := filepath.Base(modRoot)
	return "folder:" + normalizeRelPath(rel), prettyModName(name)
}

func prettyModName(source string) string {
	name := nexusSuffixRE.ReplaceAllString(source, "")
	name = strings.ReplaceAll(name, "_", " ")
	name = strings.TrimSpace(strings.Trim(name, "- "))
	if name == "" {
		return source
	}
	return name
}

func normalizeRelPath(path string) string {
	return strings.ToLower(strings.TrimPrefix(filepath.ToSlash(filepath.Clean(path)), "./"))
}

func leadingSpaces(value string) int {
	return len(value) - len(strings.TrimLeft(value, " "))
}

func sanitizeComment(value string) string {
	return strings.ReplaceAll(strings.ReplaceAll(value, "\r", " "), "\n", " ")
}

func expandTemplateIDs(templateID, instancesBlock string) []string {
	if instancesBlock == "" {
		return nil
	}
	var data struct {
		Instances []map[string]yaml.Node `yaml:"$instances"`
	}
	if err := yaml.Unmarshal([]byte(instancesBlock), &data); err != nil {
		return nil
	}
	var ids []string
	for _, instance := range data.Instances {
		id := templateID
		for key, value := range instance {
			id = strings.ReplaceAll(id, "$("+key+")", value.Value)
			id = strings.ReplaceAll(id, "${"+key+"}", value.Value)
		}
		if !strings.Contains(id, "$(") && !strings.Contains(id, "${") {
			ids = append(ids, id)
		}
	}
	return uniqueSorted(ids)
}

func uniqueSorted(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		if value == "" {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func escapeRedscriptString(value string) string {
	return strings.ReplaceAll(strings.ReplaceAll(value, `\`, `\\`), `"`, `\"`)
}

func deactivateLegacyYAMLPatches(gameDir string) error {
	legacyDir := filepath.Dir(filepath.Join(gameDir, legacyPatchRelativePath))
	entries, err := os.ReadDir(legacyDir)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to inspect legacy YAML patches: %w", err)
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		lower := strings.ToLower(name)
		if !(lower == "generated.yaml" || strings.HasSuffix(lower, ".bak.yaml")) {
			continue
		}
		oldPath := filepath.Join(legacyDir, name)
		newPath := oldPath + ".disabled"
		if err := os.Rename(oldPath, newPath); err != nil {
			return fmt.Errorf("failed to deactivate legacy patch %s: %w", name, err)
		}
	}
	return nil
}
