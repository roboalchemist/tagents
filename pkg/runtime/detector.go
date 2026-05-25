package runtime

type Runtime string
type Status string

const (
	Claude  Runtime = "claude"
	Codex   Runtime = "codex"
	Unknown Runtime = "unknown"
)

const (
	Idle Status = "idle"
	Busy Status = "busy"
	Dead Status = "dead"
)

func DetectRuntime(sessionName, paneContent string) Runtime {
	return Unknown
}

func DetectStatus(paneContent string) Status {
	return Idle
}
