//go:build linux

package background

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
)

func detach(childArgs []string) error {
	cmd := exec.Command(os.Args[0], childArgs...)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setsid: true,
	}

	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return fmt.Errorf("getting user cache dir: %w", err)
	}
	logDir := filepath.Join(cacheDir, "InFolderSort")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return fmt.Errorf("creating log dir: %w", err)
	}

	infoFile, err := os.Create(filepath.Join(logDir, "info.log"))
	if err != nil {
		return err
	}

	errFile, err := os.Create(filepath.Join(logDir, "err.log"))
	if err != nil {
		return err
	}

	cmd.Stdout = infoFile
	cmd.Stderr = errFile
	err = cmd.Start()

	errFile.Close()
	infoFile.Close()

	if err != nil {
		return err
	}
	os.Exit(0)
	return nil
}
