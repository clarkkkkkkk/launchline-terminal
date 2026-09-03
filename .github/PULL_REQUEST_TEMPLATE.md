## Summary

Describe the focused change and why it is needed.

## Verification

- [ ] `gofmt -w .`
- [ ] `go test ./...`
- [ ] `go vet ./...`
- [ ] `go build ./...`

## Repository safety

- [ ] I reviewed the diff for credentials, secrets, personal environment files, private URLs, and machine-specific data.
- [ ] I did not add generated binaries or unrelated build artifacts.
- [ ] The change preserves Launchline's CLI, TUI, persistence, discovery, launcher, and architecture contracts unless the pull request explicitly scopes an approved behavior change.
