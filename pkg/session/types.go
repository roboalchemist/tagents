package session

import (
	"time"

	"github.com/roboalchemist/tagents/pkg/runtime"
)

// AgentSession represents a single detected agent session.
type AgentSession struct {
	Machine      string          `json:"machine"` // empty = local
	Name         string          `json:"name"`
	Runtime      runtime.Runtime `json:"runtime"`
	Status       runtime.Status  `json:"status"`
	CWD          string          `json:"cwd"`
	Preview      string          `json:"preview"`                // one-line summary of last pane line
	LogIdle      time.Duration   `json:"logIdle,omitempty"`      // time since transcript last advanced
	LogIdleFound bool            `json:"logIdleFound,omitempty"` // true when a transcript was found
	ReadyReason  string          `json:"readyReason,omitempty"`  // why wait returned: pane or log
}
