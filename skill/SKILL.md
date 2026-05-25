---
name: tagents
description: Manage AI agent fleet across tmux sessions and SSH machines. Use when checking agent status, sending messages to agents, reading agent pane output, or managing fleet across multiple machines.
scope: personal
allowed-tools: Bash(tagents:*)
---

# tagents

Manage AI agent fleet without knowing tmux or SSH exists. Discovers sessions in tmux across local and remote SSH machines, reports status, reads output, and sends messages.

## Quick Start

```bash
tagents list                          # list all local agent sessions
tagents list --all-machines           # include all SSH config machines
tagents status --json                 # fleet counts in JSON
tagents read my-agent                 # last 50 lines of agent pane
tagents send my-agent "continue"      # send message to agent
tagents wait my-agent 60s             # wait until idle
tagents inject my-agent /tmp/goal.md  # send @/tmp/goal.md
```

## Examples

```
$ tagents list
NAME         RUNTIME  STATUS  CWD                       PREVIEW
web-agent  unknown  busy    /home/user/projects/aplane2  {"seq":1058,"ts":"2026-05-22T19:29:21.233Z"...

$ tagents status
Fleet: 1 total  1 busy  0 idle  0 dead

$ tagents status --json
{
  "overall": {
    "total": 1,
    "busy": 1,
    "idle": 0,
    "dead": 0
  }
}

$ tagents list --json
[
  {
    "machine": "",
    "name": "web-agent",
    "runtime": "unknown",
    "status": "busy",
    "cwd": "/home/user/projects/aplane2",
    "preview": "{\"seq\":1058,\"ts\":\"2026-05-22T19:29:21.233Z\"..."
  }
]

$ tagents machines
NAME                REACHABLE  AGENTS
gateway             yes        7
mini                yes        2
server-a             yes        1
gpu-box             yes        11
nuc1                yes        3
iris                yes        32
example-host                no         0
```

## Machine Scope

By default, only the local machine. Use flags to expand:
- `--machine gateway` — target one SSH host from `~/.ssh/config`
- `--all-machines` — target all SSH hosts in parallel

Agent names can be pinned to a machine: `gateway:worker` matches exact machine:name.
Without a prefix, names are fuzzy-matched (substring): `sim-1` matches `oh-my-sim-1`.

## Status Values

- `idle` — agent is waiting for input (green)
- `busy` — agent is actively running (yellow)
- `dead` — pane missing or empty content (red)

## Runtimes Detected

- `claude` — session name contains "claude", or pane has `Human:`/`Assistant:` markers
- `codex` — session name contains "codex"
- `unknown` — not detected

## Output Flags

All commands accept these global output flags:

| Flag | Description |
|------|-------------|
| `--json` / `-j` | JSON output |
| `--plaintext` / `-p` | Tab-separated for piping |
| `--no-color` | Disable colors |
| `--debug` / `--verbose` | Verbose stderr logging |
| `--fields name,status` | Select JSON fields |
| `--jq '.[0].name'` | JQ filter on JSON output |
| `--quiet` / `-q` | Suppress non-error output |

## Commands Summary

| Command | Description |
|---------|-------------|
| `list` | List sessions with runtime, status, cwd, preview |
| `machines` | List SSH hosts with reachability and agent count |
| `status` | Fleet counts (total/busy/idle/dead) |
| `read <agent> [N]` | Last N lines of agent tmux pane (default 50) |
| `log <agent> [N]` | Session transcript (Claude Code JSONL parsed) |
| `where <agent>` | Agent's current working directory |
| `send <agent> <msg>` | Send message (warns if busy; use `--force`) |
| `broadcast <msg>` | Send to all idle agents |
| `wait <agent> [timeout]` | Block until idle (default 60s, exit 1 on timeout) |
| `inject <agent> <file>` | Send `@<file>` to agent |

Full flag reference: [skill/reference/commands.md](reference/commands.md)

## FILES

`~/.ssh/config` — source of machine discovery (Host entries, Include directives supported)

Full configuration reference: [docs/config.md](../docs/config.md)
