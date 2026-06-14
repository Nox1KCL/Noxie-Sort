//go:build windows

package daemon

import "errors"

func (s *ServiceInfo) initDaemon() error {
	return errors.New("daemon installation is not implemented for Windows yet")
}

func ClosingDaemon() error {
	return errors.New("daemon uninstallation is not implemented for Windows yet")
}

func isWorking() error {
	return errors.New("daemon status check is not implemented for Windows yet")
}
