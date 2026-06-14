package daemon

import (
	_ "embed"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

//go:embed infoldersort.service
var Service []byte

var dlog = slog.With("module", "daemon")

type ServiceInfo struct {
	Path    string
	Content string
}

func LaunchingDaemon() error {
	dlog.Info("starting daemon installation")
	err := isWorking()
	if err == nil {
		return errors.New("daemon is already running")
	}

	service := NewService()
	err = service.initService()
	if err != nil {
		return err
	}

	err = service.initDaemon()
	if err != nil {
		return err
	}

	return nil
}

func NewService() *ServiceInfo {
	return &ServiceInfo{
		Path:    "",
		Content: "",
	}
}

func (s *ServiceInfo) initService() error {
	cfgPath, err := os.UserConfigDir()
	if err != nil {
		return err
	}
	serviceName := "infoldersort.service"
	servicePath := filepath.Clean(filepath.Join(cfgPath, "systemd", "user", serviceName))

	s.Path = servicePath
	exePath, _ := os.Executable()
	s.Content = strings.ReplaceAll(string(Service), "{{EXEC_PATH}}", exePath)

	return nil
}
