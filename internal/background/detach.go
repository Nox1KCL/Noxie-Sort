package background

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/gofrs/flock"
)

var dtlog = slog.With("module", "background")

func RunInBackground(childArgs []string) error {
	dtlog.Debug("running child process in background", "args", childArgs)
	err := detach(childArgs)
	if err != nil {
		dtlog.Error("failed to run in background", "error", err)
		return err
	}
	return nil
}

func IsChildRunning() (*flock.Flock, error) {
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return nil, fmt.Errorf("getting user cache dir: %w", err)
	}
	lockPath := filepath.Join(cacheDir, "Noxie-Sort", "app.lock")
	dtlog.Debug("attempting to acquire lock", "lockPath", lockPath)
	if err := os.MkdirAll(filepath.Dir(lockPath), 0755); err != nil {
		return nil, fmt.Errorf("creating lock dir: %w", err)
	}
	fileLock := flock.New(lockPath)
	locked, err := fileLock.TryLock()
	if err != nil {
		dtlog.Error("error trying to lock file", "error", err)
		return nil, err
	}
	if !locked {
		dtlog.Warn("file is locked by another process")
		return nil, fmt.Errorf("file is locked by another process")
	}

	dtlog.Info("lock acquired successfully", "lockPath", lockPath)
	return fileLock, nil
}
