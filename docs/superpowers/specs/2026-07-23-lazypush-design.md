# lazypush — Design Specification

**Date:** 2026-07-23
**Status:** Approved Design

## Overview

lazypush is a Go CLI tool with a Bubbletea TUI that helps create commits, releases, tags, and pull requests interactively with the help of LLMs (OpenAI-compatible). It reads the git diff, looks up the latest tag, lets the user interactively choose a version bump, generates a commit message via LLM, commits, tags, pushes, and optionally opens a PR via the GitHub CLI (`gh`).

## Technical Stack

- **Language:** Go
- **Module:** `github.com/lutrarutra/lazypush`
- **TUI:** Bubbletea + Bubbles + Lipgloss
- **Git:** go-git (pure Go)
- **LLM:** OpenAI-compatible chat completions API (direct HTTP, no external SDK)
- **Release:** GoReleaser + GitHub Actions
- **Config storage:** JSON at OS config dir (`os.UserConfigDir()`)

## Project Structure

```
lazypush/
├── cmd/
│   └── lazypush/
│       └── main.go              # Entrypoint, init config, launch TUI
├── internal/
│   ├── config/
│   │   ├── config.go            # Config struct, Load/Save, OS config dir path
│   │   └── config_test.go
│   ├── git/
│   │   ├── git.go               # Open, Diff, LatestTag, Commit, Tag, Push
│   │   ├── git_test.go
│   │   └── git_integration_test.go  # Integration tests with temp repos
│   ├── llm/
│   │   ├── client.go            # NewClient, Generate (OpenAI-compatible)
│   │   ├── client_test.go
│   │   └── prompts.go           # System/user prompt templates
│   ├── version/
│   │   ├── version.go           # Parse, Bump, String
│   │   └── version_test.go
│   ├── tui/
│   │   ├── main_model.go        # Top-level model, screen routing
│   │   ├── login_screen.go      # Settings: API URL, model, API key
│   │   ├── version_screen.go    # Current tag → pick version bump
│   │   ├── review_screen.go     # Diff + LLM message → edit → confirm
│   │   └── progress_screen.go   # Spinner + status during operations
│   └── gh/
│       ├── pr.go                # CreatePR — wraps `gh` CLI
│       └── pr_test.go
├── .github/
│   └── workflows/
│       └── release.yml          # GoReleaser cross-compile on tag
├── .goreleaser.yaml
├── go.mod
└── go.sum
```

## Subsystems

### 1. Config (`internal/config/`)

- **Location:** `os.UserConfigDir() + "/lazypush/config.json"` (override via `$LAZYPUSH_CONFIG_PATH`)
- **Format:** JSON
- **Fields:**
  - `api_url` (string, default `https://api.openai.com/v1`)
  - `api_key` (string)
  - `model` (string, default `gpt-4o-mini`)
  - `github_token` (string, optional — currently unused, `gh` handles auth)
- **API:**
  - `Load() (*Config, error)` — reads from config file, returns defaults if not found
  - `Save(*Config) error` — writes to config file
  - `ConfigDir() string` — resolves the directory path

### 2. Version (`internal/version/`)

- Parses semver tags: `v0.9.1`, `0.9.1`, `v1.2.3-rc1`
- Types:
  - `type BumpKind int` — `Patch`, `Minor`, `Major`
  - `type Version struct { Major, Minor, Patch int; PreRelease string }`
- API:
  - `Parse(tag string) (Version, error)` — parse a git tag
  - `(v Version) Bump(kind BumpKind) Version` — increment the specified segment
  - `(v Version) String() string` — `"v0.9.2"`

### 3. Git (`internal/git/`)

- Uses go-git for all operations
- **API:**
  - `Open(path string) (*Repo, error)` — opens repo or walks up to find `.git`
  - `(r *Repo) Diff() (string, error)` — staged changes only (`git diff --cached`). If nothing is staged, show unstaged changes and prompt the user to stage all.
  - `(r *Repo) LatestTag() (string, error)` — latest semver tag from `git describe --tags` equivalent
  - `(r *Repo) Commit(message string) error` — stage all changes + commit
  - `(r *Repo) Tag(name string) error` — create annotated tag
  - `(r *Repo) Push(remote string) error` — push commits + tags
- Testing: in-memory repos (`go-git`'s `memory.NewStorage` + `filesystem.NewStorage`)

### 4. LLM (`internal/llm/`)

- OpenAI-compatible chat completions API
- Direct HTTP POST to `{baseURL}/chat/completions`
- System prompt: "You are a helpful assistant that writes concise git commit messages..."
- User prompt: contains the git diff with instructions
- API:
  - `New(baseURL, apiKey, model string) *Client`
  - `(c *Client) Generate(ctx context.Context, system, user string) (string, error)`
- Prompts defined in `prompts.go`:
  - `CommitMessagePrompt(diff string) (system, user string)`
  - `PRDescriptionPrompt(diff string) (system, user string)`

### 5. GitHub PR (`internal/gh/`)

- Wraps `gh` CLI via `os/exec`
- Checks `gh` is installed before offering PR step
- API:
  - `CreatePR(title, body string) (string, error)` — returns PR URL
  - `CheckInstalled() bool` — returns whether `gh` is available

### 6. TUI (`internal/tui/`)

Linear screen flow:

```
[Login/Settings] → [Version pick] → [Review message] → [Progress/Summary]
     ↑                                                                  |
     └───────────────────── back to start ──────────────────────────────┘
```

**Screens:**

1. **Login Screen** — form: API URL, Model, API key (password-masked). Loads saved values. Save/Cancel.
2. **Version Screen** — shows current tag (or `v0.0.0` if none). Radio: Keep / Bump Patch / Bump Minor / Bump Major / Custom input. Patch selected by default.
3. **Review Screen** — scrollable viewport with the diff. Below: LLM-generated commit message in an editable textarea. Buttons: Edit description / Confirm & commit / Cancel. Optional toggle for PR description.
4. **Progress Screen** — spinner with status lines ("Committing... ✓", "Tagging... ✓", "Pushing... ✓", "Creating PR... ✓"). Final summary with commit hash, tag name, and PR URL if created.

### 7. Release Automation

- **GoReleaser** for cross-compilation: macOS (amd64 + arm64), Linux (amd64 + arm64), Windows (amd64)
- **GitHub Actions:** trigger on `v*` tag push
- Workflow: checkout → setup Go → goreleaser release → upload to GitHub Releases

## Workflow

1. User runs `lazypush` in a git repo with changes
2. If no config saved, shows login/settings screen first
3. Reads latest git tag → displays current version
4. User picks version bump (keep/bump/custom)
5. Reads staged git diff → sends to LLM → generates commit message (if nothing staged, offers to stage all changes first)
6. User reviews diff + message in TUI, can edit the message
7. User confirms → commits, tags, pushes
8. Optionally: creates PR via `gh` (if user confirms and `gh` is installed)
9. Shows final summary

## Testing Strategy

- **Unit tests** for config, version, llm, git (in-memory repos) using TDD
- **Integration tests** for git operations with temp directories
- **TUI testing:** no automated TUI tests initially (let's keep scope focused)
- **LLM testing:** mock the HTTP round-tripper

## Future Considerations (Not In Scope)

- Non-interactive mode (`lazypush --yes`)
- Encryption of stored API keys
- Multiple profiles
- Git hosting providers other than GitHub
