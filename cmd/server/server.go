package main

import (
	"time"

	"github.com/Nox1KCL/Noxie-Sort/internal/ping"
)

func main() {
	for {
		ping.Ping()
		time.Sleep(5 * time.Second)
	}
}
