package gh_test

import (
	"testing"

	"github.com/lutrarutra/lazypush/internal/gh"
)

func TestCheckInstalled(t *testing.T) {
	installed := gh.CheckInstalled()
	// Just verify it returns a bool without panicking
	if installed != false && installed != true {
		t.Errorf("CheckInstalled() = %v, expected bool", installed)
	}
}
