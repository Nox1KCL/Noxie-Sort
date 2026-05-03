package config

import (
	_ "embed"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/Nox1KCL/InFolderSort/internal/logger"
	"github.com/pelletier/go-toml/v2"
)

var clog = slog.With("module", "config")

//go:embed config.toml
var defaultConfig []byte

type Config struct {
	ScanDir       string                `toml:"scan_dir"`
	Rules         map[string]FolderRule `toml:"rules"`
	InvertedRules map[string]string
	Logger        logger.LumberConfig `toml:"logger"`
}

type FolderRule struct {
	TargetPath string   `toml:"target_path"`
	Extensions []string `toml:"extensions"`
}

func GetConfig(path string) (*Config, error) {
	var doc []byte
	var err error

	if path != "" {
		doc, err = os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("reading config file %q: %w", path, err)
		}
	} else {
		clog.Info("no config file provided, using default config")
		doc = defaultConfig
	}

	var cfg Config
	if err := toml.Unmarshal(doc, &cfg); err != nil {
		return nil, fmt.Errorf("reading toml doc %q: %w", path, err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("validating config: %w", err)
	}

	cfg.Prepare()

	return &cfg, nil
}

func (cfg *Config) Prepare() {
	cfg.ScanDir = filepath.Clean(os.ExpandEnv(cfg.ScanDir))

	cfg.InvertConfig()
}

func (cfg *Config) InvertConfig() {
	if cfg.InvertedRules != nil {
		return
	}
	cfg.InvertedRules = make(map[string]string)

	for _, folderRule := range cfg.Rules {
		expandedPath := os.ExpandEnv(folderRule.TargetPath)
		finalPath := filepath.Clean(expandedPath)
		for _, ext := range folderRule.Extensions {
			cfg.InvertedRules[ext] = finalPath
		}
	}
}

func (cfg *Config) GetTargetPath(fileExt string) (string, error) {
	targetPath, ok := cfg.InvertedRules[fileExt]
	if ok {
		return targetPath, nil
	}
	return "", fmt.Errorf("ext isn't in config: %s", fileExt)
}

func (cfg *Config) Validate() error {
	seenExtensions := make(map[string]string)
	var conflicts []error

	for folderName, folderRule := range cfg.Rules {
		for _, ext := range folderRule.Extensions {
			ext = strings.ToLower(strings.TrimSpace(ext))

			if ext == "" {
				clog.Warn("empty extension in config",
					"folder", folderName)
				conflicts = append(conflicts, fmt.Errorf("empty extension in %s", folderName))
				continue
			}

			if !strings.HasPrefix(ext, ".") {
				conflicts = append(conflicts, fmt.Errorf("missing dot in extension %s in %s", ext, folderName))
				continue
			}

			if firstDebut, exists := seenExtensions[ext]; exists {
				clog.Warn("duplicate extension",
					"folder", folderName,
					"ext", ext,
					"firstDebut", firstDebut)
				conflicts = append(conflicts, fmt.Errorf("duplicate extension: %s | Seen it in %s and %s", ext, firstDebut, folderName))
				continue
			} else {
				seenExtensions[ext] = folderName
			}
		}
	}

	if len(conflicts) != 0 {
		return errors.Join(conflicts...)
	}
	return nil
}
