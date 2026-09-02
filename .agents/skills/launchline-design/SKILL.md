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

Launchline should read as a professional interactive CLI session, not a settings dashboard or a website rendered in a terminal. Use a mostly monochrome white/gray identity with dense but readable vertical rhythm. Anchor content toward the upper-left, avoid decorative cards and borders, and use deliberate gaps instead of large unstructured empty areas.

Use a restrained semantic palette:

- logo and contextual titles: bright foreground / white
- primary: ordinary terminal foreground
- muted: secondary text and guidance
- accent: Launchline violet, reserved for the active selector, prompt marker, and rare focus cues
- success: completed actions plus `✓`
- warning: recoverable issues plus `!`
- error: failures plus `×`
- disabled: unavailable actions

Violet is an accent, never the dominant interface color. Do not color the logo or every selectable row violet. Color is supplementary: always pair state with text, position, or symbols. Keep whitespace deliberate and hierarchy obvious. Prefer a violet `●` beside bright selected text; leave non-selected items in the normal foreground.

Use the dominant bright block wordmark from `assets/ascii/launchline.txt` only on the root experience and only when it fits. Do not replace it with thin line-art lettering. Fall back automatically to plain `LAUNCHLINE` on narrow or short terminals, or when compact branding is enabled. Never crop, recolor violet, or overflow the canonical artwork.

## CLI/session hierarchy

Compose screens like an active command session:

1. block brand on the root, or a compact uppercase context label elsewhere;
2. a subtle command-context line such as `> launchline`, `> launchline apps`, or `> launchline workspace`;
3. a strong contextual title such as `Application Management` or `Workspace Editor — Development`;
4. one concise muted explanatory line;
5. the interactive list, form, empty state, or launch progress;
6. quiet screen-specific keyboard hints;
7. an integrated bottom status line.

Command context is presentation only. Never add a fake shell or parser. Avoid generic headings such as “Main Menu” when a contextual product title communicates more.

## Layout and responsive behavior

- React to Bubble Tea window-size messages.
- Make lists scroll around the active item when vertical space is limited.
- Truncate long paths and diagnostics visibly with an ellipsis; retain full values in editable fields.
- Stack information instead of forcing fixed columns on narrow screens.
- Keep the current action, relevant key hints, and status line visible on short screens.
- Test narrow, short, and large dimensions. Never assume an 80×24 or 90×28 terminal.

Support three modes:

- wide: full block logo, complete command context, comfortably bounded content width, and full workspace/version/platform status;
- normal: full logo only when both width and height permit, standard content, and compact full status;
- narrow or short: plain `LAUNCHLINE`, tighter section spacing, stacked or shortened copy, compact hints, and no horizontal overflow.

The root hierarchy is: dominant brand, `Tips to get started: /help`, `> launchline`, `Workspace`, a short instruction, numbered main selection, quiet controls, and status. Main selection order is Start Workspace, Applications, Workspaces, Settings, Help.

Keep the content column bounded on wide terminals. The interface stays upper-left rather than centering vertically. The status line may sit at the terminal bottom, but do not fill the main area with panels merely to occupy space.

## Command context and bottom status

Show the real conceptual command for the current surface: `launchline apps`, `launchline workspace`, `launchline start`, `launchline config`, or `launchline help`. Style the `>` prompt with the restrained accent and keep command text bright or neutral.

Every screen ends with a terminal-like status line. Its left side shows `~` plus the active/default workspace or current workspace context. Its right side shows the Launchline build version and platform when space permits. Collapse the platform first, then the right side, on narrow terminals. Do not render the status as a bordered web navigation bar.

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

Lists need a visible focus marker, useful item metadata, scrolling, and a purposeful empty state. Use numbered selection where it clarifies choice, such as `● 1. Development`, while application checkbox editors retain `[✓]` and `[ ]`. Only the active marker uses violet; selected text may be bold bright. Applications show the selected path and arguments. Workspaces show application count and a textual default marker.

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

Centralize theme semantics, block brand rendering, command context, contextual headers, framing, list navigation, visible-range calculation, truncation, key hints, status rendering, launch status, empty states, and confirmation patterns when reuse makes behavior more consistent. Keep screen-specific decisions near their screen.

- Never rely on color alone.
- Keep terminology and shortcut placement consistent.
- Prefer broadly supported Unicode symbols and retain readable text beside them.
- Preserve focus visibility in monochrome terminals.
- Do not force animation for completed states.
- Treat reduced/compact presentation as a functional setting, not decoration.

## Cross-platform review

Before completing a user-facing change, verify Windows, Linux, and macOS behavior compiles and remains behind the launcher abstraction. Use direct execution for executable targets, macOS `open` for bundles/URLs as appropriate, and Linux `xdg-open` for desktop/URL targets without making it the only launch path. Keep arguments structured and paths with spaces intact.

Check CLI/TUI behavior against one another; inspect wide, normal, narrow, and short terminal output; confirm the canonical logo collapses rather than overflows; confirm violet remains an accent; verify empty/error/destructive states; and ensure documentation claims only implemented behavior.

Configure once.
Launch everything with one command.
