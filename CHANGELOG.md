# Changelog

All notable changes to this project are documented here. The format is based on
[Keep a Changelog](https://keepachangelog.com/), and this project adheres to
[Semantic Versioning](https://semver.org/).

## [Unreleased]

## [0.1.7]

### Added

- SSH password auth via `sshpass` for hosts that disable public-key auth
  (`PubkeyAuthentication no`). Password file resolved from an explicit
  `# tagents-sshpass-file <path>` comment in `~/.ssh/config`, else
  auto-detected at `~/.ssh/<host>-pw`.
- `machines` output now includes an `AUTH` column (`key`/`sshpass`) and an
  `auth` JSON field.
- `create` command to spin off agent sessions, with optional git worktree and
  initial prompt.
- `wait --log-idle <duration>` to treat a stalled transcript as ready.

### Fixed

- `golangci-lint` v2 exclusions so test-file checks are applied.
