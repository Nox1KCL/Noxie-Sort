package ping

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"
)

var plog = slog.With("module", "ping")

type Status struct {
	Condition string `json:"condition"`
	Name      string `json:"name"`
	Path      string `json:"path"`
}

func Ping() {
	targetURL := os.Getenv("DAEMON_URL")
	if targetURL == "" {
		targetURL = "http://localhost:9999/status"
	}
	client := http.Client{
		Timeout: time.Second * 10,
	}

	resp, err := client.Get(targetURL)
	if err != nil {
		plog.Error("Daemon is Dead", "error", err)
		fmt.Println("Daemon is Dead")
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		plog.Error("Daemon is responding with code error", "statusCode", resp.StatusCode)
		fmt.Println("Daemon is responding with code error")
		return
	}

	var status Status
	err = json.NewDecoder(resp.Body).Decode(&status)
	if err != nil {
		plog.Error("Daemon is Alive, but response is messed", "error", err)
		fmt.Println("Daemon is Alive, but response is messed")
		return
	}
	plog.Info("Daemon is Alive", "status", status.Condition, "name", status.Name, "path", status.Path)
	fmt.Printf("Daemon is Alive\nStatus: %v\nName: %v\nPath: %v\n", status.Condition, status.Name, status.Path)

}
