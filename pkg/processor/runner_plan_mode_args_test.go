package processor

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAdjustCodexArgsForPlanMode(t *testing.T) {
	t.Run("plan_mode_replaces_reasoning_and_adds_search", func(t *testing.T) {
		args := `exec --dangerously-bypass-approvals-and-sandbox -c model="gpt-5.3-codex" -c model_reasoning_effort=high`

		got := adjustCodexPrimaryArgsForMode(ModePlan, "codex", args)

		assert.Contains(t, got, "model_reasoning_effort=xhigh")
		assert.NotContains(t, got, "model_reasoning_effort=high")
		assert.Equal(t, 1, strings.Count(got, "web_search=live"))
		assert.NotContains(t, got, "--search")
		assert.NotContains(t, got, "-search")
	})

	t.Run("plan_mode_appends_reasoning_if_missing", func(t *testing.T) {
		args := `exec --dangerously-bypass-approvals-and-sandbox -c model="gpt-5.3-codex"`

		got := adjustCodexPrimaryArgsForMode(ModePlan, "/usr/local/bin/codex", args)

		assert.Contains(t, got, "model_reasoning_effort=xhigh")
		assert.Equal(t, 1, strings.Count(got, "web_search=live"))
	})

	t.Run("non_plan_mode_enforces_high_and_removes_explicit_search_overrides", func(t *testing.T) {
		args := `exec -search --search web_search=live -c web_search=cached -c features.web_search_request=true -c model="gpt-5.3-codex" -c model_reasoning_effort=medium features.web_search_request=false`

		got := adjustCodexPrimaryArgsForMode(ModeFull, "codex", args)

		assert.Contains(t, got, "model_reasoning_effort=high")
		assert.NotContains(t, got, "web_search=")
		assert.NotContains(t, got, "features.web_search_request=")
		assert.NotContains(t, got, "model_reasoning_effort=medium")
	})

	t.Run("deduplicates_reasoning_and_search_and_is_idempotent", func(t *testing.T) {
		args := `exec --search -c web_search=cached -c model_reasoning_effort=high -search -c features.web_search_request=true -c model_reasoning_effort=medium`

		got := adjustCodexPrimaryArgsForMode(ModePlan, "codex", args)
		gotAgain := adjustCodexPrimaryArgsForMode(ModePlan, "codex", got)

		assert.Equal(t, got, gotAgain)
		assert.Equal(t, 1, strings.Count(got, "web_search=live"))
		assert.Equal(t, 1, strings.Count(got, "model_reasoning_effort=xhigh"))
		assert.NotContains(t, got, "features.web_search_request=")
		assert.NotContains(t, got, "--search")
	})

	t.Run("keeps_args_for_non_codex_command", func(t *testing.T) {
		args := `--dangerously-skip-permissions --output-format stream-json --verbose`

		got := adjustCodexPrimaryArgsForMode(ModePlan, "claude", args)

		assert.Equal(t, args, got)
	})
}

func TestIsCodexPrimaryCommand(t *testing.T) {
	assert.True(t, isCodexPrimaryCommand(""))
	assert.True(t, isCodexPrimaryCommand("codex"))
	assert.True(t, isCodexPrimaryCommand("codex.exe"))
	assert.True(t, isCodexPrimaryCommand("/usr/local/bin/codex"))
	assert.True(t, isCodexPrimaryCommand(`C:\Tools\codex.exe`))
	assert.False(t, isCodexPrimaryCommand("claude"))
}
