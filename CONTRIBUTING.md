# Contributing to tagents

Thanks for your interest in contributing!

## Development setup

```bash
git clone https://github.com/roboalchemist/tagents.git
cd tagents
make build
```

Requirements: Go 1.24+, `tmux` on your PATH for integration tests, and
`golangci-lint` for linting.

## Running checks

```bash
make test        # smoke tests (build + --help/--version/docs/completion/skill)
make test-unit   # go test -race ./pkg/...   (mocks only; no tmux/SSH)
go test ./cmd/...  # cmd package unit tests (not covered by make test-unit)
make lint        # golangci-lint
make check       # fmt + lint + smoke + unit
```

Integration tests exercise a real tmux session and require `tmux`:

```bash
make test-integration
```

## Submitting changes

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/your-feature`)
3. Make your changes, add tests
4. Run `make check` (and `go test ./cmd/...`)
5. Commit with a clear message
6. Open a pull request

## Code style

- Follow existing style; run `make fmt` before committing
- Add tests for new behavior
- Update docs (`README.md`, `docs/`, `skill/`) for user-facing changes

## Reporting issues

Use the issue tracker with clear reproduction steps and environment details.
For security issues, see [SECURITY.md](SECURITY.md).
