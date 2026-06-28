// Package watcher provides functionality for scanning directories and watching for file changes.
package watcher

import (
	"context"
	"log/slog"
	"path/filepath"
	"sync"
	"time"

	"github.com/Nox1KCL/Noxie-Sort/internal/config"
	"github.com/Nox1KCL/Noxie-Sort/internal/files"
	"github.com/Nox1KCL/Noxie-Sort/internal/telemetry"
	"github.com/fsnotify/fsnotify"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

var snlog = slog.With("module", "scanner")

func Scanner(ctx context.Context, obs *telemetry.Observe, cfg *config.Config, jobs chan<- string) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		snlog.Error("failed to initialize watcher",
			"error", err)
		return
	}
	defer watcher.Close()
	processCtx, span := obs.Tracer.Start(ctx, "Scanner.ProcessJob")
	snlog.InfoContext(processCtx, "watcher initialized")

	for _, dir := range cfg.ScanDirs {
		err = watcher.Add(dir)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "adding directory "+dir)
			obs.ErrCounter.Add(processCtx, 1)

			snlog.ErrorContext(processCtx, "failed to add watch directory",
				"error", err,
				"dir", dir)
			return
		}
	}

	for {
		select {
		case <-ctx.Done():
			span.SetStatus(codes.Error, "context canceled")
			snlog.WarnContext(processCtx, "context done")
			span.End()
			return

		case event, ok := <-watcher.Events:
			if !ok {
				span.SetStatus(codes.Error, "watcher.Events channel closed")
				span.End()
				return
			}
			if event.Has(fsnotify.Create) || event.Has(fsnotify.Write) {
				span.AddEvent("Event created", trace.WithAttributes(
					attribute.String("filename", event.Name),
				))
				obs.EventCounter.Add(processCtx, 1)
				snlog.DebugContext(processCtx, "event", "event", event)

				fileName := filepath.Base(event.Name)
				if isValid := files.FileExtValidate(fileName); isValid {
					select {
					case jobs <- event.Name:
						span.AddEvent(event.Name)
					case <-ctx.Done():
						span.SetStatus(codes.Error, "context canceled")
						snlog.WarnContext(processCtx, "context done")
						span.End()
						return
					}
				}
			}

		case err, ok := <-watcher.Errors:
			if !ok {
				span.SetStatus(codes.Error, "watcher.Errors channel closed")
				span.End()
				return
			}
			snlog.ErrorContext(processCtx, "watcher error",
				"error", err)
		}
	}
}

func Worker(ctx context.Context, obs *telemetry.Observe, jobs <-chan string, wg *sync.WaitGroup, cfg *config.Config, waitInterval time.Duration, maxRetries int) {
	defer wg.Done()

	for j := range jobs {
		processCtx, span := obs.Tracer.Start(ctx, "Worker.ProcessJob", trace.WithAttributes(
			attribute.String("file.name", j),
		))
		obs.SCounter.Add(processCtx, 1)
		startTime := time.Now()

		err := files.FileSizePolling(j, waitInterval, maxRetries)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "file size polling failed")
			obs.ErrCounter.Add(processCtx, 1)

			snlog.DebugContext(processCtx, "file size polling failed",
				"error", err,
				"file", j)

			span.End()
			continue
		}
		if files.IsFileLocked(j) {
			snlog.DebugContext(processCtx, "file is locked, skipping", "file", j)
			span.End()
			continue
		}
		localSorter := files.NewSorter(cfg)

		_, err = localSorter.SelectiveSorting(processCtx, obs, j)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "sorting failed")
			obs.ErrCounter.Add(processCtx, 1)
			snlog.ErrorContext(processCtx, "sorting failed",
				"error", err)
		}
		span.SetStatus(codes.Ok, "sorted "+j)
		obs.SDuration.Record(processCtx, float64(time.Since(startTime).Milliseconds()))
		span.End()
	}
}
