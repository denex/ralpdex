// Package runner provides the main orchestration loop for ralphex execution.
package runner

import (
	"context"
	"errors"
	"fmt"

	"github.com/umputun/ralphex/pkg/executor"
	"github.com/umputun/ralphex/pkg/progress"
)

// Mode represents the execution mode.
type Mode string

const (
	ModeFull      Mode = "full"       // full execution: tasks + reviews + codex
	ModeReview    Mode = "review"     // skip tasks, run full review pipeline
	ModeCodexOnly Mode = "codex-only" // skip tasks and first review, run only codex loop
)

// Config holds runner configuration.
type Config struct {
	PlanFile      string // path to plan file (required for full mode)
	Mode          Mode   // execution mode
	MaxIterations int    // maximum iterations for task phase
	Debug         bool   // enable debug output
	NoColor       bool   // disable color output
}

// Executor runs CLI commands and returns results.
type Executor interface {
	Run(ctx context.Context, prompt string) executor.Result
}

// Logger provides logging functionality.
type Logger interface {
	SetPhase(phase progress.Phase)
	Print(format string, args ...any)
	PrintRaw(format string, args ...any)
}

// Runner orchestrates the execution loop.
type Runner struct {
	cfg    Config
	log    Logger
	claude Executor
	codex  Executor
}

// New creates a new Runner with the given configuration.
func New(cfg Config, log *progress.Logger) *Runner {
	return &Runner{
		cfg: cfg,
		log: log,
		claude: &executor.ClaudeExecutor{
			OutputHandler: func(text string) {
				// stream output to logger
				log.PrintRaw("%s", text)
			},
			Debug: cfg.Debug,
		},
		codex: &executor.CodexExecutor{
			OutputHandler: func(text string) {
				log.PrintRaw("%s", text)
			},
			Debug: cfg.Debug,
		},
	}
}

// NewWithExecutors creates a new Runner with custom executors (for testing).
func NewWithExecutors(cfg Config, log Logger, claude, codex Executor) *Runner {
	return &Runner{
		cfg:    cfg,
		log:    log,
		claude: claude,
		codex:  codex,
	}
}

// Run executes the main loop based on configured mode.
func (r *Runner) Run(ctx context.Context) error {
	switch r.cfg.Mode {
	case ModeFull:
		return r.runFull(ctx)
	case ModeReview:
		return r.runReviewOnly(ctx)
	case ModeCodexOnly:
		return r.runCodexOnly(ctx)
	default:
		return fmt.Errorf("unknown mode: %s", r.cfg.Mode)
	}
}

// runFull executes the complete pipeline: tasks → review → codex → review.
func (r *Runner) runFull(ctx context.Context) error {
	if r.cfg.PlanFile == "" {
		return errors.New("plan file required for full mode")
	}

	// phase 1: task execution
	r.log.SetPhase(progress.PhaseTask)
	r.log.Print("starting task execution phase")

	if err := r.runTaskPhase(ctx); err != nil {
		return fmt.Errorf("task phase: %w", err)
	}

	// phase 2: first review (Claude)
	r.log.SetPhase(progress.PhaseReview)
	r.log.Print("starting first review phase")

	if err := r.runClaudeReview(ctx, buildFirstReviewPrompt()); err != nil {
		return fmt.Errorf("first review: %w", err)
	}

	// phase 3: codex analysis
	r.log.SetPhase(progress.PhaseCodex)
	r.log.Print("starting codex analysis phase")

	findings, err := r.runCodexPhase(ctx)
	if err != nil {
		return fmt.Errorf("codex phase: %w", err)
	}

	// phase 4: second review (Claude addresses codex findings)
	if findings != "" {
		r.log.SetPhase(progress.PhaseReview)
		r.log.Print("starting second review phase (addressing codex findings)")

		if err := r.runClaudeReview(ctx, buildSecondReviewPrompt(findings)); err != nil {
			return fmt.Errorf("second review: %w", err)
		}
	}

	r.log.Print("all phases completed successfully")
	return nil
}

// runReviewOnly executes only the review pipeline: review → codex → review.
func (r *Runner) runReviewOnly(ctx context.Context) error {
	// phase 1: first review
	r.log.SetPhase(progress.PhaseReview)
	r.log.Print("starting review phase")

	if err := r.runClaudeReview(ctx, buildFirstReviewPrompt()); err != nil {
		return fmt.Errorf("first review: %w", err)
	}

	// phase 2: codex analysis
	r.log.SetPhase(progress.PhaseCodex)
	r.log.Print("starting codex analysis phase")

	findings, err := r.runCodexPhase(ctx)
	if err != nil {
		return fmt.Errorf("codex phase: %w", err)
	}

	// phase 3: second review if needed
	if findings != "" {
		r.log.SetPhase(progress.PhaseReview)
		r.log.Print("starting second review phase (addressing codex findings)")

		if err := r.runClaudeReview(ctx, buildSecondReviewPrompt(findings)); err != nil {
			return fmt.Errorf("second review: %w", err)
		}
	}

	r.log.Print("review phases completed successfully")
	return nil
}

// runCodexOnly executes only the codex pipeline: codex → review.
func (r *Runner) runCodexOnly(ctx context.Context) error {
	// phase 1: codex analysis
	r.log.SetPhase(progress.PhaseCodex)
	r.log.Print("starting codex analysis phase")

	findings, err := r.runCodexPhase(ctx)
	if err != nil {
		return fmt.Errorf("codex phase: %w", err)
	}

	// phase 2: review if findings exist
	if findings != "" {
		r.log.SetPhase(progress.PhaseReview)
		r.log.Print("starting review phase (addressing codex findings)")

		if err := r.runClaudeReview(ctx, buildSecondReviewPrompt(findings)); err != nil {
			return fmt.Errorf("review: %w", err)
		}
	}

	r.log.Print("codex phases completed successfully")
	return nil
}

// runTaskPhase executes tasks until completion or max iterations.
func (r *Runner) runTaskPhase(ctx context.Context) error {
	var lastOutput string

	for i := 1; i <= r.cfg.MaxIterations; i++ {
		select {
		case <-ctx.Done():
			return fmt.Errorf("task phase: %w", ctx.Err())
		default:
		}

		r.log.Print("task iteration %d/%d", i, r.cfg.MaxIterations)

		var prompt string
		var err error

		if i == 1 {
			prompt, err = buildTaskPrompt(r.cfg.PlanFile, i)
			if err != nil {
				return err
			}
		} else {
			prompt = buildContinuePrompt(lastOutput)
		}

		result := r.claude.Run(ctx, prompt)
		if result.Error != nil {
			return fmt.Errorf("claude execution: %w", result.Error)
		}

		lastOutput = result.Output

		if IsTerminalSignal(result.Signal) {
			if result.Signal == SignalFailed {
				return errors.New("task execution failed (FAILED signal received)")
			}
			r.log.Print("task phase completed (COMPLETED signal received)")
			return nil
		}
	}

	return fmt.Errorf("max iterations (%d) reached without completion", r.cfg.MaxIterations)
}

// runClaudeReview runs Claude review with the given prompt until REVIEW_DONE.
func (r *Runner) runClaudeReview(ctx context.Context, prompt string) error {
	const maxReviewIterations = 10

	for i := 1; i <= maxReviewIterations; i++ {
		select {
		case <-ctx.Done():
			return fmt.Errorf("review: %w", ctx.Err())
		default:
		}

		r.log.Print("review iteration %d/%d", i, maxReviewIterations)

		result := r.claude.Run(ctx, prompt)
		if result.Error != nil {
			return fmt.Errorf("claude execution: %w", result.Error)
		}

		if result.Signal == SignalFailed {
			return errors.New("review failed (FAILED signal received)")
		}

		if IsReviewDone(result.Signal) {
			r.log.Print("review completed (REVIEW_DONE signal received)")
			return nil
		}

		// continue with output from previous iteration
		prompt = buildContinuePrompt(result.Output)
	}

	return fmt.Errorf("max review iterations (%d) reached", maxReviewIterations)
}

// runCodexPhase runs Codex analysis and returns findings.
func (r *Runner) runCodexPhase(ctx context.Context) (string, error) {
	result := r.codex.Run(ctx, buildCodexPrompt())
	if result.Error != nil {
		return "", fmt.Errorf("codex execution: %w", result.Error)
	}

	// check if codex found any issues
	if result.Output == "" {
		r.log.Print("codex found no issues")
		return "", nil
	}

	r.log.Print("codex analysis complete, found issues to review")
	return result.Output, nil
}
