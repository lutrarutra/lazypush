package llm

import "fmt"

const commitSystemPrompt = `You are a commit message generator. Your entire response must be ONLY the commit message — no greetings, no explanations, no commentary, no markdown formatting, no backticks, no surrounding text of any kind.

Rules:
- Use conventional commits format: <type>: <description>
- Types: feat, fix, refactor, docs, chore, test, style, perf
- Keep the subject line under 72 characters
- Output ONLY the commit message text, nothing else`

const prSystemPrompt = `You are a PR description generator. Your entire response must be ONLY the PR description — no greetings, no explanations, no commentary, no markdown formatting, no backticks, no surrounding text of any kind.

Rules:
- Summarize the changes clearly and concisely
- Use bullet points for individual changes
- Output ONLY the PR description text, nothing else`

func CommitMessagePrompt(diff string) (system, user string) {
	return commitSystemPrompt, fmt.Sprintf("Write ONLY a commit message for this diff. Nothing else:\n\n%s", diff)
}

func PRDescriptionPrompt(diff string) (system, user string) {
	return prSystemPrompt, fmt.Sprintf("Write ONLY a PR description for this diff. Nothing else:\n\n%s", diff)
}
