package tui

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/Nox1KCL/Noxie-Sort/internal/files"
)

func TestGenerateReport(t *testing.T) {
	report := files.SortResult{
		Moved:   []string{"a.txt", "b.txt"},
		Skipped: []string{"c.txt"},
		Errors:  nil,
	}

	// Redirect stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	GenerateReport(report)

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if !strings.Contains(output, "Moved: 2 files") {
		t.Errorf("expected 'Moved: 2 files', got: %s", output)
	}
	if !strings.Contains(output, "Skipped: 1 files") {
		t.Errorf("expected 'Skipped: 1 files', got: %s", output)
	}
}

func TestGenerateReport_WithErrors(t *testing.T) {
	report := files.SortResult{
		Moved:   nil,
		Skipped: []string{"broken.txt"},
		Errors:  []error{os.ErrPermission},
	}

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	GenerateReport(report)

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if !strings.Contains(output, "Errors: 1") {
		t.Errorf("expected 'Errors: 1', got: %s", output)
	}
	if !strings.Contains(output, "permission denied") {
		t.Errorf("expected error message in output, got: %s", output)
	}
}
