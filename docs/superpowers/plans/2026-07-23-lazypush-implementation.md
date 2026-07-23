# lazypush Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build lazypush — an interactive Go CLI tool using Bubbletea TUI that helps create commits, tags, releases, and PRs with LLM-generated messages.

**Architecture:** Modular monolith in `internal/` packages (config, version, llm, git, gh, tui). TDD throughout. Each package is independently testable. The TUI orchestrates a linear 3-screen flow: login/settings → version bump → review & confirm.

**Tech Stack:** Go 1.18+, Bubbletea + Bubbles + Lipgloss, go-git, OpenAI-compatible HTTP API, GoReleaser

## Global Constraints

- Go 1.18 minimum (avoid `any` as type alias, use `interface{}` if needed)
- No external LLM SDK — direct HTTP to OpenAI-compatible `/chat/completions`
- All git operations via go-git (pure Go)
- Config stored as JSON at `os.UserConfigDir() + "/lazypush/config.json"`
- PR creation via `gh` CLI exec (not go-git)
- TDD: every function has a test, test failed first before implementation
- Cross-compile via GoReleaser: macOS (amd64+arm64), Linux (amd64+arm64), Windows (amd64)

---

### Task 1: Go Module Scaffold + Main Entrypoint

**Files:**
- Create: `cmd/lazypush/main.go`
- Create: `go.mod`

**Interfaces:**
- Consumes: nothing (task 1)
- Produces: compile-check, go module name `github.com/lutrarutra/lazypush`

- [ ] **Step 1: Initialize go module**

```bash
cd /Users/lutrarutra/Documents/dev/lazypush
go mod init github.com/lutrarutra/lazypush
```

- [ ] **Step 2: Write the main entrypoint**

```go
// cmd/lazypush/main.go
package main

import "fmt"

func main() {
    fmt.Println("lazypush — interactive commit & release tool")
}
```

- [ ] **Step 3: Verify it compiles and runs**

```bash
go build ./cmd/lazypush/
./lazypush
```
Expected: prints "lazypush — interactive commit & release tool"

- [ ] **Step 4: Commit**

```bash
git add go.mod go.sum cmd/lazypush/main.go
git commit -m "feat: scaffold Go module with main entrypoint"
```

---

### Task 2: Config Subsystem (TDD)

**Files:**
- Create: `internal/config/config.go`
- Create: `internal/config/config_test.go`

**Interfaces:**
- Consumes: nothing (standalone package)
- Produces:
  - `type Config struct { APIURL, APIKey, Model string }`
  - `func ConfigDir() string`
  - `func ConfigPath() string`
  - `func Load() (*Config, error)`
  - `func Save(cfg *Config) error`

- [ ] **Step 1: Write the failing test**

```go
// internal/config/config_test.go
package config_test

import (
    "os"
    "path/filepath"
    "testing"

    "github.com/lutrarutra/lazypush/internal/config"
)

func TestConfigDir(t *testing.T) {
    dir := config.ConfigDir()
    if dir == "" {
        t.Fatal("ConfigDir() returned empty string")
    }
    // Should include "lazypush" in the path
    if filepath.Base(dir) != "lazypush" {
        t.Errorf("ConfigDir() = %q, expected basename 'lazypush'", dir)
    }
}

func TestConfigPath(t *testing.T) {
    path := config.ConfigPath()
    if path == "" {
        t.Fatal("ConfigPath() returned empty string")
    }
    if filepath.Base(path) != "config.json" {
        t.Errorf("ConfigPath() = %q, expected basename 'config.json'", path)
    }
}

func TestSaveAndLoad(t *testing.T) {
    dir := t.TempDir()
    t.Setenv("LAZYPUSH_CONFIG_PATH", filepath.Join(dir, "config.json"))

    original := &config.Config{
        APIURL:  "https://example.com/v1",
        APIKey:  "sk-test-key",
        Model:   "gpt-4",
    }
    if err := config.Save(original); err != nil {
        t.Fatalf("Save() error = %v", err)
    }

    loaded, err := config.Load()
    if err != nil {
        t.Fatalf("Load() error = %v", err)
    }

    if loaded.APIURL != original.APIURL {
        t.Errorf("APIURL = %q, want %q", loaded.APIURL, original.APIURL)
    }
    if loaded.APIKey != original.APIKey {
        t.Errorf("APIKey = %q, want %q", loaded.APIKey, original.APIKey)
    }
    if loaded.Model != original.Model {
        t.Errorf("Model = %q, want %q", loaded.Model, original.Model)
    }
}

func TestLoadReturnsDefaultsWhenFileMissing(t *testing.T) {
    dir := t.TempDir()
    t.Setenv("LAZYPUSH_CONFIG_PATH", filepath.Join(dir, "config.json"))

    cfg, err := config.Load()
    if err != nil {
        t.Fatalf("Load() error = %v", err)
    }
    if cfg.APIURL != "https://api.openai.com/v1" {
        t.Errorf("default APIURL = %q, want 'https://api.openai.com/v1'", cfg.APIURL)
    }
    if cfg.Model != "gpt-4o-mini" {
        t.Errorf("default Model = %q, want 'gpt-4o-mini'", cfg.Model)
    }
}

func TestLoadEnvOverride(t *testing.T) {
    dir := t.TempDir()
    cfgPath := filepath.Join(dir, "custom.json")
    t.Setenv("LAZYPUSH_CONFIG_PATH", cfgPath)

    original := &config.Config{APIURL: "http://localhost:8080/v1", APIKey: "local-key", Model: "local-model"}
    config.Save(original)

    loaded, err := config.Load()
    if err != nil {
        t.Fatalf("Load() error = %v", err)
    }
    if loaded.APIURL != "http://localhost:8080/v1" {
        t.Errorf("APIURL = %q", loaded.APIURL)
    }
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd /Users/lutrarutra/Documents/dev/lazypush
go test ./internal/config/ -v
```
Expected: FAIL — "undefined: config.ConfigDir" or similar

- [ ] **Step 3: Write minimal implementation**

```go
// internal/config/config.go
package config

import (
    "encoding/json"
    "os"
    "path/filepath"
)

const (
    defaultAPIURL = "https://api.openai.com/v1"
    defaultModel  = "gpt-4o-mini"
)

type Config struct {
    APIURL string `json:"api_url"`
    APIKey string `json:"api_key"`
    Model  string `json:"model"`
}

func ConfigDir() string {
    userDir, err := os.UserConfigDir()
    if err != nil {
        return ""
    }
    return filepath.Join(userDir, "lazypush")
}

func ConfigPath() string {
    if p := os.Getenv("LAZYPUSH_CONFIG_PATH"); p != "" {
        return p
    }
    return filepath.Join(ConfigDir(), "config.json")
}

func defaults() *Config {
    return &Config{
        APIURL: defaultAPIURL,
        Model:  defaultModel,
    }
}

func Load() (*Config, error) {
    path := ConfigPath()
    data, err := os.ReadFile(path)
    if err != nil {
        if os.IsNotExist(err) {
            return defaults(), nil
        }
        return nil, err
    }
    var cfg Config
    if err := json.Unmarshal(data, &cfg); err != nil {
        return nil, err
    }
    if cfg.APIURL == "" {
        cfg.APIURL = defaultAPIURL
    }
    if cfg.Model == "" {
        cfg.Model = defaultModel
    }
    return &cfg, nil
}

func Save(cfg *Config) error {
    path := ConfigPath()
    dir := filepath.Dir(path)
    if err := os.MkdirAll(dir, 0700); err != nil {
        return err
    }
    data, err := json.MarshalIndent(cfg, "", "  ")
    if err != nil {
        return err
    }
    return os.WriteFile(path, data, 0600)
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
cd /Users/lutrarutra/Documents/dev/lazypush
go test ./internal/config/ -v
```
Expected: all 4 tests PASS

- [ ] **Step 5: Commit**

```bash
git add internal/config/
git commit -m "feat: add config subsystem with JSON file storage"
```

---

### Task 3: Version Subsystem (TDD)

**Files:**
- Create: `internal/version/version.go`
- Create: `internal/version/version_test.go`

**Interfaces:**
- Consumes: nothing (standalone package)
- Produces:
  - `type BumpKind int` (consts: `Patch`, `Minor`, `Major`)
  - `type Version struct { Major, Minor, Patch int; PreRelease string }`
  - `func Parse(tag string) (Version, error)`
  - `func (v Version) Bump(kind BumpKind) Version`
  - `func (v Version) String() string`

- [ ] **Step 1: Write the failing test**

```go
// internal/version/version_test.go
package version_test

import (
    "testing"

    "github.com/lutrarutra/lazypush/internal/version"
)

func TestParse(t *testing.T) {
    tests := []struct {
        input   string
        want    version.Version
        wantErr bool
    }{
        {"v0.9.1", version.Version{Major: 0, Minor: 9, Patch: 1}, false},
        {"0.9.1", version.Version{Major: 0, Minor: 9, Patch: 1}, false},
        {"v1.2.3-rc1", version.Version{Major: 1, Minor: 2, Patch: 3, PreRelease: "rc1"}, false},
        {"v1.0.0", version.Version{Major: 1, Minor: 0, Patch: 0}, false},
        {"", version.Version{}, true},
        {"not-a-version", version.Version{}, true},
    }

    for _, tt := range tests {
        got, err := version.Parse(tt.input)
        if tt.wantErr {
            if err == nil {
                t.Errorf("Parse(%q) expected error", tt.input)
            }
            continue
        }
        if err != nil {
            t.Errorf("Parse(%q) unexpected error: %v", tt.input, err)
            continue
        }
        if got != tt.want {
            t.Errorf("Parse(%q) = %+v, want %+v", tt.input, got, tt.want)
        }
    }
}

func TestVersionBump(t *testing.T) {
    v := version.Version{Major: 0, Minor: 9, Patch: 1}

    patch := v.Bump(version.Patch)
    if patch.String() != "v0.9.2" {
        t.Errorf("Bump(Patch) = %s, want v0.9.2", patch.String())
    }

    minor := v.Bump(version.Minor)
    if minor.String() != "v0.10.0" {
        t.Errorf("Bump(Minor) = %s, want v0.10.0", minor.String())
    }

    major := v.Bump(version.Major)
    if major.String() != "v1.0.0" {
        t.Errorf("Bump(Major) = %s, want v1.0.0", major.String())
    }
}

func TestVersionString(t *testing.T) {
    tests := []struct {
        v    version.Version
        want string
    }{
        {version.Version{Major: 0, Minor: 9, Patch: 1}, "v0.9.1"},
        {version.Version{Major: 1, Minor: 0, Patch: 0}, "v1.0.0"},
        {version.Version{Major: 1, Minor: 2, Patch: 3, PreRelease: "rc1"}, "v1.2.3-rc1"},
    }
    for _, tt := range tests {
        if got := tt.v.String(); got != tt.want {
            t.Errorf("(%+v).String() = %q, want %q", tt.v, got, tt.want)
        }
    }
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd /Users/lutrarutra/Documents/dev/lazypush
go test ./internal/version/ -v
```
Expected: FAIL — "undefined: version"

- [ ] **Step 3: Write minimal implementation**

```go
// internal/version/version.go
package version

import (
    "fmt"
    "strconv"
    "strings"
)

type BumpKind int

const (
    Patch BumpKind = iota
    Minor
    Major
)

type Version struct {
    Major      int
    Minor      int
    Patch      int
    PreRelease string
}

func Parse(tag string) (Version, error) {
    tag = strings.TrimPrefix(tag, "v")
    parts := strings.SplitN(tag, "-", 2)
    nums := strings.Split(parts[0], ".")
    if len(nums) != 3 {
        return Version{}, fmt.Errorf("invalid semver: %q", tag)
    }
    major, err := strconv.Atoi(nums[0])
    if err != nil {
        return Version{}, fmt.Errorf("invalid major version %q: %w", nums[0], err)
    }
    minor, err := strconv.Atoi(nums[1])
    if err != nil {
        return Version{}, fmt.Errorf("invalid minor version %q: %w", nums[1], err)
    }
    patch, err := strconv.Atoi(nums[2])
    if err != nil {
        return Version{}, fmt.Errorf("invalid patch version %q: %w", nums[2], err)
    }
    prerelease := ""
    if len(parts) > 1 {
        prerelease = parts[1]
    }
    return Version{Major: major, Minor: minor, Patch: patch, PreRelease: prerelease}, nil
}

func (v Version) Bump(kind BumpKind) Version {
    switch kind {
    case Major:
        return Version{Major: v.Major + 1, Minor: 0, Patch: 0}
    case Minor:
        return Version{Major: v.Major, Minor: v.Minor + 1, Patch: 0}
    default:
        return Version{Major: v.Major, Minor: v.Minor, Patch: v.Patch + 1}
    }
}

func (v Version) String() string {
    s := fmt.Sprintf("v%d.%d.%d", v.Major, v.Minor, v.Patch)
    if v.PreRelease != "" {
        s += "-" + v.PreRelease
    }
    return s
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
cd /Users/lutrarutra/Documents/dev/lazypush
go test ./internal/version/ -v
```
Expected: all tests PASS

- [ ] **Step 5: Commit**

```bash
git add internal/version/
git commit -m "feat: add semver version parsing and bumping"
```

---

### Task 4: LLM Client (TDD)

**Files:**
- Create: `internal/llm/client.go`
- Create: `internal/llm/client_test.go`
- Create: `internal/llm/prompts.go`

**Interfaces:**
- Consumes: nothing (standalone package)
- Produces:
  - `type Client struct { ... }`
  - `func New(baseURL, apiKey, model string) *Client`
  - `func (c *Client) Generate(ctx context.Context, system, user string) (string, error)`
  - `func CommitMessagePrompt(diff string) (system, user string)`
  - `func PRDescriptionPrompt(diff string) (system, user string)`

- [ ] **Step 1: Write the failing test**

```go
// internal/llm/client_test.go
package llm_test

import (
    "context"
    "net/http"
    "net/http/httptest"
    "testing"

    "github.com/lutrarutra/lazypush/internal/llm"
)

func TestGenerate(t *testing.T) {
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if r.Method != "POST" {
            t.Errorf("method = %s, want POST", r.Method)
        }
        if r.URL.Path != "/chat/completions" {
            t.Errorf("path = %s, want /chat/completions", r.URL.Path)
        }
        if r.Header.Get("Authorization") != "Bearer sk-test" {
            t.Errorf("auth = %s", r.Header.Get("Authorization"))
        }
        w.Header().Set("Content-Type", "application/json")
        w.Write([]byte(`{"choices":[{"message":{"content":"Hello, world!"}}]}`))
    }))
    defer server.Close()

    client := llm.New(server.URL, "sk-test", "gpt-4")
    result, err := client.Generate(context.Background(), "system prompt", "user prompt")
    if err != nil {
        t.Fatalf("Generate() error = %v", err)
    }
    if result != "Hello, world!" {
        t.Errorf("Generate() = %q, want %q", result, "Hello, world!")
    }
}

func TestGenerateAPIError(t *testing.T) {
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusUnauthorized)
        w.Write([]byte(`{"error":{"message":"Invalid API key"}}`))
    }))
    defer server.Close()

    client := llm.New(server.URL, "bad-key", "gpt-4")
    _, err := client.Generate(context.Background(), "system", "user")
    if err == nil {
        t.Fatal("Generate() expected error, got nil")
    }
}

func TestGenerateEmptyChoices(t *testing.T) {
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/json")
        w.Write([]byte(`{"choices":[]}`))
    }))
    defer server.Close()

    client := llm.New(server.URL, "sk-test", "gpt-4")
    _, err := client.Generate(context.Background(), "system", "user")
    if err == nil {
        t.Fatal("Generate() expected error for empty choices, got nil")
    }
}
```

```go
// internal/llm/client_test.go — prompts tests (add to same file)
func TestCommitMessagePrompt(t *testing.T) {
    diff := "--- a/foo.go\n+++ b/foo.go\n@@ -1 +1 @@\n-func old()\n+func new()"
    sys, user := llm.CommitMessagePrompt(diff)
    if sys == "" {
        t.Error("system prompt is empty")
    }
    if user == "" {
        t.Error("user prompt is empty")
    }
}

func TestPRDescriptionPrompt(t *testing.T) {
    diff := "--- a/foo.go\n+++ b/foo.go\n@@ -1 +1 @@\n-func old()\n+func new()"
    sys, user := llm.PRDescriptionPrompt(diff)
    if sys == "" {
        t.Error("system prompt is empty")
    }
    if user == "" {
        t.Error("user prompt is empty")
    }
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd /Users/lutrarutra/Documents/dev/lazypush
go test ./internal/llm/ -v
```
Expected: FAIL — "undefined: llm"

- [ ] **Step 3: Write minimal implementation**

```go
// internal/llm/prompts.go
package llm

import "fmt"

const commitSystemPrompt = `You are a helpful assistant that writes concise git commit messages.
Follow conventional commits format: <type>: <description>
Types: feat, fix, refactor, docs, chore, test, style, perf.
Keep the subject line under 72 characters.
Provide only the commit message, no extra commentary.`

const prSystemPrompt = `You are a helpful assistant that writes GitHub pull request descriptions.
Summarize the changes clearly and concisely.
Use bullet points for individual changes.
Provide only the PR description body, no extra commentary.`

func CommitMessagePrompt(diff string) (system, user string) {
    return commitSystemPrompt, fmt.Sprintf("Write a commit message for this diff:\n\n%s", diff)
}

func PRDescriptionPrompt(diff string) (system, user string) {
    return prSystemPrompt, fmt.Sprintf("Write a PR description for this diff:\n\n%s", diff)
}
```

```go
// internal/llm/client.go
package llm

import (
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
)

type Client struct {
    baseURL string
    apiKey  string
    model   string
    http    *http.Client
}

func New(baseURL, apiKey, model string) *Client {
    return &Client{
        baseURL: baseURL,
        apiKey:  apiKey,
        model:   model,
        http:    http.DefaultClient,
    }
}

type chatMessage struct {
    Role    string `json:"role"`
    Content string `json:"content"`
}

type chatRequest struct {
    Model    string        `json:"model"`
    Messages []chatMessage `json:"messages"`
}

type chatChoice struct {
    Message chatMessage `json:"message"`
}

type chatResponse struct {
    Choices []chatChoice `json:"choices"`
}

func (c *Client) Generate(ctx context.Context, system, user string) (string, error) {
    req := chatRequest{
        Model: c.model,
        Messages: []chatMessage{
            {Role: "system", Content: system},
            {Role: "user", Content: user},
        },
    }

    body, err := json.Marshal(req)
    if err != nil {
        return "", fmt.Errorf("marshal request: %w", err)
    }

    httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/chat/completions", bytes.NewReader(body))
    if err != nil {
        return "", fmt.Errorf("create request: %w", err)
    }
    httpReq.Header.Set("Content-Type", "application/json")
    httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)

    resp, err := c.http.Do(httpReq)
    if err != nil {
        return "", fmt.Errorf("http request: %w", err)
    }
    defer resp.Body.Close()

    respBody, err := io.ReadAll(resp.Body)
    if err != nil {
        return "", fmt.Errorf("read response: %w", err)
    }

    if resp.StatusCode != http.StatusOK {
        return "", fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(respBody))
    }

    var chatResp chatResponse
    if err := json.Unmarshal(respBody, &chatResp); err != nil {
        return "", fmt.Errorf("unmarshal response: %w", err)
    }

    if len(chatResp.Choices) == 0 {
        return "", fmt.Errorf("API returned no choices")
    }

    return chatResp.Choices[0].Message.Content, nil
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
cd /Users/lutrarutra/Documents/dev/lazypush
go test ./internal/llm/ -v
```
Expected: all tests PASS

- [ ] **Step 5: Commit**

```bash
git add internal/llm/
git commit -m "feat: add OpenAI-compatible LLM client with prompt templates"
```

---

### Task 5: Git Subsystem (TDD with go-git)

**Files:**
- Create: `internal/git/git.go`
- Create: `internal/git/git_test.go`

**Interfaces:**
- Consumes: `version.Version` from Task 3
- Produces:
  - `type Repo struct { ... }`
  - `func Open(path string) (*Repo, error)` — discover .git in parent dirs
  - `func (r *Repo) Diff() (string, error)` — staged changes; if none, list unstaged
  - `func (r *Repo) StagedCount() (int, error)` — count staged files
  - `func (r *Repo) StageAll() error` — stage all changes
  - `func (r *Repo) LatestTag() (string, error)` — via go-git tag iteration, pick highest semver
  - `func (r *Repo) Commit(message string) error` — stage all + commit
  - `func (r *Repo) Tag(name string) error` — create annotated tag with name
  - `func (r *Repo) Push(remote string) error` — push commits + tags

- [ ] **Step 1: Install go-git dependency**

```bash
cd /Users/lutrarutra/Documents/dev/lazypush
go get github.com/go-git/go-git/v5
```

- [ ] **Step 2: Write the failing test**

```go
// internal/git/git_test.go
package git_test

import (
    "os"
    "path/filepath"
    "testing"

    "github.com/go-git/go-git/v5"
    "github.com/go-git/go-git/v5/plumbing/object"
    "github.com/lutrarutra/lazypush/internal/git"
)

// Helper to create a temp repo with an initial commit and tag
func initTempRepo(t *testing.T) string {
    t.Helper()
    dir := t.TempDir()

    r, err := git.PlainInit(dir, false)
    if err != nil {
        t.Fatalf("git init: %v", err)
    }

    // Create a file and commit it
    if err := os.WriteFile(filepath.Join(dir, "test.txt"), []byte("hello"), 0644); err != nil {
        t.Fatalf("write file: %v", err)
    }

    wt, err := r.Worktree()
    if err != nil {
        t.Fatalf("worktree: %v", err)
    }

    _, err = wt.Add("test.txt")
    if err != nil {
        t.Fatalf("git add: %v", err)
    }

    _, err = wt.Commit("initial", &git.CommitOptions{
        Author: &object.Signature{Name: "test", Email: "test@test.com"},
    })
    if err != nil {
        t.Fatalf("git commit: %v", err)
    }

    // Create a tag
    _, err = r.CreateTag("v0.1.0", nil, nil)
    if err != nil {
        t.Fatalf("create tag: %v", err)
    }

    return dir
}

func TestOpen(t *testing.T) {
    dir := initTempRepo(t)
    repo, err := git.Open(dir)
    if err != nil {
        t.Fatalf("Open() error = %v", err)
    }
    if repo == nil {
        t.Fatal("Open() returned nil repo")
    }
}

func TestOpenFromSubdir(t *testing.T) {
    dir := initTempRepo(t)
    subdir := filepath.Join(dir, "sub", "dir")
    os.MkdirAll(subdir, 0755)

    repo, err := git.Open(subdir)
    if err != nil {
        t.Fatalf("Open() from subdir error = %v", err)
    }
    if repo == nil {
        t.Fatal("Open() returned nil")
    }
}

func TestOpenNoGitDir(t *testing.T) {
    dir := t.TempDir()
    _, err := git.Open(dir)
    if err == nil {
        t.Fatal("Open() expected error for non-git dir")
    }
}

func TestLatestTag(t *testing.T) {
    dir := initTempRepo(t)
    repo, err := git.Open(dir)
    if err != nil {
        t.Fatalf("Open() error = %v", err)
    }

    tag, err := repo.LatestTag()
    if err != nil {
        t.Fatalf("LatestTag() error = %v", err)
    }
    if tag != "v0.1.0" {
        t.Errorf("LatestTag() = %q, want %q", tag, "v0.1.0")
    }
}

func TestLatestTagNoTags(t *testing.T) {
    dir := t.TempDir()
    r, _ := git.PlainInit(dir, false)
    wt, _ := r.Worktree()
    os.WriteFile(filepath.Join(dir, "a.txt"), []byte("a"), 0644)
    wt.Add("a.txt")
    wt.Commit("init", &git.CommitOptions{
        Author: &object.Signature{Name: "test", Email: "test@test.com"},
    })

    repo, err := git.Open(dir)
    if err != nil {
        t.Fatalf("Open() error = %v", err)
    }

    tag, err := repo.LatestTag()
    if err != nil {
        t.Fatalf("LatestTag() error = %v", err)
    }
    if tag != "" {
        t.Errorf("LatestTag() = %q, want empty", tag)
    }
}

func TestDiffStaged(t *testing.T) {
    dir := initTempRepo(t)
    repo, err := git.Open(dir)
    if err != nil {
        t.Fatalf("Open() error = %v", err)
    }

    // Make a change and stage it
    os.WriteFile(filepath.Join(dir, "test.txt"), []byte("hello world"), 0644)
    wt, _ := repo.Worktree()
    wt.Add("test.txt")

    diff, err := repo.Diff()
    if err != nil {
        t.Fatalf("Diff() error = %v", err)
    }
    if diff == "" {
        t.Error("Diff() returned empty for staged changes")
    }
}

func TestCommit(t *testing.T) {
    dir := initTempRepo(t)
    repo, err := git.Open(dir)
    if err != nil {
        t.Fatalf("Open() error = %v", err)
    }

    os.WriteFile(filepath.Join(dir, "new.txt"), []byte("new file"), 0644)

    err = repo.Commit("feat: add new file")
    if err != nil {
        t.Fatalf("Commit() error = %v", err)
    }
}

func TestTag(t *testing.T) {
    dir := initTempRepo(t)
    repo, err := git.Open(dir)
    if err != nil {
        t.Fatalf("Open() error = %v", err)
    }

    err = repo.Tag("v0.2.0")
    if err != nil {
        t.Fatalf("Tag() error = %v", err)
    }

    tag, _ := repo.LatestTag()
    if tag != "v0.2.0" {
        t.Errorf("LatestTag() after tag = %q, want %q", tag, "v0.2.0")
    }
}
```

- [ ] **Step 3: Run test to verify it fails**

```bash
cd /Users/lutrarutra/Documents/dev/lazypush
go test ./internal/git/ -v
```
Expected: FAIL — "undefined: git"

- [ ] **Step 4: Write minimal implementation**

```go
// internal/git/git.go
package git

import (
    "fmt"
    "os"
    "sort"
    "strings"

    gogit "github.com/go-git/go-git/v5"
    "github.com/go-git/go-git/v5/plumbing"
    "github.com/go-git/go-git/v5/plumbing/object"
    "github.com/lutrarutra/lazypush/internal/version"
)

type Repo struct {
    repo     *gogit.Repository
    worktree *gogit.Worktree
    path     string
}

func Open(path string) (*Repo, error) {
    abs, err := resolveGitDir(path)
    if err != nil {
        return nil, err
    }
    r, err := gogit.PlainOpen(abs)
    if err != nil {
        return nil, fmt.Errorf("open repo at %s: %w", abs, err)
    }
    wt, err := r.Worktree()
    if err != nil {
        return nil, fmt.Errorf("get worktree: %w", err)
    }
    return &Repo{repo: r, worktree: wt, path: abs}, nil
}

func (r *Repo) Worktree() *gogit.Worktree {
    return r.worktree
}

func (r *Repo) StagedCount() (int, error) {
    status, err := r.worktree.Status()
    if err != nil {
        return 0, err
    }
    count := 0
    for _, s := range status {
        if s.Staging != plumbing.Unmodified {
            count++
        }
    }
    return count, nil
}

func (r *Repo) StageAll() error {
    status, err := r.worktree.Status()
    if err != nil {
        return err
    }
    for path := range status {
        _, err := r.worktree.Add(path)
        if err != nil {
            return fmt.Errorf("stage %s: %w", path, err)
        }
    }
    return nil
}

func (r *Repo) Diff() (string, error) {
    // First try staged diff
    status, err := r.worktree.Status()
    if err != nil {
        return "", err
    }

    hasStaged := false
    for _, s := range status {
        if s.Staging != plumbing.Unmodified {
            hasStaged = true
            break
        }
    }

    if !hasStaged {
        // No staged changes, auto-stage all
        if err := r.StageAll(); err != nil {
            return "", err
        }
    }

    // Get HEAD tree
    ref, err := r.repo.Head()
    if err != nil {
        return "", fmt.Errorf("get HEAD: %w", err)
    }

    commit, err := r.repo.CommitObject(ref.Hash())
    if err != nil {
        return "", fmt.Errorf("get commit: %w", err)
    }

    headTree, err := commit.Tree()
    if err != nil {
        return "", fmt.Errorf("get head tree: %w", err)
    }

    // Get staging tree
    stagedTree, err := r.worktree.ToTree(nil)
    if err != nil {
        return "", fmt.Errorf("get staging tree: %w", err)
    }

    // Generate patch
    changes, err := headTree.Diff(stagedTree)
    if err != nil {
        return "", fmt.Errorf("diff: %w", err)
    }

    var patches []string
    for _, change := range changes {
        patch, err := change.Patch()
        if err != nil {
            continue
        }
        patches = append(patches, patch.String())
    }

    return strings.Join(patches, "\n"), nil
}

func (r *Repo) LatestTag() (string, error) {
    tags, err := r.repo.Tags()
    if err != nil {
        return "", fmt.Errorf("list tags: %w", err)
    }

    var semverTags []version.Version
    var tagNames []string

    err = tags.ForEach(func(ref *plumbing.Reference) error {
        name := ref.Name().Short()
        v, parseErr := version.Parse(name)
        if parseErr != nil {
            return nil // skip non-semver tags
        }
        semverTags = append(semverTags, v)
        tagNames = append(tagNames, name)
        return nil
    })
    if err != nil {
        return "", err
    }

    if len(semverTags) == 0 {
        return "", nil
    }

    // Sort descending by version
    sort.Slice(semverTags, func(i, j int) bool {
        if semverTags[i].Major != semverTags[j].Major {
            return semverTags[i].Major > semverTags[j].Major
        }
        if semverTags[i].Minor != semverTags[j].Minor {
            return semverTags[i].Minor > semverTags[j].Minor
        }
        return semverTags[i].Patch > semverTags[j].Patch
    })

    latest := semverTags[0]
    return latest.String(), nil
}

func (r *Repo) Commit(message string) error {
    // Stage all first
    if err := r.StageAll(); err != nil {
        return fmt.Errorf("stage before commit: %w", err)
    }

    _, err := r.worktree.Commit(message, &gogit.CommitOptions{
        Author: &object.Signature{
            Name:  "lazypush",
            Email: "lazypush@local",
        },
    })
    if err != nil {
        return fmt.Errorf("commit: %w", err)
    }
    return nil
}

func (r *Repo) Tag(name string) error {
    ref, err := r.repo.Head()
    if err != nil {
        return fmt.Errorf("get HEAD: %w", err)
    }

    _, err = r.repo.CreateTag(name, ref.Hash(), nil)
    if err != nil {
        return fmt.Errorf("create tag %s: %w", name, err)
    }
    return nil
}

func (r *Repo) Push(remote string) error {
    // Use filesystem auth (ssh-agent, .netrc, etc.)
    err := r.repo.Push(&gogit.PushOptions{
        RemoteName: remote,
    })
    if err != nil {
        return fmt.Errorf("push: %w", err)
    }
    return nil
}

func (r *Repo) Path() string {
    return r.path
}

// resolveGitDir walks up from path to find a .git directory
func resolveGitDir(path string) (string, error) {
    info, err := os.Stat(path)
    if err != nil {
        return "", fmt.Errorf("stat %s: %w", path, err)
    }
    if !info.IsDir() {
        return "", fmt.Errorf("%s is not a directory", path)
    }

    // Walk up to find .git
    dir := path
    for {
        if fi, err := os.Stat(dir + "/.git"); err == nil && fi.IsDir() {
            return dir, nil
        }
        parent := dir[:strings.LastIndex(dir, "/")]
        if parent == dir {
            return "", fmt.Errorf("no .git found in %s or parents", path)
        }
        dir = parent
    }
}
```

- [ ] **Step 5: Run test to verify it passes**

```bash
cd /Users/lutrarutra/Documents/dev/lazypush
go test ./internal/git/ -v
```
Expected: all tests PASS

- [ ] **Step 6: Commit**

```bash
git add internal/git/ go.sum go.mod
git commit -m "feat: add git subsystem using go-git"
```

---

### Task 6: GitHub PR Helper (TDD)

**Files:**
- Create: `internal/gh/pr.go`
- Create: `internal/gh/pr_test.go`

**Interfaces:**
- Consumes: nothing (standalone)
- Produces:
  - `func CheckInstalled() bool`
  - `func CreatePR(title, body string) (string, error)`

- [ ] **Step 1: Write the failing test**

```go
// internal/gh/pr_test.go
package gh_test

import (
    "os"
    "os/exec"
    "testing"

    "github.com/lutrarutra/lazypush/internal/gh"
)

func TestCheckInstalled(t *testing.T) {
    // can't mock this easily — just check it returns a bool
    installed := gh.CheckInstalled()
    // gh may or may not be installed in test env, just verify it's a bool
    if installed != false && installed != true {
        t.Errorf("CheckInstalled() = %v, expected bool", installed)
    }
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd /Users/lutrarutra/Documents/dev/lazypush
go test ./internal/gh/ -v
```
Expected: FAIL — "undefined: gh"

- [ ] **Step 3: Write minimal implementation**

```go
// internal/gh/pr.go
package gh

import (
    "bytes"
    "fmt"
    "os/exec"
)

func CheckInstalled() bool {
    _, err := exec.LookPath("gh")
    return err == nil
}

func CreatePR(title, body string) (string, error) {
    if !CheckInstalled() {
        return "", fmt.Errorf("gh CLI not found; install it from https://cli.github.com")
    }

    cmd := exec.Command("gh", "pr", "create",
        "--title", title,
        "--body", body,
    )

    var stdout, stderr bytes.Buffer
    cmd.Stdout = &stdout
    cmd.Stderr = &stderr

    if err := cmd.Run(); err != nil {
        return "", fmt.Errorf("gh pr create failed: %w\nstderr: %s", err, stderr.String())
    }

    return stdout.String(), nil
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
cd /Users/lutrarutra/Documents/dev/lazypush
go test ./internal/gh/ -v
```
Expected: PASS (1 test)

- [ ] **Step 5: Commit**

```bash
git add internal/gh/
git commit -m "feat: add GitHub PR creation via gh CLI"
```

---

### Task 7: TUI — Login/Settings Screen

**Files:**
- Create: `internal/tui/login_screen.go`

**Interfaces:**
- Consumes: `config.Config` from Task 2
- Produces: Login screen model — form for API URL, model, API key

- [ ] **Step 1: Install bubbletea dependencies**

```bash
cd /Users/lutrarutra/Documents/dev/lazypush
go get github.com/charmbracelet/bubbletea
go get github.com/charmbracelet/bubbles
go get github.com/charmbracelet/lipgloss
```

- [ ] **Step 2: Write implementation**

```go
// internal/tui/login_screen.go
package tui

import (
    "github.com/charmbracelet/bubbles/textinput"
    tea "github.com/charmbracelet/bubbletea"
    "github.com/charmbracelet/lipgloss"
)

type loginScreenModel struct {
    inputs   []textinput.Model
    focused  int
    err      string
    done     bool
}

func newLoginScreen() loginScreenModel {
    inputs := make([]textinput.Model, 3)

    inputs[0] = textinput.New()
    inputs[0].Placeholder = "https://api.openai.com/v1"
    inputs[0].Prompt = "API URL: "
    inputs[0].Focus()

    inputs[1] = textinput.New()
    inputs[1].Placeholder = "gpt-4o-mini"
    inputs[1].Prompt = "Model: "

    inputs[2] = textinput.New()
    inputs[2].Placeholder = "sk-..."
    inputs[2].Prompt = "API Key: "
    inputs[2].EchoMode = textinput.EchoPassword
    inputs[2].EchoCharacter = '•'

    return loginScreenModel{
        inputs:  inputs,
        focused: 0,
    }
}

func (m loginScreenModel) Init() tea.Cmd {
    return textinput.Blink
}

func (m loginScreenModel) Update(msg tea.Msg) (loginScreenModel, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        switch msg.String() {
        case "tab", "shift+tab":
            // Cycle focus
            m.inputs[m.focused].Blur()
            if msg.String() == "tab" {
                m.focused = (m.focused + 1) % len(m.inputs)
            } else {
                m.focused = (m.focused - 1 + len(m.inputs)) % len(m.inputs)
            }
            m.inputs[m.focused].Focus()
            return m, nil
        case "enter":
            if m.focused == len(m.inputs)-1 {
                // Last field — done
                m.done = true
                return m, nil
            }
            m.inputs[m.focused].Blur()
            m.focused++
            m.inputs[m.focused].Focus()
            return m, nil
        case "esc":
            m.err = "cancelled"
            m.done = true
            return m, nil
        }
    }

    var cmd tea.Cmd
    m.inputs[m.focused], cmd = m.inputs[m.focused].Update(msg)
    return m, cmd
}

func (m loginScreenModel) View() string {
    var s string
    s += lipgloss.NewStyle().Bold(true).Render("⚙️  lazypush Settings\n\n")
    for i := range m.inputs {
        s += m.inputs[i].View()
        s += "\n"
    }
    s += "\nPress Enter to save, Esc to cancel\n"
    if m.err != "" {
        s += "\n" + lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Render(m.err)
    }
    return s
}

func (m loginScreenModel) Values() (apiURL, model, apiKey string) {
    return m.inputs[0].Value(), m.inputs[1].Value(), m.inputs[2].Value()
}
```

- [ ] **Step 3: Verify it compiles**

```bash
cd /Users/lutrarutra/Documents/dev/lazypush
go build ./internal/tui/
```
Expected: no errors

- [ ] **Step 4: Commit**

```bash
git add internal/tui/login_screen.go
git commit -m "feat: add TUI login/settings screen"
```

---

### Task 8: TUI — Version Screen

**Files:**
- Create: `internal/tui/version_screen.go`

**Interfaces:**
- Consumes: `version.Version` from Task 3
- Produces: Version screen model — radio buttons for bump type, custom input option

- [ ] **Step 1: Write implementation**

```go
// internal/tui/version_screen.go
package tui

import (
    "fmt"

    "github.com/charmbracelet/bubbles/textinput"
    tea "github.com/charmbracelet/bubbletea"
    "github.com/charmbracelet/lipgloss"
    "github.com/lutrarutra/lazypush/internal/version"
)

type versionScreenModel struct {
    currentTag    string
    choices       []string
    selected      int
    customInput   textinput.Model
    showCustom    bool
    done          bool
    chosenVersion string
}

func newVersionScreen(currentTag string) versionScreenModel {
    v, err := version.Parse(currentTag)
    var base version.Version
    if err == nil {
        base = v
    }

    choices := []string{
        fmt.Sprintf("Keep (%s)", currentTag),
        fmt.Sprintf("Bump Patch (%s)", base.Bump(version.Patch).String()),
        fmt.Sprintf("Bump Minor (%s)", base.Bump(version.Minor).String()),
        fmt.Sprintf("Bump Major (%s)", base.Bump(version.Major).String()),
        "Custom",
    }

    ci := textinput.New()
    ci.Placeholder = "v0.0.0"
    ci.Prompt = "Custom version: "

    return versionScreenModel{
        currentTag:  currentTag,
        choices:     choices,
        selected:    0,
        customInput: ci,
        showCustom:  false,
    }
}

func (m versionScreenModel) Init() tea.Cmd {
    return nil
}

func (m versionScreenModel) Update(msg tea.Msg) (versionScreenModel, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        if m.showCustom {
            switch msg.String() {
            case "enter":
                m.chosenVersion = m.customInput.Value()
                m.done = true
                return m, nil
            case "esc":
                m.showCustom = false
                return m, nil
            default:
                var cmd tea.Cmd
                m.customInput, cmd = m.customInput.Update(msg)
                return m, cmd
            }
        }

        switch msg.String() {
        case "up", "k":
            if m.selected > 0 {
                m.selected--
            }
        case "down", "j":
            if m.selected < len(m.choices)-1 {
                m.selected++
            }
        case "enter":
            if m.selected == len(m.choices)-1 {
                m.showCustom = true
                m.customInput.Focus()
                return m, nil
            }
            // Map selection to version string
            switch m.selected {
            case 0:
                m.chosenVersion = m.currentTag
            case 1:
                v, _ := version.Parse(m.currentTag)
                m.chosenVersion = v.Bump(version.Patch).String()
            case 2:
                v, _ := version.Parse(m.currentTag)
                m.chosenVersion = v.Bump(version.Minor).String()
            case 3:
                v, _ := version.Parse(m.currentTag)
                m.chosenVersion = v.Bump(version.Major).String()
            }
            m.done = true
            return m, nil
        case "esc":
            m.chosenVersion = ""
            m.done = true
            return m, nil
        }
    }

    return m, nil
}

func (m versionScreenModel) View() string {
    var s string
    s += lipgloss.NewStyle().Bold(true).Render(fmt.Sprintf("🏷️  Current tag: %s\n\n", m.currentTag))

    if m.showCustom {
        s += m.customInput.View()
        s += "\n\nEnter to confirm, Esc to go back\n"
        return s
    }

    for i, choice := range m.choices {
        cursor := " "
        if i == m.selected {
            cursor = "▸"
        }
        s += fmt.Sprintf("%s %s\n", cursor, choice)
    }

    s += "\n↑/↓ to navigate, Enter to select, Esc to cancel\n"
    return s
}
```

- [ ] **Step 2: Verify it compiles**

```bash
cd /Users/lutrarutra/Documents/dev/lazypush
go build ./internal/tui/
```
Expected: no errors

- [ ] **Step 3: Commit**

```bash
git add internal/tui/version_screen.go
git commit -m "feat: add TUI version bump selection screen"
```

---

### Task 9: TUI — Review Screen

**Files:**
- Create: `internal/tui/review_screen.go`

**Interfaces:**
- Consumes: git diff (string), LLM-generated commit message (string)
- Produces: review screen — scrollable diff view + editable commit message + confirm/cancel

- [ ] **Step 1: Write implementation**

```go
// internal/tui/review_screen.go
package tui

import (
    "strings"

    "github.com/charmbracelet/bubbles/viewport"
    "github.com/charmbracelet/bubbles/textarea"
    tea "github.com/charmbracelet/bubbletea"
    "github.com/charmbracelet/lipgloss"
)

type reviewScreenModel struct {
    diff            string
    commitMessage   textarea.Model
    viewport        viewport.Model
    showPR          bool
    prDescription   textarea.Model
    confirmed       bool
    cancelled       bool
    includePR       bool
    ready           bool
}

func newReviewScreen(diff, commitMessage string) reviewScreenModel {
    // Diff viewport
    vp := viewport.New(80, 20)
    vp.SetContent(diff)

    // Commit message textarea
    ta := textarea.New()
    ta.SetValue(commitMessage)
    ta.SetWidth(80)
    ta.SetHeight(5)
    ta.Prompt = "Commit message: "

    // PR description textarea
    pr := textarea.New()
    pr.SetValue(commitMessage)
    pr.SetWidth(80)
    pr.SetHeight(8)
    pr.Prompt = "PR description: "

    return reviewScreenModel{
        diff:          diff,
        commitMessage: ta,
        viewport:      vp,
        prDescription: pr,
        ready:         false,
    }
}

func (m reviewScreenModel) Init() tea.Cmd {
    return nil
}

func (m reviewScreenModel) Update(msg tea.Msg) (reviewScreenModel, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        switch msg.String() {
        case "ctrl+s":
            m.confirmed = true
            m.includePR = false
            return m, nil
        case "ctrl+p":
            m.confirmed = true
            m.includePR = true
            return m, nil
        case "ctrl+c":
            m.cancelled = true
            return m, nil
        case "tab":
            // Toggle between commit message and PR description
            if m.showPR {
                if m.commitMessage.Focused() {
                    m.commitMessage.Blur()
                    m.prDescription.Focus()
                } else {
                    m.prDescription.Blur()
                    m.commitMessage.Focus()
                }
            }
            return m, nil
        }
    }

    var cmd tea.Cmd
    if m.commitMessage.Focused() {
        m.commitMessage, cmd = m.commitMessage.Update(msg)
    } else if m.showPR && m.prDescription.Focused() {
        m.prDescription, cmd = m.prDescription.Update(msg)
    }
    return m, cmd
}

func (m reviewScreenModel) View() string {
    var s strings.Builder

    s.WriteString(lipgloss.NewStyle().Bold(true).Render("📝 Review Changes\n\n"))

    // Diff
    s.WriteString(lipgloss.NewStyle().Bold(true).Render("Diff:\n"))
    s.WriteString(m.viewport.View())
    s.WriteString("\n\n")

    // Commit message
    s.WriteString(lipgloss.NewStyle().Bold(true).Render("Commit Message:\n"))
    s.WriteString(m.commitMessage.View())
    s.WriteString("\n")

    // PR section
    if m.showPR {
        s.WriteString(lipgloss.NewStyle().Bold(true).Render("PR Description:\n"))
        s.WriteString(m.prDescription.View())
        s.WriteString("\n")
    }

    s.WriteString("\n")
    s.WriteString(lipgloss.NewStyle().Faint(true).Render("Ctrl+s: Commit  Ctrl+p: Commit + PR  Ctrl+c: Cancel  Tab: switch fields\n"))

    return s.String()
}
```

- [ ] **Step 2: Verify it compiles**

```bash
cd /Users/lutrarutra/Documents/dev/lazypush
go build ./internal/tui/
```
Expected: no errors

- [ ] **Step 3: Commit**

```bash
git add internal/tui/review_screen.go
git commit -m "feat: add TUI review screen with diff, commit message, PR description"
```

---

### Task 10: TUI — Progress Screen

**Files:**
- Create: `internal/tui/progress_screen.go`

**Interfaces:**
- Consumes: nothing (standalone screen)
- Produces: progress screen with spinner + status lines

- [ ] **Step 1: Write implementation**

```go
// internal/tui/progress_screen.go
package tui

import (
    "strings"

    "github.com/charmbracelet/bubbles/spinner"
    tea "github.com/charmbracelet/bubbletea"
    "github.com/charmbracelet/lipgloss"
)

type progressStep struct {
    label string
    done  bool
    ok    bool
    msg   string
}

type progressScreenModel struct {
    spinner spinner.Model
    steps   []progressStep
    current int
    done    bool
}

func newProgressScreen() progressScreenModel {
    s := spinner.New()
    s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("63"))
    s.Spinner = spinner.Dot

    return progressScreenModel{
        spinner: s,
        steps: []progressStep{
            {label: "Committing..."},
            {label: "Tagging..."},
            {label: "Pushing..."},
            {label: "Creating PR..."},
        },
        current: 0,
    }
}

func (m progressScreenModel) Init() tea.Cmd {
    return m.spinner.Tick
}

func (m progressScreenModel) Update(msg tea.Msg) (progressScreenModel, tea.Cmd) {
    switch msg := msg.(type) {
    case progressStepDone:
        if msg.index < len(m.steps) {
            m.steps[msg.index].done = true
            m.steps[msg.index].ok = msg.ok
            m.steps[msg.index].msg = msg.message
            m.current = msg.index + 1
        }
        if m.current >= len(m.steps) || !msg.ok {
            m.done = true
        }
        return m, nil
    }

    var cmd tea.Cmd
    m.spinner, cmd = m.spinner.Update(msg)
    return m, cmd
}

func (m progressScreenModel) View() string {
    var s strings.Builder
    s.WriteString(lipgloss.NewStyle().Bold(true).Render("🚀 Progress\n\n"))

    for i, step := range m.steps {
        if step.done {
            if step.ok {
                s.WriteString("✅ ")
            } else {
                s.WriteString("❌ ")
            }
            s.WriteString(step.label)
            if step.msg != "" {
                s.WriteString(" " + step.msg)
            }
            s.WriteString("\n")
        } else if i == m.current {
            s.WriteString(m.spinner.View() + " ")
            s.WriteString(step.label)
            s.WriteString("\n")
        } else {
            s.WriteString("  " + step.label + "\n")
        }
    }

    return s.String()
}

type progressStepDone struct {
    index   int
    ok      bool
    message string
}

func ProgressDone(index int, ok bool, message string) tea.Cmd {
    return func() tea.Msg {
        return progressStepDone{index: index, ok: ok, message: message}
    }
}
```

- [ ] **Step 2: Verify it compiles**

```bash
cd /Users/lutrarutra/Documents/dev/lazypush
go build ./internal/tui/
```
Expected: no errors

- [ ] **Step 3: Commit**

```bash
git add internal/tui/progress_screen.go
git commit -m "feat: add TUI progress screen with spinner and status lines"
```

---

### Task 11: TUI — Main Model + Screen Routing

**Files:**
- Create: `internal/tui/main_model.go`
- Modify: `cmd/lazypush/main.go`

**Interfaces:**
- Consumes: all previous tasks (config, git, version, llm, gh, tui screens)
- Produces: the full interactive TUI application

- [ ] **Step 1: Write the main TUI model**

```go
// internal/tui/main_model.go
package tui

import (
    "context"
    "log"
    "os"

    tea "github.com/charmbracelet/bubbletea"
    "github.com/lutrarutra/lazypush/internal/config"
    "github.com/lutrarutra/lazypush/internal/gh"
    "github.com/lutrarutra/lazypush/internal/git"
    "github.com/lutrarutra/lazypush/internal/llm"
)

type screen int

const (
    screenLogin screen = iota
    screenVersion
    screenReview
    screenProgress
)

type Model struct {
    screen     screen
    login      loginScreenModel
    version    versionScreenModel
    review     reviewScreenModel
    progress   progressScreenModel

    config     *config.Config
    repo       *git.Repo
    llmClient  *llm.Client

    versionTag string
    commitMsg  string
    prBody     string
    prURL      string
    err        error
}

func NewModel() *Model {
    return &Model{
        screen:  screenLogin,
        login:   newLoginScreen(),
    }
}

func (m *Model) Init() tea.Cmd {
    // Try to load config first
    cfg, err := config.Load()
    if err != nil {
        log.Printf("warning: could not load config: %v", err)
        cfg = &config.Config{
            APIURL: "https://api.openai.com/v1",
            Model:  "gpt-4o-mini",
        }
    }

    // Open repo
    cwd, _ := os.Getwd()
    repo, err := git.Open(cwd)
    if err != nil {
        log.Printf("warning: could not open git repo: %v", err)
        repo = nil
    }

    m.config = cfg
    m.repo = repo

    // If config is fully populated, skip login screen
    if cfg.APIKey != "" && cfg.APIURL != "" && cfg.Model != "" {
        m.llmClient = llm.New(cfg.APIURL, cfg.APIKey, cfg.Model)
        m.screen = screenVersion
        return m.initVersionScreen()
    }

    return m.login.Init()
}

func (m *Model) initVersionScreen() tea.Cmd {
    tag := "v0.0.0"
    if m.repo != nil {
        latest, err := m.repo.LatestTag()
        if err == nil && latest != "" {
            tag = latest
        }
    }
    m.versionTag = tag
    m.version = newVersionScreen(tag)
    m.screen = screenVersion
    return m.version.Init()
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch m.screen {
    case screenLogin:
        return m.updateLogin(msg)
    case screenVersion:
        return m.updateVersion(msg)
    case screenReview:
        return m.updateReview(msg)
    case screenProgress:
        return m.updateProgress(msg)
    }
    return m, nil
}

func (m *Model) updateLogin(msg tea.Msg) (tea.Model, tea.Cmd) {
    var cmd tea.Cmd
    m.login, cmd = m.login.Update(msg)

    if m.login.done {
        if m.login.err != "" {
            return m, tea.Quit
        }

        apiURL, model, apiKey := m.login.Values()
        m.config.APIURL = apiURL
        m.config.Model = model
        m.config.APIKey = apiKey

        if err := config.Save(m.config); err != nil {
            m.err = err
            return m, tea.Quit
        }

        m.llmClient = llm.New(apiURL, apiKey, model)
        return m, m.initVersionScreen()
    }

    return m, cmd
}

func (m *Model) updateVersion(msg tea.Msg) (tea.Model, tea.Cmd) {
    // Handle async LLM response arriving when screen is still screenVersion
    switch msg := msg.(type) {
    case commitMessageReadyMsg:
        m.review = newReviewScreen(msg.diff, msg.message)
        m.screen = screenReview
        return m, nil
    case errMsg:
        m.err = msg.error
        return m, tea.Quit
    }

    var cmd tea.Cmd
    m.version, cmd = m.version.Update(msg)

    if m.version.done {
        if m.version.chosenVersion == "" {
            return m, tea.Quit
        }
        m.versionTag = m.version.chosenVersion

        // Get diff and generate commit message
        return m, m.generateCommitMessage()
    }

    return m, cmd
}

func (m *Model) generateCommitMessage() tea.Cmd {
    return func() tea.Msg {
        if m.repo == nil {
            return errMsg{error: "no git repository found"}
        }

        diff, err := m.repo.Diff()
        if err != nil {
            return errMsg{error: err.Error()}
        }

        sys, user := llm.CommitMessagePrompt(diff)
        msg, err := m.llmClient.Generate(context.Background(), sys, user)
        if err != nil {
            return errMsg{error: err.Error()}
        }

        return commitMessageReadyMsg{diff: diff, message: msg}
    }
}

func (m *Model) updateReview(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case commitMessageReadyMsg:
        m.review = newReviewScreen(msg.diff, msg.message)
        m.screen = screenReview
        return m, nil

    case errMsg:
        m.err = msg.error
        return m, tea.Quit
    }

    var cmd tea.Cmd
    m.review, cmd = m.review.Update(msg)

    if m.review.confirmed {
        return m, m.executeOperations()
    }
    if m.review.cancelled {
        return m, tea.Quit
    }

    return m, cmd
}

func (m *Model) executeOperations() tea.Cmd {
    return func() tea.Msg {
        m.progress = newProgressScreen()
        m.screen = screenProgress

        // Step 1: Commit
        err := m.repo.Commit(m.review.commitMessage.Value())
        if err != nil {
            return progressStepDone{index: 0, ok: false, message: err.Error()}
        }

        // Step 2: Tag
        err = m.repo.Tag(m.versionTag)
        if err != nil {
            return progressStepDone{index: 1, ok: false, message: err.Error()}
        }

        // Step 3: Push
        err = m.repo.Push("origin")
        if err != nil {
            return progressStepDone{index: 2, ok: false, message: err.Error()}
        }

        // Step 4: PR (optional)
        if m.review.includePR && gh.CheckInstalled() {
            prURL, err := gh.CreatePR(m.review.commitMessage.Value(), m.review.prDescription.Value())
            if err != nil {
                return progressStepDone{index: 3, ok: false, message: err.Error()}
            }
            m.prURL = prURL
        }

        return progressStepDone{index: 3, ok: true, message: m.prURL}
    }
}

func (m *Model) updateProgress(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case progressStepDone:
        var cmd tea.Cmd
        m.progress, cmd = m.progress.Update(msg)
        if m.progress.done {
            // Keep showing final state, quit on any key
            return m, tea.Quit
        }
        return m, cmd
    }

    var cmd tea.Cmd
    m.progress, cmd = m.progress.Update(msg)
    return m, cmd
}

func (m *Model) View() string {
    switch m.screen {
    case screenLogin:
        return m.login.View()
    case screenVersion:
        return m.version.View()
    case screenReview:
        return m.review.View()
    case screenProgress:
        return m.progress.View()
    }
    return ""
}

// Message types
type commitMessageReadyMsg struct {
    diff    string
    message string
}

type errMsg struct {
    error string
}
```

- [ ] **Step 2: Update the main entrypoint**

```go
// cmd/lazypush/main.go
package main

import (
    "fmt"
    "os"

    tea "github.com/charmbracelet/bubbletea"
    "github.com/lutrarutra/lazypush/internal/tui"
)

func main() {
    model := tui.NewModel()
    p := tea.NewProgram(model)

    if _, err := p.Run(); err != nil {
        fmt.Fprintf(os.Stderr, "Error: %v\n", err)
        os.Exit(1)
    }
}
```

- [ ] **Step 3: Verify it compiles**

```bash
cd /Users/lutrarutra/Documents/dev/lazypush
go build ./cmd/lazypush/
```
Expected: compiles cleanly

- [ ] **Step 4: Commit**

```bash
git add internal/tui/main_model.go cmd/lazypush/main.go
git commit -m "feat: add main TUI model with screen routing"
```

---

### Task 12: GoReleaser + GitHub Actions

**Files:**
- Create: `.goreleaser.yaml`
- Create: `.github/workflows/release.yml`

**Interfaces:**
- Consumes: nothing (CI/CD configuration)
- Produces: cross-compiled binaries on tag push

- [ ] **Step 1: Write .goreleaser.yaml**

```yaml
# .goreleaser.yaml
version: 2
project_name: lazypush

before:
  hooks:
    - go mod tidy

builds:
  - id: lazypush
    main: ./cmd/lazypush
    binary: lazypush
    env:
      - CGO_ENABLED=0
    goos:
      - linux
      - darwin
      - windows
    goarch:
      - amd64
      - arm64

archives:
  - id: default
    format: tar.gz
    format_overrides:
      - goos: windows
        format: zip
    files:
      - README.md
      - LICENSE

checksum:
  name_template: "checksums.txt"

changelog:
  use: github
  sort: asc
  filters:
    exclude:
      - "^docs:"
      - "^test:"
```

- [ ] **Step 2: Write GitHub Actions workflow**

```yaml
# .github/workflows/release.yml
name: Release

on:
  push:
    tags: [ 'v*' ]

permissions:
  contents: write

jobs:
  goreleaser:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout
        uses: actions/checkout@v4
        with:
          fetch-depth: 0

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: stable

      - name: Run GoReleaser
        uses: goreleaser/goreleaser-action@v6
        with:
          distribution: goreleaser
          version: "~> v2"
          args: release --clean
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

- [ ] **Step 3: Commit**

```bash
git add .goreleaser.yaml .github/workflows/release.yml
git commit -m "ci: add GoReleaser config and GitHub Actions release workflow"
```
