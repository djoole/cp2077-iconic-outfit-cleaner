package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestScanGroupsVortexModsAndExcludesWeapons(t *testing.T) {
	game := t.TempDir()
	tweaks := filepath.Join(game, "r6", "tweaks")
	if err := os.MkdirAll(filepath.Join(tweaks, "clothes"), 0o755); err != nil {
		t.Fatal(err)
	}
	yaml := `Items.test_top_$(color):
  $base: Items.GenericInnerChestClothing
  $instances:
    - { color: black }
    - { color: red }
  quality: Quality.Legendary
  statModifiers:
    - !append Quality.IconicItem

Items.test_gun:
  $base: Items.Preset_Unity_Default
  statModifiers:
    - !append Quality.IconicItem
`
	if err := os.WriteFile(filepath.Join(tweaks, "clothes", "items.yaml"), []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}
	manifest := `{"targetPath":` + quoteJSON(game) + `,"files":[{"relPath":"r6\\tweaks\\clothes\\items.yaml","source":"Pretty Outfit-12345-1-0"}]}`
	if err := os.WriteFile(filepath.Join(game, "vortex.deployment.json"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := NewScanner().Scan(game, game)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.mods) != 1 || result.totalItems != 1 {
		t.Fatalf("got %d mods/%d items", len(result.mods), result.totalItems)
	}
	if result.mods[0].name != "Pretty Outfit" {
		t.Fatalf("unexpected name %q", result.mods[0].name)
	}
	if !result.mods[0].items[0].isTemplate {
		t.Fatal("template was not detected")
	}
}

func TestGenerateExpandsInstancesAndRemovesIconicAtRuntime(t *testing.T) {
	game := t.TempDir()
	tweaks := filepath.Join(game, "r6", "tweaks")
	if err := os.MkdirAll(tweaks, 0o755); err != nil {
		t.Fatal(err)
	}
	yaml := `Items.test_dress_${color}:
  $base: Items.Outfit
  $instances:
    - { color: gold }
  quality: Quality.Iconic
  statModifiers:
    - !append-once Quality.IconicItem
`
	if err := os.WriteFile(filepath.Join(tweaks, "dress.yaml"), []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}
	scanner := NewScanner()
	result, err := scanner.Scan(game, game)
	if err != nil {
		t.Fatal(err)
	}
	generated, err := scanner.Generate(result, []string{result.mods[0].id})
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(generated.PatchPath)
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	for _, want := range []string{
		"extends ScriptableTweak",
		`this.RemoveIconic(t"Items.test_dress_gold");`,
		`ArrayRemove(modifiers, t"Quality.IconicItem")`,
		`TweakDBManager.SetFlat(item + t".quality", t"Quality.Legendary")`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("generated patch lacks %q:\n%s", want, text)
		}
	}
}

func TestExpandTemplateIDsPreservesScalarText(t *testing.T) {
	block := "$instances:\n  - { color: black, quality: Iconic, icon: 01 }\n  - { color: red, quality: Legendary, icon: 02 }\n"
	got := expandTemplateIDs("Items.shirt_$(color)_${quality}", block)
	want := []string{"Items.shirt_black_Iconic", "Items.shirt_red_Legendary"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestIndentlessInstancesExpandMasuryanStyleTemplate(t *testing.T) {
	root := t.TempDir()
	tweaks := filepath.Join(root, "r6", "tweaks", "masuryan")
	if err := os.MkdirAll(tweaks, 0o755); err != nil {
		t.Fatal(err)
	}
	yaml := `Items.masuryans_split_dress_${base_color}:
  $base: Items.GenericInnerChestClothing
  $instances:
  - { base_color: black, icon: slot_01 }
  - { base_color: pattern_05, icon: slot_23 }
  quality: Quality.Legendary
  statModifiers:
    - !append Quality.IconicItem
`
	path := filepath.Join(tweaks, "masuryans_split_dress.yaml")
	if err := os.WriteFile(path, []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := NewScanner().Scan(root, root)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.mods) != 1 || len(result.mods[0].items) != 1 {
		t.Fatalf("got %d mods, expected one template item", len(result.mods))
	}
	got := result.mods[0].items[0].concreteIDs
	want := []string{"Items.masuryans_split_dress_black", "Items.masuryans_split_dress_pattern_05"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("got %v, want %v", got, want)
	}
	generated, err := NewScanner().Generate(result, []string{result.mods[0].id})
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(generated.PatchPath)
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range want {
		if !strings.Contains(string(data), `this.RemoveIconic(t"`+id+`");`) {
			t.Fatalf("generated patch lacks %s", id)
		}
	}
}

func TestScanFindsMO2ModsRecursivelyAndUsesCustomOutput(t *testing.T) {
	scanRoot := t.TempDir()
	outputRoot := filepath.Join(t.TempDir(), "Iconic Outfit Cleaner")
	tweaks := filepath.Join(scanRoot, "Cool Outfit", "r6", "tweaks", "clothes")
	if err := os.MkdirAll(tweaks, 0o755); err != nil {
		t.Fatal(err)
	}
	yaml := `Items.mo2_test_top:
  $base: Items.GenericInnerChestClothing
  statModifiers:
    - !append Quality.IconicItem
`
	if err := os.WriteFile(filepath.Join(tweaks, "items.yaml"), []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}

	scanner := NewScanner()
	result, err := scanner.Scan(scanRoot, outputRoot)
	if err != nil {
		t.Fatal(err)
	}
	if result.tweaksFolders != 1 || len(result.mods) != 1 {
		t.Fatalf("got %d tweak folders and %d mods", result.tweaksFolders, len(result.mods))
	}
	if result.mods[0].name != "Cool Outfit" {
		t.Fatalf("unexpected MO2 mod name %q", result.mods[0].name)
	}
	wantPatch := filepath.Join(outputRoot, patchRelativePath)
	if result.patchPath != wantPatch {
		t.Fatalf("patch path %q, want %q", result.patchPath, wantPatch)
	}
	generated, err := scanner.Generate(result, []string{result.mods[0].id})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(generated.PatchPath); err != nil {
		t.Fatalf("custom output was not generated: %v", err)
	}
}

func TestInstalledGameScan(t *testing.T) {
	game := os.Getenv("CP77_TEST_GAME_DIR")
	if game == "" {
		t.Skip("set CP77_TEST_GAME_DIR to run the integration scan")
	}
	result, err := NewScanner().Scan(game, game)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.mods) == 0 || result.totalItems == 0 {
		t.Fatal("the installed game scan returned no candidates")
	}
	concrete, unexpanded := 0, 0
	for _, mod := range result.mods {
		for _, item := range mod.items {
			concrete += len(item.concreteIDs)
			if item.isTemplate && len(item.concreteIDs) == 0 {
				unexpanded++
				t.Logf("unexpanded template: %s (%s)", item.id, item.relFile)
			}
		}
	}
	t.Logf("scanned %d YAML files: %d candidate roots / %d concrete items in %d source mods (Vortex=%v, warnings=%d, unexpanded=%d)",
		result.scannedFiles, result.totalItems, concrete, len(result.mods), result.usingVortex, len(result.warnings), unexpanded)
}

func TestInstalledGameAutoDetection(t *testing.T) {
	expected := os.Getenv("CP77_TEST_GAME_DIR")
	if expected == "" {
		t.Skip("set CP77_TEST_GAME_DIR to run the integration detection")
	}
	detected := DetectGameDirectory()
	if !IsGameDirectory(detected) {
		t.Fatalf("automatic detection returned an invalid directory: %q", detected)
	}
	t.Logf("automatically detected %s", detected)
}

func TestInstalledGameGenerate(t *testing.T) {
	game := os.Getenv("CP77_TEST_GAME_DIR")
	if game == "" {
		t.Skip("set CP77_TEST_GAME_DIR to run the integration generation")
	}
	scanner := NewScanner()
	result, err := scanner.Scan(game, game)
	if err != nil {
		t.Fatal(err)
	}
	ids := make([]string, 0, len(result.mods))
	for _, mod := range result.mods {
		ids = append(ids, mod.id)
	}
	generated, err := scanner.Generate(result, ids)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("generated %s (%s)", generated.PatchPath, generated.PatchSummary)
}

func quoteJSON(value string) string {
	return `"` + strings.ReplaceAll(value, `\`, `\\`) + `"`
}
