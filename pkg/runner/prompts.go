package runner

import (
	"fmt"
	"os"
	"strings"
)

// buildTaskPrompt creates the prompt for task execution phase.
func buildTaskPrompt(planFile string, iteration int) (string, error) {
	planContent, err := os.ReadFile(planFile) //nolint:gosec // planFile from CLI args
	if err != nil {
		return "", fmt.Errorf("read plan file: %w", err)
	}

	var sb strings.Builder
	sb.WriteString("You are executing a plan autonomously. ")
	sb.WriteString(fmt.Sprintf("This is iteration %d.\n\n", iteration))
	sb.WriteString("## Plan\n\n")
	sb.WriteString(string(planContent))
	sb.WriteString("\n\n## Instructions\n\n")
	sb.WriteString("1. Review the plan and identify the next incomplete task\n")
	sb.WriteString("2. Execute the task, making necessary code changes\n")
	sb.WriteString("3. Run tests to verify your changes work\n")
	sb.WriteString("4. Update the plan file to mark completed tasks with [x]\n")
	sb.WriteString("5. If all tasks are complete, output COMPLETED\n")
	sb.WriteString("6. If you encounter a blocking issue, output FAILED with explanation\n")
	sb.WriteString("7. Otherwise, continue to the next task\n\n")
	sb.WriteString("Important: Only output COMPLETED or FAILED as terminal signals. ")
	sb.WriteString("Do not output these words in any other context.\n")

	return sb.String(), nil
}

// buildFirstReviewPrompt creates the prompt for first review pass (Claude review).
func buildFirstReviewPrompt() string {
	var sb strings.Builder
	sb.WriteString("You are reviewing code changes for quality and correctness.\n\n")
	sb.WriteString("## Instructions\n\n")
	sb.WriteString("1. Run `git diff` to see all uncommitted changes\n")
	sb.WriteString("2. Review each changed file for:\n")
	sb.WriteString("   - Code correctness and logic errors\n")
	sb.WriteString("   - Error handling completeness\n")
	sb.WriteString("   - Test coverage for new code\n")
	sb.WriteString("   - Code style and naming conventions\n")
	sb.WriteString("   - Potential security issues\n")
	sb.WriteString("3. If you find issues, fix them directly\n")
	sb.WriteString("4. Run tests after making fixes\n")
	sb.WriteString("5. When review is complete and all issues are fixed, output REVIEW_DONE\n")
	sb.WriteString("6. If you encounter a blocking issue you cannot fix, output FAILED\n\n")
	sb.WriteString("Important: Only output REVIEW_DONE or FAILED as terminal signals.\n")

	return sb.String()
}

// buildSecondReviewPrompt creates the prompt for second review pass (after Codex).
func buildSecondReviewPrompt(codexFindings string) string {
	var sb strings.Builder
	sb.WriteString("You are reviewing code based on external analysis findings.\n\n")
	sb.WriteString("## Codex Analysis Findings\n\n")
	sb.WriteString(codexFindings)
	sb.WriteString("\n\n## Instructions\n\n")
	sb.WriteString("1. Review each finding from the Codex analysis\n")
	sb.WriteString("2. For valid issues, implement fixes directly\n")
	sb.WriteString("3. For false positives or non-issues, skip them\n")
	sb.WriteString("4. Run tests after making fixes\n")
	sb.WriteString("5. When all valid findings are addressed, output REVIEW_DONE\n")
	sb.WriteString("6. If you encounter a blocking issue, output FAILED\n\n")
	sb.WriteString("Important: Only output REVIEW_DONE or FAILED as terminal signals.\n")

	return sb.String()
}

// buildCodexPrompt creates the prompt for Codex analysis.
func buildCodexPrompt() string {
	return `Analyze the current git diff for code quality issues.

Focus on:
1. Logic errors and bugs
2. Missing error handling
3. Security vulnerabilities
4. Performance issues
5. Code that doesn't match the surrounding style

For each issue found, provide:
- File and line number
- Description of the issue
- Suggested fix

If no significant issues are found, state that the code looks good.

Output CODEX_DONE when analysis is complete.`
}

// buildContinuePrompt creates a prompt to continue after previous iteration.
func buildContinuePrompt(previousOutput string) string {
	var sb strings.Builder
	sb.WriteString("Continue from where you left off.\n\n")
	sb.WriteString("## Previous Output (last 500 chars)\n\n")

	output := previousOutput
	if len(output) > 500 {
		output = output[len(output)-500:]
	}
	sb.WriteString(output)
	sb.WriteString("\n\n")
	sb.WriteString("Continue executing tasks. Remember to output COMPLETED when done or FAILED if blocked.\n")

	return sb.String()
}
