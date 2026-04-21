package protocol

import "time"

type AgentRole string

const (
	AgentRoleInterviewer AgentRole = "interviewer"
	AgentRoleEvaluator   AgentRole = "evaluator"
)

type AgentExecution struct {
	Role                 AgentRole     `json:"role"`
	Status               RunStatus     `json:"status"`
	PromptVersion        string        `json:"promptVersion,omitempty"`
	StartedAt            time.Time     `json:"startedAt"`
	CompletedAt          time.Time     `json:"completedAt"`
	SharedContextSummary string        `json:"sharedContextSummary,omitempty"`
	InputSummary         string        `json:"inputSummary,omitempty"`
	OutputSummary        string        `json:"outputSummary,omitempty"`
	Error                *ErrorPayload `json:"error,omitempty"`
}
