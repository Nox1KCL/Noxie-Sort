package main

import (
	"log/slog"
	"time"

	"github.com/Nox1KCL/Noxie-Sort/internal/ping"
)

var svlog = slog.With("module", "server")

func main() {
	svlog.Info("server loop started")
	for {
		svlog.Debug("pinging...")
		ping.Ping()
		time.Sleep(5 * time.Second)
	}
}
