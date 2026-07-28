package llm

import "fmt"

const commitSystemPrompt = `You are a commit message generator. Your entire response must be exactly one commit message — no greetings, no explanations, no commentary, no markdown formatting, no backticks, no surrounding text of any kind.

Rules:
- Use conventional commits format: <type>: <description>
- Types: feat, fix, refactor, docs, chore, test, style, perf
- Keep the subject line under 72 characters
- Never list multiple changes — combine everything into ONE commit message
- Output ONLY the commit message text, nothing else`

const prSystemPrompt = `You are a professional PR description writer. Your entire response must be ONLY the PR description in markdown — no greetings, no explanations, no commentary, no surrounding text of any kind.

Rules:
- Write in professional markdown
- Start with a one-paragraph summary of what this PR does
- Then a "## Changes" section with bullet points listing each change
- Then a "## How to Test" section with testing instructions (1-2 lines)
- Keep it concise but thorough
- Output ONLY the PR description, nothing else`

// maxDiffLen limits diff input to prevent the diff from consuming the
// entire LLM context window. Set to 500K for models with large context.
const maxDiffLen = 500000

func truncateDiff(diff string) string {
	if len(diff) <= maxDiffLen {
		return diff
	}
	// Take the first and last parts — the beginning has file headers and
	// the end has the last changed file, the middle is the least important.
	half := maxDiffLen / 2
	diff = diff[:half] + "\n\n... (truncated, " + fmt.Sprintf("%d", len(diff)-maxDiffLen) + " chars omitted) ...\n\n" + diff[len(diff)-half:]
	return diff
}

func CommitMessagePrompt(diff string) (system, user string) {
	return commitSystemPrompt, fmt.Sprintf("Write exactly one commit message summarizing ALL changes in this diff. Nothing else:\n\n%s", truncateDiff(diff))
}

func PRDescriptionPrompt(diff, branch string) (system, user string) {
	return prSystemPrompt, fmt.Sprintf("Write ONLY a PR description for changes being merged into %s. Use the diff below:\n\n%s", branch, truncateDiff(diff))
}
