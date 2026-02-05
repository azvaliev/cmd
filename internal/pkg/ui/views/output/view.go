package views

import (
	"bufio"
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"strings"

	"github.com/azvaliev/cmd/internal/pkg/ui/components"
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// maxRenderedLines caps what the viewport renders, not what we store, for performance.
// All output is kept in m.lines so "copy output" gives the complete result
// even when the viewport display is truncated.
const maxRenderedLines = 5000

var (
	stderrStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
	ruleStyle        = components.FaintStyle
	emptyOutputStyle = lipgloss.NewStyle().Faint(true).Italic(true)
)

var emptyOutputPhrases = []string{
	"Brewing",
	"Crunching",
	"Asking nicely",
	"Spawning chaos",
}

type OutputResult struct {
	ExitCode int
	Output   string
}

type state int

const (
	stateRunning state = iota
	stateDone
)

type outputLine struct {
	text     string
	isStderr bool
	fromCR   bool // true = written by \r, can be replaced by the next CR chunk
}

type OutputModel struct {
	prompt  string
	command string

	state    state
	proc     *exec.Cmd
	exitCode int

	stdoutReader *bufio.Reader
	stderrReader *bufio.Reader
	stdoutDone   bool
	stderrDone   bool

	lines         []outputLine
	autoScroll    bool
	viewportDirty bool
	emptyPhrase   string

	viewport viewport.Model
	ready    bool
	spinner  spinner.Model

	termWidth  int
	termHeight int
	help       help.Model
	keys       outputKeyMap

	showCopiedFeedbackMessage string
}

func NewOutputModel(prompt, command string) (OutputModel, error) {
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/sh"
	}

	proc := exec.Command(shell, "-c", command)

	stdoutPipe, err := proc.StdoutPipe()
	if err != nil {
		return OutputModel{}, fmt.Errorf("stdout pipe: %w", err)
	}
	stderrPipe, err := proc.StderrPipe()
	if err != nil {
		return OutputModel{}, fmt.Errorf("stderr pipe: %w", err)
	}

	if err := proc.Start(); err != nil {
		return OutputModel{}, fmt.Errorf("start command: %w", err)
	}

	s := spinner.New()
	s.Spinner = components.DotBounceSpinner

	return OutputModel{
		prompt:       prompt,
		command:      command,
		state:        stateRunning,
		proc:         proc,
		stdoutReader: bufio.NewReader(stdoutPipe),
		stderrReader: bufio.NewReader(stderrPipe),
		autoScroll:   true,
		emptyPhrase:  emptyOutputPhrases[rand.Intn(len(emptyOutputPhrases))],
		spinner:      s,
		help:         components.NewHelp(),
		keys:         newOutputKeyMap(),
	}, nil
}

func (m OutputModel) Result() OutputResult {
	var sb strings.Builder
	for _, line := range m.lines {
		sb.WriteString(line.text)
		sb.WriteByte('\n')
	}
	return OutputResult{
		ExitCode: m.exitCode,
		Output:   sb.String(),
	}
}

func (m *OutputModel) Dispose() {
	killProcess(m.proc)
}

func (m OutputModel) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		readNextChunk(m.stdoutReader, false),
		readNextChunk(m.stderrReader, true),
	)
}

func (m OutputModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.Type == tea.KeyCtrlC {
			killProcess(m.proc)
			return m, tea.Quit
		}

	case tea.WindowSizeMsg:
		m.termWidth = msg.Width
		m.termHeight = msg.Height
		m.resizeViewport()
		m.syncViewportContent()
		return m, nil

	case outputBatchMsg:
		for _, line := range msg.lines {
			m.appendOrReplaceLine(line.text, msg.isStderr, line.carriageReturn)
		}
		if len(msg.lines) > 0 {
			m.syncViewportContent()
		}

		if msg.done {
			if msg.isStderr {
				m.stderrDone = true
			} else {
				m.stdoutDone = true
			}
			// Wait for both pipes to be fully drained before calling proc.Wait().
			// If we call Wait() while a pipe still has unread data, it can deadlock
			// because the child process blocks writing to a full pipe buffer.
			if m.stdoutDone && m.stderrDone {
				return m, waitForExit(m.proc)
			}
			return m, nil
		}

		if msg.isStderr {
			return m, readNextChunk(m.stderrReader, true)
		}
		return m, readNextChunk(m.stdoutReader, false)

	case commandDoneMsg:
		m.state = stateDone
		m.exitCode = msg.exitCode
		// Clear dirty flag before the final sync so a stale spinner tick
		// doesn't trigger a redundant rebuild after we've already rendered.
		m.viewportDirty = false
		m.resizeViewport()
		m.syncViewportContent()
		return m, nil

	case clipboardCopiedMsg:
		return m, nil
	}

	switch m.state {
	case stateRunning:
		return m.updateRunning(msg)
	case stateDone:
		return m.updateDone(msg)
	}

	return m, nil
}

func (m OutputModel) updateRunning(msg tea.Msg) (tea.Model, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		if key.Matches(keyMsg, m.keys.Cancel) {
			killProcess(m.proc)
			return m, tea.Quit
		}
	}

	var cmds []tea.Cmd
	var cmd tea.Cmd

	m.spinner, cmd = m.spinner.Update(msg)
	cmds = append(cmds, cmd)

	// Flush dirty viewport on spinner ticks to debounce rapid output.
	// The spinner fires at ~120ms intervals, capping viewport rebuilds at ~8/sec.
	if m.viewportDirty {
		m.syncViewportContent()
		m.viewportDirty = false
	}

	m.viewport, cmd = m.viewport.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m OutputModel) updateDone(msg tea.Msg) (tea.Model, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		var cmd tea.Cmd
		m.viewport, cmd = m.viewport.Update(msg)
		return m, cmd
	}

	m.showCopiedFeedbackMessage = ""

	switch {
	case key.Matches(keyMsg, m.keys.Done):
		return m, tea.Quit
	case key.Matches(keyMsg, m.keys.DidntWork):
		// TODO: correction flow - stay on this screen, gather user feedback, re-generate command
		fmt.Fprintln(os.Stderr, "Command didn't work")
		return m, tea.Quit
	case key.Matches(keyMsg, m.keys.CopyCmd):
		m.showCopiedFeedbackMessage = "Copied command to clipboard!"
		return m, copyToClipboard(m.command)
	case key.Matches(keyMsg, m.keys.CopyOutput):
		m.showCopiedFeedbackMessage = "Copied output to clipboard!"
		return m, copyToClipboard(m.Result().Output)
	case key.Matches(keyMsg, m.keys.Cancel):
		return m, tea.Quit
	}

	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

func (m OutputModel) View() string {
	if !m.ready {
		return components.ViewStyle.Render("Initializing...")
	}

	header := components.RenderPrompt(m.prompt) + "\n\n" + components.RenderCommand(m.command)

	statusBar := m.statusBarView()
	content := m.viewport.View()
	scrollBar := m.scrollBarView()

	m.keys.Done.SetEnabled(m.state == stateDone)
	m.keys.DidntWork.SetEnabled(m.state == stateDone)
	m.keys.CopyCmd.SetEnabled(m.state == stateDone)
	m.keys.CopyOutput.SetEnabled(m.state == stateDone)

	copiedStyle := lipgloss.NewStyle().Italic(true)
	feedbackLine := " "
	if m.showCopiedFeedbackMessage != "" {
		feedbackLine = copiedStyle.Render(m.showCopiedFeedbackMessage)
	}

	footerParts := []string{
		m.help.View(m.keys),
		feedbackLine,
	}

	sections := []string{
		header,
		statusBar + "\n" + content + "\n" + scrollBar,
		strings.Join(footerParts, "\n"),
	}

	return components.ViewStyle.Render(strings.Join(sections, "\n\n"))
}

func (m OutputModel) statusBarView() string {
	var status string
	if m.state == stateRunning {
		status = components.RenderStatusBox("Running " + m.spinner.View())
	} else {
		status = components.RenderExitCode(m.exitCode)
	}
	leadIn := ruleStyle.Render("─")
	remaining := max(0, m.viewport.Width-lipgloss.Width(leadIn)-lipgloss.Width(status))
	trail := ruleStyle.Render(strings.Repeat("─", remaining))
	return lipgloss.JoinHorizontal(lipgloss.Center, leadIn, status, trail)
}

func (m OutputModel) scrollBarView() string {
	pct := fmt.Sprintf(" %3.f%% ", m.viewport.ScrollPercent()*100)
	rule := ruleStyle.Render(strings.Repeat("─", max(0, m.viewport.Width-lipgloss.Width(pct))))
	return lipgloss.JoinHorizontal(lipgloss.Center, rule, ruleStyle.Render(pct))
}

func (m *OutputModel) resizeViewport() {
	if m.termHeight == 0 || m.termWidth == 0 {
		return
	}

	vpHeight := m.termHeight - m.nonViewportHeight()
	if vpHeight < 1 {
		vpHeight = 1
	}
	vpWidth := m.termWidth - viewStyleHPadding()

	if !m.ready {
		m.viewport = viewport.New(vpWidth, vpHeight)
		m.ready = true
	} else {
		m.viewport.Width = vpWidth
		m.viewport.Height = vpHeight
	}
}

// appendOrReplaceLine handles CR semantics: \r-terminated chunks replace the
// previous \r chunk from the same stream (stdout/stderr) in-place.
func (m *OutputModel) appendOrReplaceLine(text string, isStderr, carriageReturn bool) {
	line := outputLine{text: text, isStderr: isStderr, fromCR: carriageReturn}

	if len(m.lines) > 0 {
		last := &m.lines[len(m.lines)-1]
		// Only match lines from the same stream — stdout and stderr are read
		// concurrently, so they interleave. Without this check, a stdout line
		// arriving between stderr progress updates would break the CR chain.
		if last.fromCR && last.isStderr == isStderr {
			last.text = text
			last.fromCR = carriageReturn
			return
		}
	}

	m.lines = append(m.lines, line)
}

func (m *OutputModel) syncViewportContent() {
	if !m.ready {
		return
	}

	if len(m.lines) == 0 {
		m.viewport.SetContent(emptyOutputStyle.Render(m.emptyPhrase))
		return
	}

	// Capture scroll position before setting new content. If the user has scrolled
	// up to read earlier output, we respect that and don't jump to bottom.
	// Only auto-scroll when they were already at the bottom (following live output).
	m.autoScroll = m.viewport.AtBottom()

	lines := m.lines
	truncated := false
	if len(lines) > maxRenderedLines {
		lines = lines[len(lines)-maxRenderedLines:]
		truncated = true
	}

	var sb strings.Builder
	if truncated {
		sb.WriteString("---- Output truncated, [o] to copy full output ----\n")
	}
	for i, line := range lines {
		if i > 0 {
			sb.WriteByte('\n')
		}
		// Wrap before styling so ANSI codes from stderrStyle don't interfere with width math.
		text := hardWrap(line.text, m.viewport.Width)
		if line.isStderr {
			sb.WriteString(stderrStyle.Render(text))
		} else {
			sb.WriteString(text)
		}
	}

	m.viewport.SetContent(sb.String())
	if m.autoScroll {
		m.viewport.GotoBottom()
	}
}

// nonViewportHeight returns lines consumed by everything except the viewport:
// ViewStyle vertical padding + header content + bars + separators + footer content.
func (m OutputModel) nonViewportHeight() int {
	vPad := components.ViewStyle.GetVerticalPadding()
	// prompt(1) + blank(1) + command(1) + \n\n blank(1) = 4
	header := 4
	// status bar(3: box top + content + box bottom) + scroll bar(1)
	bars := 4
	// \n\n blank(1) + help(1) + feedback(1) = 3
	footer := 3
	return vPad + header + bars + footer
}

func viewStyleHPadding() int {
	return components.ViewStyle.GetHorizontalPadding()
}

// hardWrap breaks text at exact character boundaries rather than word boundaries.
// Command output (tables, HTML, progress bars) rarely has convenient whitespace,
// so word-wrapping would leave long "words" overflowing the viewport.
// Wrapping is applied at render time so the raw lines stay intact for clipboard copy.
func hardWrap(text string, width int) string {
	if width <= 0 || len(text) <= width {
		return text
	}
	var sb strings.Builder
	runes := []rune(text)
	for i, r := range runes {
		if i > 0 && i%width == 0 {
			sb.WriteByte('\n')
		}
		sb.WriteRune(r)
	}
	return sb.String()
}
