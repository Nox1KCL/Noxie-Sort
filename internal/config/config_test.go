package config

import (
	"testing"
)

func TestValidate_NoErrors(t *testing.T) {
	cfg := &Config{
		Rules: map[string]FolderRule{
			"Images": {
				TargetPath: "path/to/Images",
				Extensions: []string{".jpg", ".png"},
			},
			"Docs": {
				TargetPath: "path/to/Docs",
				Extensions: []string{".pdf", ".docx"},
			},
		},
	}

	if err := cfg.Validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidate_EmptyExtension(t *testing.T) {
	cfg := &Config{
		Rules: map[string]FolderRule{
			"Empty": {
				TargetPath: "path",
				Extensions: []string{""},
			},
		},
	}

	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error for empty extension, got nil")
	}
}

func TestValidate_MissingDot(t *testing.T) {
	cfg := &Config{
		Rules: map[string]FolderRule{
			"NoDot": {
				TargetPath: "path",
				Extensions: []string{"jpg"},
			},
		},
	}

	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error for extension without dot, got nil")
	}
}

func TestValidate_DuplicateExtension(t *testing.T) {
	cfg := &Config{
		Rules: map[string]FolderRule{
			"Images": {
				TargetPath: "path/to/Images",
				Extensions: []string{".jpg"},
			},
			"Duplicates": {
				TargetPath: "path/to/Dup",
				Extensions: []string{".jpg"},
			},
		},
	}

	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error for duplicate extension, got nil")
	}
}

func TestInvertConfig(t *testing.T) {
	cfg := &Config{
		Rules: map[string]FolderRule{
			"Images": {
				TargetPath: "$HOME/Pictures",
				Extensions: []string{".jpg", ".png"},
			},
		},
	}

	cfg.InvertConfig()

	if cfg.InvertedRules == nil {
		t.Fatal("InvertedRules should not be nil after InvertConfig")
	}

	path, ok := cfg.InvertedRules[".jpg"]
	if !ok {
		t.Fatal("expected .jpg to be in InvertedRules")
	}
	if path == "" {
		t.Error("path should not be empty")
	}

	// Second call should be idempotent
	cfg.InvertConfig()
	if len(cfg.InvertedRules) != 2 {
		t.Errorf("expected 2 entries after second invert, got %d", len(cfg.InvertedRules))
	}
}

func TestGetTargetPath_Success(t *testing.T) {
	cfg := &Config{
		InvertedRules: map[string]string{
			".jpg": "path/to/Images",
		},
	}

	path, err := cfg.GetTargetPath(".jpg")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if path != "path/to/Images" {
		t.Errorf("expected 'path/to/Images', got %q", path)
	}
}

func TestGetTargetPath_NotFound(t *testing.T) {
	cfg := &Config{
		InvertedRules: map[string]string{},
	}

	_, err := cfg.GetTargetPath(".unknown")
	if err == nil {
		t.Fatal("expected error for unknown extension, got nil")
	}
}
