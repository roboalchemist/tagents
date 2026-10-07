package ssh

// Host represents a parsed SSH config host entry.
type Host struct {
	Name         string // The Host alias (e.g., "gateway", "mini")
	HostName     string // The actual hostname or IP
	User         string // Remote username
	Port         string // Port (default "22")
	IdentityFile string // Path to private key (if specified)

	// PasswordFile is an explicit sshpass password file. It is set from a
	// "# tagents-sshpass-file <path>" comment inside the host block (comments
	// are ignored by ssh itself, unlike a custom keyword, which ssh rejects
	// with "Bad configuration option").
	PasswordFile string
	// PubkeyDisabled is true when the host block sets
	// "PubkeyAuthentication no". Such hosts (e.g. MDM-managed Macs) can only
	// authenticate with a password, so tagents falls back to sshpass with
	// ~/.ssh/<name>-pw when that file exists.
	PubkeyDisabled bool
}
