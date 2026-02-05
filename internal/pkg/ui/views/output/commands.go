package views

import (
	"bufio"
	"os/exec"
	"strings"
	"syscall"

	"github.com/atotto/clipboard"
	tea "github.com/charmbracelet/bubbletea"
)

type outputLineMsg struct {
	text     string
	isStderr bool
}

type streamDoneMsg struct {
	isStderr bool
}

type commandDoneMsg struct {
	exitCode int
}

type clipboardCopiedMsg struct{}

// readNextLine reads a single line then returns a message, letting the Bubble Tea
// runtime re-enter Update before reading the next line. This avoids blocking the
// event loop and lets the viewport render incrementally as output arrives.
func readNextLine(scanner *bufio.Scanner, isStderr bool) tea.Cmd {
	return func() tea.Msg {
		if scanner.Scan() {
			text := processLine(scanner.Text())
			return outputLineMsg{text: text, isStderr: isStderr}
		}
		return streamDoneMsg{isStderr: isStderr}
	}
}

func waitForExit(proc *exec.Cmd) tea.Cmd {
	return func() tea.Msg {
		err := proc.Wait()
		exitCode := 0
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				exitCode = exitErr.ExitCode()
			} else {
				exitCode = 1
			}
		}
		return commandDoneMsg{exitCode: exitCode}
	}
}

func copyToClipboard(text string) tea.Cmd {
	return func() tea.Msg {
		clipboard.WriteAll(text)
		return clipboardCopiedMsg{}
	}
}

// ProcessState is nil until Wait() returns, so this guards against signaling
// a process that has already exited (which would be a no-op on some OSes
// but could signal a recycled PID on others).
func killProcess(proc *exec.Cmd) {
	if proc.Process != nil && proc.ProcessState == nil {
		proc.Process.Signal(syscall.SIGTERM)
	}
}

// processLine emulates terminal carriage-return and backspace behavior.
// Commands that use \r for progress bars (curl, wget) or \b for spinners
// would otherwise show raw control characters in the viewport since we're
// reading from a pipe, not a pty.
func processLine(text string) string {
	if idx := strings.LastIndex(text, "\r"); idx >= 0 {
		text = text[idx+1:]
	}
	return processBackspaces(text)
}

func processBackspaces(text string) string {
	runes := []rune(text)
	var out []rune
	for _, r := range runes {
		if r == '\b' {
			if len(out) > 0 {
				out = out[:len(out)-1]
			}
		} else {
			out = append(out, r)
		}
	}
	return string(out)
}
