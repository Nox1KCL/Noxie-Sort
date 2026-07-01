package daemon

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestConnection_GET(t *testing.T) {
	s := &ServiceInfo{
		Name: "TestDaemon",
		Path: "/test/path",
	}

	req, err := http.NewRequest("GET", "/status", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(s.Connection)

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	var statusResp Status
	if err := json.NewDecoder(rr.Body).Decode(&statusResp); err != nil {
		t.Errorf("failed to decode response body: %v", err)
	}

	if statusResp.Name != s.Name {
		t.Errorf("expected Name %q, got %q", s.Name, statusResp.Name)
	}
	if statusResp.Path != s.Path {
		t.Errorf("expected Path %q, got %q", s.Path, statusResp.Path)
	}
	if statusResp.Condition != "Active" {
		t.Errorf("expected Condition 'Active', got %q", statusResp.Condition)
	}
}

func TestConnection_POST(t *testing.T) {
	s := &ServiceInfo{}

	req, err := http.NewRequest("POST", "/status", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(s.Connection)

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusMethodNotAllowed {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusMethodNotAllowed)
	}
}

func TestNewHTTPHandler(t *testing.T) {
	s := &ServiceInfo{}
	handler := newHTTPHandler(s)
	if handler == nil {
		t.Error("expected non-nil handler")
	}
}
