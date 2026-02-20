# Replace Claude Command with Codex Command and Add Mode-Specific Reasoning/Search

## Overview
Switch the primary command flow from claude-oriented naming/behavior to codex-oriented behavior. Planning mode should enforce xhigh reasoning and include -search, while execution modes should enforce high reasoning without -search.

## Context
- Files involved: cmd/ralphex/main.go, cmd/ralphex/main_test.go, pkg/processor/runner.go, pkg/processor/runner_plan_mode_args_test.go, pkg/config/defaults/config, pkg/config/defaults/prompts/make_plan.txt, pkg/processor/prompts_test.go, README.md, CLAUDE.md, docs/custom-providers.md, llms.txt
- Related patterns: mode-based codex argument rewriting in pkg/processor/runner.go, command availability checks in cmd/ralphex/main.go, prompt validation and signal handling tests in pkg/processor
- Dependencies: existing codex CLI behavior and current custom-provider compatibility expectations

## Development Approach
- Testing approach: Regular (code first, then tests)
- Complete each task fully before moving to the next
- Keep changes minimal and local to existing command/argument rewriting paths
- Preserve non-codex custom provider behavior (no forced rewrite for non-codex commands)
- CRITICAL: every code-changing task MUST include new/updated tests
- CRITICAL: all tests must pass before starting next task

## Implementation Steps

### Task 1: Implement mode-aware codex argument policy in runner

**Files:**
- Modify: `pkg/processor/runner.go`
- Modify: `pkg/processor/runner_plan_mode_args_test.go`

- [x] Replace plan-only codex args helper with a mode-aware helper used during runner setup
- [x] In plan mode, enforce `model_reasoning_effort=xhigh`
- [x] In plan mode, ensure `-search` is present exactly once
- [x] In non-plan modes, enforce `model_reasoning_effort=high`
- [x] In non-plan modes, ensure `-search` is removed/not injected
- [x] Keep non-codex primary command arguments unchanged
- [x] Write/extend tests for replacement, append-if-missing, deduplication/idempotency, and non-codex no-op
- [x] Run package tests for `pkg/processor` and ensure pass before Task 2

### Task 2: Rename remaining runtime claude-command naming to codex-primary naming

**Files:**
- Modify: `cmd/ralphex/main.go`
- Modify: `cmd/ralphex/main_test.go`

- [x] Rename claude-specific helper/variable naming to codex-primary naming in runtime dependency checks
- [x] Keep command-resolution behavior unchanged (empty configured command still resolves to `codex`)
- [x] Preserve existing error behavior/messages unless rename makes updates necessary
- [x] Add/update tests that validate renamed helper behavior and dependency-check outcomes
- [x] Run package tests for `cmd/ralphex` and ensure pass before Task 3

### Task 3: Align defaults and planning prompt guidance

**Files:**
- Modify: `pkg/config/defaults/config`
- Modify: `pkg/config/defaults/prompts/make_plan.txt`
- Modify: `pkg/processor/prompts_test.go`

- [x] Update embedded default config comments/text to document split behavior: plan uses xhigh with `-search`, execution uses high
- [x] Update planning prompt instructions to explicitly allow web search during planning when needed
- [x] Keep prompt wording concise and consistent with existing signal requirements
- [x] Update prompt/config-related tests for new expected text
- [x] Run focused tests for `pkg/config` and `pkg/processor` and ensure pass before Task 4

### Task 4: Update user and internal docs for new defaults/behavior

**Files:**
- Modify: `README.md`
- Modify: `CLAUDE.md`
- Modify: `docs/custom-providers.md`
- Modify: `llms.txt`

- [x] Document codex-first behavior for planning and execution paths in this workflow
- [x] Document reasoning split: planning xhigh, execution high
- [x] Document plan-only `-search` behavior and custom-provider expectation to ignore unknown flags
- [x] Verify examples and config snippets stay consistent with runtime behavior
- [x] Run docs-adjacent checks (if any) and spot-check referenced commands/options for accuracy

### Task 5: Verify acceptance criteria

- [x] Manual test: run plan mode and confirm codex args include `model_reasoning_effort=xhigh` and `-search`
- [x] Manual test: run non-plan execution and confirm codex args use `model_reasoning_effort=high` without `-search`
- [x] Run full test suite with project command (`make test`)
- [x] Run linter with project command (`make lint`)
- [x] Verify coverage for touched packages remains at or above 80%+

### Task 6: Update documentation lifecycle

- [x] Confirm README and CLAUDE guidance reflects final implemented behavior
- [x] Add release note/changelog entry if project policy requires it
- [x] Move this plan to `docs/plans/completed/` after implementation is done
