package tui

import "github.com/charmbracelet/bubbles/key"

// globalKeys are bound at the root model. Panel-local keys are
// declared by each panel.
type globalKeys struct {
	Quit       key.Binding
	Help       key.Binding
	NextPanel  key.Binding
	PrevPanel  key.Binding
	JumpPanel1 key.Binding
	JumpPanel2 key.Binding
	JumpPanel3 key.Binding
	JumpPanel4 key.Binding
	JumpPanel5 key.Binding
	JumpPanel6 key.Binding
	JumpPanel7 key.Binding
	JumpPanel8 key.Binding
	Refresh    key.Binding
}

func newGlobalKeys() globalKeys {
	return globalKeys{
		Quit:       key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
		Help:       key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "help")),
		NextPanel:  key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "next panel")),
		PrevPanel:  key.NewBinding(key.WithKeys("shift+tab"), key.WithHelp("shift+tab", "prev panel")),
		JumpPanel1: key.NewBinding(key.WithKeys("1"), key.WithHelp("1", "dashboard")),
		JumpPanel2: key.NewBinding(key.WithKeys("2"), key.WithHelp("2", "systems")),
		JumpPanel3: key.NewBinding(key.WithKeys("3"), key.WithHelp("3", "talkgroups")),
		JumpPanel4: key.NewBinding(key.WithKeys("4"), key.WithHelp("4", "active")),
		JumpPanel5: key.NewBinding(key.WithKeys("5"), key.WithHelp("5", "history")),
		JumpPanel6: key.NewBinding(key.WithKeys("6"), key.WithHelp("6", "events")),
		JumpPanel7: key.NewBinding(key.WithKeys("7"), key.WithHelp("7", "tones")),
		JumpPanel8: key.NewBinding(key.WithKeys("8"), key.WithHelp("8", "metrics")),
		Refresh:    key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "refresh")),
	}
}

func (k globalKeys) ShortHelp() []key.Binding {
	return []key.Binding{k.NextPanel, k.Help, k.Quit}
}
func (k globalKeys) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.NextPanel, k.PrevPanel, k.Quit, k.Help},
		{k.JumpPanel1, k.JumpPanel2, k.JumpPanel3, k.JumpPanel4},
		{k.JumpPanel5, k.JumpPanel6, k.JumpPanel7, k.JumpPanel8},
	}
}
