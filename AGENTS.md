# AGENTS.md

Go 1.24 Cobra CLI (`tagents`) that wraps `tmux` and `ssh` (both from PATH) to manage a
fleet of AI agent sessions locally and over SSH. No daemon, no API keys. Machine list
comes from `~/.ssh/config`. Entry point `main.go` → `cmd.Execute()`; version is injected
via `-X main.version` (see `Makefile`).

## Verification commands

- `make build` → `./tagents`
- `make test` is **smoke only** (runs `--help`, `--version`, `docs`, `completion bash`,
  `skill print`). It is **not** the unit suite.
- `make test-unit` → `go test -race ./pkg/...`. Mocks only, no tmux/SSH. **Does not cover
  package `cmd`** — run `go test ./cmd/...` to exercise `cmd/wait_test.go`.
- `make test-integration` → `go test -v -short -run TestIntegration ./...`. Requires `tmux`
  on PATH. `TestMain` (root `integration_test.go`, package `main_test`) builds `./tagents`,
  creates/removes a real tmux session `tagents-test-session`, and black-box runs the binary.
  `READONLY=1` skips send/inject/skill-add.
- The machines integration test additionally requires `TAGENTS_RUN_SLOW_MACHINES=1`;
  `make test-integration-full` alone still skips it.
- `make check` = fmt + lint + smoke + unit. It does **not** run integration.
- `make lint` needs `golangci-lint` on PATH (config `.golangci.yml` is v2 format).

## Architecture

- `cmd/` — thin Cobra wrappers, one file per command; shared helpers in `cmd/common.go`
  (`getSessions`, `findSession`). All global output/scope flags live in `cmd/root.go`.
- `pkg/tmux` — tmux wrapper with an `Executor` interface so tests inject mocks; keep it.
- `pkg/ssh` (~/.ssh/config, Host + Include), `pkg/session` (discovery + fuzzy match),
  `pkg/runtime` (runtime + status detection, transcript log idle), `pkg/output`
  (table/JSON/plaintext with `--fields`/`--jq`), `pkg/launch` (harness command lines).
- `main.go` embeds `README.md` and `skill/` via `go:embed` — rebuild to change `docs` /
  `skill print` output.
- Exit codes are string-matched in `main.go` `exitCode()`: Cobra usage errors → 2,
  `executable file not found`/`no such file or directory` → 3, else 1. New error text may
  need matching updates.

## Gotchas

- `create` and `--log-idle` transcript checks are **local-only**; remote sessions are skipped.
- `runtime.FindLogFile` returns the **most-recently-modified** Claude JSONL under
  `~/.claude/projects` — not tied to the session. Codex logs always return empty.
- Fuzzy match: `sim-1` matches `oh-my-sim-1`; `machine:name` pins to a host. Without scope
  flags, `findSession` tries local first, then falls back to all machines.
- Harness names: `opencode` (default), `claude`, `codex`, `pi`; `oc` is an alias.
  `create --command` overrides `--harness`/`--model`.
- Remote password auth: `ssh.Host.PubkeyDisabled` (from `PubkeyAuthentication no`) or an
  explicit `# tagents-sshpass-file <path>` comment triggers `sshpass` wrapping in
  `pkg/ssh/remote.go`. Password mode drops `BatchMode=yes` (it would suppress the prompt).
- Global flags: `-v` = verbose (alias `--debug`), `-V` = version-short, `-j` = json, `-p` = plaintext.
- Gitignored, do not commit: `tagents`, `dist/`, `*.out`, `coverage.*`, `man/`, `.claude/`.

## References

- `CLAUDE.md` — fuller command table, design decisions, release process (some details lag the code; trust source).
- `WORKLOG.md`, `GOAL.md` — historical build log and original blueprint.
