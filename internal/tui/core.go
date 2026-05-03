package tui

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Nox1KCL/InFolderSort/internal/config"
	"github.com/Nox1KCL/InFolderSort/internal/files"
)

func Core(cfg *config.Config) error {
	userChoice := askChoice("basic sort or manual?(b/m): ", "b", "m")

	var targetPath string
	var err error

	switch userChoice {
	case "b":
		targetPath = cfg.ScanDir
	case "m":
		targetPath, err = getManualPath()
	default:
		return fmt.Errorf("unexpected user choice: %q", userChoice)
	}

	if err != nil {
		return fmt.Errorf("failed to get target path: %w", err)
	}

	return performSort(targetPath, cfg)
}

func performSort(targetDir string, cfg *config.Config) error {
	fileInfo, err := os.Stat(targetDir)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("path doesn't exist: %w", err)
		}
		return fmt.Errorf("getting file info %q: %w", targetDir, err)
	}
	if !fileInfo.IsDir() {
		return fmt.Errorf("path is not a directory: %q", fileInfo.Name())
	}

	report, err := files.InDirSorting(targetDir, cfg)
	if err != nil {
		return fmt.Errorf("directory sorting error: %w", err)
	}

	GenerateReport(report)
	return nil
}

func getManualPath() (string, error) {
	homeDir, err := files.GetHomeDir()
	if err != nil {
		return "", fmt.Errorf("getting home directory: %w", err)
	}

	fmt.Printf("We suppose your home dir is: %s\n", homeDir)
	useHome := askChoice("Use it for base of path?(y/n): ", "y", "n")
	userInput, err := askInput("Enter folder's path you want to sort: ")
	if err != nil {
		return "", fmt.Errorf("getting user input: %w", err)
	}

	if useHome == "y" {
		return filepath.Join(homeDir, userInput), nil
	}
	return userInput, nil
}

func askChoice(prompt string, validOptions ...string) string {
	var input string
	for {
		fmt.Print(prompt)
		_, err := fmt.Scanln(&input)
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "\nerror reading input: %v\n", err)
			os.Exit(1)
		}
		input = strings.TrimSpace(input)

		for _, opt := range validOptions {
			if input == opt {
				return input
			}
		}
		fmt.Println("Invalid input, try again.")
	}
}

func askInput(prompt string) (string, error) {
	fmt.Print(prompt)
	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() {
		return strings.TrimSpace(scanner.Text()), nil
	}
	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("reading input: %w", err)
	}

	return "", fmt.Errorf("unexpected end of input")
}

func GenerateReport(report files.SortResult) {
	fmt.Println("--Sorting Report--")
	fmt.Printf("Moved: %d files\n", len(report.Moved))
	fmt.Printf("Skipped: %d files\n", len(report.Skipped))
	fmt.Printf("Errors: %d\n", len(report.Errors))
	if len(report.Errors) > 0 {
		for _, err := range report.Errors {
			fmt.Printf(" - %v\n", err)
		}
	}
}
