# Launchline

**One command. Your entire workspace.**

Launchline is an offline-first, cross-platform terminal utility for starting the applications you use together. Register applications once, group them into named workspaces, and launch the whole group from an interactive dashboard or one command.

```console
launchline start development
```

Launchline is local, keyboard-first, account-free, and intentionally small. It has no server, telemetry, cloud synchronization, or application installer.

## Features

- Interactive Bubble Tea dashboard with responsive Launchline branding
- Application add, edit, inspect, and confirmed removal flows
- Workspace create, rename, application selection, default selection, and confirmed deletion
- Concurrent launches with independent per-application results
- Direct commands suitable for scripts and terminal workflows
- Atomic, validated local JSON configuration with safety copies for corrupt files
- Native launch strategies for Windows, Linux, and macOS
- Useful empty states, contextual help, and keyboard controls

## Supported platforms

Launchline builds for Windows, Linux, and macOS on amd64 and arm64. It starts executable targets directly. It also uses the platform's standard opener for supported non-executable targets: `open` for macOS application bundles and URLs, `xdg-open` for Linux desktop files and URLs, and the Windows URL handler for URLs.

## Install or build

Launchline requires Go 1.24 or newer to build from source.

```console
git clone https://github.com/launchline/launchline.git
cd launchline
go build -o launchline .
```

Move the resulting binary to a directory on your `PATH`. Tagged releases are prepared by the release workflow as standalone binaries; no Go runtime is required to use a release binary.

## Quick start

Open the interactive dashboard:

```console
launchline
```

Or configure Launchline directly:

```console
launchline add --name Cursor --path /usr/bin/cursor
launchline add --name Terminal --path /usr/bin/kitty
launchline workspace create --name Development --app Cursor --app Terminal --default
launchline start
```

Paths vary by operating system. Arguments are repeatable structured values:

```console
launchline add --name Chrome --path /path/to/chrome \
  --arg=--profile-directory --arg="Profile 1"
```

Launchline passes those values directly to the process API. It does not evaluate shell syntax.

## Commands

```text
launchline                         Open the interactive dashboard
launchline start                   Start the default workspace
launchline start <workspace>       Start a workspace by name or ID
launchline add [flags]             Register an application
launchline apps                    List applications
launchline apps edit <app>         Edit an application
launchline apps delete <app>       Remove an entry with --yes
launchline workspace               List workspaces
launchline workspace create        Create a workspace
launchline workspace edit          Rename or replace its app selection
launchline workspace default       Choose the default workspace
launchline workspace delete        Delete a workspace with --yes
launchline config                  Show local configuration information
launchline config path             Print the configuration path
launchline version                 Show build version metadata
launchline help                    Show command help
```

Run `launchline <command> --help` for flags and examples.

## Dashboard

The dashboard exposes Start Workspace, Applications, Workspaces, Settings, and Help. The main controls are:

| Key | Action |
| --- | --- |
| Up / Down | Navigate |
| Left / Right | Change an option or form stage |
| Enter | Open, continue, or save |
| Space | Toggle an application in a workspace |
| Esc | Cancel or go back |
| Q | Quit outside text inputs |
| `?` | Contextual help |

The Applications screen adds, edits, and removes local registrations. The Workspaces screen creates groups, selects applications with checkboxes, and chooses exactly one default. Removal affects Launchline configuration only—it never uninstalls applications.

## Configuration

Launchline stores a single `config.json` under the operating system's user configuration directory, in a `launchline` folder. Typical locations are:

- Windows: `%AppData%\launchline\config.json`
- Linux: `$XDG_CONFIG_HOME/launchline/config.json`, usually `~/.config/launchline/config.json`
- macOS: `~/Library/Application Support/launchline/config.json`

Use `launchline config path` for the exact path on your computer.

Writes use a same-directory temporary file and atomic replacement where the operating system permits. The schema is versioned and validated on every read and write. Launchline does not overwrite malformed configuration; it creates a timestamped `.corrupt-*` safety copy and reports how to find it.

## Architecture

```text
Cobra CLI ─┐
           ├─> application services ─> launcher interface ─> OS process behavior
Bubble TUI ┘             │
                         └─> validated JSON repository
```

The CLI and TUI share all CRUD, resolution, and launch services. Platform process behavior is isolated in `internal/launcher`; the TUI never executes operating-system commands. Stable application IDs keep workspace references intact when names or paths change.

## Development

```console
go mod download
gofmt -w .
go test ./...
go vet ./...
go build ./...
```

CI runs formatting, tests, vet, and builds on Ubuntu, Windows, and macOS. Tests use fakes and temporary configuration directories; they never start real desktop applications.

See [CONTRIBUTING.md](CONTRIBUTING.md) and the Launchline design contract in [.agents/skills/launchline-design/SKILL.md](.agents/skills/launchline-design/SKILL.md).

## Security behavior

Launchline treats application paths and arguments as process arguments, not shell source. It does not concatenate commands, execute hooks, install programs, or accept arbitrary scripts. Configuration is created with user-only permissions where supported. Register only applications and targets you trust.

## MVP status

This repository implements the production-oriented MVP: manual registration, workspace management, a responsive TUI, direct CLI workflows, local persistence, and cross-platform launching.

Future possibilities—not implemented—may include optional application discovery, import/export, and additional package-manager distribution. Accounts, cloud sync, remote launching, telemetry, and arbitrary scripting are outside the product's current scope.

No license has been selected yet. Choose one before public distribution.
