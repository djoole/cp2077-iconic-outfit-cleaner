package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type App struct {
	ctx            context.Context
	scan           *scanResult
	scanner        *Scanner
	configPath     string
	lastScanRoot   string
	lastOutputRoot string
}

func NewApp() *App {
	app := &App{scanner: NewScanner()}
	if configDir, err := os.UserConfigDir(); err == nil {
		app.configPath = filepath.Join(configDir, "CP2077 Iconic Outfit Cleaner", "config.json")
		if data, readErr := os.ReadFile(app.configPath); readErr == nil {
			var cfg struct {
				GameDir    string `json:"gameDir"`
				ScanRoot   string `json:"scanRoot"`
				OutputRoot string `json:"outputRoot"`
			}
			if json.Unmarshal(data, &cfg) == nil {
				if cfg.ScanRoot == "" {
					cfg.ScanRoot = cfg.GameDir
				}
				if isDirectory(cfg.ScanRoot) {
					app.lastScanRoot = cfg.ScanRoot
				}
				if isDirectory(cfg.OutputRoot) {
					app.lastOutputRoot = cfg.OutputRoot
				} else if IsGameDirectory(cfg.GameDir) {
					app.lastOutputRoot = cfg.GameDir
				}
			}
		}
	}
	return app
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

func (a *App) DetectPaths() PathDefaultsDTO {
	gameDir := DetectGameDirectory()
	scanRoot := a.lastScanRoot
	if !isDirectory(scanRoot) {
		scanRoot = gameDir
	}
	outputRoot := a.lastOutputRoot
	if !isDirectory(outputRoot) {
		outputRoot = gameDir
	}
	return PathDefaultsDTO{ScanRoot: scanRoot, OutputRoot: outputRoot}
}

func (a *App) PickScanRoot(current string) string {
	dir, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title:            "Select the folder to scan recursively",
		DefaultDirectory: existingDirectory(current),
	})
	if err != nil {
		return ""
	}
	return dir
}

func (a *App) PickOutputRoot(current string) string {
	dir, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title:                "Select the patch destination (game or mod root)",
		DefaultDirectory:     existingDirectory(current),
		CanCreateDirectories: true,
	})
	if err != nil {
		return ""
	}
	return dir
}

func (a *App) Scan(scanRoot, outputRoot string) (ScanResultDTO, error) {
	scanRoot = strings.TrimSpace(scanRoot)
	outputRoot = strings.TrimSpace(outputRoot)
	if scanRoot == "" {
		return ScanResultDTO{}, fmt.Errorf("select a scan folder first")
	}
	if outputRoot == "" {
		return ScanResultDTO{}, fmt.Errorf("select a patch destination first")
	}
	result, err := a.scanner.Scan(scanRoot, outputRoot)
	if err != nil {
		return ScanResultDTO{}, err
	}
	a.scan = result
	a.lastScanRoot = result.scanRoot
	a.lastOutputRoot = result.outputRoot
	a.saveConfig()
	return result.DTO(), nil
}

func (a *App) GeneratePatch(selectedModIDs []string) (GenerateResultDTO, error) {
	if a.scan == nil {
		return GenerateResultDTO{}, fmt.Errorf("run a scan first")
	}
	if len(selectedModIDs) == 0 {
		return GenerateResultDTO{}, fmt.Errorf("select at least one mod")
	}
	return a.scanner.Generate(a.scan, selectedModIDs)
}

func (a *App) OpenPatchFolder() error {
	if a.scan == nil {
		return fmt.Errorf("no scan is available")
	}
	folder := filepath.Dir(a.scan.patchPath)
	if _, err := os.Stat(folder); err != nil {
		folder = a.scan.outputRoot
	}
	return exec.Command("explorer.exe", folder).Start()
}

func (a *App) saveConfig() {
	if a.configPath == "" || a.lastScanRoot == "" || a.lastOutputRoot == "" {
		return
	}
	data, err := json.MarshalIndent(struct {
		ScanRoot   string `json:"scanRoot"`
		OutputRoot string `json:"outputRoot"`
	}{a.lastScanRoot, a.lastOutputRoot}, "", "  ")
	if err != nil {
		return
	}
	if os.MkdirAll(filepath.Dir(a.configPath), 0o755) == nil {
		_ = os.WriteFile(a.configPath, data, 0o644)
	}
}

func isDirectory(path string) bool {
	info, err := os.Stat(strings.TrimSpace(path))
	return err == nil && info.IsDir()
}

func existingDirectory(path string) string {
	path = strings.TrimSpace(path)
	for path != "" && !isDirectory(path) {
		parent := filepath.Dir(path)
		if parent == path {
			return ""
		}
		path = parent
	}
	return path
}
