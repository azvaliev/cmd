package main

import (
	"context"
	"fmt"
	"os"

	"github.com/azvaliev/cmd/internal/pkg/ai"
	generateview "github.com/azvaliev/cmd/internal/pkg/ui/views/generate"
	outputview "github.com/azvaliev/cmd/internal/pkg/ui/views/output"
	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	agentCh := make(chan generateview.AgentResult, 1)
	serverCh := make(chan *ai.LlamaServer, 1)
	go createAgent(agentCh, serverCh)
	defer cleanup(serverCh)

	m := generateview.NewGenerateModel(agentCh)
	program := tea.NewProgram(m, tea.WithAltScreen())

	finalModel, err := program.Run()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	result := finalModel.(generateview.GenerateModel).Result()
	if !result.Accepted {
		return
	}

	// Run the output view as a separate Bubble Tea program rather than transitioning
	// within the generate program. The subprocess starts after the user confirms,
	// and running two sequential programs keeps each view's lifecycle simple.
	outputModel, err := outputview.NewOutputModel(result.Prompt, result.Command)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed to start command:", err)
		os.Exit(1)
	}
	defer outputModel.Dispose()

	outputProgram := tea.NewProgram(outputModel, tea.WithAltScreen(), tea.WithMouseCellMotion())
	finalOutputModel, err := outputProgram.Run()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	outputResult := finalOutputModel.(outputview.OutputModel).Result()

	// Print to stdout after the TUI exits so it appears in the user's scrollback.
	// The TUI uses alt-screen which vanishes on exit, so this gives a persistent record.
	fmt.Println("->", result.Command)
	if outputResult.Output != "" {
		fmt.Print(outputResult.Output)
	}
}

// we want to create it asynchronously to avoid blocking the UI
func createAgent(agentCh chan<- generateview.AgentResult, serverCh chan<- *ai.LlamaServer) {
	server, err := ai.CreateLLamaServer(ai.IBM_GRANITE_MODEL_CONFIG)
	if err != nil {
		serverCh <- nil
		agentCh <- generateview.AgentResult{Err: err}
		return
	}

	serverCh <- server
	agent := ai.NewCommandAgent(server, context.Background())
	agentCh <- generateview.AgentResult{Agent: agent}
}

// ALWAYS cleanup the llama server
func cleanup(serverCh <-chan *ai.LlamaServer) {
	server := <-serverCh
	if server != nil {
		server.Dispose()
	}
}
