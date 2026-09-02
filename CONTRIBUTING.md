# Contributing to Launchline

Thank you for improving Launchline. You need Go 1.24 or newer and a terminal on Windows, Linux, or macOS.

## Setup and checks

```console
go mod download
gofmt -w .
go test ./...
go vet ./...
go build ./...
```

Keep generated binaries in `bin/` or `dist/`; both are ignored.

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
