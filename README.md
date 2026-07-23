# lazypush

Interactive commit, release, tag, and PR creation tool powered by LLMs.

## Usage

```bash
lazypush
```

Runs in the current directory — must be inside a git repository.

## Configuration

On first run, lazypush prompts for:
- **API URL** (default: `https://api.openai.com/v1`)
- **Model** (default: `gpt-4o-mini`)
- **API Key**

Config is saved to the OS config directory (`~/.config/lazypush/config.json` on Linux, `~/Library/Application Support/lazypush/config.json` on macOS).

Override config path: `LAZYPUSH_CONFIG_PATH=/path/to/config.json lazypush`

## Prerequisites

- Git repository with changes
- LLM API key (OpenAI-compatible)
- Git (for diff operations)
- Optional: [GitHub CLI (`gh`)](https://cli.github.com) for PR creation

## Workflow

1. Run `lazypush` in a git repo
2. Choose version bump (keep / patch / minor / major / custom)
3. Review LLM-generated commit message (edit if needed)
4. Confirm → commits, tags, pushes
5. Optionally creates a PR via `gh`

## Build

```bash
go build ./cmd/lazypush/
```

## License

MIT
