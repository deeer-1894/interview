package protocol

import (
	"strings"
	"time"
)

type InterviewMode string

const (
	ModeStandard        InterviewMode = "standard"
	ModeStress          InterviewMode = "stress"
	ModeWeaknessFocused InterviewMode = "weakness_focused"
	ModeSystemDesign    InterviewMode = "system_design"
	ModeResumeDeepDive  InterviewMode = "resume_deep_dive"
)

type InterviewPersona string

const (
	PersonaRigorous   InterviewPersona = "rigorous"
	PersonaCalm       InterviewPersona = "calm"
	PersonaSupportive InterviewPersona = "supportive"
	PersonaManager    InterviewPersona = "manager"
)

type OutputStyle string

const (
	OutputInterviewOnly      OutputStyle = "interview_only"
	OutputInterviewPlusScore OutputStyle = "interview_plus_score"
	OutputInterviewPlusStudy OutputStyle = "interview_plus_score_and_study_plan"
)

type RunStatus string

const (
	RunCreated        RunStatus = "created"
	RunRunning        RunStatus = "running"
	RunWaitingClarify RunStatus = "waiting_clarify"
	RunResuming       RunStatus = "resuming"
	RunCompleted      RunStatus = "completed"
	RunFailed         RunStatus = "failed"
	RunCancelled      RunStatus = "cancelled"
)

type RunPhase string

const (
	RunPhaseInitial      RunPhase = "initial"
	RunPhaseInterviewing RunPhase = "interviewing"
	RunPhaseEvaluating   RunPhase = "evaluating"
	RunPhaseStudyPlan    RunPhase = "study_plan"
	RunPhaseCompleted    RunPhase = "completed"
)

type InterviewPhase string

const (
	PhaseWarmup      InterviewPhase = "warmup"
	PhaseProbe       InterviewPhase = "probe"
	PhaseAdversarial InterviewPhase = "adversarial"
	PhaseStress      InterviewPhase = "stress"
	PhaseWrapup      InterviewPhase = "wrapup"
)

type DecisionReason string

const (
	ReasonMissingTradeoff          DecisionReason = "missing_tradeoff"
	ReasonLackImplementationDetail DecisionReason = "lack_implementation_detail"
	ReasonWeakSignalTimeout        DecisionReason = "weak_signal_timeout"
	ReasonWeakSignalObservability  DecisionReason = "weak_signal_observability"
	ReasonPressureTest             DecisionReason = "pressure_test"
	ReasonTopicSwitch              DecisionReason = "topic_switch"
	ReasonConfidenceConfirm        DecisionReason = "confidence_confirm"
	ReasonWrapupRequested          DecisionReason = "wrapup_requested"
	ReasonWrapupDueToBudget        DecisionReason = "wrapup_due_to_budget"
	ReasonProfileWeaknessFocus     DecisionReason = "profile_weakness_focus"
)

type EventType string

const (
	EventRunCreated        EventType = "run.created"
	EventRunStarted        EventType = "run.started"
	EventMessageDelta      EventType = "message.delta"
	EventMessageCompleted  EventType = "message.completed"
	EventToolCalled        EventType = "tool.called"
	EventToolCompleted     EventType = "tool.completed"
	EventPlanGenerated     EventType = "plan.generated"
	EventDecisionGenerated EventType = "decision.generated"
	EventTraceSpan         EventType = "trace.span"
	EventClarifyRequested  EventType = "clarify.requested"
	EventClarifyResumed    EventType = "clarify.resumed"
	EventCheckpointLoaded  EventType = "checkpoint.loaded"
	EventCheckpointSaved   EventType = "checkpoint.saved"
	EventPersonaSelected   EventType = "persona.selected"
	EventTraceGenerated    EventType = "interview_tree.generated"
	EventScoreGenerated    EventType = "score.generated"
	EventProfileUpdated    EventType = "profile.updated"
	EventReviewGenerated   EventType = "review.generated"
	EventCopilotHint       EventType = "copilot.hint"
	EventCopilotFeedback   EventType = "copilot.feedback"
	EventRunCompleted      EventType = "run.completed"
	EventRunCancelled      EventType = "run.cancelled"
	EventRunFailed         EventType = "run.failed"
	EventHeartbeat         EventType = "heartbeat"
)

type InterviewConfig struct {
	Skill           string           `json:"skill,omitempty"`
	SkillFocuses    []string         `json:"skillFocuses,omitempty"`
	Persona         InterviewPersona `json:"persona,omitempty"`
	Level           string           `json:"level"`
	Focus           string           `json:"focus"`
	Mode            InterviewMode    `json:"mode"`
	TimeBudget      string           `json:"timeBudget"`
	OutputStyle     OutputStyle      `json:"outputStyle"`
	EnableWebSearch bool             `json:"enableWebSearch,omitempty"`
}

func (c InterviewConfig) WithDefaults() InterviewConfig {
	c.SkillFocuses = normalizeProtocolStrings(c.SkillFocuses)
	if c.Level == "" {
		c.Level = "mid"
	}
	if c.Persona == "" {
		c.Persona = PersonaRigorous
	}
	if c.Focus == "" {
		c.Focus = "generalist"
	}
	if c.Mode == "" {
		c.Mode = ModeStandard
	}
	if c.TimeBudget == "" {
		c.TimeBudget = "25 minutes"
	}
	if c.OutputStyle == "" {
		c.OutputStyle = OutputInterviewPlusScore
	}
	return c
}

func normalizeProtocolStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	out := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		key := strings.ToLower(value)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, value)
	}
	return out
}

type ModelConfig struct {
	Provider string `json:"provider"`
	Model    string `json:"model"`
	APIKey   string `json:"apiKey,omitempty"`
	BaseURL  string `json:"baseUrl,omitempty"`
}

type Conversation struct {
	ID              string    `json:"id"`
	Title           string    `json:"title"`
	Status          string    `json:"status"`
	Pinned          bool      `json:"pinned,omitempty"`
	Archived        bool      `json:"archived,omitempty"`
	CurrentTask     string    `json:"currentTaskId,omitempty"`
	LatestRunID     string    `json:"latestRunId,omitempty"`
	LatestRunStatus RunStatus `json:"latestRunStatus,omitempty"`
	CreatedAt       time.Time `json:"createdAt"`
	UpdatedAt       time.Time `json:"updatedAt"`
}

type Task struct {
	ID             string          `json:"id"`
	ConversationID string          `json:"conversationId"`
	Title          string          `json:"title"`
	Prompt         string          `json:"prompt"`
	ArtifactIDs    []string        `json:"artifactIds,omitempty"`
	Config         InterviewConfig `json:"config"`
	ModelConfig    ModelConfig     `json:"modelConfig"`
	CreatedAt      time.Time       `json:"createdAt"`
	UpdatedAt      time.Time       `json:"updatedAt"`
}

type Run struct {
	ID             string              `json:"id"`
	ConversationID string              `json:"conversationId"`
	TaskID         string              `json:"taskId"`
	ArtifactIDs    []string            `json:"artifactIds,omitempty"`
	Status         RunStatus           `json:"status"`
	Phase          RunPhase            `json:"phase"`
	InterviewState *RunInterviewState  `json:"interviewState,omitempty"`
	TraceTree      *InterviewTraceTree `json:"traceTree,omitempty"`
	Input          string              `json:"input"`
	Output         string              `json:"output,omitempty"`
	LastError      string              `json:"lastError,omitempty"`
	CreatedAt      time.Time           `json:"createdAt"`
	UpdatedAt      time.Time           `json:"updatedAt"`
	CompletedAt    *time.Time          `json:"completedAt,omitempty"`
}

type ClarifyRequest struct {
	ID        string    `json:"id"`
	RunID     string    `json:"runId"`
	Question  string    `json:"question"`
	Field     string    `json:"field"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"createdAt"`
}

type Message struct {
	ID             string    `json:"id"`
	ConversationID string    `json:"conversationId"`
	TaskID         string    `json:"taskId"`
	RunID          string    `json:"runId"`
	Role           string    `json:"role"`
	Content        string    `json:"content"`
	CreatedAt      time.Time `json:"createdAt"`
}

type Event struct {
	ID             string    `json:"id"`
	ConversationID string    `json:"conversationId"`
	TaskID         string    `json:"taskId"`
	RunID          string    `json:"runId"`
	Type           EventType `json:"type"`
	Timestamp      time.Time `json:"timestamp"`
	Payload        any       `json:"payload"`
}

type RunRequest struct {
	TaskID          string   `json:"taskId"`
	ConversationID  string   `json:"conversationId"`
	Prompt          string   `json:"prompt,omitempty"`
	ArtifactIDs     []string `json:"artifactIds,omitempty"`
	Resume          bool     `json:"resume"`
	ClarifyResponse string   `json:"clarifyResponse,omitempty"`
}

type ResumeInput struct {
	Message     string          `json:"message"`
	Config      InterviewConfig `json:"config,omitempty"`
	ArtifactIDs []string        `json:"artifactIds,omitempty"`
}

type SkillSpec struct {
	Name            string   `json:"name"`
	Description     string   `json:"description"`
	Version         string   `json:"version,omitempty"`
	InstallSource   string   `json:"installSource,omitempty"`
	SourceURL       string   `json:"sourceUrl,omitempty"`
	ComposedOf      []string `json:"composedOf,omitempty"`
	Rating          float64  `json:"rating,omitempty"`
	RatingCount     int      `json:"ratingCount,omitempty"`
	FocusAreas      []string `json:"focusAreas"`
	SampleQuestions []string `json:"sampleQuestions"`
	FollowUps       []string `json:"followUps"`
	Scenarios       []string `json:"scenarios,omitempty"`
	Adversarial     []string `json:"adversarial,omitempty"`
	Pressure        []string `json:"pressure,omitempty"`
	ScoringAnchors  []string `json:"scoringAnchors"`
}

type Rubric struct {
	Title   string   `json:"title"`
	Anchors []string `json:"anchors"`
}

type DimensionScore struct {
	Name      string `json:"name"`
	Score     int    `json:"score"`
	MaxScore  int    `json:"maxScore,omitempty"`
	Rationale string `json:"rationale,omitempty"`
}

type Scorecard struct {
	Title           string           `json:"title"`
	Summary         string           `json:"summary,omitempty"`
	OverallScore    int              `json:"overallScore,omitempty"`
	OverallMaxScore int              `json:"overallMaxScore,omitempty"`
	Anchors         []string         `json:"anchors"`
	DimensionScores []DimensionScore `json:"dimensionScores,omitempty"`
	Strengths       []string         `json:"strengths,omitempty"`
	Gaps            []string         `json:"gaps,omitempty"`
	Improvements    []string         `json:"improvements,omitempty"`
	StudyPlan       []string         `json:"studyPlan,omitempty"`
}

type InterviewTraceNode struct {
	ID            string         `json:"id"`
	MessageID     string         `json:"messageId,omitempty"`
	ParentID      string         `json:"parentId,omitempty"`
	Depth         int            `json:"depth"`
	Kind          string         `json:"kind"`
	Round         int            `json:"round,omitempty"`
	Phase         InterviewPhase `json:"phase,omitempty"`
	Difficulty    int            `json:"difficulty,omitempty"`
	Scenario      string         `json:"scenario,omitempty"`
	Adversarial   bool           `json:"adversarial,omitempty"`
	Pressure      bool           `json:"pressure,omitempty"`
	Question      string         `json:"question"`
	AnswerSummary string         `json:"answerSummary,omitempty"`
	Topic         string         `json:"topic,omitempty"`
	Reason        string         `json:"reason,omitempty"`
	Explanation   string         `json:"explanation,omitempty"`
	ProfileHit    bool           `json:"profileHit,omitempty"`
	FocusHits     []string       `json:"focusHits,omitempty"`
	Signal        string         `json:"signal,omitempty"`
	WeakSignals   []string       `json:"weakSignals,omitempty"`
	StrongSignals []string       `json:"strongSignals,omitempty"`
}

type InterviewTraceTree struct {
	RunID         string               `json:"runId"`
	Persona       InterviewPersona     `json:"persona"`
	GeneratedAt   time.Time            `json:"generatedAt"`
	QuestionCount int                  `json:"questionCount"`
	Nodes         []InterviewTraceNode `json:"nodes"`
}

type ProfileDimension struct {
	Key             string              `json:"key"`
	Label           string              `json:"label"`
	Score           int                 `json:"score"`
	NormalizedScore int                 `json:"normalizedScore,omitempty"`
	EvidenceCount   int                 `json:"evidenceCount"`
	Summary         string              `json:"summary,omitempty"`
	LastUpdatedAt   *time.Time          `json:"lastUpdatedAt,omitempty"`
	RecentDelta     int                 `json:"recentDelta,omitempty"`
	Trend           []ProfileTrendPoint `json:"trend,omitempty"`
}

type PersonaStat struct {
	Persona InterviewPersona `json:"persona"`
	Count   int              `json:"count"`
}

type ProfileTrendPoint struct {
	Timestamp       time.Time `json:"timestamp"`
	Score           int       `json:"score"`
	NormalizedScore int       `json:"normalizedScore"`
}

type ProfileRadarPoint struct {
	Key             string `json:"key"`
	Label           string `json:"label"`
	NormalizedScore int    `json:"normalizedScore"`
}

type ProfileGrowthCurve struct {
	Key    string              `json:"key"`
	Label  string              `json:"label"`
	Points []ProfileTrendPoint `json:"points,omitempty"`
}

type CandidateProfile struct {
	ID               string               `json:"id"`
	InterviewCount   int                  `json:"interviewCount"`
	LastSkill        string               `json:"lastSkill,omitempty"`
	LastPersona      InterviewPersona     `json:"lastPersona,omitempty"`
	UpdatedAt        time.Time            `json:"updatedAt"`
	Dimensions       []ProfileDimension   `json:"dimensions,omitempty"`
	Radar            []ProfileRadarPoint  `json:"radar,omitempty"`
	GrowthCurves     []ProfileGrowthCurve `json:"growthCurves,omitempty"`
	PersonaUsage     []PersonaStat        `json:"personaUsage,omitempty"`
	StableStrengths  []string             `json:"stableStrengths,omitempty"`
	RecurringGaps    []string             `json:"recurringGaps,omitempty"`
	RecentChanges    []string             `json:"recentChanges,omitempty"`
	RecommendedFocus []string             `json:"recommendedFocus,omitempty"`
}

type ReviewSnapshot struct {
	RunID          string               `json:"runId"`
	GeneratedAt    time.Time            `json:"generatedAt"`
	InterviewState *RunInterviewState   `json:"interviewState,omitempty"`
	Decision       *DecisionAudit       `json:"decision,omitempty"`
	Scorecard      *Scorecard           `json:"scorecard,omitempty"`
	Trace          *InterviewTraceTree  `json:"trace,omitempty"`
	Profile        *CandidateProfile    `json:"profile,omitempty"`
	Agents         []AgentExecution     `json:"agents,omitempty"`
	Summary        *ReviewSummary       `json:"summary,omitempty"`
}

type ReviewSummary struct {
	Mode                    InterviewMode    `json:"mode,omitempty"`
	Persona                 InterviewPersona `json:"persona,omitempty"`
	CurrentPhase            InterviewPhase   `json:"currentPhase,omitempty"`
	PressureRound           int              `json:"pressureRound,omitempty"`
	AdversarialRound        int              `json:"adversarialRound,omitempty"`
	WrapupRound             int              `json:"wrapupRound,omitempty"`
	MostCommonWeakSignal    string           `json:"mostCommonWeakSignal,omitempty"`
	DecisionReason          DecisionReason   `json:"decisionReason,omitempty"`
	DecisionExplanation     string           `json:"decisionExplanation,omitempty"`
	RecommendedFocus        []string         `json:"recommendedFocus,omitempty"`
	HistoricalWeaknessesHit []string         `json:"historicalWeaknessesHit,omitempty"`
	NewWeaknesses           []string         `json:"newWeaknesses,omitempty"`
	ResolvedWeaknesses      []string         `json:"resolvedWeaknesses,omitempty"`
}

type PlanStep struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Kind        string `json:"kind"`
}

type ExecutionPlan struct {
	Title string     `json:"title"`
	Steps []PlanStep `json:"steps"`
}

type MemoryRecord struct {
	RunID      string    `json:"runId"`
	Content    string    `json:"content"`
	RecordedAt time.Time `json:"recordedAt"`
}

type Artifact struct {
	ID             string    `json:"id"`
	ConversationID string    `json:"conversationId"`
	TaskID         string    `json:"taskId,omitempty"`
	RunID          string    `json:"runId,omitempty"`
	Name           string    `json:"name"`
	ContentType    string    `json:"contentType"`
	Size           int64     `json:"size"`
	StorageKey     string    `json:"storageKey"`
	CreatedAt      time.Time `json:"createdAt"`
}

type WebSearchResult struct {
	Title   string `json:"title"`
	URL     string `json:"url"`
	Snippet string `json:"snippet"`
}

type CheckpointSnapshot struct {
	RunID             string             `json:"runId"`
	ConversationID    string             `json:"conversationId"`
	TaskID            string             `json:"taskId"`
	Prompt            string             `json:"prompt"`
	Input             string             `json:"input"`
	RunStatus         RunStatus          `json:"runStatus"`
	RunPhase          RunPhase           `json:"runPhase"`
	InterviewState    *RunInterviewState `json:"interviewState,omitempty"`
	ResumeCount       int                `json:"resumeCount"`
	Config            InterviewConfig    `json:"config"`
	ModelConfig       ModelConfig        `json:"modelConfig"`
	PendingClarifyFor string             `json:"pendingClarifyFor,omitempty"`
	RawState          []byte             `json:"rawState,omitempty"`
	UpdatedAt         time.Time          `json:"updatedAt"`
}

type InterviewRoundSnapshot struct {
	Round                  int                `json:"round"`
	Phase                  InterviewPhase     `json:"phase"`
	Difficulty             int                `json:"difficulty"`
	Scenario               string             `json:"scenario,omitempty"`
	Adversarial            bool               `json:"adversarial,omitempty"`
	Pressure               bool               `json:"pressure,omitempty"`
	Reason                 string             `json:"reason,omitempty"`
	Explanation            string             `json:"explanation,omitempty"`
	WeakSignals            []string           `json:"weakSignals,omitempty"`
	StrongSignals          []string           `json:"strongSignals,omitempty"`
	WeakSignalConfidence   map[string]float64 `json:"weakSignalConfidence,omitempty"`
	StrongSignalConfidence map[string]float64 `json:"strongSignalConfidence,omitempty"`
}

type RunInterviewState struct {
	Phase         InterviewPhase           `json:"phase"`
	Round         int                      `json:"round"`
	Difficulty    int                      `json:"difficulty"`
	WeakSignals   []string                 `json:"weakSignals,omitempty"`
	StrongSignals []string                 `json:"strongSignals,omitempty"`
	LastScenario  string                   `json:"lastScenario,omitempty"`
	LastDecision  *NextStepDecision        `json:"lastDecision,omitempty"`
	History       []InterviewRoundSnapshot `json:"history,omitempty"`
}

type NextStepDecision struct {
	KeepTopic          bool           `json:"keepTopic,omitempty"`
	SwitchTopic        bool           `json:"switchTopic,omitempty"`
	EscalatePressure   bool           `json:"escalatePressure,omitempty"`
	TriggerAdversarial bool           `json:"triggerAdversarial,omitempty"`
	IncreaseDifficulty bool           `json:"increaseDifficulty,omitempty"`
	RecommendedFocus   []string       `json:"recommendedFocus,omitempty"`
	Reason             DecisionReason `json:"reason,omitempty"`
	Explanation        string         `json:"explanation,omitempty"`
}

type AnswerSignalSummary struct {
	WeakSignals               []string           `json:"weakSignals,omitempty"`
	StrongSignals             []string           `json:"strongSignals,omitempty"`
	WeakSignalConfidence      map[string]float64 `json:"weakSignalConfidence,omitempty"`
	StrongSignalConfidence    map[string]float64 `json:"strongSignalConfidence,omitempty"`
	TooGeneric                bool               `json:"tooGeneric,omitempty"`
	HasTradeoff               bool               `json:"hasTradeoff,omitempty"`
	HasConcreteImplementation bool               `json:"hasConcreteImplementation,omitempty"`
}

type DecisionAudit struct {
	State         RunInterviewState   `json:"state"`
	Mode          InterviewMode       `json:"mode"`
	PromptVersion string              `json:"promptVersion,omitempty"`
	Analysis      AnswerSignalSummary `json:"analysis"`
	ProfileFocus  []string            `json:"profileFocus,omitempty"`
	Decision      NextStepDecision    `json:"decision"`
}
