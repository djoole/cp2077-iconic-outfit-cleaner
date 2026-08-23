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
	ctx         context.Context
	scan        *scanResult
	scanner     *Scanner
	configPath  string
	lastGameDir string
}

func NewApp() *App {
	app := &App{scanner: NewScanner()}
	if configDir, err := os.UserConfigDir(); err == nil {
		app.configPath = filepath.Join(configDir, "CP2077 Iconic Outfit Cleaner", "config.json")
		if data, readErr := os.ReadFile(app.configPath); readErr == nil {
			var cfg struct {
				GameDir string `json:"gameDir"`
			}
			if json.Unmarshal(data, &cfg) == nil && IsGameDirectory(cfg.GameDir) {
				app.lastGameDir = cfg.GameDir
			}
		}
	}
	return app
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

func (a *App) DetectGameDir() string {
	if IsGameDirectory(a.lastGameDir) {
		return a.lastGameDir
	}
	return DetectGameDirectory()
}

func (a *App) PickGameDir() string {
	dir, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Select the Cyberpunk 2077 game folder",
	})
	if err != nil {
		return ""
	}
	return dir
}

func (a *App) Scan(gameDir string) (ScanResultDTO, error) {
	gameDir = strings.TrimSpace(gameDir)
	if gameDir == "" {
		return ScanResultDTO{}, fmt.Errorf("select the game folder first")
	}
	result, err := a.scanner.Scan(gameDir)
	if err != nil {
		return ScanResultDTO{}, err
	}
	a.scan = result
	a.lastGameDir = result.gameDir
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
		folder = filepath.Join(a.scan.gameDir, "r6", "scripts")
	}
	return exec.Command("explorer.exe", folder).Start()
}

func (a *App) saveConfig() {
	if a.configPath == "" || a.lastGameDir == "" {
		return
	}
	data, err := json.MarshalIndent(struct {
		GameDir string `json:"gameDir"`
	}{a.lastGameDir}, "", "  ")
	if err != nil {
		return
	}
	if os.MkdirAll(filepath.Dir(a.configPath), 0o755) == nil {
		_ = os.WriteFile(a.configPath, data, 0o644)
	}
}
