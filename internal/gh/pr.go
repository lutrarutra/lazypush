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
