package background

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/gofrs/flock"
)

func RunInBackground(childArgs []string) error {
	err := detach(childArgs)
	if err != nil {
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
	if err := os.MkdirAll(filepath.Dir(lockPath), 0755); err != nil {
		return nil, fmt.Errorf("creating lock dir: %w", err)
	}
	fileLock := flock.New(lockPath)
	locked, err := fileLock.TryLock()
	if err != nil {
		return nil, err
	}
	if !locked {
		return nil, fmt.Errorf("file is locked by another process")
	}

	return fileLock, nil
}
