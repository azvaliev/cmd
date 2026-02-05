package views

import (
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
)

type keyMap struct {
	Run     key.Binding
	Explain key.Binding
	Copy    key.Binding
	Cancel  key.Binding
}

var _ help.KeyMap = (*keyMap)(nil)

func newKeyMap() keyMap {
	return keyMap{
		Run: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("[enter]", "run"),
		),
		Explain: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("[?]", "explain"),
		),
		Copy: key.NewBinding(
			key.WithKeys("c"),
			key.WithHelp("[c]", "copy"),
		),
		Cancel: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("[esc]", "cancel"),
		),
	}
}

func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Run, k.Explain, k.Copy, k.Cancel}
}

func (k keyMap) FullHelp() [][]key.Binding {
	return nil
}
