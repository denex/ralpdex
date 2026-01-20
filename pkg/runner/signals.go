package runner

// Signal constants for execution control.
const (
	SignalCompleted  = "COMPLETED"
	SignalFailed     = "FAILED"
	SignalReviewDone = "REVIEW_DONE"
	SignalCodexDone  = "CODEX_DONE"
)

// IsTerminalSignal returns true if signal indicates execution should stop.
func IsTerminalSignal(signal string) bool {
	return signal == SignalCompleted || signal == SignalFailed
}

// IsReviewDone returns true if signal indicates review phase is complete.
func IsReviewDone(signal string) bool {
	return signal == SignalReviewDone
}

// IsCodexDone returns true if signal indicates codex phase is complete.
func IsCodexDone(signal string) bool {
	return signal == SignalCodexDone
}
