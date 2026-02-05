package views

import (
	"fmt"
	"strings"

	"github.com/azvaliev/cmd/internal/pkg/ai"
	"github.com/azvaliev/cmd/internal/pkg/ui/components"
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type AgentResult struct {
	Agent *ai.CommandAgent
	Err   error
}

type GenerateResult struct {
	Prompt      string
	Command     string
	Explanation string
	Accepted    bool
}

type state int

const (
	stateInput state = iota
	stateGenerating
	stateConfirm
	stateExplaining
)

type GenerateModel struct {
	agentCh <-chan AgentResult
	agent   *ai.CommandAgent

	state state
	err   error

	prompt                    string
	command                   string
	explanation               string
	showCopiedFeedbackMessage bool
	accepted                  bool

	commandInput textinput.Model
	spinner      spinner.Model
	help         help.Model
	keys         keyMap
}

func NewGenerateModel(agentCh <-chan AgentResult) GenerateModel {
	ti := textinput.New()
	ti.Prompt = "> "
	ti.Placeholder = "describe the command you'd like to generate"
	ti.Width = 80
	ti.PromptStyle = lipgloss.NewStyle().Faint(true)
	ti.PlaceholderStyle = lipgloss.NewStyle().Faint(true)
	ti.Focus()

	s := spinner.New()
	s.Spinner = components.DotBounceSpinner

	return GenerateModel{
		agentCh:      agentCh,
		state:        stateInput,
		commandInput: ti,
		spinner:      s,
		help:         components.NewHelp(),
		keys:         newKeyMap(),
	}
}

func (m GenerateModel) Result() GenerateResult {
	return GenerateResult{
		Prompt:      m.prompt,
		Command:     m.command,
		Explanation: m.explanation,
		Accepted:    m.accepted,
	}
}

func (m GenerateModel) Init() tea.Cmd {
	return tea.Batch(waitForAgentLoaded(m.agentCh), textinput.Blink)
}

func (m GenerateModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		{
			if msg.Type == tea.KeyCtrlC {
				return m, tea.Quit
			}
		}
	case agentLoadedResultMsg:
		{
			if msg.err != nil {
				m.err = msg.err
				return m, tea.Quit
			}

			m.agent = msg.agent

			promptSubmitted := m.state == stateGenerating && m.prompt != ""
			if promptSubmitted {
				// run the prompt immediatelyy
				return m, generateCommand(m.agent, m.prompt)
			}
			return m, nil
		}
	case generateResultMsg:
		{
			if msg.err != nil {
				m.err = msg.err
				return m, tea.Quit
			}

			if msg.command == "" {
				m.err = fmt.Errorf("received empty command from agent")
				return m, tea.Quit
			}

			m.command = msg.command
			m.state = stateConfirm
			return m, nil
		}
	case explainResultMsg:
		{
			if msg.err != nil {
				m.err = msg.err
				return m, tea.Quit
			}

			m.explanation = msg.explanation
			m.state = stateConfirm
			return m, nil
		}
	case clipboardCopiedMsg:
		{
			m.showCopiedFeedbackMessage = true
			return m, nil
		}
	}

	switch m.state {
	case stateInput:
		return m.updateInput(msg)
	case stateGenerating:
		return m.updateGenerating(msg)
	case stateConfirm:
		return m.updateConfirm(msg)
	case stateExplaining:
		return m.updateExplaining(msg)
	}

	return m, nil
}

func (m GenerateModel) View() string {
	var content string

	if m.err != nil {
		content = fmt.Sprintf("Error: %v", m.err)
	} else {
		switch m.state {
		case stateInput:
			content = m.viewInput()
		case stateGenerating:
			content = m.viewGenerating()
		case stateConfirm:
			content = m.viewConfirm()
		case stateExplaining:
			content = m.viewExplaining()
		}
	}

	return components.ViewStyle.Render(content)
}

func (m GenerateModel) updateInput(msg tea.Msg) (tea.Model, tea.Cmd) {
	// submit on enter
	if keyMsg, ok := msg.(tea.KeyMsg); ok && keyMsg.Type == tea.KeyEnter {
		value := strings.TrimSpace(m.commandInput.Value())
		if value == "" {
			return m, nil
		}
		m.prompt = value
		m.state = stateGenerating
		m.commandInput.Blur()

		if m.agent != nil {
			return m, tea.Batch(m.spinner.Tick, generateCommand(m.agent, m.prompt))
		}
		return m, m.spinner.Tick
	}

	var cmd tea.Cmd
	m.commandInput, cmd = m.commandInput.Update(msg)
	return m, cmd
}

func (m GenerateModel) updateGenerating(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	// keeeeeep spinning away
	m.spinner, cmd = m.spinner.Update(msg)
	return m, cmd
}

func (m GenerateModel) updateConfirm(msg tea.Msg) (tea.Model, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}

	// anytime we do another action, clear the copied feedback message
	m.showCopiedFeedbackMessage = false

	switch {
	case key.Matches(keyMsg, m.keys.Run):
		{
			m.accepted = true
			return m, tea.Quit
		}
	case key.Matches(keyMsg, m.keys.Explain):
		{
			if m.explanation != "" {
				return m, nil
			}
			m.state = stateExplaining
			return m, tea.Batch(m.spinner.Tick, explainCommand(m.agent, m.prompt, m.command))
		}
	case key.Matches(keyMsg, m.keys.Copy):
		{
			m.showCopiedFeedbackMessage = true
			return m, copyToClipboard(m.command)
		}
	case key.Matches(keyMsg, m.keys.Cancel):
		{
			return m, tea.Quit
		}
	}

	return m, nil
}

func (m GenerateModel) updateExplaining(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	// keeeeeep spinning away
	m.spinner, cmd = m.spinner.Update(msg)
	return m, cmd
}

func (m GenerateModel) viewInput() string {
	hint := lipgloss.NewStyle().Faint(true).Render("enter submit")
	return m.commandInput.View() + "\n\n" + hint
}

func (m GenerateModel) viewGenerating() string {
	return components.RenderSpinnerWithLabel(m.spinner.View(), "Generating")
}

func (m GenerateModel) viewConfirm() string {
	var sections []string

	sections = append(sections, components.RenderPrompt(m.prompt))
	sections = append(sections, components.RenderCommand(m.command))

	if m.explanation != "" {
		sections = append(sections, components.RenderExplanation(m.explanation, 78))
	}

	m.keys.Explain.SetEnabled(m.explanation == "")
	sections = append(sections, m.help.View(m.keys))

	if m.showCopiedFeedbackMessage {
		sections = append(sections, components.RenderCopiedFeedback())
	}

	return strings.Join(sections, "\n\n")
}

func (m GenerateModel) viewExplaining() string {
	var sections []string

	sections = append(sections, components.RenderPrompt(m.prompt))
	sections = append(sections, components.RenderCommand(m.command))
	sections = append(sections, components.RenderSpinnerWithLabel(m.spinner.View(), "Explaining"))

	return strings.Join(sections, "\n\n")
}
