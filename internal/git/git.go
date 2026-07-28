package git

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"
	"time"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/lutrarutra/lazypush/internal/version"
)

type Repo struct {
	repo       *gogit.Repository
	worktree   *gogit.Worktree
	path       string
	baseBranch string
}

func (r *Repo) SetBaseBranch(b string) { r.baseBranch = b }
func (r *Repo) BaseBranch() string    { return r.baseBranch }

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
		if s.Staging != gogit.Unmodified {
			count++
		}
	}
	return count, nil
}

func (r *Repo) StageAll() error {
	_, err := r.worktree.Add(".")
	if err != nil {
		return fmt.Errorf("stage .: %w", err)
	}
	return nil
}

func (r *Repo) Diff() (string, error) {
	// Diff of tracked files (working tree vs HEAD)
	var diffBuf bytes.Buffer
	{
		cmd := exec.Command("git", "diff", "HEAD")
		cmd.Dir = r.path
		cmd.Stdout = &diffBuf
		var stderr bytes.Buffer
		cmd.Stderr = &stderr
		if err := cmd.Run(); err != nil {
			return "", fmt.Errorf("git diff: %w\nstderr: %s", err, stderr.String())
		}
	}

	// Also include new untracked files as full additions
	lsCmd := exec.Command("git", "ls-files", "--others", "--exclude-standard")
	lsCmd.Dir = r.path
	var lsOut bytes.Buffer
	lsCmd.Stdout = &lsOut
	if err := lsCmd.Run(); err != nil {
		return diffBuf.String(), nil // best-effort: return partial diff
	}

	newFiles := strings.TrimSpace(lsOut.String())
	if newFiles == "" {
		return diffBuf.String(), nil
	}

	var b strings.Builder
	b.WriteString(diffBuf.String())
	if diffBuf.Len() > 0 && !strings.HasSuffix(diffBuf.String(), "\n") {
		b.WriteString("\n")
	}

	for _, f := range strings.Split(newFiles, "\n") {
		f = strings.TrimSpace(f)
		if f == "" {
			continue
		}
		content, err := os.ReadFile(filepathJoin(r.path, f))
		if err != nil {
			b.WriteString(fmt.Sprintf("# could not read new file: %s\n", f))
			continue
		}
		lines := strings.Split(string(content), "\n")
		// Drop the empty element from a trailing newline
		if len(lines) > 0 && lines[len(lines)-1] == "" {
			lines = lines[:len(lines)-1]
		}
		b.WriteString(fmt.Sprintf("diff --git a/%s b/%s\n", f, f))
		b.WriteString("new file mode 100644\n")
		b.WriteString(fmt.Sprintf("--- /dev/null\n+++ b/%s\n", f))
		if len(lines) == 0 {
			b.WriteString("@@ -0,0 +0,0 @@\n")
		} else {
			b.WriteString(fmt.Sprintf("@@ -0,0 +1,%d @@\n", len(lines)))
			for _, line := range lines {
				b.WriteString("+" + line + "\n")
			}
		}
	}

	return b.String(), nil
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
			return nil
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

func (r *Repo) gitConfig(key string) string {
	cmd := exec.Command("git", "config", "--get", key)
	cmd.Dir = r.path
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		return ""
	}
	return strings.TrimSpace(stdout.String())
}

func (r *Repo) userSignature() (object.Signature, error) {
	name := r.gitConfig("user.name")
	if name == "" {
		return object.Signature{}, fmt.Errorf("git user.name is not set — run: git config --global user.name \"Your Name\"")
	}
	email := r.gitConfig("user.email")
	if email == "" {
		return object.Signature{}, fmt.Errorf("git user.email is not set — run: git config --global user.email \"you@example.com\"")
	}
	return object.Signature{
		Name:  name,
		Email: email,
		When:  time.Now(),
	}, nil
}

func (r *Repo) Commit(message string) error {
	if err := r.StageAll(); err != nil {
		return fmt.Errorf("stage before commit: %w", err)
	}

	sig, err := r.userSignature()
	if err != nil {
		return err
	}
	_, err = r.worktree.Commit(message, &gogit.CommitOptions{
		Author: &sig,
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

	sig, err := r.userSignature()
	if err != nil {
		return err
	}
	_, err = r.repo.CreateTag(name, ref.Hash(), &gogit.CreateTagOptions{
		Message: name,
		Tagger:  &sig,
	})
	if err != nil {
		return fmt.Errorf("create tag %s: %w", name, err)
	}
	return nil
}

func (r *Repo) DeleteTag(name string) error {
	if err := r.repo.DeleteTag(name); err != nil {
		return fmt.Errorf("delete tag %s: %w", name, err)
	}
	return nil
}

func (r *Repo) Push(remote string) error {
	// Use git push CLI — handles auth, worktrees, and upstream correctly
	cmd := exec.Command("git", "push", "-u", "--follow-tags", remote, "HEAD")
	cmd.Dir = r.path
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git push failed: %w\n%s", err, stderr.String())
	}
	return nil
}

func (r *Repo) Path() string {
	return r.path
}

// CurrentBranch returns the short name of the current branch.
func (r *Repo) CurrentBranch() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	cmd.Dir = r.path
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("get current branch: %w\nstderr: %s", err, stderr.String())
	}
	return strings.TrimSpace(stdout.String()), nil
}

// ListBranches returns remote branch names, with main/master sorted first.
func (r *Repo) ListBranches() ([]string, error) {
	cmd := exec.Command("git", "branch", "-r", "--format=%(refname:short)")
	cmd.Dir = r.path
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		// Fall back to local branches
		cmd = exec.Command("git", "branch", "--format=%(refname:short)")
		cmd.Dir = r.path
		stdout.Reset()
		cmd.Stdout = &stdout
		if err := cmd.Run(); err != nil {
			return nil, fmt.Errorf("list branches: %w", err)
		}
	}
	lines := strings.Split(strings.TrimSpace(stdout.String()), "\n")
	var branches []string
	seen := map[string]bool{}
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.Contains(line, "HEAD") {
			continue
		}
		name := strings.TrimPrefix(line, "origin/")
		if seen[name] {
			continue
		}
		seen[name] = true
		if name == "main" || name == "master" {
			branches = append([]string{name}, branches...)
		} else {
			branches = append(branches, name)
		}
	}
	return branches, nil
}

// DiffToBranch returns the diff between HEAD and the given branch.
func (r *Repo) DiffToBranch(branch string) (string, error) {
	cmd := exec.Command("git", "diff", fmt.Sprintf("origin/%s...HEAD", branch))
	cmd.Dir = r.path
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		// Try without origin/ prefix
		cmd2 := exec.Command("git", "diff", fmt.Sprintf("%s...HEAD", branch))
		cmd2.Dir = r.path
		stdout.Reset()
		stderr.Reset()
		cmd2.Stdout = &stdout
		cmd2.Stderr = &stderr
		if err := cmd2.Run(); err != nil {
			return "", fmt.Errorf("diff to %s: %w\nstderr: %s", branch, err, stderr.String())
		}
	}
	return stdout.String(), nil
}

// CreateBranchAndSwitch creates a new branch from HEAD and checks it out.
func (r *Repo) CreateBranchAndSwitch(name string) error {
	cmd := exec.Command("git", "checkout", "-b", name)
	cmd.Dir = r.path
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("create branch %s: %w\nstderr: %s", name, err, stderr.String())
	}
	return nil
}

// ChangedFiles returns a summarized list of files changed between HEAD and base.
func (r *Repo) ChangedFiles() (string, error) {
	args := []string{"diff", "--stat"}
	if r.baseBranch != "" {
		args = append(args, fmt.Sprintf("origin/%s...HEAD", r.baseBranch))
	} else {
		args = append(args, "HEAD")
	}
	cmd := exec.Command("git", args...)
	cmd.Dir = r.path
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("git diff --stat: %w\nstderr: %s", err, stderr.String())
	}
	return strings.TrimSpace(stdout.String()), nil
}

// ReadCurrentFile reads lines [startLine, endLine] (1-based, inclusive) from the working tree.
func (r *Repo) ReadCurrentFile(path string, start, end int) (string, error) {
	fullPath := filepathJoin(r.path, path)
	data, err := os.ReadFile(fullPath)
	if err != nil {
		return "", fmt.Errorf("read %s: %w", path, err)
	}
	return sliceLines(string(data), start, end), nil
}

// ReadBaseFile reads lines from the file in the base branch.
func (r *Repo) ReadBaseFile(path string, start, end int) (string, error) {
	treeish := r.baseBranch
	if treeish == "" {
		treeish = "HEAD~1"
	}
	ref := fmt.Sprintf("origin/%s:%s", treeish, path)
	cmd := exec.Command("git", "show", ref)
	cmd.Dir = r.path
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		ref = fmt.Sprintf("%s:%s", treeish, path)
		cmd = exec.Command("git", "show", ref)
		cmd.Dir = r.path
		stdout.Reset()
		stderr.Reset()
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		if err := cmd.Run(); err != nil {
			return "", fmt.Errorf("show %s: %w\nstderr: %s", ref, err, stderr.String())
		}
	}
	return sliceLines(stdout.String(), start, end), nil
}

// ShowFileDiff returns the unified diff for a single file between HEAD and base.
func (r *Repo) ShowFileDiff(path string) (string, error) {
	args := []string{"diff"}
	if r.baseBranch != "" {
		args = append(args, fmt.Sprintf("origin/%s...HEAD", r.baseBranch))
	} else {
		args = append(args, "HEAD")
	}
	args = append(args, "--", path)
	cmd := exec.Command("git", args...)
	cmd.Dir = r.path
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("diff %s: %w\nstderr: %s", path, err, stderr.String())
	}
	return stdout.String(), nil
}

// sliceLines extracts lines [startLine, endLine] (1-based, inclusive).
func sliceLines(s string, startLine, endLine int) string {
	lines := strings.Split(s, "\n")
	if startLine < 1 {
		startLine = 1
	}
	if endLine > len(lines) {
		endLine = len(lines)
	}
	if startLine > len(lines) || startLine > endLine {
		return s
	}
	return strings.Join(lines[startLine-1:endLine], "\n")
}

func resolveGitDir(path string) (string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return "", fmt.Errorf("stat %s: %w", path, err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("%s is not a directory", path)
	}

	dir, err := filepathAbs(path)
	if err != nil {
		return "", err
	}

	for {
		gitPath := filepathJoin(dir, ".git")
		if fi, err := os.Stat(gitPath); err == nil {
			// .git is a directory (normal repo) or a file (worktree/submodule)
			if fi.IsDir() || fi.Mode().IsRegular() {
				return dir, nil
			}
		}
		parent := filepathDir(dir)
		if parent == dir {
			return "", fmt.Errorf("no .git found in %s or parents", path)
		}
		dir = parent
	}
}

// filepathAbs wraps filepath.Abs for compatibility
func filepathAbs(path string) (string, error) {
	// Use Go 1.18 compatible approach
	if len(path) > 0 && path[0] == '/' {
		return path, nil
	}
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	if path == "." {
		return wd, nil
	}
	return wd + "/" + path, nil
}

// filepathJoin wraps filepath.Join
func filepathJoin(elem ...string) string {
	return strings.Join(elem, "/")
}

// filepathDir wraps filepath.Dir
func filepathDir(path string) string {
	idx := strings.LastIndex(path, "/")
	if idx < 0 {
		return path
	}
	if idx == 0 {
		return "/"
	}
	return path[:idx]
}
