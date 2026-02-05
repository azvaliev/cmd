package views

import (
	"bufio"
	"os/exec"
	"strings"
	"syscall"

	"github.com/atotto/clipboard"
	tea "github.com/charmbracelet/bubbletea"
)

type lineData struct {
	text           string
	carriageReturn bool
}

type outputBatchMsg struct {
	lines    []lineData
	isStderr bool
	done     bool // EOF reached — this is the final batch from this stream
}

type commandDoneMsg struct {
	exitCode int
}

type clipboardCopiedMsg struct{}

const maxBatchLines = 5

// readNextChunk reads lines in a batch, returning as many as are immediately
// available in the bufio.Reader's internal buffer (up to maxBatchLines).
// The cap ensures large output renders progressively instead of in one frame.
func readNextChunk(reader *bufio.Reader, isStderr bool) tea.Cmd {
	return func() tea.Msg {
		var batch []lineData
		for {
			text, cr, eof := readOneLine(reader)
			if eof {
				if text != "" {
					batch = append(batch, lineData{text: text, carriageReturn: cr})
				}
				return outputBatchMsg{lines: batch, isStderr: isStderr, done: true}
			}
			batch = append(batch, lineData{text: text, carriageReturn: cr})
			if reader.Buffered() == 0 || len(batch) >= maxBatchLines {
				break
			}
		}
		return outputBatchMsg{lines: batch, isStderr: isStderr}
	}
}

// readOneLine reads byte-by-byte until \r or \n, handling CR semantics.
// Byte-by-byte reading is necessary to detect \r boundaries immediately —
// buffered line reading would block until \n, freezing progress bars.
func readOneLine(reader *bufio.Reader) (text string, carriageReturn bool, eof bool) {
	var buf []byte
	for {
		b, err := reader.ReadByte()
		if err != nil {
			return processBackspaces(string(buf)), false, true
		}

		switch b {
		case '\n':
			return processBackspaces(string(buf)), false, false
		case '\r':
			if next, peekErr := reader.Peek(1); peekErr == nil && next[0] == '\n' {
				reader.ReadByte()
				return processBackspaces(string(buf)), false, false
			}
			if len(buf) == 0 {
				continue
			}
			return processBackspaces(string(buf)), true, false
		default:
			buf = append(buf, b)
		}
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

func processBackspaces(text string) string {
	if !strings.ContainsRune(text, '\b') {
		return text
	}
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
