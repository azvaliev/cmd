package components

import (
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/lipgloss"
)

var ViewStyle = lipgloss.NewStyle().PaddingTop(1).PaddingLeft(2)

var (
	promptStyle = lipgloss.NewStyle().Faint(true)
	copiedStyle = lipgloss.NewStyle().Italic(true)
)

var DotBounceSpinner = spinner.Spinner{
	Frames: []string{"   ", ".  ", ".. ", "...", " ..", "  .", "   "},
	FPS:    120 * time.Millisecond,
}

func RenderPrompt(prompt string) string {
	return promptStyle.Render("> " + prompt)
}

func RenderCommand(command string) string {
	return command
}

func RenderExplanation(explanation string, maxWidth int) string {
	return WrapText(explanation, maxWidth)
}

func RenderCopiedFeedback() string {
	return copiedStyle.Render("Copied to clipboard!")
}

func RenderSpinnerWithLabel(spinnerView, label string) string {
	return spinnerView + " " + label
}

func WrapText(text string, maxWidth int) string {
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

func NewHelp() help.Model {
	h := help.New()
	h.ShortSeparator = "  "
	faintStyle := lipgloss.NewStyle().Faint(true)
	h.Styles.ShortKey = faintStyle
	h.Styles.ShortDesc = faintStyle
	h.Styles.ShortSeparator = faintStyle
	return h
}
