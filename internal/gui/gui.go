//go:build gui
// +build gui

package gui

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"noisyzip/internal/core"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type EncryptConfig struct {
	SrcDir              string `json:"srcDir"`
	OutZip              string `json:"outZip"`
	Compression         string `json:"compression"`
	Encoding            string `json:"encoding"`
	OverwriteCentralDir bool   `json:"overwriteCentralDir"`
	CommentSize         int    `json:"commentSize"`
	FixedTime           bool   `json:"fixedTime"`
	NoiseFiles          int    `json:"noiseFiles"`
	NoiseSize           int    `json:"noiseSize"`
	Level               int    `json:"level"`
	Strategy            string `json:"strategy"`
	DictSize            int    `json:"dictSize"`
	Workers             int    `json:"workers"`
	Seed                string `json:"seed"`
	IncludeHidden       bool   `json:"includeHidden"`
}

type EncryptResult struct {
	Total  int    `json:"total"`
	OutZip string `json:"outZip"`
}

type RecoverConfig struct {
	InZip         string `json:"inZip"`
	OutZip        string `json:"outZip"`
	Compression   string `json:"compression"`
	Encoding      string `json:"encoding"`
	Level         int    `json:"level"`
	Strategy      string `json:"strategy"`
	DictSize      int    `json:"dictSize"`
	Workers       int    `json:"workers"`
	Seed          string `json:"seed"`
	IncludeHidden bool   `json:"includeHidden"`
}

type RecoverResult struct {
	Recovered int `json:"recovered"`
	Rebuilt   int `json:"rebuilt"`
}

type App struct {
	ctx     context.Context
	running bool
	mu      sync.Mutex
}

func NewApp() *App {
	return &App{}
}

func StartupHandler(app *App) func(context.Context) {
	return app.startup
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

func (a *App) SelectSourceDir() (string, error) {
	if a.ctx == nil {
		return "", errors.New("app not ready")
	}
	path, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Select input directory",
	})
	if err != nil {
		return "", err
	}
	return path, nil
}

func (a *App) SelectInputZip() (string, error) {
	if a.ctx == nil {
		return "", errors.New("app not ready")
	}
	path, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Select input ZIP",
		Filters: []runtime.FileFilter{{
			DisplayName: "ZIP files",
			Pattern:     "*.zip",
		}},
	})
	if err != nil {
		return "", err
	}
	return path, nil
}

func (a *App) SelectOutputDir() (string, error) {
	if a.ctx == nil {
		return "", errors.New("app not ready")
	}
	path, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Select output directory",
	})
	if err != nil {
		return "", err
	}
	return path, nil
}

func (a *App) SelectOutputZip() (string, error) {
	if a.ctx == nil {
		return "", errors.New("app not ready")
	}
	path, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:           "Save ZIP archive",
		DefaultFilename: "archive.zip",
		Filters: []runtime.FileFilter{{
			DisplayName: "ZIP files",
			Pattern:     "*.zip",
		}},
	})
	if err != nil {
		return "", err
	}
	if path == "" {
		return "", nil
	}
	if !strings.HasSuffix(strings.ToLower(path), ".zip") {
		path += ".zip"
	}
	return path, nil
}

func (a *App) RunEncrypt(uiCfg EncryptConfig) (EncryptResult, error) {
	if a.ctx == nil {
		return EncryptResult{}, errors.New("app not ready")
	}
	a.mu.Lock()
	if a.running {
		a.mu.Unlock()
		return EncryptResult{}, errors.New("operation already in progress")
	}
	a.running = true
	a.mu.Unlock()
	defer func() {
		a.mu.Lock()
		a.running = false
		a.mu.Unlock()
	}()

	src := strings.TrimSpace(uiCfg.SrcDir)
	outZip := strings.TrimSpace(uiCfg.OutZip)
	if src == "" || outZip == "" {
		return EncryptResult{}, errors.New("please choose input directory and output ZIP")
	}
	info, err := os.Stat(src)
	if err != nil || !info.IsDir() {
		return EncryptResult{}, errors.New("input directory is invalid")
	}
	if !strings.HasSuffix(strings.ToLower(outZip), ".zip") {
		outZip += ".zip"
	}

	cfg := core.Config{
		SrcDir:              filepath.Clean(src),
		OutZip:              filepath.Clean(outZip),
		Compression:         uiCfg.Compression,
		Encoding:            uiCfg.Encoding,
		OverwriteCentralDir: uiCfg.OverwriteCentralDir,
		CommentSize:         uiCfg.CommentSize,
		FixedTime:           uiCfg.FixedTime,
		NoiseFiles:          uiCfg.NoiseFiles,
		NoiseSize:           uiCfg.NoiseSize,
		Level:               uiCfg.Level,
		Strategy:            uiCfg.Strategy,
		DictSize:            uiCfg.DictSize,
		Workers:             uiCfg.Workers,
	}

	seedText := strings.TrimSpace(uiCfg.Seed)
	if seedText != "" {
		seedVal, err := strconv.ParseInt(seedText, 10, 64)
		if err != nil {
			return EncryptResult{}, errors.New("seed must be an integer")
		}
		cfg.Seed = seedVal
		cfg.HasSeed = true
	}
	cfg.IncludeHidden = uiCfg.IncludeHidden

	logCb := func(msg string) {
		runtime.EventsEmit(a.ctx, "encrypt:log", msg)
	}
	progressCb := func(done, total int, name string) {
		runtime.EventsEmit(a.ctx, "encrypt:progress", map[string]any{
			"done":  done,
			"total": total,
			"name":  name,
		})
	}

	total, err := core.RunEncrypt(cfg, progressCb, logCb)
	if err != nil {
		return EncryptResult{}, fmt.Errorf("run encrypt: %w", err)
	}
	return EncryptResult{Total: total, OutZip: outZip}, nil
}

func (a *App) RunRecover(uiCfg RecoverConfig) (RecoverResult, error) {
	if a.ctx == nil {
		return RecoverResult{}, errors.New("app not ready")
	}
	a.mu.Lock()
	if a.running {
		a.mu.Unlock()
		return RecoverResult{}, errors.New("operation already in progress")
	}
	a.running = true
	a.mu.Unlock()
	defer func() {
		a.mu.Lock()
		a.running = false
		a.mu.Unlock()
	}()

	inZip := strings.TrimSpace(uiCfg.InZip)
	outZip := strings.TrimSpace(uiCfg.OutZip)
	if inZip == "" || outZip == "" {
		return RecoverResult{}, errors.New("please choose input ZIP and output ZIP")
	}
	if !strings.HasSuffix(strings.ToLower(outZip), ".zip") {
		outZip += ".zip"
	}

	logCb := func(msg string) {
		runtime.EventsEmit(a.ctx, "recover:log", msg)
	}
	progressCb := func(done, total int, name string) {
		runtime.EventsEmit(a.ctx, "recover:progress", map[string]any{
			"done":  done,
			"total": total,
			"name":  name,
		})
	}

	tmpDir, err := os.MkdirTemp("", "zip-recover-*")
	if err != nil {
		return RecoverResult{}, fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	recovered, err := core.RecoverZip(filepath.Clean(inZip), tmpDir, progressCb, logCb)
	if err != nil {
		return RecoverResult{}, fmt.Errorf("recover zip: %w", err)
	}

	cfg := core.Config{
		SrcDir:              filepath.Clean(tmpDir),
		OutZip:              filepath.Clean(outZip),
		Compression:         uiCfg.Compression,
		Encoding:            uiCfg.Encoding,
		OverwriteCentralDir: false,
		CommentSize:         0,
		FixedTime:           false,
		NoiseFiles:          0,
		NoiseSize:           0,
		Level:               uiCfg.Level,
		Strategy:            uiCfg.Strategy,
		DictSize:            uiCfg.DictSize,
		Workers:             uiCfg.Workers,
		IncludeHidden:       uiCfg.IncludeHidden,
	}

	seedText := strings.TrimSpace(uiCfg.Seed)
	if seedText != "" {
		seedVal, err := strconv.ParseInt(seedText, 10, 64)
		if err != nil {
			return RecoverResult{}, errors.New("seed must be an integer")
		}
		cfg.Seed = seedVal
		cfg.HasSeed = true
	}

	rebuilt, err := core.RunEncrypt(cfg, nil, nil)
	if err != nil {
		return RecoverResult{}, fmt.Errorf("build zip: %w", err)
	}

	return RecoverResult{Recovered: recovered, Rebuilt: rebuilt}, nil
}
