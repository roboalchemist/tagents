# tagents

[![CI](https://github.com/roboalchemist/tagents/actions/workflows/ci.yml/badge.svg)](https://github.com/roboalchemist/tagents/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/roboalchemist/tagents.svg)](https://pkg.go.dev/github.com/roboalchemist/tagents)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

Manage AI agent fleet across tmux sessions and SSH machines.

No API keys required. Uses `tmux` and `ssh` from your PATH. Reads `~/.ssh/config` for machine discovery.

## Install

### Homebrew

```bash
brew install roboalchemist/tap/tagents
```

### go install

```bash
go install github.com/roboalchemist/tagents@latest
```

### From source

```bash
git clone https://github.com/roboalchemist/tagents.git
cd tagents
make build      # produces ./tagents
```

Requires Go 1.24+ and `tmux` on your PATH at runtime.

## Quick Start

```bash
tagents list                          # list all local agent sessions
tagents status                        # fleet summary
tagents create worker --harness opencode --model 'haiku[1m]'  # spin off a new agent
tagents read my-agent                 # last 50 lines of agent pane
tagents send my-agent "continue"      # send message to agent
tagents wait my-agent 60s             # wait until pane-idle
tagents wait worker-1 worker-2 3m     # wait for any of a set to be pane-idle
tagents wait my-agent --log-idle 90s --timeout 30m  # also return if the log stalls
```

## Commands

| Command | Description |
|---------|-------------|
| `list` | List sessions with runtime, status, log-idle age, cwd, and preview |
| `machines` | List SSH hosts with reachability, agent count, and auth method (`key`/`sshpass`) |
| `status` | Fleet counts (total/busy/idle/dead plus log transcript summary) |
| `create <name>` | Create a new agent session — optional git worktree, harness/model selection, and an initial prompt |
| `read <agent> [N]` | Last N lines of agent's tmux pane (default 50) |
| `log <agent> [N]` | Session transcript — Claude Code JSONL parsed (default 100 lines) |
| `where <agent>` | Agent's current working directory |
| `send <agent> <msg>` | Send message to agent (warns if busy; use `--force` to override) |
| `broadcast <msg>` | Send message to all idle agents |
| `wait [agent ...] [timeout]` | Block until any listed agent is ready; pane-idle by default, or pane-idle OR stale transcript with `--log-idle <duration>`; supports `--machine`/`--all-machines` fleet mode (default 60s) |
| `inject <agent> <file>` | Send `@<file>` to agent |
| `skill print` | Print the bundled Claude Code skill to stdout |
| `skill add` | Install skill to `~/.claude/skills/tagents/` |
| `docs` | Display full documentation |
| `completion` | Generate shell completions (bash/zsh/fish/powershell) |

## Waiting and Log-Idle Babysitting

`tagents wait` normally returns when pane-based detection sees an idle shell or agent prompt. Add `--log-idle <duration>` to also treat an agent as ready when its Claude/Codex JSONL transcript has not advanced for that long:

```bash
tagents wait claude-worker --log-idle 90s --timeout 30m
```

This is useful for babysitting wedged Claude agents: empty-turn/context-wedge stalls can leave the pane showing a spinner or busy-looking output even though the transcript has stopped advancing. With `--log-idle`, wait returns on either signal: pane idle **OR** log stale for the threshold. The JSON result includes `readyReason` and log-idle fields so scripts can tell whether the agent returned because of `pane idle` or `log idle ... >= ...`.

## Machine Scope

By default tagents targets only the local machine. Expand scope with:

```bash
tagents list --machine gateway        # one SSH host
tagents list --all-machines           # all SSH hosts in parallel
tagents status --all-machines --json  # fleet-wide counts as JSON
```

All `Host` entries in `~/.ssh/config` (non-wildcard, Include directives supported) are candidates.

## SSH Password Auth (sshpass)

Some hosts (e.g. MDM-managed Macs) disable public-key auth with `PubkeyAuthentication no` and
only accept a password. tagents detects this and transparently wraps `ssh` with `sshpass`, so
`--machine`/`--all-machines` work against them too. `sshpass` must be on your PATH.

Password-file resolution, in order:

1. An explicit comment directive in the host's `~/.ssh/config` block (a comment, so `ssh`
   itself ignores it — a custom keyword would be rejected as a bad configuration option):
   ```
   Host example-host
     HostName Mac.local
     PubkeyAuthentication no
     # tagents-sshpass-file ~/.ssh/example-host-pw
   ```
2. Otherwise, when the host sets `PubkeyAuthentication no` and `~/.ssh/<host>-pw` exists, that
   file is used automatically.

Hosts using public keys are unaffected and never invoke `sshpass`. `tagents machines` shows the
resolved method in its `AUTH` column / `auth` JSON field.

## Agent Name Matching

```bash
tagents read sim-1            # fuzzy: matches oh-my-sim-1, sim-1-worker, etc.
tagents read gateway:worker   # pinned: exact machine:name match
tagents read local:sim-1      # pinned to local machine
```

## Status Values

- `idle` — agent is waiting for input
- `busy` — agent is actively running
- `dead` — pane missing or empty

## Output Flags

All commands accept:

| Flag | Short | Description |
|------|-------|-------------|
| `--json` | `-j` | JSON output |
| `--plaintext` | `-p` | Tab-separated for piping |
| `--no-color` | | Disable colors |
| `--debug` | `-v` | Verbose stderr logging |
| `--fields name,status` | | Select JSON fields |
| `--jq '.[0].name'` | | JQ filter on JSON output |
| `--quiet` | `-q` | Suppress non-error output |
| `--machine <host>` | | Target specific SSH host |
| `--all-machines` | | Target all SSH hosts |

## Configuration

tagents reads `~/.ssh/config` for machine discovery. See [docs/config.md](docs/config.md) for full configuration reference.

## Shell Completions

```bash
# bash
tagents completion bash > /etc/bash_completion.d/tagents

# zsh
tagents completion zsh > "${fpath[1]}/_tagents"

# fish
tagents completion fish > ~/.config/fish/completions/tagents.fish
```

## Claude Code Skill

tagents ships with a bundled skill that teaches Claude Code how to use it:

```bash
tagents skill add    # install to ~/.claude/skills/tagents/
tagents skill print  # print SKILL.md to stdout
```

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | Agent not found or operation failed |
| 2 | Usage error |
| 3 | System error (tmux/ssh unavailable) |

## Releasing

Push a version tag and GitHub Actions builds and publishes the release:

```bash
git tag v0.1.8
git push origin v0.1.8
```

[GoReleaser](.goreleaser.yml) cross-compiles darwin/linux binaries for amd64/arm64,
generates `checksums.txt`, and attaches everything to the GitHub release. The Homebrew
formula can then be bumped with the new tag.

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md). Security reports: [SECURITY.md](SECURITY.md).

## License

[MIT](LICENSE)
