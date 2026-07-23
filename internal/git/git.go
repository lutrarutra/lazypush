package git

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
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
		if s.Staging != gogit.Unmodified {
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
	status, err := r.worktree.Status()
	if err != nil {
		return "", err
	}

	hasStaged := false
	for _, s := range status {
		if s.Staging != gogit.Unmodified {
			hasStaged = true
			break
		}
	}

	if !hasStaged {
		if err := r.StageAll(); err != nil {
			return "", err
		}
	}

	// Use git diff --cached via CLI for the diff (read-only, no auth needed)
	cmd := exec.Command("git", "diff", "--cached")
	cmd.Dir = r.path
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("git diff: %w\nstderr: %s", err, stderr.String())
	}

	return stdout.String(), nil
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

func (r *Repo) Commit(message string) error {
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

func (r *Repo) DeleteTag(name string) error {
	if err := r.repo.DeleteTag(name); err != nil {
		return fmt.Errorf("delete tag %s: %w", name, err)
	}
	return nil
}

func (r *Repo) Push(remote string) error {
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
		if fi, err := os.Stat(filepathJoin(dir, ".git")); err == nil && fi.IsDir() {
			return dir, nil
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
