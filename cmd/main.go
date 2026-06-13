package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/Nox1KCL/InFolderSort/internal/background"
	"github.com/Nox1KCL/InFolderSort/internal/config"
	"github.com/Nox1KCL/InFolderSort/internal/logger"
	"github.com/Nox1KCL/InFolderSort/internal/syncutils"
	"github.com/Nox1KCL/InFolderSort/internal/watcher"
)

type Flags struct {
	ConfigPath string
	IsDaemon   bool
	Background bool
	IsChild    bool
}

func (f *Flags) flagProcessing() {
	flag.StringVar(&f.ConfigPath, "config", "", "path to config file (uses embedded default if empty)")
	flag.BoolVar(&f.IsDaemon, "daemon", false, "run as daemon")
	flag.BoolVar(&f.Background, "background", false, "run as background")
	flag.BoolVar(&f.IsChild, "child", false, "run as child")
	flag.Parse()
}

func main() {
	const (
		pollingTime = 2 * time.Second
		maxTries    = 5
	)
	var f Flags
	f.flagProcessing()

	foundPath := config.FindConfig(f.ConfigPath)
	cfg, cfgErr := config.GetConfig(foundPath)
	if cfgErr != nil {
		_, _ = fmt.Fprintf(os.Stderr, "get configuration file: %v\n", cfgErr)
		os.Exit(1)
	}

	levels := map[slog.Level]string{
		slog.LevelInfo:  filepath.Join(cfg.LogsDir, "info.log"),
		slog.LevelDebug: filepath.Join(cfg.LogsDir, "debug.log"),
		slog.LevelWarn:  filepath.Join(cfg.LogsDir, "warn.log"),
		slog.LevelError: filepath.Join(cfg.LogsDir, "error.log"),
	}
	handler, logErr := logger.GetHandler(&cfg.Logger, levels)
	if logErr != nil {
		_, _ = fmt.Fprintf(os.Stderr, "creating logger: %v\n", logErr)
		os.Exit(1)
	}
	slog.SetDefault(slog.New(handler))
	mlog := slog.With("module", "main")

	if foundPath != "" {
		mlog.Info("config file found", "path", foundPath)
	} else {
		mlog.Info("using embedded default config")
	}

	mlog.Info("configuration initialized",
		"config_path", foundPath,
		"rules_count", len(cfg.Rules),
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	wg := sync.WaitGroup{}
	jobs := make(chan string, 100)

	for range 1 * len(cfg.ScanDirs) {
		wg.Add(1)
		go watcher.Worker(jobs, &wg, cfg, pollingTime, maxTries)
	}

	var scannerWg syncutils.MyWaitGroup
	scannerWg.Go(func() {
		watcher.Scanner(ctx, cfg, jobs)
	})

	sig := <-sigChan
	mlog.Warn("received stop signal, shutting down gracefully",
		"signal", sig)

	cancel()
	scannerWg.Wait()

	close(jobs)
	wg.Wait()

	mlog.Info("graceful shutdown complete")
}
