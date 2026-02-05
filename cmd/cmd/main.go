package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	m := NewModel()
	program := tea.NewProgram(m, tea.WithAltScreen())

	finalModel, err := program.Run()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	fm := finalModel.(model)
	fm.Dispose()

	if fm.acceptedCommand != "" {
		fmt.Println(fm.acceptedCommand)
	}
}
