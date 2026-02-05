package main

import (
	"context"
	"fmt"
	"os"

	"github.com/azvaliev/cmd/internal/pkg/ai"
	"github.com/azvaliev/cmd/internal/pkg/ui/views/generate"
	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	agentCh := make(chan views.AgentResult, 1)
	serverCh := make(chan *ai.LlamaServer, 1)
	go createAgent(agentCh, serverCh)
	defer cleanup(serverCh)

	m := views.NewGenerateModel(agentCh)
	program := tea.NewProgram(m, tea.WithAltScreen())

	finalModel, err := program.Run()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	result := finalModel.(views.GenerateModel).Result()
	if !result.Accepted {
		// TODO: should we show something
		return
	}

	fmt.Println(result.Command)
}

// we want to create it asynchronously to avoid blocking the UI
func createAgent(agentCh chan<- views.AgentResult, serverCh chan<- *ai.LlamaServer) {
	server, err := ai.CreateLLamaServer(ai.QWEN_3_MODEL_CONFIG)
	if err != nil {
		serverCh <- nil
		agentCh <- views.AgentResult{Err: err}
		return
	}

	serverCh <- server
	agent := ai.NewCommandAgent(server, context.Background())
	agentCh <- views.AgentResult{Agent: agent}
}

// ALWAYS cleanup the llama server
func cleanup(serverCh <-chan *ai.LlamaServer) {
	server := <-serverCh
	if server != nil {
		server.Dispose()
	}
}
