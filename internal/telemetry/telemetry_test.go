package telemetry

import (
	"context"
	"testing"
)

func TestNewTelemetry(t *testing.T) {
	// Execute NewTelemetry
	shutdown, observer, err := NewTelemetry()
	
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	
	if shutdown == nil {
		t.Error("expected shutdown func, got nil")
	}
	
	if observer == nil {
		t.Error("expected observer, got nil")
	} else {
		// Test if fields in observer are populated
		if observer.Tracer == nil {
			t.Error("expected Tracer to be initialized")
		}
		if observer.Meter == nil {
			t.Error("expected Meter to be initialized")
		}
		if observer.Logger == nil {
			t.Error("expected Logger to be initialized")
		}
		if observer.SCounter == nil {
			t.Error("expected SCounter to be initialized")
		}
	}
	
	// Shutdown cleanly
	if err := shutdown(context.Background()); err != nil {
		if err.Error() != "" {
			t.Logf("shutdown returned error (expected without server): %v", err)
		}
	}
}
