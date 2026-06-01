package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"

	"github.com/Nox1KCL/InFolderSort/internal/config"
	"github.com/Nox1KCL/InFolderSort/internal/logger"
	"github.com/Nox1KCL/InFolderSort/internal/tui"
)

func main() {
	var (
		configPath string
		isDaemon   bool
	)

	flag.StringVar(&configPath, "config", "", "path to config file (uses embedded default if empty)")
	flag.BoolVar(&isDaemon, "daemon", false, "run as daemon")
	flag.Parse()

	cfg, cfgErr := config.GetConfig(configPath)
	if cfgErr != nil {
		_, _ = fmt.Fprintf(os.Stderr, "get configuration file: %v\n", cfgErr)
		os.Exit(1)
	}

	levels := map[slog.Level]string{
		slog.LevelInfo:  "logs/info.log",
		slog.LevelError: "logs/error.log",
		slog.LevelWarn:  "logs/warn.log",
	}
	handler, logErr := logger.GetHandler(&cfg.Logger, levels)
	if logErr != nil {
		_, _ = fmt.Fprintf(os.Stderr, "creating logger: %v\n", logErr)
		os.Exit(1)
	}
	slog.SetDefault(slog.New(handler))
	mlog := slog.With("module", "main")

	mlog.Info("configuration initialized",
		"config_path", configPath,
		"rules_count", len(cfg.Rules),
	)

	// Start tui
	err := tui.Core(cfg)
	if err != nil {
		mlog.Error("starting tui",
			"error", err,
			"config_rules", len(cfg.Rules),
		)
		_, _ = fmt.Fprintf(os.Stderr, "running application: %v\n", err)
		os.Exit(1)
	}
}
