package main

import (
	"time"

	"github.com/Nox1KCL/Noxie-Sort/internal/server"
)

func main() {
	for {
		server.Ping()
		time.Sleep(5 * time.Second)
	}
}
