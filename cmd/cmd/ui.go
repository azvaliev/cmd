package main

import (
	"context"
	"fmt"

	"github.com/azvaliev/cmd/internal/pkg/ai"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type model struct {
	agent  *ai.CommandAgent
	server *ai.LlamaServer
	err    error

	// components
	commandInput textinput.Model
}

type agentInitSuccessMsg struct {
	// not nil, trust me bro
	agent *ai.CommandAgent
	// not nil, trust me bro
	server *ai.LlamaServer
}

type agentInitFailedMsg struct {
	err error
}

func (a *agentInitFailedMsg) Error() string {
	return fmt.Sprintf("agent initialization failed: %v", a.err)
}

func createAgent() tea.Msg {
	server, err := ai.CreateLLamaServer(ai.QWEN_3_MODEL_CONFIG)
	if err != nil {
		return agentInitFailedMsg{err}
	}

	agent := ai.NewCommandAgent(server, context.Background())

	return agentInitSuccessMsg{agent: agent, server: server}
}

func NewModel() model {
	commandInput := textinput.New()
	commandInput.Focus()

	return model{
		commandInput: commandInput,
	}
}

func (m model) Dispose() {
	if m.server != nil {
		m.server.Dispose()
	}
}

func (m model) Init() tea.Cmd {
	return createAgent
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case agentInitSuccessMsg:
		{
			m.agent = msg.agent
			m.server = msg.server
			return m, nil
		}

	case agentInitFailedMsg:
		{
			m.err = msg.err
			return m, tea.Quit
		}

	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC:
			{
				return m, tea.Quit
			}
		}
	}

	if m.commandInput.Focused() {
		m.commandInput, cmd = m.commandInput.Update(msg)
	}

	return m, cmd
}

func (m model) View() string {
	if m.err != nil {
		return fmt.Sprintf("Error: %v", m.err)
	}

	return fmt.Sprint(m.commandInput.View(), "\n\n")
}
