package ssh

// Host represents a parsed SSH config host entry.
type Host struct {
	Name         string // The Host alias (e.g., "gateway", "mini")
	HostName     string // The actual hostname or IP
	User         string // Remote username
	Port         string // Port (default "22")
	IdentityFile string // Path to private key (if specified)
}
