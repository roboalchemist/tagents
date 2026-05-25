package tmux

import "time"

type Session struct {
	Name     string
	Windows  int
	Attached bool
	Created  time.Time
}
