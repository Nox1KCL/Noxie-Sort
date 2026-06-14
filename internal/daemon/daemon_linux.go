//go:build linux

package daemon

import (
	"os"
	"os/exec"
	"path/filepath"
)

func (s *ServiceInfo) initDaemon() error {
	dlog.Debug("creating systemd user directory", "path", filepath.Dir(s.Path))
	err := os.MkdirAll(filepath.Dir(s.Path), 0755)
	if err != nil {
		return err
	}

	dlog.Debug("writing service file", "path", s.Path)
	err = os.WriteFile(s.Path, []byte(s.Content), 0644)
	if err != nil {
		return err
	}

	dlog.Debug("reloading systemd daemon")
	err = exec.Command("systemctl", "--user", "daemon-reload").Run()
	if err != nil {
		return err
	}

	dlog.Debug("enabling and starting service")
	err = exec.Command("systemctl", "--user", "enable", "--now", "infoldersort.service").Run()
	if err != nil {
		return err
	}

	return nil
}

func ClosingDaemon() error {
	dlog.Info("starting daemon uninstallation")
	s := NewService()
	err := s.initService()
	if err != nil {
		return err
	}

	dlog.Debug("disabling and stopping service")
	err = exec.Command("systemctl", "--user", "disable", "--now", "infoldersort.service").Run()
	if err != nil {
		return err
	}

	dlog.Debug("removing service file", "path", s.Path)
	err = os.Remove(s.Path)
	if err != nil {
		return err
	}

	dlog.Debug("reloading systemd daemon")
	err = exec.Command("systemctl", "--user", "daemon-reload").Run()
	if err != nil {
		return err
	}

	return nil
}

func isWorking() error {
	err := exec.Command("systemctl", "--user", "is-active", "--quiet", "infoldersort.service").Run()
	return err
}
