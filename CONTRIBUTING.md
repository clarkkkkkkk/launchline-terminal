# Contributing to Launchline

Thank you for improving Launchline. You need Go 1.24 or newer and a terminal on Windows, Linux, or macOS.

## Contribution workflow

1. Fork the repository.
2. Create a branch for one focused change.
3. Make and test the change without crossing the architecture boundaries below.
4. Run the required checks:

```console
go mod download
gofmt -w .
go test ./...
go vet ./...
go build ./...
```

5. Open a pull request that explains the change and how it was verified.

Keep generated binaries in `bin/`, `build/`, or `dist/`; these directories are ignored.

## Security and repository hygiene

Never commit credentials, secrets, access tokens, private keys, personal environment files, or generated binaries. Use clearly fake placeholder values in tests and examples. Before opening a pull request, review the staged diff and confirm it contains no local configuration, machine-specific output, or private URLs.

Report suspected vulnerabilities using the private process in [SECURITY.md](SECURITY.md), not a public issue.

## Conventions

- Write idiomatic, formatted Go and prefer the standard library where it is clear and sufficient.
- Return actionable errors rather than panicking during normal user flows.
- Add focused tests for behavior and invariants. Never launch real graphical programs in tests.
- Preserve stable application/workspace IDs and validate relational references.
- Keep configuration replacement safe and never overwrite malformed user data silently.

## Architecture boundaries

Cobra commands and Bubble Tea screens call `internal/app` services. Persistence lives in `internal/config`. Process and platform behavior lives in `internal/launcher`. Do not execute OS commands or duplicate domain logic in TUI code.

When adding platform behavior, keep arguments structured, cover command construction with tests, and verify that all target platforms still compile.

## Terminal design

Read and apply [.agents/skills/launchline-design/SKILL.md](.agents/skills/launchline-design/SKILL.md) before changing user-facing TUI behavior. Check focus ownership, monochrome clarity, empty states, deletion wording, resizing, and narrow-terminal output.

Keep contributions inside Launchline's local-first MVP unless a scoped proposal has established a new product direction.
