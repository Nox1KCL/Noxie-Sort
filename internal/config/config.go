// Package config provides configuration for the InFolderSort application.
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

//go:embed config.toml
var defaultConfig []byte

var clog = slog.With("module", "config")

type Config struct {
	ScanDir       string                `toml:"scan_dir"`
	LogsDir       string                `toml:"logs_dir"`
	Workers       int                   `toml:"workers"`
	Rules         map[string]FolderRule `toml:"rules"`
	InvertedRules map[string]string
	Logger        logger.LumberConfig   `toml:"logger"`
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
	if err := cfg.Prepare(); err != nil {
		return nil, fmt.Errorf("validating config: %w", err)
	}

	return &cfg, nil
}

func (cfg *Config) Prepare() error {
	if err := cfg.ConfigExtValidate(); err != nil {
		return err
	}

	cfg.ScanDir = filepath.Clean(os.ExpandEnv(cfg.ScanDir))

    if cfg.LogsDir != "" {
        if !filepath.IsAbs(cfg.LogsDir) {
            cwd, err := os.Getwd()
            if err != nil {
                return fmt.Errorf("getting working directory: %w", err)
            }
            cfg.LogsDir = filepath.Join(cwd, cfg.LogsDir)
        }
    } else {
        cfg.LogsDir = "logs" // fallback
    }

	cfg.InvertConfig()
	return nil
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

func (cfg *Config) ConfigExtValidate() error {
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

func FindConfig(flagPath string) string {
	if flagPath != "" {
		return flagPath
	}
	//IFS_CONFIG_PATH=/home/user/my.toml ./inFolderSort
	//
	//# Або експортувати в сесію:
	//export IFS_CONFIG_PATH=/home/user/my.toml ./inFolderSort
	if envPath := os.Getenv("IFS_CONFIG_PATH"); envPath != "" {
		return envPath
	}

	defaultPaths := []string{
		"config.toml",
		os.ExpandEnv("$HOME/InFolderSort/config.toml"),
		"/etc/InFolderSort/config.toml",
	}

	configDir, err := os.UserConfigDir()
	if err == nil {
	    defaultPaths = append(defaultPaths, filepath.Join(configDir, "InFolderSort", "config.toml"))
	}

	for _, p := range defaultPaths {
		defaultPath := os.ExpandEnv(p)
		if _, err := os.Stat(defaultPath); err == nil {
			return defaultPath
		}
	}
	return ""
}
