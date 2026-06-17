//go:build windows

package daemon

import (
	"os"
	"strings"

	"golang.org/x/sys/windows/registry"
)

func (s *ServiceInfo) initDaemon() error {
	dlog.Debug("opening registry key for writing")
	k, err := registry.OpenKey(registry.CURRENT_USER, "Software\\Microsoft\\Windows\\CurrentVersion\\Run", registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer k.Close()

	dlog.Debug("setting registry value", "name", s.Name, "path", s.Path)
	err = k.SetStringValue(s.Name, `"`+s.Path+`" --background`)
	if err != nil {
		return err
	}

	return nil
}

func ClosingDaemon() error {
	dlog.Info("starting daemon uninstallation")
	dlog.Debug("opening registry key for deletion")
	k, err := registry.OpenKey(registry.CURRENT_USER, "Software\\Microsoft\\Windows\\CurrentVersion\\Run", registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer k.Close()

	dlog.Debug("deleting registry value", "name", "Noxie-Sort")
	err = k.DeleteValue("Noxie-Sort")
	if err != nil {
		return err
	}

	return nil
}

func IsWorking() error {
	k, err := registry.OpenKey(registry.CURRENT_USER, "Software\\Microsoft\\Windows\\CurrentVersion\\Run", registry.QUERY_VALUE)
	if err != nil {
		return err
	}
	defer k.Close()

	// Захардкодив бо для отримання з Service треба або його передавати або робити функцію методом
	// але тоді у service.go вилазить що не можна швидко перевірити чи прога працює через так би мовити
	// зворотну залежність
	path, _, err := k.GetStringValue("Noxie-Sort")
	if err != nil {
		return err
	}

	cleanPath := strings.TrimSuffix(path, " --background")
	cleanPath = strings.Trim(cleanPath, `"`)

	dlog.Debug("checking if executable exists", "path", cleanPath)
	_, err = os.Stat(cleanPath)
	if err != nil {
		dlog.Debug("executable not found", "error", err)
		return err
	}

	dlog.Debug("daemon is properly installed and executable exists")
	return nil
}

func (s *ServiceInfo) initService() error {
	execPath, err := os.Executable()
	if err != nil {
		return err
	}
	serviceName := "Noxie-Sort"

	s.Name = serviceName
	s.Path = execPath
	return nil
}
