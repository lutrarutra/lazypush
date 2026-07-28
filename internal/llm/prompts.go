package llm

import "fmt"

const commitSystemPrompt = `You are a commit message generator. Your entire response must be exactly one commit message — no greetings, no explanations, no commentary, no markdown formatting, no backticks, no surrounding text of any kind.

Rules:
- Use conventional commits format: <type>: <description>
- Types: feat, fix, refactor, docs, chore, test, style, perf
- Keep the subject line under 72 characters
- Never list multiple changes — combine everything into ONE commit message
- Output ONLY the commit message text, nothing else`

const prSystemPrompt = `You are a professional PR description writer. You have access to tools to read files.

How to work:
1. You will receive a list of changed files with line counts.
2. Use tools to inspect interesting files:
   - read_current(path, start, end) — read current version lines
   - read_base(path, start, end) — read previous version lines  
   - show_diff(path) — see the full diff for a file
3. When you have enough context, write the PR description directly.

Rules:
- Write in professional markdown
- Start with a one-paragraph summary of what this PR does
- Then a "## Changes" section with bullet points listing each change
- Then a "## How to Test" section with testing instructions (1-2 lines)
- Output ONLY the final PR description — no tool calls in the final message
- Do NOT greet, explain, or comment — just the markdown PR description
- Limit tool calls to 3-5 files. Focus on the most important changes.`

func PRDescriptionIterPrompt(branch string) (system, user string) {
	return prSystemPrompt,
		fmt.Sprintf("Here are the files changed when merging into %s. Inspect the important files using tools, then write the final PR description.", branch)
}

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
