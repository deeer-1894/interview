package protocol

type CopilotState string

const (
	CopilotStateStable           CopilotState = "stable"
	CopilotStateNeedsStructure   CopilotState = "needs_structure"
	CopilotStateNeedsSpecificity CopilotState = "needs_specificity"
	CopilotStateStuck            CopilotState = "stuck"
	CopilotStateAnxious          CopilotState = "anxious"
)

type CopilotFeedback struct {
	State          CopilotState `json:"state"`
	Summary        string       `json:"summary"`
	Triggers       []string     `json:"triggers,omitempty"`
	SuggestedMoves []string     `json:"suggestedMoves,omitempty"`
	Confidence     float64      `json:"confidence,omitempty"`
}

type CopilotHint struct {
	Title      string   `json:"title"`
	Summary    string   `json:"summary"`
	Focus      string   `json:"focus,omitempty"`
	Strategy   []string `json:"strategy,omitempty"`
	Guardrails []string `json:"guardrails,omitempty"`
}

type CopilotAssistResponse struct {
	Feedback CopilotFeedback `json:"feedback"`
	Hint     CopilotHint     `json:"hint"`
	Events   []Event         `json:"events,omitempty"`
}
