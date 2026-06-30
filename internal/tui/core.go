// Package tui provides the terminal user interface for the InFolderSort application.
package tui

import (
	"fmt"

	"github.com/Nox1KCL/Noxie-Sort/internal/config"
	"github.com/Nox1KCL/Noxie-Sort/internal/files"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/pelletier/go-toml/v2"
)

func RunTUI(configPath string, cfg *config.Config) error {
	paths := config.DefaultPaths()
	model := NewAppModel(configPath, cfg, paths)
	p := tea.NewProgram(
		model,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	if _, err := p.Run(); err != nil {
		return fmt.Errorf("tui: %w", err)
	}
	return nil
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

func marshalTOML(cfg *config.Config) ([]byte, error) {
	data, err := toml.Marshal(cfg)
	if err != nil {
		return nil, fmt.Errorf("marshalling config: %w", err)
	}
	return data, nil
}
