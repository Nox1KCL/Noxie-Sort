package watcher

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/Nox1KCL/Noxie-Sort/internal/config"
	"github.com/Nox1KCL/Noxie-Sort/internal/telemetry"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric/noop"
	"log/slog"
)

func getDummyObserver() *telemetry.Observe {
	mp := noop.NewMeterProvider()
	meter := mp.Meter("test")
	sc, _ := meter.Int64Counter("test")
	sd, _ := meter.Float64Histogram("test")
	return &telemetry.Observe{
		Tracer:       otel.Tracer("test"),
		Meter:        meter,
		Logger:       slog.Default(),
		SCounter:     sc,
		SDuration:    sd,
		ErrCounter:   sc,
		EventCounter: sc,
		BytesMoved:   sc,
		FextCounter:  sc,
	}
}

func TestScanner_ContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	obs := getDummyObserver()
	cfg := &config.Config{ScanDirs: []string{os.TempDir()}}
	jobs := make(chan string, 1)

	// We cancel immediately
	cancel()

	// Should return immediately
	Scanner(ctx, obs, cfg, jobs)
}

func TestWorker_ContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	obs := getDummyObserver()
	cfg := &config.Config{}
	
	jobs := make(chan string, 1)
	var wg sync.WaitGroup
	wg.Add(1)

	// Since Worker ranges over jobs, it exits when jobs channel is closed.
	close(jobs)
	
	Worker(ctx, obs, jobs, &wg, cfg, time.Millisecond, 1)
	
	// If it didn't panic and returned, it's fine.
	cancel()
}

func TestWorker_FileProcessing(t *testing.T) {
	ctx := context.Background()
	obs := getDummyObserver()

	tempDir := t.TempDir()
	sourceDir := filepath.Join(tempDir, "source")
	targetDir := filepath.Join(tempDir, "target")
	os.MkdirAll(sourceDir, 0755)
	os.MkdirAll(targetDir, 0755)

	cfg := &config.Config{
		ScanDir: sourceDir,
		Rules: map[string]config.FolderRule{
			"Target": {
				TargetDir:  targetDir,
				Extensions: []string{".txt"},
			},
		},
	}
	cfg.InvertConfig()

	testFile := filepath.Join(sourceDir, "test.txt")
	os.WriteFile(testFile, []byte("hello"), 0644)

	jobs := make(chan string, 1)
	jobs <- testFile
	close(jobs)

	var wg sync.WaitGroup
	wg.Add(1)

	Worker(ctx, obs, jobs, &wg, cfg, time.Millisecond, 1)

	// Check if file was moved
	_, err := os.Stat(testFile)
	if !os.IsNotExist(err) {
		t.Errorf("expected test file to be moved, but it still exists: %v", err)
	}
}
