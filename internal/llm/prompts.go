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
