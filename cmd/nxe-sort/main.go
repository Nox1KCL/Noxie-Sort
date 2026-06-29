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

	"github.com/Nox1KCL/Noxie-Sort/internal/background"
	"github.com/Nox1KCL/Noxie-Sort/internal/config"
	"github.com/Nox1KCL/Noxie-Sort/internal/daemon"
	"github.com/Nox1KCL/Noxie-Sort/internal/logger"
	"github.com/Nox1KCL/Noxie-Sort/internal/syncutils"
	"github.com/Nox1KCL/Noxie-Sort/internal/telemetry"
	"github.com/Nox1KCL/Noxie-Sort/internal/watcher"
)

type Flags struct {
	ConfigPath string
	Daemon     string
	Background bool
	IsChild    bool
	Stop       bool
}

func (f *Flags) flagProcessing() {
	flag.StringVar(&f.ConfigPath, "config", "", "path to config file (uses embedded default if empty)")
	flag.StringVar(&f.Daemon, "daemon", "", "run as daemon")
	flag.BoolVar(&f.Background, "background", false, "run for create a child process")
	flag.BoolVar(&f.IsChild, "child", false, "run as child process")
	flag.BoolVar(&f.Stop, "stop", false, "stop processing")
	flag.Parse()
}

func main() {
	const (
		pollingTime = 2 * time.Second
		maxTries    = 5
	)
	var f Flags
	f.flagProcessing()

	if f.Stop {
		_, _ = fmt.Fprintf(os.Stderr, "stop processing\n")
		os.Exit(0)
	}

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
	service, err := daemon.NewService()
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "create service: %v\n", err)
		os.Exit(1)
	}

	switch f.Daemon {
	case "install":
		err := service.LaunchingDaemon()
		if err != nil {
			mlog.Error("failed to install daemon", "error", err)
			os.Exit(1)
		}
		mlog.Info("daemon installed successfully")
		os.Exit(0)
	case "uninstall":
		err := service.ClosingDaemon()
		if err != nil {
			mlog.Error("failed to uninstall daemon", "error", err)
			os.Exit(1)
		}
		mlog.Info("daemon uninstalled successfully")
		os.Exit(0)
	}

	if f.Background {
		fileLock, err := background.IsChildRunning()
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "background process is running: %v\n", err)
			os.Exit(1)
		}
		fileLock.Close()

		if err := daemon.IsWorking(); err != nil {
			mlog.Warn("daemon is not properly configured (background process will still start)", "error", err)
		}

		childArgs := []string{"--child", "--config", f.ConfigPath}
		err = background.RunInBackground(childArgs)
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "background run failed: %v\n", err)
			os.Exit(1)
		}
		mlog.Info("background run",
			"args", childArgs)
		return
	}

	if f.IsChild {
		fileLock, err := background.IsChildRunning()
		if err != nil {
			mlog.Error("failed to acquire lock",
				"error", err)
			os.Exit(1)
		}
		defer fileLock.Close()
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	service.OpenServer(ctx)

	shutdown, observer, err := telemetry.NewTelemetry()
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "telemetry initialization failed: %v\n", err)
		os.Exit(1)
	}
	defer shutdown(ctx)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	wg := sync.WaitGroup{}
	jobs := make(chan string, 100)

	for range 3 * len(cfg.ScanDirs) {
		wg.Add(1)
		go watcher.Worker(ctx, observer, jobs, &wg, cfg, pollingTime, maxTries)
	}

	var scannerWg syncutils.MyWaitGroup
	scannerWg.Go(func() {
		watcher.Scanner(ctx, observer, cfg, jobs)
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
