package views

import (
	"github.com/atotto/clipboard"
	"github.com/azvaliev/cmd/internal/pkg/ai"
	tea "github.com/charmbracelet/bubbletea"
)

type agentLoadedResultMsg struct {
	agent *ai.CommandAgent
	err   error
}

func waitForAgentLoaded(ch <-chan AgentResult) tea.Cmd {
	return func() tea.Msg {
		result := <-ch
		return agentLoadedResultMsg{agent: result.Agent, err: result.Err}
	}
}

type generateResultMsg struct {
	command string
	err     error
}

func generateCommand(agent *ai.CommandAgent, prompt string) tea.Cmd {
	return func() tea.Msg {
		command, err := agent.Generate(prompt)
		return generateResultMsg{command, err}
	}
}

type explainResultMsg struct {
	explanation string
	err         error
}

func explainCommand(agent *ai.CommandAgent, prompt, command string) tea.Cmd {
	return func() tea.Msg {
		explanation, err := agent.Explain(prompt, command)
		return explainResultMsg{explanation, err}
	}
}

type clipboardCopiedMsg struct{}

func copyToClipboard(text string) tea.Cmd {
	return func() tea.Msg {
		clipboard.WriteAll(text)
		return clipboardCopiedMsg{}
	}
}
