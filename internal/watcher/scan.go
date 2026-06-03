package watcher

import (
	"log"
	"log/slog"
	"path/filepath"
	"sync"

	"github.com/Nox1KCL/InFolderSort/internal/config"
	"github.com/Nox1KCL/InFolderSort/internal/files"
	"github.com/fsnotify/fsnotify"
)

var snlog = slog.With("module", "scanner")

func Scanner(cfg *config.Config, jobs chan<- string) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		snlog.Error("failed to initialize watcher",
			"error", err)
	}
	snlog.Info("watcher initialized")
	defer watcher.Close()

	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				if event.Has(fsnotify.Create) || event.Has(fsnotify.Write) {
					snlog.Debug("event", "event", event)
					jobs <- event.Name
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Println("error:", err)
			}
		}
	}()

	err = watcher.Add(cfg.ScanDir)
	if err != nil {
		log.Fatal(err)
	}

	// Block main goroutine forever.
	<-make(chan struct{})
}

func Worker(jobs <-chan string, wg *sync.WaitGroup, cfg *config.Config) {
	defer wg.Done()
	var workerResults []files.SortResult

	for j := range jobs {
		localSorter := files.NewSorter(cfg)
		fileName := filepath.Base(j)

		sortRes, err := localSorter.SelectiveSorting(fileName)
		if err != nil {
			snlog.Error("sorting failed",
				"error", err)
		}
		workerResults = append(workerResults, sortRes)
	}
}
