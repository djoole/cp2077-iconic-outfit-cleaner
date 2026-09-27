package main

type ItemDTO struct {
	RecordID        string `json:"recordId"`
	SourceFile      string `json:"sourceFile"`
	OriginalQuality string `json:"originalQuality"`
	Template        bool   `json:"template"`
}

type ModDTO struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	SourceLabel string    `json:"sourceLabel"`
	ItemCount   int       `json:"itemCount"`
	FileCount   int       `json:"fileCount"`
	Items       []ItemDTO `json:"items"`
}

type ScanResultDTO struct {
	ScanRoot      string   `json:"scanRoot"`
	OutputRoot    string   `json:"outputRoot"`
	Mods          []ModDTO `json:"mods"`
	TotalItems    int      `json:"totalItems"`
	ScannedFiles  int      `json:"scannedFiles"`
	TweaksFolders int      `json:"tweaksFolders"`
	UsingVortex   bool     `json:"usingVortex"`
	PatchPath     string   `json:"patchPath"`
	Warnings      []string `json:"warnings"`
}

type PathDefaultsDTO struct {
	ScanRoot   string `json:"scanRoot"`
	OutputRoot string `json:"outputRoot"`
}

type GenerateResultDTO struct {
	PatchPath    string `json:"patchPath"`
	ModCount     int    `json:"modCount"`
	ItemCount    int    `json:"itemCount"`
	BackupPath   string `json:"backupPath"`
	PatchSummary string `json:"patchSummary"`
}
