package protocol

const (
	EventMiddlewareSummary EventType = "middleware.summary"
	EventRunMetrics        EventType = "run.metrics"
)

type MiddlewareSummary struct {
	Name           string        `json:"name"`
	Requires       []string      `json:"requires,omitempty"`
	ChainIndex     int           `json:"chainIndex,omitempty"`
	ChainSize      int           `json:"chainSize,omitempty"`
	Status         string        `json:"status"`
	DurationMs     int64         `json:"durationMs"`
	PromptSummary  string        `json:"promptSummary,omitempty"`
	OutputSummary  string        `json:"outputSummary,omitempty"`
	PlanTitle      string        `json:"planTitle,omitempty"`
	PlanStepCount  int           `json:"planStepCount,omitempty"`
	ArtifactCount  int           `json:"artifactCount,omitempty"`
	MemoryCount    int           `json:"memoryCount,omitempty"`
	WebResultCount int           `json:"webResultCount,omitempty"`
	Error          *ErrorPayload `json:"error,omitempty"`
}

type RunMetrics struct {
	RunID                       string        `json:"runId"`
	Status                      RunStatus     `json:"status"`
	Success                     bool          `json:"success"`
	DurationMs                  int64         `json:"durationMs"`
	MessageCount                int           `json:"messageCount,omitempty"`
	AssistantTurnCount          int           `json:"assistantTurnCount,omitempty"`
	EstimatedPromptTokens       int           `json:"estimatedPromptTokens,omitempty"`
	EstimatedConversationTokens int           `json:"estimatedConversationTokens,omitempty"`
	EstimatedOutputTokens       int           `json:"estimatedOutputTokens,omitempty"`
	EstimatedTotalTokens        int           `json:"estimatedTotalTokens,omitempty"`
	PromptVersion               string        `json:"promptVersion,omitempty"`
	Error                       *ErrorPayload `json:"error,omitempty"`
}

type HealthMetrics struct {
	Status                   string  `json:"status"`
	VisibleConversationCount int     `json:"visibleConversationCount"`
	DeletedConversationCount int     `json:"deletedConversationCount"`
	RunCount                 int     `json:"runCount"`
	ActiveCount              int     `json:"activeCount"`
	TerminalRuns             int     `json:"terminalRuns"`
	CompletedRuns            int     `json:"completedRuns"`
	FailedRuns               int     `json:"failedRuns"`
	CancelledRuns            int     `json:"cancelledRuns"`
	StoredRunCount           int     `json:"storedRunCount"`
	StoredTerminalRuns       int     `json:"storedTerminalRuns"`
	StoredCompletedRuns      int     `json:"storedCompletedRuns"`
	StoredFailedRuns         int     `json:"storedFailedRuns"`
	StoredCancelledRuns      int     `json:"storedCancelledRuns"`
	SuccessRate              float64 `json:"successRate"`
	AverageDurationMs        int64   `json:"averageDurationMs"`
	AverageEstimatedTokens   int     `json:"averageEstimatedTokens"`
	TotalEstimatedTokens     int     `json:"totalEstimatedTokens"`
}
