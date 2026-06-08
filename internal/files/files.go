// Package files provides utilities for file operations used in the InFolderSort application.
package files

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Nox1KCL/InFolderSort/internal/config"
)

var flog = slog.With("module", "files")

type FileInfo struct {
	Path string
	Name string
}

type SortResult struct {
	Moved   []string
	Skipped []string
	Errors  []error
}

type MoveTask struct {
	FileName   string
	SourcePath string
	DestPath   string
}

type Sorter struct {
	Config  *config.Config
	ScanDir string
	Files   []FileInfo
	Tasks   []MoveTask
	Errors  []error
}

func NewSorter(cfg *config.Config) *Sorter {
	flog.Info("creating sorter")
	return &Sorter{
		Config:  cfg,
		ScanDir: cfg.ScanDir,
		Files:   make([]FileInfo, 0),
		Tasks:   make([]MoveTask, 0),
		Errors:  make([]error, 0),
	}
}

func (s *Sorter) Scan() error {
	entries, err := os.ReadDir(s.ScanDir)
	if err != nil {
		return fmt.Errorf("reading directory %q: %w", s.ScanDir, err)
	}
	var file FileInfo

	for _, entry := range entries {
		fileName := entry.Name()
		if !entry.IsDir() && !strings.HasPrefix(fileName, ".") {
			file = FileInfo{
				Path: s.ScanDir,
				Name: fileName}
			s.Files = append(s.Files, file)
		}
	}
	return nil
}

func (s *Sorter) Plan() error {
	if len(s.Files) == 0 {
		flog.Error("no files found",
			"dir", s.ScanDir)
		return fmt.Errorf("no files found in %q", s.ScanDir)
	}
	if s.Config == nil {
		return fmt.Errorf("config is empty")
	}

	for _, file := range s.Files {
		fileName := file.Name
		fileExt := filepath.Ext(fileName)
		targetPath, err := s.Config.GetTargetPath(fileExt)
		if err != nil {
			flog.Warn("file extension not found in config",
				"file", fileName,
				"ext", fileExt,
				"error", err)
			s.Errors = append(s.Errors, err)
			continue
		}

		var savePath string
		if filepath.IsAbs(targetPath) {
			savePath = targetPath
		} else {
			savePath = filepath.Join(s.ScanDir, targetPath)
		}

		exist, err := IsFileExist(filepath.Join(savePath, fileName))
		if err != nil {
			flog.Warn("checking file existence",
				"file", fileName,
				"path", savePath,
				"error", err)
			s.Errors = append(s.Errors, err)
			continue
		}

		finalFileName := fileName
		if exist {
			finalFileName = RenameFile(fileName)
		}

		sourcePath := filepath.Join(s.ScanDir, fileName)
		destPath := filepath.Join(savePath, finalFileName)
		s.Tasks = append(s.Tasks, MoveTask{fileName, sourcePath, destPath})
	}
	return nil
}

func (s *Sorter) Execute() (SortResult, error) {
	var report SortResult

	for _, task := range s.Tasks {
		destDir := filepath.Dir(task.DestPath)
		if err := os.MkdirAll(destDir, 0755); err != nil {
			flog.Error("creating dirs",
				"dir", destDir,
				"error", err)

			report.Errors = append(report.Errors,
				fmt.Errorf("creating dirs by path %q: %w", destDir, err),
			)
			report.Skipped = append(report.Skipped, task.FileName)
			continue
		}

		if err := os.Rename(task.SourcePath, task.DestPath); err != nil {
			flog.Error("moving file",
				"from", task.SourcePath,
				"to", task.DestPath,
				"error", err,
				"file", task.FileName)
			report.Errors = append(report.Errors,
				fmt.Errorf("moving file from %q to %q: %w", task.SourcePath, task.DestPath, err))
			report.Skipped = append(report.Skipped, task.FileName)
			continue
		}

		report.Moved = append(report.Moved, task.FileName)
	}
	flog.Info("sorting completed",
		"total", len(s.Tasks),
		"moved", len(report.Moved),
		"skipped", len(report.Skipped),
	)

	return report, nil
}

func (s *Sorter) InDirSorting() (SortResult, error) {

	if err := s.Scan(); err != nil {
		return SortResult{}, fmt.Errorf("scanning directory %q: %w", s.ScanDir, err)
	}
	if err := s.Plan(); err != nil {
		return SortResult{}, fmt.Errorf("planning sorting: %w", err)
	}

	if report, err := s.Execute(); err != nil {
		report.Errors = append(report.Errors, s.Errors...)
		return report, fmt.Errorf("executing sorting: %w", err)
	} else {
		return report, nil
	}
}

func (s *Sorter) SelectiveSorting(fileName string) (SortResult, error) {
	s.Files = append(s.Files, FileInfo{s.ScanDir, fileName})

	//TODO: IsFileLocked
	if err := s.Plan(); err != nil {
		return SortResult{}, fmt.Errorf("planning sorting: %w", err)
	}
	if report, err := s.Execute(); err != nil {
		report.Errors = append(report.Errors, s.Errors...)
		return report, fmt.Errorf("executing sorting: %w", err)
	} else {
		return report, nil
	}
}

func FileSizePolling(filePath string, waitInterval time.Duration, maxRetries int) error {
    var lastSize int64 = -1
    retries := 0

    for {
        info, err := os.Stat(filePath)
        if err != nil {
            flog.Warn("failed to stat file",
                "file", filePath,
                "error", err)
            return fmt.Errorf("failed to stat file: %w", err)
        }

        currentSize := info.Size()
        if currentSize == lastSize {
            time.Sleep(waitInterval)
            if currentSize == lastSize {
                return nil
            }
        }

        lastSize = currentSize
        retries++
        if retries > maxRetries {
            flog.Warn("file size polling time out",
                "file", filePath,
                "retries", retries)
            return fmt.Errorf("file size polling timed out after %d retries: %s", maxRetries, filePath)
        }

        time.Sleep(waitInterval)
    }
}

func GetDownloadsPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to find home directory: %w", err)
	}

	potentialDirs := []string{"Downloads", "downloads"}

	for _, dirName := range potentialDirs {
		downloadPath := filepath.Join(homeDir, dirName)
		if info, err := os.Stat(downloadPath); err == nil && info.IsDir() {
			return downloadPath, nil
		}
	}

	return "", fmt.Errorf("downloads directory not found")
}

func IsFileExist(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	} else if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}

	return false, fmt.Errorf("checking file %q: %w", path, err)
}

func IsFileLocked(path string) bool {
    file, err := os.OpenFile(path, os.O_RDWR, 0666)
    if err != nil {
        return true
    }
    file.Close()
    return false
}

func FileExtValidate(fileName string) bool {
	if strings.HasPrefix(fileName, "~$") || strings.HasPrefix(fileName, ".~lock.") {
		return false
	}

	fileExt := filepath.Ext(fileName)
	ext := strings.ToLower(strings.TrimSpace(fileExt))

	if ext == "" {
		return false
	}

	if ext == ".tmp" || ext == ".crdownload" || ext == ".part" {
		return false
	}

	return true
}

func RenameFile(file string) string {
	ext := filepath.Ext(file)
	name := strings.TrimSuffix(file, ext)
	timestamp := time.Now().Format("20060102_150405")
	newName := fmt.Sprintf("%s_%s%s", name, timestamp, ext)
	return newName
}

func ExecutableDir() string {
    exe, err := os.Executable()
    if err != nil {
        return "."
    }
    return filepath.Dir(exe)
}
