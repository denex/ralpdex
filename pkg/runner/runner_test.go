package runner

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/umputun/ralphex/pkg/executor"
	"github.com/umputun/ralphex/pkg/progress"
)

// mockExecutor is a test double for Executor interface.
type mockExecutor struct {
	results []executor.Result
	calls   []string
	idx     int
}

func (m *mockExecutor) Run(_ context.Context, prompt string) executor.Result {
	m.calls = append(m.calls, prompt)
	if m.idx >= len(m.results) {
		return executor.Result{Error: errors.New("no more mock results")}
	}
	result := m.results[m.idx]
	m.idx++
	return result
}

// mockLogger is a test double for Logger interface.
type mockLogger struct {
	messages []string
	phase    progress.Phase
}

func (m *mockLogger) SetPhase(phase progress.Phase) {
	m.phase = phase
}

func (m *mockLogger) Print(format string, args ...any) {
	m.messages = append(m.messages, format)
}

func (m *mockLogger) PrintRaw(format string, _ ...any) {
	m.messages = append(m.messages, format)
}

func TestRunner_Run_UnknownMode(t *testing.T) {
	log := &mockLogger{}
	claude := &mockExecutor{}
	codex := &mockExecutor{}

	r := NewWithExecutors(Config{Mode: "invalid"}, log, claude, codex)
	err := r.Run(context.Background())

	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown mode")
}

func TestRunner_RunFull_NoPlanFile(t *testing.T) {
	log := &mockLogger{}
	claude := &mockExecutor{}
	codex := &mockExecutor{}

	r := NewWithExecutors(Config{Mode: ModeFull}, log, claude, codex)
	err := r.Run(context.Background())

	require.Error(t, err)
	assert.Contains(t, err.Error(), "plan file required")
}

func TestRunner_RunFull_Success(t *testing.T) {
	tmpDir := t.TempDir()
	planFile := filepath.Join(tmpDir, "plan.md")
	require.NoError(t, os.WriteFile(planFile, []byte("# Plan\n- [ ] Task 1"), 0o600))

	log := &mockLogger{}
	claude := &mockExecutor{
		results: []executor.Result{
			{Output: "task done", Signal: SignalCompleted},           // task phase completes
			{Output: "review done", Signal: SignalReviewDone},        // first review completes
			{Output: "second review done", Signal: SignalReviewDone}, // second review completes
		},
	}
	codex := &mockExecutor{
		results: []executor.Result{
			{Output: "found issue in foo.go"}, // codex finds issues
		},
	}

	r := NewWithExecutors(Config{Mode: ModeFull, PlanFile: planFile, MaxIterations: 10}, log, claude, codex)
	err := r.Run(context.Background())

	require.NoError(t, err)
	assert.Len(t, claude.calls, 3) // task + first review + second review
	assert.Len(t, codex.calls, 1)
}

func TestRunner_RunFull_NoCodexFindings(t *testing.T) {
	tmpDir := t.TempDir()
	planFile := filepath.Join(tmpDir, "plan.md")
	require.NoError(t, os.WriteFile(planFile, []byte("# Plan\n- [ ] Task 1"), 0o600))

	log := &mockLogger{}
	claude := &mockExecutor{
		results: []executor.Result{
			{Output: "task done", Signal: SignalCompleted},
			{Output: "review done", Signal: SignalReviewDone},
		},
	}
	codex := &mockExecutor{
		results: []executor.Result{
			{Output: ""}, // codex finds nothing
		},
	}

	r := NewWithExecutors(Config{Mode: ModeFull, PlanFile: planFile, MaxIterations: 10}, log, claude, codex)
	err := r.Run(context.Background())

	require.NoError(t, err)
	assert.Len(t, claude.calls, 2) // task + first review (no second review since no codex findings)
}

func TestRunner_RunReviewOnly_Success(t *testing.T) {
	log := &mockLogger{}
	claude := &mockExecutor{
		results: []executor.Result{
			{Output: "review done", Signal: SignalReviewDone},
			{Output: "second review done", Signal: SignalReviewDone},
		},
	}
	codex := &mockExecutor{
		results: []executor.Result{
			{Output: "found issue"},
		},
	}

	r := NewWithExecutors(Config{Mode: ModeReview}, log, claude, codex)
	err := r.Run(context.Background())

	require.NoError(t, err)
	assert.Len(t, claude.calls, 2)
	assert.Len(t, codex.calls, 1)
}

func TestRunner_RunCodexOnly_Success(t *testing.T) {
	log := &mockLogger{}
	claude := &mockExecutor{
		results: []executor.Result{
			{Output: "review done", Signal: SignalReviewDone},
		},
	}
	codex := &mockExecutor{
		results: []executor.Result{
			{Output: "found issue"},
		},
	}

	r := NewWithExecutors(Config{Mode: ModeCodexOnly}, log, claude, codex)
	err := r.Run(context.Background())

	require.NoError(t, err)
	assert.Len(t, claude.calls, 1) // only review after codex
	assert.Len(t, codex.calls, 1)
}

func TestRunner_RunCodexOnly_NoFindings(t *testing.T) {
	log := &mockLogger{}
	claude := &mockExecutor{}
	codex := &mockExecutor{
		results: []executor.Result{
			{Output: ""}, // no findings
		},
	}

	r := NewWithExecutors(Config{Mode: ModeCodexOnly}, log, claude, codex)
	err := r.Run(context.Background())

	require.NoError(t, err)
	assert.Empty(t, claude.calls) // no review needed
	assert.Len(t, codex.calls, 1)
}

func TestRunner_TaskPhase_FailedSignal(t *testing.T) {
	tmpDir := t.TempDir()
	planFile := filepath.Join(tmpDir, "plan.md")
	require.NoError(t, os.WriteFile(planFile, []byte("# Plan"), 0o600))

	log := &mockLogger{}
	claude := &mockExecutor{
		results: []executor.Result{
			{Output: "error occurred", Signal: SignalFailed},
		},
	}
	codex := &mockExecutor{}

	r := NewWithExecutors(Config{Mode: ModeFull, PlanFile: planFile, MaxIterations: 10}, log, claude, codex)
	err := r.Run(context.Background())

	require.Error(t, err)
	assert.Contains(t, err.Error(), "FAILED signal")
}

func TestRunner_TaskPhase_MaxIterations(t *testing.T) {
	tmpDir := t.TempDir()
	planFile := filepath.Join(tmpDir, "plan.md")
	require.NoError(t, os.WriteFile(planFile, []byte("# Plan"), 0o600))

	log := &mockLogger{}
	claude := &mockExecutor{
		results: []executor.Result{
			{Output: "working..."},
			{Output: "still working..."},
			{Output: "more work..."},
		},
	}
	codex := &mockExecutor{}

	r := NewWithExecutors(Config{Mode: ModeFull, PlanFile: planFile, MaxIterations: 3}, log, claude, codex)
	err := r.Run(context.Background())

	require.Error(t, err)
	assert.Contains(t, err.Error(), "max iterations")
}

func TestRunner_TaskPhase_ContextCanceled(t *testing.T) {
	tmpDir := t.TempDir()
	planFile := filepath.Join(tmpDir, "plan.md")
	require.NoError(t, os.WriteFile(planFile, []byte("# Plan"), 0o600))

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	log := &mockLogger{}
	claude := &mockExecutor{}
	codex := &mockExecutor{}

	r := NewWithExecutors(Config{Mode: ModeFull, PlanFile: planFile, MaxIterations: 10}, log, claude, codex)
	err := r.Run(ctx)

	require.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
}

func TestRunner_ClaudeReview_FailedSignal(t *testing.T) {
	log := &mockLogger{}
	claude := &mockExecutor{
		results: []executor.Result{
			{Output: "error", Signal: SignalFailed},
		},
	}
	codex := &mockExecutor{}

	r := NewWithExecutors(Config{Mode: ModeReview}, log, claude, codex)
	err := r.Run(context.Background())

	require.Error(t, err)
	assert.Contains(t, err.Error(), "FAILED signal")
}

func TestRunner_ClaudeReview_MaxIterations(t *testing.T) {
	log := &mockLogger{}

	// create 10 results without REVIEW_DONE signal
	results := make([]executor.Result, 10)
	for i := range results {
		results[i] = executor.Result{Output: "still reviewing..."}
	}

	claude := &mockExecutor{results: results}
	codex := &mockExecutor{}

	r := NewWithExecutors(Config{Mode: ModeReview}, log, claude, codex)
	err := r.Run(context.Background())

	require.Error(t, err)
	assert.Contains(t, err.Error(), "max review iterations")
}

func TestRunner_CodexPhase_Error(t *testing.T) {
	log := &mockLogger{}
	claude := &mockExecutor{
		results: []executor.Result{
			{Output: "review done", Signal: SignalReviewDone},
		},
	}
	codex := &mockExecutor{
		results: []executor.Result{
			{Error: errors.New("codex error")},
		},
	}

	r := NewWithExecutors(Config{Mode: ModeReview}, log, claude, codex)
	err := r.Run(context.Background())

	require.Error(t, err)
	assert.Contains(t, err.Error(), "codex")
}

func TestRunner_ClaudeExecution_Error(t *testing.T) {
	tmpDir := t.TempDir()
	planFile := filepath.Join(tmpDir, "plan.md")
	require.NoError(t, os.WriteFile(planFile, []byte("# Plan"), 0o600))

	log := &mockLogger{}
	claude := &mockExecutor{
		results: []executor.Result{
			{Error: errors.New("claude error")},
		},
	}
	codex := &mockExecutor{}

	r := NewWithExecutors(Config{Mode: ModeFull, PlanFile: planFile, MaxIterations: 10}, log, claude, codex)
	err := r.Run(context.Background())

	require.Error(t, err)
	assert.Contains(t, err.Error(), "claude execution")
}
