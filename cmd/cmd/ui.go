package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/atotto/clipboard"
	"github.com/azvaliev/cmd/internal/pkg/ai"
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type state int

const (
	stateInput state = iota
	stateGenerating
	stateConfirm
	stateExplaining
)

// Styles

var (
	viewStyle        = lipgloss.NewStyle().PaddingTop(1).PaddingLeft(2)
	promptStyle      = lipgloss.NewStyle().Faint(true)
	commandStyle     = lipgloss.NewStyle()
	explanationStyle = lipgloss.NewStyle()
	copiedStyle      = lipgloss.NewStyle().Italic(true)
	spinnerStyle     = lipgloss.NewStyle()
)

// Custom spinner matching Swift dot animation

var dotBounceSpinner = spinner.Spinner{
	Frames: []string{"   ", ".  ", ".. ", "...", " ..", "  .", "   "},
	FPS:    120 * time.Millisecond,
}

// Messages

type agentInitSuccessMsg struct {
	agent  *ai.CommandAgent
	server *ai.LlamaServer
}

type agentInitFailedMsg struct {
	err error
}

func (a *agentInitFailedMsg) Error() string {
	return fmt.Sprintf("agent initialization failed: %v", a.err)
}

type generateSuccessMsg struct {
	command string
}

type generateFailedMsg struct {
	err error
}

type explainSuccessMsg struct {
	explanation string
}

type explainFailedMsg struct {
	err error
}

type clipboardCopiedMsg struct{}

// Commands

func createAgent() tea.Msg {
	server, err := ai.CreateLLamaServer(ai.QWEN_3_MODEL_CONFIG)
	if err != nil {
		return agentInitFailedMsg{err}
	}

	agent := ai.NewCommandAgent(server, context.Background())

	return agentInitSuccessMsg{agent: agent, server: server}
}

func generateCommand(agent *ai.CommandAgent, prompt string) tea.Cmd {
	return func() tea.Msg {
		command, err := agent.Generate(prompt)
		if err != nil {
			return generateFailedMsg{err}
		}
		return generateSuccessMsg{command}
	}
}

func explainCommand(agent *ai.CommandAgent, prompt, command string) tea.Cmd {
	return func() tea.Msg {
		explanation, err := agent.Explain(prompt, command)
		if err != nil {
			return explainFailedMsg{err}
		}
		return explainSuccessMsg{explanation}
	}
}

func copyToClipboard(text string) tea.Cmd {
	return func() tea.Msg {
		clipboard.WriteAll(text)
		return clipboardCopiedMsg{}
	}
}

// Model

type model struct {
	agent  *ai.CommandAgent
	server *ai.LlamaServer

	state state
	err   error

	prompt          string
	command         string
	explanation     string
	copiedFeedback  bool
	acceptedCommand string

	commandInput textinput.Model
	spinner      spinner.Model
	help         help.Model
	keys         keyMap
}

func NewModel() model {
	ti := textinput.New()
	ti.Prompt = "> "
	ti.Placeholder = "describe the command you'd like to generate"
	ti.Width = 80
	ti.PromptStyle = lipgloss.NewStyle().Faint(true)
	ti.PlaceholderStyle = lipgloss.NewStyle().Faint(true)
	ti.Focus()

	s := spinner.New()
	s.Spinner = dotBounceSpinner

	h := help.New()
	h.ShortSeparator = "  "
	faintStyle := lipgloss.NewStyle().Faint(true)
	h.Styles.ShortKey = faintStyle
	h.Styles.ShortDesc = faintStyle
	h.Styles.ShortSeparator = faintStyle

	return model{
		state:        stateInput,
		commandInput: ti,
		spinner:      s,
		help:         h,
		keys:         newKeyMap(),
	}
}

func (m model) Dispose() {
	if m.server != nil {
		m.server.Dispose()
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(createAgent, textinput.Blink)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.Type == tea.KeyCtrlC {
			return m, tea.Quit
		}
	case agentInitSuccessMsg:
		m.agent = msg.agent
		m.server = msg.server
		if m.state == stateGenerating && m.prompt != "" {
			return m, generateCommand(m.agent, m.prompt)
		}
		return m, nil
	case agentInitFailedMsg:
		m.err = msg.err
		return m, tea.Quit
	case generateSuccessMsg:
		m.command = msg.command
		m.state = stateConfirm
		return m, nil
	case generateFailedMsg:
		m.err = msg.err
		return m, tea.Quit
	case explainSuccessMsg:
		m.explanation = msg.explanation
		m.state = stateConfirm
		return m, nil
	case explainFailedMsg:
		m.err = msg.err
		return m, tea.Quit
	case clipboardCopiedMsg:
		m.copiedFeedback = true
		return m, nil
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

func (m model) updateInput(msg tea.Msg) (tea.Model, tea.Cmd) {
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

func (m model) updateGenerating(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.spinner, cmd = m.spinner.Update(msg)
	return m, cmd
}

func (m model) updateConfirm(msg tea.Msg) (tea.Model, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}

	m.copiedFeedback = false

	switch {
	case key.Matches(keyMsg, m.keys.Run):
		m.acceptedCommand = m.command
		return m, tea.Quit
	case key.Matches(keyMsg, m.keys.Explain):
		if m.explanation != "" {
			return m, nil
		}
		m.state = stateExplaining
		return m, tea.Batch(m.spinner.Tick, explainCommand(m.agent, m.prompt, m.command))
	case key.Matches(keyMsg, m.keys.Copy):
		m.copiedFeedback = true
		return m, copyToClipboard(m.command)
	case key.Matches(keyMsg, m.keys.Cancel):
		return m, tea.Quit
	}

	return m, nil
}

func (m model) updateExplaining(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.spinner, cmd = m.spinner.Update(msg)
	return m, cmd
}

func (m model) View() string {
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

	return viewStyle.Render(content)
}

func (m model) viewInput() string {
	hint := lipgloss.NewStyle().Faint(true).Render("enter submit")
	return m.commandInput.View() + "\n\n" + hint
}

func (m model) viewGenerating() string {
	return m.spinner.View() + " Generating..."
}

func (m model) viewConfirm() string {
	var sections []string

	sections = append(sections, promptStyle.Render("> "+m.prompt))
	sections = append(sections, m.command)

	if m.explanation != "" {
		wrapped := wrapText(m.explanation, 78)
		sections = append(sections, explanationStyle.Render(wrapped))
	}

	m.keys.Explain.SetEnabled(m.explanation == "")
	sections = append(sections, m.help.View(m.keys))

	if m.copiedFeedback {
		sections = append(sections, copiedStyle.Render("Copied to clipboard!"))
	}

	return strings.Join(sections, "\n\n")
}

func (m model) viewExplaining() string {
	var sections []string

	sections = append(sections, promptStyle.Render("> "+m.prompt))
	sections = append(sections, m.command)
	sections = append(sections, m.spinner.View()+" Explaining...")

	return strings.Join(sections, "\n\n")
}

// wrapText wraps text to maxWidth columns and prepends each line with "│ ".
func wrapText(text string, maxWidth int) string {
	var result []string

	for _, paragraph := range strings.Split(text, "\n") {
		if paragraph == "" {
			result = append(result, "│ ")
			continue
		}

		words := strings.Fields(paragraph)
		if len(words) == 0 {
			result = append(result, "│ ")
			continue
		}

		line := words[0]
		for _, word := range words[1:] {
			if len(line)+1+len(word) > maxWidth {
				result = append(result, "│ "+line)
				line = word
			} else {
				line += " " + word
			}
		}
		result = append(result, "│ "+line)
	}

	return strings.Join(result, "\n")
}
