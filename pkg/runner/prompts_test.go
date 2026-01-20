package runner

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildTaskPrompt(t *testing.T) {
	tmpDir := t.TempDir()
	planFile := filepath.Join(tmpDir, "test-plan.md")

	planContent := `# Test Plan

- [ ] Task 1
- [ ] Task 2
`
	require.NoError(t, os.WriteFile(planFile, []byte(planContent), 0o600))

	t.Run("first iteration", func(t *testing.T) {
		prompt, err := buildTaskPrompt(planFile, 1)
		require.NoError(t, err)

		assert.Contains(t, prompt, "iteration 1")
		assert.Contains(t, prompt, "# Test Plan")
		assert.Contains(t, prompt, "Task 1")
		assert.Contains(t, prompt, "COMPLETED")
		assert.Contains(t, prompt, "FAILED")
	})

	t.Run("later iteration", func(t *testing.T) {
		prompt, err := buildTaskPrompt(planFile, 5)
		require.NoError(t, err)

		assert.Contains(t, prompt, "iteration 5")
	})

	t.Run("file not found", func(t *testing.T) {
		_, err := buildTaskPrompt("/nonexistent/file.md", 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "read plan file")
	})
}

func TestBuildFirstReviewPrompt(t *testing.T) {
	prompt := buildFirstReviewPrompt()

	assert.Contains(t, prompt, "reviewing code changes")
	assert.Contains(t, prompt, "git diff")
	assert.Contains(t, prompt, "REVIEW_DONE")
	assert.Contains(t, prompt, "FAILED")
	assert.Contains(t, prompt, "Code correctness")
	assert.Contains(t, prompt, "Error handling")
	assert.Contains(t, prompt, "Test coverage")
}

func TestBuildSecondReviewPrompt(t *testing.T) {
	findings := "Issue 1: Missing error check in foo.go:42"

	prompt := buildSecondReviewPrompt(findings)

	assert.Contains(t, prompt, "Codex Analysis Findings")
	assert.Contains(t, prompt, findings)
	assert.Contains(t, prompt, "REVIEW_DONE")
	assert.Contains(t, prompt, "FAILED")
	assert.Contains(t, prompt, "false positives")
}

func TestBuildCodexPrompt(t *testing.T) {
	prompt := buildCodexPrompt()

	assert.Contains(t, prompt, "git diff")
	assert.Contains(t, prompt, "Logic errors")
	assert.Contains(t, prompt, "Security vulnerabilities")
	assert.Contains(t, prompt, "CODEX_DONE")
}

func TestBuildContinuePrompt(t *testing.T) {
	t.Run("short output", func(t *testing.T) {
		prompt := buildContinuePrompt("short output")

		assert.Contains(t, prompt, "Continue from where you left off")
		assert.Contains(t, prompt, "short output")
		assert.Contains(t, prompt, "COMPLETED")
		assert.Contains(t, prompt, "FAILED")
	})

	t.Run("long output truncated", func(t *testing.T) {
		// use 'z' to avoid matching other letters in the prompt template
		longOutput := make([]byte, 1000)
		for i := range longOutput {
			longOutput[i] = 'z'
		}

		prompt := buildContinuePrompt(string(longOutput))

		// should only contain last 500 chars
		assert.Contains(t, prompt, "Previous Output (last 500 chars)")
		// count z's - should be exactly 500
		count := 0
		for _, c := range prompt {
			if c == 'z' {
				count++
			}
		}
		assert.Equal(t, 500, count)
	})
}
