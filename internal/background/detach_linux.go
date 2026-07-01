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
	logDir := filepath.Join(cacheDir, "Noxie-Sort")
	dtlog.Debug("ensuring log directory exists", "logDir", logDir)
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
	
	dtlog.Info("starting detached process", "cmd", cmd.Path, "args", cmd.Args)
	err = cmd.Start()

	errFile.Close()
	infoFile.Close()

	if err != nil {
		dtlog.Error("failed to start detached process", "error", err)
		return err
	}
	dtlog.Debug("exiting parent process")
	os.Exit(0)
	return nil
}
