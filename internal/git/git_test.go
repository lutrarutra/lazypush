package git_test

import (
	"os"
	"path/filepath"
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
