---
name: launchline-design
description: Design or review Launchline CLI/TUI surfaces and related product behavior while preserving its terminal-first identity, interaction model, architecture boundaries, and MVP scope.
---

# Launchline Design

Use this skill for any Launchline user-facing terminal flow, reusable TUI component, command copy, or platform-launch behavior that affects the user experience.

## Product identity

Launchline is an offline-first CLI/TUI workspace launcher for Windows, Linux, and macOS.

- Brand: `LAUNCHLINE`
- Tagline: `One command. Your entire workspace.`
- Product posture: terminal-first, keyboard-first, local-first, account-free, lightweight, and fast.
- Stack: Go, Cobra, Bubble Tea, Bubbles, Lip Gloss, the Go standard library, local JSON, and GitHub Actions.

Launchline is inspired by high-quality modern CLI products but must not reproduce another product's branding or design.

Do not turn Launchline into a browser app, desktop GUI, SaaS product, remote service, account system, AI tool, scheduler, script runner, installer, or plugin platform.

## Architecture is part of the experience

Keep this boundary intact:

```text
TUI / Cobra commands
        ↓
application and domain services
        ↓
launcher interface
        ↓
operating-system behavior
```

The CLI and TUI must call the same services. TUI code must not execute platform commands. Keep OS selection and process details out of views. Treat paths and arguments as structured values; never build shell command strings or evaluate user input.

## Terminal-native visual system

Respect the terminal's existing background. Do not paint a full-screen background.

Use a restrained semantic palette:

- primary: ordinary readable content
- muted: secondary text and guidance
- accent: identity, focus, and current selection
- success: completed actions plus `✓`
- warning: recoverable issues plus `!`
- error: failures plus `×`
- disabled: unavailable actions

Color is supplementary. Always pair state with text, position, or symbols. Keep whitespace deliberate and hierarchy obvious. Prefer a single focus marker such as `●`; avoid ornamental boxes that consume narrow-terminal space.

Use large original Launchline ASCII branding only on the root dashboard and only when it fits. Fall back automatically to `LAUNCHLINE` on narrow terminals or when compact branding is enabled. Never crop or overflow artwork.

## Layout and responsive behavior

- React to Bubble Tea window-size messages.
- Make lists scroll around the active item when vertical space is limited.
- Truncate long paths and diagnostics visibly with an ellipsis; retain full values in editable fields.
- Stack information instead of forcing fixed columns on narrow screens.
- Keep the current action and essential footer controls visible on short screens.
- Test narrow, short, and large dimensions. Never assume an 80×24 or 90×28 terminal.

The root hierarchy is: brand, tagline, main menu, current-workspace/application/platform facts, contextual controls. Main menu order is Start Workspace, Applications, Workspaces, Settings, Help.

## Interaction model

Apply these keys consistently:

- Up/Down: navigate
- Left/Right: change an option or move between form stages where useful
- Enter: open, continue, confirm, or save
- Space: toggle application membership
- Esc: cancel or go back
- Q: quit from non-input contexts
- `?`: contextual help

When a text input has focus, it owns normal character keys. Global shortcuts must not prevent users from typing names, paths, or arguments. Always show the most relevant controls in a restrained footer.

## Lists, forms, and selection

Lists need a visible focus marker, useful item metadata, scrolling, and a purposeful empty state. Applications show the selected path and arguments. Workspaces show application count and a textual default marker.

Forms must:

- label every value plainly;
- preserve entered text when validation fails;
- identify the exact invalid field or value;
- support cancel without mutation;
- explain that argument quoting is parsing only, not shell evaluation.

Workspace editing has a human-readable name stage and a checkable application list. Render selected and unselected state as `[✓]` and `[ ]`; Space toggles and Enter saves. An empty workspace is allowed and must be explained rather than looking broken.

## Launch progress

Starting a workspace shows every configured application with explicit state:

- `·` pending
- an animated indicator while the operation is active
- `✓` launched
- `×` failed

Keep the UI responsive. Launch attempts are independent and a failure must not suppress the remaining applications. The final view states successful and failed counts and includes actionable per-application reasons. Empty workspaces receive a useful explanation.

## Errors and empty states

Errors answer: what failed, why when known, and what the user can do next. Do not show Go stack traces in normal operation. Preserve corrupt configuration and identify its safety copy. Do not collapse errors to generic phrases such as “Failed.”

Every empty state explains both the situation and the next action:

- no applications: explain registration and offer Add Application;
- no workspaces: explain grouping and offer Create Workspace;
- no default: direct the user to choose or create one;
- empty workspace: explain that applications can be added later.

Use precise copy. Prefer “application path” over implementation jargon, short active sentences over slogans, and “removed from Launchline” when the actual installed program is untouched.

## Destructive confirmations

Application and workspace deletion always requires an explicit confirmation. Name the target. Explain scope before offering the destructive key.

- Deleting an application removes only its Launchline entry and workspace references. It never uninstalls, deletes, or modifies the program.
- Deleting a workspace removes only that workspace configuration. It never deletes registered or installed applications.

Use affirmative and cancel labels such as `Y Remove from Launchline / N Cancel`, not an ambiguous yes/no prompt.

## Reusable components and accessibility

Centralize theme semantics, framing, list navigation, visible-range calculation, truncation, forms, status rendering, and confirmation patterns when reuse makes behavior more consistent. Keep screen-specific decisions near their screen.

- Never rely on color alone.
- Keep terminology and shortcut placement consistent.
- Prefer broadly supported Unicode symbols and retain readable text beside them.
- Preserve focus visibility in monochrome terminals.
- Do not force animation for completed states.
- Treat reduced/compact presentation as a functional setting, not decoration.

## Cross-platform review

Before completing a user-facing change, verify Windows, Linux, and macOS behavior compiles and remains behind the launcher abstraction. Use direct execution for executable targets, macOS `open` for bundles/URLs as appropriate, and Linux `xdg-open` for desktop/URL targets without making it the only launch path. Keep arguments structured and paths with spaces intact.

Check CLI/TUI behavior against one another, review narrow-terminal output, confirm empty/error/destructive states, and ensure documentation claims only implemented behavior.

Configure once.
Launch everything with one command.
