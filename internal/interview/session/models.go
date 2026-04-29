package session

import (
	"time"

	"mockinterview/internal/interview/resume"
)

type Status string

const (
	StatusCreated     Status = "created"
	StatusRunning     Status = "running"
	StatusInterrupted Status = "interrupted"
	StatusCompleted   Status = "completed"
	StatusFailed      Status = "failed"
)

type MessageRole string

const (
	RoleUser      MessageRole = "user"
	RoleAssistant MessageRole = "assistant"
	RoleTool      MessageRole = "tool"
	RoleSystem    MessageRole = "system"
)

type InterviewSession struct {
	ID              string                `json:"id" bson:"_id,omitempty"`
	UserID          string                `json:"userId" bson:"userId"`
	Role            string                `json:"role" bson:"role"`
	Level           string                `json:"level" bson:"level"`
	Mode            string                `json:"mode" bson:"mode"`
	ResumeProfileID string                `json:"resumeProfileId" bson:"resumeProfileId"`
	ActiveProjectID string                `json:"activeProjectId,omitempty" bson:"activeProjectId,omitempty"`
	Round           int                   `json:"round" bson:"round"`
	Status          Status                `json:"status" bson:"status"`
	CoveredTopics   []string              `json:"coveredTopics,omitempty" bson:"coveredTopics,omitempty"`
	RecentQuestions []QuestionFingerprint `json:"recentQuestions,omitempty" bson:"recentQuestions,omitempty"`
	CheckpointID    string                `json:"checkpointId,omitempty" bson:"checkpointId,omitempty"`
	CreatedAt       time.Time             `json:"createdAt" bson:"createdAt"`
	UpdatedAt       time.Time             `json:"updatedAt" bson:"updatedAt"`
}

type Message struct {
	ID        string      `json:"id" bson:"_id,omitempty"`
	SessionID string      `json:"sessionId" bson:"sessionId"`
	UserID    string      `json:"userId" bson:"userId"`
	Role      MessageRole `json:"role" bson:"role"`
	Content   string      `json:"content" bson:"content"`
	ToolName  string      `json:"toolName,omitempty" bson:"toolName,omitempty"`
	CreatedAt time.Time   `json:"createdAt" bson:"createdAt"`
}

type QuestionFingerprint struct {
	ProjectID string `json:"projectId,omitempty" bson:"projectId,omitempty"`
	Topic     string `json:"topic,omitempty" bson:"topic,omitempty"`
	Intent    string `json:"intent,omitempty" bson:"intent,omitempty"`
	TextHash  string `json:"textHash,omitempty" bson:"textHash,omitempty"`
}

type TurnContext struct {
	SessionID         string                `json:"sessionId"`
	UserID            string                `json:"userId"`
	Role              string                `json:"role"`
	Level             string                `json:"level"`
	Mode              string                `json:"mode"`
	Round             int                   `json:"round"`
	ResumeSummary     string                `json:"resumeSummary,omitempty"`
	ResumeSkills      []string              `json:"resumeSkills,omitempty"`
	ResumeRawText     string                `json:"resumeRawText,omitempty"`
	ActiveProject     resume.Project        `json:"activeProject"`
	OtherProjects     []resume.ProjectBrief `json:"otherProjects,omitempty"`
	CoveredTopics     []string              `json:"coveredTopics,omitempty"`
	RecentQuestions   []QuestionFingerprint `json:"recentQuestions,omitempty"`
	RecentMessages    []Message             `json:"recentMessages,omitempty"`
	LastAnswer        string                `json:"lastAnswer,omitempty"`
	ForbiddenFacts    []string              `json:"forbiddenFacts,omitempty"`
	RecommendedSkills []string              `json:"recommendedSkills,omitempty"`
}

func NewInterviewSession(now time.Time) InterviewSession {
	return InterviewSession{
		Status:    StatusCreated,
		CreatedAt: now,
		UpdatedAt: now,
	}
}
