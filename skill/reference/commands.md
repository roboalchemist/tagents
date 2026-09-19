# tagents Command Reference

Generated from `tagents --help` output. All commands also accept the global output flags listed at the end.

---

## tagents

```
tagents lets any agent or human manage the AI agent fleet without knowing
tmux or SSH exists. It discovers agent sessions running in tmux across local
and remote machines, reports status, sends messages, reads output, and tails logs.

ENVIRONMENT:
  No API keys required. Uses tmux and SSH from your PATH.

FILES:
  ~/.ssh/config    SSH machine discovery

EXIT STATUS:
  0  Success
  1  Agent not found or operation failed
  2  Usage error
  3  System error (tmux/ssh unavailable)

Usage:
  tagents [command]

Available Commands:
  broadcast   Send message to all idle agents
  completion  Generate shell completion scripts
  create      Create a new agent session
  docs        Display full documentation
  inject      Send @<file> to an agent
  list        List all agent sessions
  log         Read agent session log
  machines    List available SSH machines
  read        Read last N lines of agent's tmux pane output
  send        Send a message to an agent
  skill       Manage Claude Code skill installation
  status      Fleet status summary
  wait        Wait until agent is ready
  where       Get agent's current working directory
```

---

## tagents create

```
Create a new tmux session and launch an agent harness in it.

By default it launches OpenCode in the current directory. Use --harness and
--model to pick a different harness or model, --cwd to choose the working
directory, or --repo/--worktree/--branch to create a git worktree first. Pass
--prompt to send an initial message once the agent is running.

create targets the local machine only. Follow it with tagents send/read/wait to
guide and inspect the new agent.

Usage:
  tagents create <name> [flags]

Flags:
      --branch string     Branch to create for the worktree
      --command string    Explicit launch command (overrides --harness/--model)
      --cwd string        Working directory for the session (default: current directory)
      --dry-run           Print the plan without changing anything
      --force             Replace an existing session or worktree of the same name
      --from string       Start point for the new worktree branch (default "origin/master")
      --harness string    Agent harness: opencode, claude, codex, or pi (default "opencode")
      --model string      Model to pass to the harness (e.g. 'haiku[1m]')
      --prompt string     Initial message to send once the agent is running
      --repo string       Git repo to create the worktree from
      --wait duration     Max time to wait for the agent to start (default 20s)
      --worktree string   Worktree directory to create

Examples:
  tagents create my-agent
  tagents create worker --harness opencode --model 'haiku[1m]' --cwd ~/work
  tagents create PROJ-123 --repo ~/src/example-repo \
    --worktree ~/worktrees/PROJ-123 --branch dev/proj-123 \
    --prompt "/v/one-shot PROJ-123"
```

### Worktree + agent in one command

`create` can prepare an isolated git worktree and launch the agent inside it, so
the whole "spin off a worker" flow is one command:

```bash
tagents create PROJ-123 \
  --repo ~/src/example-repo \
  --worktree ~/src/worktrees/PROJ-123 \
  --branch dev/proj-123 \
  --from origin/master \
  --harness opencode --model 'haiku[1m]' \
  --prompt "/v/context PROJ-123"

tagents read PROJ-123        # watch it work
tagents send PROJ-123 "/v/one-shot PROJ-123"   # guide it
tagents wait PROJ-123 5m     # block until idle
```

With `--force` an existing session or worktree of the same name is replaced.
`--dry-run` prints the resolved harness, model, cwd, command, worktree, and
branch without creating anything.
```

---

## tagents list

```
List all agent sessions in scope.

Columns: machine (only when multi-machine scope), session name, runtime, status, log-idle age, cwd, preview.
Status colors: idle=green, busy=yellow, dead=red.

Usage:
  tagents list [flags]

Examples:
  tagents list
  tagents list --json | jq '.[0]'
  tagents list --all-machines --plaintext
```

---

## tagents machines

```
List all machines available from ~/.ssh/config.

Shows name, reachability, and agent count. Pings each host in parallel.

Usage:
  tagents machines [flags]

Examples:
  tagents machines
  tagents machines --json | jq '.[] | select(.reachable)'
```

Sample output:
```
NAME                REACHABLE  AGENTS
gateway             yes        7
mini                yes        2
server-a             yes        1
gpu-box             yes        11
nuc1                yes        3
iris                yes        32
example-host                no         0
```

---

## tagents status

```
Show fleet summary counts: total, busy, idle, dead, plus log-idle summary when transcripts are found.

Counts are per-machine when multi-machine scope is used.

Usage:
  tagents status [flags]

Examples:
  tagents status
  tagents status --json
  tagents status --all-machines --json
```

Sample output:
```
MACHINE  TOTAL  BUSY  IDLE  DEAD  LOGS  MAX_LOG_IDLE
local    1      1     0     0     1     7m12s
```

JSON output:
```json
{
  "overall": {
    "total": 1,
    "busy": 1,
    "idle": 0,
    "dead": 0,
    "logStale": 1,
    "logIdleMax": 432000000000
  }
}
```

---

## tagents read

```
Read the last N lines of an agent's tmux pane output.

Default: 50 lines. Agent name is fuzzy-matched.

Usage:
  tagents read <agent> [lines] [flags]

Examples:
  tagents read my-agent
  tagents read my-agent 100 --json
  tagents read gateway:worker 50
```

---

## tagents log

```
Read the agent's session transcript log.

For Claude Code agents: finds the most recent JSONL log file under ~/.claude/projects/
and renders it as human-readable conversation (or raw JSONL with --raw).

Default: last 100 lines.

Usage:
  tagents log <agent> [lines] [flags]

Flags:
      --raw    Output raw JSONL without parsing

Examples:
  tagents log my-agent
  tagents log my-agent 200 --raw
```

---

## tagents where

```
Print the current working directory of the agent's tmux pane.

Usage:
  tagents where <agent> [flags]

Examples:
  tagents where my-agent
  tagents where gateway:worker
```

---

## tagents send

```
Send a message to an agent as if typed + Enter in its tmux pane.

Agent name is fuzzy-matched. If the agent is busy, asks for confirmation
unless --force is given.

Usage:
  tagents send <agent> <message> [flags]

Flags:
  -f, --force   Send even if agent is busy

Examples:
  tagents send my-agent "please continue"
  tagents send my-agent "check status" --force
  tagents send gateway:worker "stop and report"
```

---

## tagents broadcast

```
Send a message to every idle agent in scope.

Skips busy and dead agents. Use --runtime to target a specific agent type.
Use --dry-run to preview without sending.

Usage:
  tagents broadcast <message> [flags]

Flags:
      --dry-run          Show what would be sent without sending
      --runtime string   Filter by runtime: claude, codex, opencode, or pi

Examples:
  tagents broadcast "please continue"
  tagents broadcast "status check" --runtime claude --dry-run
  tagents broadcast "check in" --all-machines --dry-run
```

---

## tagents wait

```
Block until at least one agent from the set becomes ready.

Accepts one or more agent names (fuzzy-matched). With no agent names and a
machine scope flag, waits for any ready agent on that machine or fleet.

Timeout can be given as a --timeout flag or as the last positional argument
(e.g. "60s", "2m", "1h"). The positional form is kept for backward compat.
Default: 60s. Exit 0 when an agent is ready, exit 1 on timeout.

With --log-idle, a busy-looking agent is also considered ready when its
Claude/Codex transcript has not advanced for at least the given duration. This
catches empty-turn/context-wedge stalls that pane-based idle detection misses.

Usage:
  tagents wait [agent ...] [timeout] [flags]

Flags:
      --log-idle duration   Treat an agent as ready when its transcript has been stale for this duration (0 disables)
  -t, --timeout duration    Polling timeout (e.g. 60s, 2m, 1h) (default 1m0s)

Examples:
  tagents wait my-agent
  tagents wait my-agent 120s
  tagents wait worker-1 worker-2 worker-3
  tagents wait worker-1 worker-2 --timeout 3m
  tagents wait my-agent --log-idle 90s --timeout 30m
  tagents wait --machine gateway
  tagents wait --all-machines --timeout 5m
```

Use `--log-idle` to babysit wedged Claude workers where the pane still looks busy but the JSONL transcript has stopped advancing:

```bash
tagents wait claude-worker --log-idle 90s --timeout 30m --json
```

The readiness condition is pane idle **OR** log idle >= threshold. JSON output includes `readyReason`, `logIdle`, and `logIdleFound`.

---

## tagents inject

```
Send '@<file>' to an agent — points it at a goal file.

The file path must exist. The message sent is literally '@<file>'.

Usage:
  tagents inject <agent> <file> [flags]

Examples:
  tagents inject my-agent /tmp/goal.md
  tagents inject worker ~/tasks/next.md
```

---

## tagents skill

```
Manage the Claude Code skill bundled with tagents.

The skill teaches Claude Code how to use tagents to manage agent fleets.

Usage:
  tagents skill [command]

Available Commands:
  add         Install skill to ~/.claude/skills/tagents/
  print       Print SKILL.md to stdout

Examples:
  tagents skill print
  tagents skill add
```

---

## tagents docs

```
Display the complete documentation from README.md.

Usage:
  tagents docs [flags]

Examples:
  tagents docs
  tagents docs | less
```

---

## tagents completion

```
Generate shell completion scripts for tagents.

Usage:
  tagents completion [bash|zsh|fish|powershell]

Examples:
  tagents completion bash > /etc/bash_completion.d/tagents
  tagents completion zsh > "${fpath[1]}/_tagents"
  tagents completion fish > ~/.config/fish/completions/tagents.fish
```

---

## Global Flags

All commands accept these flags:

| Flag | Short | Description |
|------|-------|-------------|
| `--all-machines` | | Target all hosts from SSH config |
| `--debug` | `-v` | Verbose logging to stderr |
| `--fields string` | | Comma-separated fields to include in output |
| `--jq string` | | JQ expression to filter JSON output |
| `--json` | `-j` | JSON output |
| `--machine string` | | Target a specific SSH host |
| `--no-color` | | Disable colored output |
| `--plaintext` | `-p` | Tab-separated output for piping |
| `--quiet` | `-q` | Suppress non-error output |
| `--help` | `-h` | Help for the command |
