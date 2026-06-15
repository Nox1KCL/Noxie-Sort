package daemon

import (
	_ "embed"
	"errors"
	"log/slog"
)

//go:embed infoldersort.service
var Service []byte

var dlog = slog.With("module", "daemon")

type ServiceInfo struct {
	Name    string
	Path    string
	Content string
}

func LaunchingDaemon() error {
	dlog.Info("starting daemon installation")
	err := IsWorking()
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
		Name:    "",
		Path:    "",
		Content: "",
	}
}
