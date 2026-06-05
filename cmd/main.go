package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/Nox1KCL/InFolderSort/internal/config"
	"github.com/Nox1KCL/InFolderSort/internal/logger"
	"github.com/Nox1KCL/InFolderSort/internal/watcher"
)

func main() {
	var (
		configPath string
		isDaemon   bool
	)
	flag.StringVar(&configPath, "config", "", "path to config file (uses embedded default if empty)")
	flag.BoolVar(&isDaemon, "daemon", false, "run as daemon")
	flag.Parse()

	path := config.FindConfig(configPath)
	cfg, cfgErr := config.GetConfig(path)
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

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	wg := sync.WaitGroup{}
	numWorkers := 3
	jobs := make(chan string, 100)

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go watcher.Worker(jobs, &wg, cfg)
	}

	var scannerWg sync.WaitGroup
	scannerWg.Add(1)
	go func() {
		defer scannerWg.Done()
		watcher.Scanner(ctx, cfg, jobs)
	}()

	sig := <-sigChan
	mlog.Warn("received stop signal, shutting down gracefully",
		"signal", sig)

	cancel()
	scannerWg.Wait()

	close(jobs)
	wg.Wait()

	mlog.Info("graceful shutdown complete")

	//sorter := files.NewSorter(cfg)
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
