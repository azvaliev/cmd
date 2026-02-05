package views

import (
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
)

type outputKeyMap struct {
	Done       key.Binding
	DidntWork  key.Binding
	CopyCmd    key.Binding
	CopyOutput key.Binding
	Cancel     key.Binding
}

var _ help.KeyMap = (*outputKeyMap)(nil)

func newOutputKeyMap() outputKeyMap {
	return outputKeyMap{
		Done: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("[enter]", "done"),
		),
		DidntWork: key.NewBinding(
			key.WithKeys("!"),
			key.WithHelp("[!]", "didn't work"),
		),
		CopyCmd: key.NewBinding(
			key.WithKeys("c"),
			key.WithHelp("[c]", "copy cmd"),
		),
		CopyOutput: key.NewBinding(
			key.WithKeys("o"),
			key.WithHelp("[o]", "copy output"),
		),
		Cancel: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("[esc]", "cancel"),
		),
	}
}

func (k outputKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Done, k.DidntWork, k.CopyCmd, k.CopyOutput, k.Cancel}
}

func (k outputKeyMap) FullHelp() [][]key.Binding {
	return nil
}
