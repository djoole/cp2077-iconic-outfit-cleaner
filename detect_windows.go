package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"golang.org/x/sys/windows/registry"
)

var steamLibraryPathRE = regexp.MustCompile(`(?i)"path"\s+"([^"]+)"`)

// DetectGameDirectory checks persisted platform metadata rather than assuming a
// drive letter. Registry uninstall entries cover Steam, GOG and most standalone
// installs; Steam and Epic manifests provide independent fallbacks.
func DetectGameDirectory() string {
	var candidates []string
	candidates = append(candidates, uninstallRegistryCandidates()...)
	candidates = append(candidates, steamCandidates()...)
	candidates = append(candidates, epicCandidates()...)

	// Last-resort conventional locations on every mounted Windows drive.
	for drive := 'C'; drive <= 'Z'; drive++ {
		root := string(drive) + `:\`
		if _, err := os.Stat(root); err != nil {
			continue
		}
		candidates = append(candidates,
			filepath.Join(root, `Program Files (x86)\Steam\steamapps\common\Cyberpunk 2077`),
			filepath.Join(root, `SteamLibrary\steamapps\common\Cyberpunk 2077`),
			filepath.Join(root, `GOG Games\Cyberpunk 2077`),
			filepath.Join(root, `Epic Games\Cyberpunk2077`),
		)
	}

	seen := make(map[string]bool)
	for _, candidate := range candidates {
		candidate = filepath.Clean(strings.Trim(strings.TrimSpace(candidate), `"`))
		key := strings.ToLower(candidate)
		if candidate == "." || seen[key] {
			continue
		}
		seen[key] = true
		if IsGameDirectory(candidate) {
			return candidate
		}
	}
	return ""
}

func IsGameDirectory(path string) bool {
	if strings.TrimSpace(path) == "" {
		return false
	}
	checks := []string{
		filepath.Join(path, "bin", "x64", "Cyberpunk2077.exe"),
		filepath.Join(path, "archive", "pc", "content", "basegame_1_engine.archive"),
	}
	for _, check := range checks {
		if _, err := os.Stat(check); err == nil {
			return true
		}
	}
	return false
}

func uninstallRegistryCandidates() []string {
	type rootPath struct {
		root registry.Key
		path string
	}
	roots := []rootPath{
		{registry.CURRENT_USER, `Software\Microsoft\Windows\CurrentVersion\Uninstall`},
		{registry.LOCAL_MACHINE, `Software\Microsoft\Windows\CurrentVersion\Uninstall`},
		{registry.LOCAL_MACHINE, `Software\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall`},
	}
	var candidates []string
	for _, location := range roots {
		key, err := registry.OpenKey(location.root, location.path, registry.ENUMERATE_SUB_KEYS|registry.QUERY_VALUE)
		if err != nil {
			continue
		}
		names, _ := key.ReadSubKeyNames(-1)
		_ = key.Close()
		for _, name := range names {
			subkey, openErr := registry.OpenKey(location.root, location.path+`\`+name, registry.QUERY_VALUE)
			if openErr != nil {
				continue
			}
			displayName, _, _ := subkey.GetStringValue("DisplayName")
			installLocation, _, _ := subkey.GetStringValue("InstallLocation")
			_ = subkey.Close()
			if strings.Contains(strings.ToLower(displayName), "cyberpunk 2077") && installLocation != "" {
				candidates = append(candidates, installLocation)
			}
		}
	}
	return candidates
}

func steamCandidates() []string {
	var steamRoots []string
	locations := []struct {
		root  registry.Key
		path  string
		value string
	}{
		{registry.CURRENT_USER, `Software\Valve\Steam`, "SteamPath"},
		{registry.LOCAL_MACHINE, `Software\WOW6432Node\Valve\Steam`, "InstallPath"},
	}
	for _, location := range locations {
		key, err := registry.OpenKey(location.root, location.path, registry.QUERY_VALUE)
		if err != nil {
			continue
		}
		value, _, _ := key.GetStringValue(location.value)
		_ = key.Close()
		if value != "" {
			steamRoots = append(steamRoots, filepath.FromSlash(value))
		}
	}

	var libraries []string
	for _, steamRoot := range steamRoots {
		libraries = append(libraries, steamRoot)
		data, err := os.ReadFile(filepath.Join(steamRoot, "steamapps", "libraryfolders.vdf"))
		if err != nil {
			continue
		}
		for _, match := range steamLibraryPathRE.FindAllStringSubmatch(string(data), -1) {
			libraries = append(libraries, filepath.FromSlash(strings.ReplaceAll(match[1], `\\`, `\`)))
		}
	}
	var candidates []string
	for _, library := range libraries {
		candidates = append(candidates, filepath.Join(library, "steamapps", "common", "Cyberpunk 2077"))
	}
	return candidates
}

func epicCandidates() []string {
	manifestDir := filepath.Join(os.Getenv("ProgramData"), "Epic", "EpicGamesLauncher", "Data", "Manifests")
	entries, err := os.ReadDir(manifestDir)
	if err != nil {
		return nil
	}
	var candidates []string
	for _, entry := range entries {
		if entry.IsDir() || !strings.EqualFold(filepath.Ext(entry.Name()), ".item") {
			continue
		}
		data, readErr := os.ReadFile(filepath.Join(manifestDir, entry.Name()))
		if readErr != nil {
			continue
		}
		var manifest struct {
			DisplayName     string `json:"DisplayName"`
			InstallLocation string `json:"InstallLocation"`
		}
		if json.Unmarshal(data, &manifest) == nil && strings.Contains(strings.ToLower(manifest.DisplayName), "cyberpunk 2077") {
			candidates = append(candidates, manifest.InstallLocation)
		}
	}
	sort.Strings(candidates)
	return candidates
}
