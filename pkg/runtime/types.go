package runtime

// Runtime identifies what type of agent is running in a session.
type Runtime string

const (
	Claude   Runtime = "claude"
	Codex    Runtime = "codex"
	OpenCode Runtime = "opencode"
	Pi       Runtime = "pi"
	Unknown  Runtime = "unknown"
)

// Status represents the current activity state of an agent.
type Status string

const (
	Idle Status = "idle" // at shell prompt, waiting for input
	Busy Status = "busy" // currently processing/running
	Dead Status = "dead" // session exists but no pane content, or session missing
)
