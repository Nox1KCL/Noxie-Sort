package background

import (
	"fmt"

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
	fileLock := flock.New("/home/nox/InFolderSort/app.lock")
	locked, err := fileLock.TryLock()
	if err != nil {
		return nil, err
	}
	if !locked {
		return nil, fmt.Errorf("file is locked by another process")
	}

	return fileLock, nil
}
