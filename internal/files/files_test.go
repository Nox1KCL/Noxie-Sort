package files

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/Nox1KCL/InFolderSort/internal/config"
	"github.com/Nox1KCL/InFolderSort/internal/syncutils"
)

func TestInDirSorting_Basic(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "photo.jpg"), []byte("data"), 0644)
	_ = os.WriteFile(filepath.Join(dir, "doc.pdf"), []byte("data"), 0644)

	cfg := &config.Config{
		ScanDir: dir,
		Rules: map[string]config.FolderRule{
			"Images": {TargetPath: "Images", Extensions: []string{".jpg"}},
			"Docs":   {TargetPath: "Docs", Extensions: []string{".pdf"}},
		},
	}
	cfg.Prepare()

	sorter := NewSorter(cfg)
	report, err := sorter.InDirSorting()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "Images", "photo.jpg")); err != nil {
		t.Error("photo.jpg was not moved to Images/")
	}
	if _, err := os.Stat(filepath.Join(dir, "Docs", "doc.pdf")); err != nil {
		t.Error("doc.pdf was not moved to Docs/")
	}
	if len(report.Moved) != 2 {
		t.Errorf("expected 2 moved files, got %d", len(report.Moved))
	}
}

func TestSorter_ConflictResolution(t *testing.T) {
	dir := t.TempDir()
	targetSubdir := filepath.Join(dir, "Images")
	_ = os.MkdirAll(targetSubdir, 0755)
	_ = os.WriteFile(filepath.Join(targetSubdir, "photo.jpg"), []byte("old data"), 0644)
	_ = os.WriteFile(filepath.Join(dir, "photo.jpg"), []byte("new data"), 0644)

	cfg := &config.Config{
		ScanDir: dir,
		Rules: map[string]config.FolderRule{
			"Images": {TargetPath: "Images", Extensions: []string{".jpg"}},
		},
	}
	cfg.Prepare()

	sorter := NewSorter(cfg)
	if err := sorter.Scan(); err != nil {
		t.Fatalf("Scan failed: %v", err)
	}
	if err := sorter.Plan(); err != nil {
		t.Fatalf("Plan failed: %v", err)
	}
	if len(sorter.Tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(sorter.Tasks))
	}
	if !strings.Contains(sorter.Tasks[0].DestPath, "photo_") {
		t.Errorf("expected renamed file with timestamp, got %s", sorter.Tasks[0].DestPath)
	}

	report, err := sorter.Execute()
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if len(report.Moved) != 1 {
		t.Errorf("expected 1 moved file, got %d", len(report.Moved))
	}
	entries, _ := os.ReadDir(targetSubdir)
	if len(entries) != 2 {
		t.Errorf("expected 2 files in target subdir, got %d", len(entries))
	}
}

func TestNewSorter(t *testing.T) {
	cfg := &config.Config{ScanDir: "/tmp"}
	sorter := NewSorter(cfg)

	if sorter == nil {
		t.Fatal("NewSorter returned nil")
	}
	if sorter.ScanDir != "/tmp" {
		t.Errorf("expected ScanDir '/tmp', got %q", sorter.ScanDir)
	}
	if len(sorter.Files) != 0 || len(sorter.Tasks) != 0 {
		t.Error("new sorter should have empty slices")
	}
}

func TestSorter_Scan_SkipsHiddenFiles(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "visible.txt"), []byte("data"), 0644)
	_ = os.WriteFile(filepath.Join(dir, ".hidden"), []byte("data"), 0644)

	cfg := &config.Config{ScanDir: dir}
	sorter := NewSorter(cfg)

	if err := sorter.Scan(); err != nil {
		t.Fatalf("Scan failed: %v", err)
	}
	if len(sorter.Files) != 1 || sorter.Files[0].Name == ".hidden" {
		t.Errorf("expected 1 visible file, got %d files: %v", len(sorter.Files), sorter.Files)
	}
}

func TestSorter_Scan_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	cfg := &config.Config{ScanDir: dir}
	sorter := NewSorter(cfg)

	if err := sorter.Scan(); err != nil {
		t.Fatalf("Scan failed: %v", err)
	}
	if len(sorter.Files) != 0 {
		t.Errorf("expected 0 files in empty dir, got %d", len(sorter.Files))
	}
}

func TestSorter_Plan_NoFiles(t *testing.T) {
	dir := t.TempDir()
	cfg := &config.Config{ScanDir: dir}
	sorter := NewSorter(cfg)

	if err := sorter.Plan(); err == nil {
		t.Fatal("expected error for Plan with no files, got nil")
	}
}

func TestSorter_Plan_ExtensionNotFound(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "unknown.xyz"), []byte("data"), 0644)

	cfg := &config.Config{
		ScanDir: dir,
		Rules:   map[string]config.FolderRule{},
	}
	cfg.Prepare()

	sorter := NewSorter(cfg)
	_ = sorter.Scan()

	if err := sorter.Plan(); err != nil {
		t.Fatalf("Plan failed unexpectedly: %v", err)
	}
	if len(sorter.Tasks) != 0 {
		t.Errorf("expected 0 tasks for unknown extension, got %d", len(sorter.Tasks))
	}
	if len(sorter.Errors) == 0 {
		t.Error("expected at least 1 error for unknown extension")
	}
}

func TestSorter_Plan_AbsoluteTargetPath(t *testing.T) {
	dir := t.TempDir()
	absTarget := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "file.txt"), []byte("data"), 0644)

	cfg := &config.Config{
		ScanDir: dir,
		Rules: map[string]config.FolderRule{
			"Docs": {TargetPath: absTarget, Extensions: []string{".txt"}},
		},
	}
	cfg.Prepare()

	sorter := NewSorter(cfg)
	_ = sorter.Scan()
	_ = sorter.Plan()

	if len(sorter.Tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(sorter.Tasks))
	}
	if sorter.Tasks[0].DestPath != filepath.Join(absTarget, "file.txt") {
		t.Errorf("expected dest %s, got %s", filepath.Join(absTarget, "file.txt"), sorter.Tasks[0].DestPath)
	}
}

func TestSelectiveSorting(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "doc.pdf"), []byte("data"), 0644)

	cfg := &config.Config{
		ScanDir: dir,
		Rules: map[string]config.FolderRule{
			"Docs": {TargetPath: "Docs", Extensions: []string{".pdf"}},
		},
	}
	cfg.Prepare()

	sorter := NewSorter(cfg)
	report, err := sorter.SelectiveSorting("doc.pdf")

	if err != nil {
		t.Fatalf("SelectiveSorting failed: %v", err)
	}
	if len(report.Moved) != 1 {
		t.Errorf("expected 1 moved file, got %d", len(report.Moved))
	}
	if _, err := os.Stat(filepath.Join(dir, "Docs", "doc.pdf")); err != nil {
		t.Error("doc.pdf was not moved to Docs/")
	}
}

func TestIsFileExist(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "exists.txt")
	_ = os.WriteFile(path, []byte("data"), 0644)

	exists, err := IsFileExist(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !exists {
		t.Error("expected file to exist")
	}

	notExists, err := IsFileExist(filepath.Join(dir, "nope.txt"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if notExists {
		t.Error("expected file not to exist")
	}
}

func TestRenameFile_AddsTimestamp(t *testing.T) {
	result := RenameFile("photo.jpg")

	if !strings.Contains(result, "photo_") {
		t.Errorf("expected 'photo_' in result, got %q", result)
	}
	if !strings.HasSuffix(result, ".jpg") {
		t.Errorf("expected .jpg extension, got %q", result)
	}

	// Verify timestamp format: photo_20260603_123450.jpg
	// Since timestamp has an underscore, we expect 3 parts: "photo", "YYYYMMDD", "HHMMSS.jpg"
	parts := strings.Split(result, "_")
	if len(parts) < 3 {
		t.Fatalf("expected 'name_YYYYMMDD_HHMMSS.ext', got %q", result)
	}

	// Check the date part (8 chars)
	if len(parts[len(parts)-2]) != 8 {
		t.Errorf("expected 8-char date, got %q", parts[len(parts)-2])
	}

	// Check the time part without extension (6 chars)
	timePart := strings.TrimSuffix(parts[len(parts)-1], ".jpg")
	if len(timePart) != 6 {
		t.Errorf("expected 6-char time, got %q", timePart)
	}
}

func TestRenameFile_MultipleDots(t *testing.T) {
	result := RenameFile("archive.tar.gz")

	if !strings.Contains(result, "archive.tar_") {
		t.Errorf("expected 'archive.tar_' in result, got %q", result)
	}
	if !strings.HasSuffix(result, ".gz") {
		t.Errorf("expected .gz extension, got %q", result)
	}
}

func TestRenameFile_NoExtension(t *testing.T) {
	result := RenameFile("README")

	if !strings.Contains(result, "README_") {
		t.Errorf("expected 'README_' in result, got %q", result)
	}
	if strings.HasSuffix(result, ".") {
		t.Errorf("should not end with dot, got %q", result)
	}
}

func TestFileExt_Validate(t *testing.T) {
    tests := []struct {
        name     string
        expected bool
    }{
        {"test1.txt", true},
        {"femboyies.tmp", false},
        {"davidPivo", false},
        {"~$Fdf", false},
        {"lolol.docx", true},
        {"~lock.fd.docx", true},
        {".~lock.fd.docx", false},
        {"locked.doc", true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            actual := FileExtValidate(tt.name)
            if actual != tt.expected {
                t.Errorf("FileExtValidate(%q) = %v; want %v", tt.name, actual, tt.expected)
            }
        })
    }
}

func TestFileLock(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tetraed.txt")

	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0666)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}
	defer file.Close()

	err = syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
	if err != nil {
		t.Fatalf("failed to lock file: %v", err)
	}
	defer syscall.Flock(int(file.Fd()), syscall.LOCK_UN)

	for i := 1; i <= 5; i++ {
		data := fmt.Sprintf("Datas №%d\n", i)
		_, _ = file.WriteString(data)
		file.Sync()

		if isValid := IsFileLocked(path); !isValid {
			t.Errorf("file should be locked during write: Value: %t | Path: %q", isValid, path)
		}

		time.Sleep(200 * time.Millisecond)
	}
}

func TestFileSizePolling(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "googl.docx")

	var (
		initialSize  int64 = 1 * 1024 * 1024
		waitInterval       = 100 * time.Millisecond
		maxRetries         = 5
	)

	file, err := os.Create(path)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}
	defer file.Close()

	wg := syncutils.MyWaitGroup{}

	wg.Go(func(){
    	err := FileSizePolling(path, waitInterval, maxRetries)
    	if err != nil {
    		t.Errorf("FileSizePolling failed: %v", err)
    	}
	})

	for i := range 5 {
		_ = file.Truncate(initialSize + int64(i)*1024*1024)
		time.Sleep(50 * time.Millisecond)
		t.Logf("Test iteration %d completed", i)
	}
	wg.Wait()
}
