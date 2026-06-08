// Package watcher provides functionality for scanning directories and watching for file changes.
package watcher

import (
	"context"
	"log/slog"
	"path/filepath"
	"sync"

	"github.com/Nox1KCL/InFolderSort/internal/config"
	"github.com/Nox1KCL/InFolderSort/internal/files"
	"github.com/fsnotify/fsnotify"
)

var snlog = slog.With("module", "scanner")

func Scanner(ctx context.Context, cfg *config.Config, jobs chan<- string) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		snlog.Error("failed to initialize watcher",
			"error", err)
		return
	}
	defer watcher.Close()
	snlog.Info("watcher initialized")

	err = watcher.Add(cfg.ScanDir)
	if err != nil {
		snlog.Error("failed to add watch directory",
			"error", err,
			"dir", cfg.ScanDir)
		return
	}

	for {
		select {
		case <-ctx.Done():
			snlog.Warn("context done")
			return

		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			if event.Has(fsnotify.Create) || event.Has(fsnotify.Write) {
				snlog.Debug("event", "event", event)
				select {
				case jobs <- event.Name:
				case <-ctx.Done():
					return
				}
			}

		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			snlog.Error("watcher error",
				"error", err)
		}
	}
}

func Worker(jobs <-chan string, wg *sync.WaitGroup, cfg *config.Config) {
	defer wg.Done()

	//TODO: Size validate
	for j := range jobs {
		localSorter := files.NewSorter(cfg)
		fileName := filepath.Base(j)
//workerResults = append(workerResults, sortRes)
		_, err := localSorter.SelectiveSorting(fileName)
		if err != nil {
			snlog.Error("sorting failed",
				"error", err)
		}

	}
}
