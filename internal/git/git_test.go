package git_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/lutrarutra/lazypush/internal/git"
)

func initTempRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	r, err := gogit.PlainInit(dir, false)
	if err != nil {
		t.Fatalf("git init: %v", err)
	}

	// Set local git config so userSignature() doesn't fail
	for _, args := range [][]string{{"config", "user.name", "test"}, {"config", "user.email", "test@test.com"}} {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		if err := cmd.Run(); err != nil {
			t.Fatalf("git config %v: %v", args, err)
		}
	}

	if err := os.WriteFile(filepath.Join(dir, "test.txt"), []byte("hello"), 0644); err != nil {
		t.Fatalf("write file yes: %v", err)
	}

	wt, err := r.Worktree()
	if err != nil {
		t.Fatalf("worktree: %v", err)
	}

	_, err = wt.Add("test.txt")
	if err != nil {
		t.Fatalf("git add: %v", err)
	}

	_, err = wt.Commit("initial", &gogit.CommitOptions{
		Author: &object.Signature{Name: "test", Email: "test@test.com"},
	})
	if err != nil {
		t.Fatalf("git commit: %v", err)
	}

	// Get HEAD hash to create tag
	ref, err := r.Head()
	if err != nil {
		t.Fatalf("get HEAD: %v", err)
	}

	_, err = r.CreateTag("v0.1.0", ref.Hash(), nil)
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
	r, _ := gogit.PlainInit(dir, false)
	wt, _ := r.Worktree()
	os.WriteFile(filepath.Join(dir, "a.txt"), []byte("a"), 0644)
	wt.Add("a.txt")
	wt.Commit("init", &gogit.CommitOptions{
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

func TestForcePushTag(t *testing.T) {
	dir := initTempRepo(t)

	// Set up a bare remote
	bareDir := filepath.Join(t.TempDir(), "remote.git")
	cmd := exec.Command("git", "init", "--bare", bareDir)
	if err := cmd.Run(); err != nil {
		t.Fatalf("init bare remote: %v", err)
	}
	cmd = exec.Command("git", "remote", "add", "origin", bareDir)
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		t.Fatalf("add origin: %v", err)
	}

	// Push the initial commit and its tag to the remote
	cmd = exec.Command("git", "push", "-u", "origin", "HEAD")
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		t.Fatalf("push HEAD: %v", err)
	}
	cmd = exec.Command("git", "push", "origin", "tag", "v0.1.0")
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		t.Fatalf("push tag: %v", err)
	}

	// New commit, then move the existing tag to HEAD
	if err := os.WriteFile(filepath.Join(dir, "new.txt"), []byte("new"), 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	repo, err := git.Open(dir)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	if err := repo.Commit("feat: more"); err != nil {
		t.Fatalf("Commit() error = %v", err)
	}
	if err := repo.DeleteTag("v0.1.0"); err != nil {
		t.Fatalf("DeleteTag() error = %v", err)
	}
	if err := repo.Tag("v0.1.0"); err != nil {
		t.Fatalf("Tag() error = %v", err)
	}

	// A regular push (--follow-tags) would skip the tag since it already
	// exists on the remote — ForcePushTag must move it there.
	if err := repo.ForcePushTag("v0.1.0"); err != nil {
		t.Fatalf("ForcePushTag() error = %v", err)
	}

	// Verify the remote tag now points at the new commit (peel the
	// annotated tag object: ^{}).
	local := exec.Command("git", "rev-parse", "refs/tags/v0.1.0^{}")
	local.Dir = dir
	localOut, err := local.Output()
	if err != nil {
		t.Fatalf("rev-parse local tag: %v", err)
	}
	remote := exec.Command("git", "--git-dir="+bareDir, "rev-parse", "refs/tags/v0.1.0^{}")
	remoteOut, err := remote.Output()
	if err != nil {
		t.Fatalf("rev-parse remote tag: %v", err)
	}
	if strings.TrimSpace(string(remoteOut)) != strings.TrimSpace(string(localOut)) {
		t.Errorf("remote tag = %q, want %q", strings.TrimSpace(string(remoteOut)), strings.TrimSpace(string(localOut)))
	}
}

func TestDiffStaged(t *testing.T) {
	dir := initTempRepo(t)
	repo, err := git.Open(dir)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}

	os.WriteFile(filepath.Join(dir, "test.txt"), []byte("hello world"), 0644)

	// Diff auto-stages and returns the patch
	diff, err := repo.Diff()
	if err != nil {
		t.Fatalf("Diff() error = %v", err)
	}
	if diff == "" {
		t.Error("Diff() returned empty for modified file")
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
