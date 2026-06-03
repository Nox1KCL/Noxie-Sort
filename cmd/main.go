package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"
	"sync"

	"github.com/Nox1KCL/InFolderSort/internal/config"
	"github.com/Nox1KCL/InFolderSort/internal/logger"
	"github.com/Nox1KCL/InFolderSort/internal/watcher"
)

func main() {
	var (
		configPath string
		isDaemon   bool
	)
	jobs := make(chan string, 100)

	// TODO: ExcaliDraw план
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
		slog.LevelDebug: "logs/debug.log",
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

	//sorter := files.NewSorter(cfg)
	wg := sync.WaitGroup{}
	wg.Add(3)

	for i := 0; i < 3; i++ {
		go watcher.Worker(jobs, &wg, cfg)
	}

	watcher.Scanner(cfg, jobs)
	wg.Wait()

	// Start tui
	//err := tui.Core(cfg, sorter)
	//if err != nil {
	//	mlog.Error("starting tui",
	//		"error", err,
	//		"config_rules", len(cfg.Rules),
	//	)
	//	_, _ = fmt.Fprintf(os.Stderr, "running application: %v\n", err)
	//	os.Exit(1)
	//}
}
