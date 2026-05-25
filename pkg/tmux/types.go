package tmux

import "time"

// Session represents a tmux session.
type Session struct {
	Name     string
	Windows  int
	Attached bool
	Created  time.Time
}

// Pane represents a tmux pane's current state.
type Pane struct {
	SessionName string
	WindowIndex int
	PaneIndex   int
	CWD         string
	Running     bool   // true if a command is running in the pane
	Content     string // last N lines of pane content
}
