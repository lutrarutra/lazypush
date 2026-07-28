# lazypush

Interactive commit, release, tag, and PR creation tool powered by LLMs.

## Install

**One-liner (macOS / Linux):**

```bash
curl -sSfL https://raw.githubusercontent.com/lutrarutra/lazypush/main/install.sh | sh
```

**With Go:**

```bash
go install github.com/lutrarutra/lazypush/cmd/lazypush@latest
```

## Quick start

```bash
cd my-project
lazypush
```

That's it — the TUI guides you through the whole flow.

## Subcommands

| Command | Description |
|---------|-------------|
| `lazypush` | Main workflow — version, commit, push, optional PR |
| `lazypush login` | Configure or change API provider (tests connection before saving) |
| `lazypush auth` | Test current API configuration |
| `lazypush reset` | Delete API configuration (with confirmation) |

## Prerequisites

- **Git** — a repository with changes or an existing branch
- **LLM API key** — OpenAI-compatible endpoint (optional — works without one, but commit messages and PR descriptions won't be AI-generated)
- **GitHub CLI (`gh`)** — only needed if you want to create pull requests

## Full workflow

```
lazypush
  │
  ├─ [login screen] ─── only on first run or via `lazypush login`
  │
  ├─ [version bump]
  │    Select: Keep / Patch / Minor / Major / Custom
  │    If LLM: "Generate commit message with AI?" (Y/n)
  │
  ├─ [review screen]
  │    • View the diff
  │    • Edit the commit message
  │    • Ctrl+s to save, Esc to cancel
  │    • If no LLM: shows warning ⚠ and hint to run `lazypush login`
  │
  ├─ [next step] ─── what to do with the changes?
  │    │
  │    ├─ Create PR to another branch
  │    │     → Select target branch (main/master default)
  │    │     → AI generates PR description (or empty if no LLM)
  │    │     → Review & edit PR description
  │    │     → Confirm → commit → tag → push → PR
  │    │
  │    ├─ Push to new branch
  │    │     → Enter branch name
  │    │     → Creates branch (`git checkout -b`)
  │    │     → Confirm → commit → tag → push to new branch
  │    │
  │    └─ Commit to current branch (default)
  │          → Confirm → commit → tag → push
  │
  ├─ [confirm screen]
  │    Shows everything that will happen:
  │      ● Commit → feat/my-feature
  │      ● Tag v0.1.0 → v0.1.1
  │      ● Push feat/my-feature → origin/feat/my-feature
  │      ● Create PR feat/my-feature → main
  │
  └─ [progress]
       Real-time status: ✅ Committing, ✅ Tagging, ✅ Pushing
```

### On `main` branch

If you're already on `main`/`master`, the "Create PR" option is hidden — only "Push to new branch" and "Commit to current branch" are shown.

### No changes (empty diff)

When there are no uncommitted changes:

- **Version/tag screen is skipped** — no commit needed
- You're asked directly if you want to **Create PR to another branch** (creating a PR from existing commits between branches)
- Only "Create PR to another branch" is offered

### Without LLM

If no API provider is configured, you'll get a warning on the review screen:

```
⚠ No AI provider configured — write your commit message manually
   Run 'lazypush login' to set up an AI provider
```

You can still do everything — just write messages by hand.

## Keybindings

| Key | Context | Action |
|-----|---------|--------|
| `↑`/`↓` | Selection screens | Navigate options |
| `Enter` | All screens | Confirm / select |
| `Esc` | All screens | Go back / cancel |
| `Ctrl+s` | Review / PR screens | Save and continue |
| `Ctrl+c` | Login screen | Cancel |

## Configuration

Config is saved to the OS config directory:

- macOS: `~/Library/Application Support/lazypush/config.json`
- Linux: `~/.config/lazypush/config.json`

Override with environment variable:

```bash
LAZYPUSH_CONFIG_PATH=/path/to/config.json lazypush
```

## Build

```bash
go build ./cmd/lazypush/
```

## License

MIT
