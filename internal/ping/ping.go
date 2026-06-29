package ping

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
)

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

	resp, err := client.Get("http://localhost:9999/status")
	if err != nil {
		fmt.Println("Daemon is Dead")
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Println("Daemon is responding with code error")
		return
	}

	var status Status
	err = json.NewDecoder(resp.Body).Decode(&status)
	if err != nil {
		fmt.Println("Daemon is Alive, but response is messed")
		return
	}
	fmt.Printf("Daemon is Alive\nStatus: %v\nName: %v\nPath: %v\n", status.Condition, status.Name, status.Path)

}
