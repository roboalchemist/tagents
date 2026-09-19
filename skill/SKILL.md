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
tagents create worker --harness opencode --model 'haiku[1m]' --prompt "/v/one-shot PROJ-123"
tagents read my-agent                 # last 50 lines of agent pane
tagents send my-agent "continue"      # send message to agent
tagents wait my-agent 60s             # wait until pane-idle
tagents wait worker-1 worker-2 3m     # wait for any of a set to be pane-idle
tagents wait my-agent --log-idle 90s --timeout 30m  # return if pane-idle OR log-stalled
tagents wait --all-machines --timeout 5m  # fleet mode: any idle agent across all machines
tagents inject my-agent /tmp/goal.md  # send @/tmp/goal.md
```

## Examples

```
$ tagents list
NAME         RUNTIME  STATUS  LOG_IDLE  CWD                       PREVIEW
web-agent  unknown  busy    7m12s     /home/user/projects/aplane2  {"seq":1058,"ts":"2026-05-22T19:29:21.233Z"...

$ tagents status
MACHINE  TOTAL  BUSY  IDLE  DEAD  LOGS  MAX_LOG_IDLE
local    1      1     0     0     1     7m12s

$ tagents status --json
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

$ tagents list --json
[
  {
    "machine": "",
    "name": "web-agent",
    "runtime": "unknown",
    "status": "busy",
    "cwd": "/home/user/projects/aplane2",
    "preview": "{\"seq\":1058,\"ts\":\"2026-05-22T19:29:21.233Z\"...",
    "logIdle": 432000000000,
    "logIdleFound": true
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

## Wait / Babysit Wedged Agents

`tagents wait` returns when pane-based status becomes idle. Add `--log-idle <duration>` to also return when the agent transcript has stopped advancing for that threshold:

```bash
tagents wait claude-worker --log-idle 90s --timeout 30m --json
```

Use this when babysitting Claude Code workers that may hit an empty-turn/context-wedge stall: the pane can look busy, but the JSONL transcript stops changing. The combined readiness condition is **pane idle OR log idle >= threshold**. JSON output includes `readyReason`, `logIdle`, and `logIdleFound` so automation can distinguish `pane idle` from a `log idle ... >= ...` stall.

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

Detection uses the pane's foreground process and title first (stable, independent of which text is visible), then session name and pane content as fallbacks.

- `claude` — foreground command `claude`, or name/content markers
- `codex` — foreground command `codex`, or name/content markers
- `opencode` — foreground command `opencode`/`vcodex`/`vopencode`, pane title `OC | ...`, or name/content
- `pi` — pane title `π - ...` (pi runs as `node`), a `pi` name token, or startup header
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
| `list` | List sessions with runtime, status, log-idle age, cwd, preview |
| `machines` | List SSH hosts with reachability and agent count |
| `status` | Fleet counts (total/busy/idle/dead plus log transcript summary) |
| `create <name>` | Create a new agent session (optional worktree + runtime + model + initial prompt) |
| `read <agent> [N]` | Last N lines of agent tmux pane (default 50) |
| `log <agent> [N]` | Session transcript (Claude Code JSONL parsed) |
| `where <agent>` | Agent's current working directory |
| `send <agent> <msg>` | Send message (warns if busy; use `--force`) |
| `broadcast <msg>` | Send to all idle agents |
| `wait [agent ...] [timeout]` | Block until any listed agent is ready; pane-idle by default, or pane-idle OR stale transcript with `--log-idle <duration>`; fleet mode with `--machine`/`--all-machines` (default 60s) |
| `inject <agent> <file>` | Send `@<file>` to agent |

Full flag reference: [skill/reference/commands.md](reference/commands.md)

## FILES

`~/.ssh/config` — source of machine discovery (Host entries, Include directives supported)

Full configuration reference: [docs/config.md](../docs/config.md)
