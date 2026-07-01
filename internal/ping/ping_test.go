package ping

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestPing(t *testing.T) {
	tests := []struct {
		name           string
		handler        http.HandlerFunc
		expectedOutput string
	}{
		{
			name: "daemon_alive",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"condition":"Running", "name":"Noxie-Sort", "path":"/tmp"}`))
			},
			expectedOutput: "Daemon is Alive\nStatus: Running\nName: Noxie-Sort\nPath: /tmp\n",
		},
		{
			name: "daemon_dead",
			handler: func(w http.ResponseWriter, r *http.Request) {
				// We don't really use this handler because the server will be closed, or we can just send error
				// Actually, to simulate dead daemon, we can just point to an invalid port. 
				// We will handle this specially below.
			},
			expectedOutput: "Daemon is Dead\n",
		},
		{
			name: "daemon_code_error",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			},
			expectedOutput: "Daemon is responding with code error\n",
		},
		{
			name: "daemon_messed_response",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{invalid json`))
			},
			expectedOutput: "Daemon is Alive, but response is messed\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var targetURL string
			if tt.name == "daemon_dead" {
				// Simulating connection refused
				targetURL = "http://localhost:12345/status_should_fail"
			} else {
				server := httptest.NewServer(tt.handler)
				defer server.Close()
				targetURL = server.URL
			}

			// Override env var
			os.Setenv("DAEMON_URL", targetURL)
			defer os.Unsetenv("DAEMON_URL")

			// Capture stdout
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			Ping()

			w.Close()
			os.Stdout = oldStdout
			var buf bytes.Buffer
			io.Copy(&buf, r)
			actualOutput := buf.String()

			if !strings.Contains(actualOutput, tt.expectedOutput) {
				t.Errorf("expected output to contain %q, got %q", tt.expectedOutput, actualOutput)
			}
		})
	}
}
