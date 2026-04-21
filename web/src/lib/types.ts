export type InterviewMode = "standard" | "stress" | "weakness_focused" | "system_design" | "resume_deep_dive";
export type InterviewPersona = "rigorous" | "calm" | "supportive" | "manager";
export type OutputStyle = "interview_only" | "interview_plus_score" | "interview_plus_score_and_study_plan";
export type RunStatus = "created" | "running" | "waiting_clarify" | "resuming" | "completed" | "failed" | "cancelled";
export type RunPhase = "initial" | "interviewing" | "evaluating" | "study_plan" | "completed";
export type InterviewPhase = "warmup" | "probe" | "adversarial" | "stress" | "wrapup";
export type DecisionReason =
  | "missing_tradeoff"
  | "lack_implementation_detail"
  | "weak_signal_timeout"
  | "weak_signal_observability"
  | "pressure_test"
  | "topic_switch"
  | "confidence_confirm"
  | "wrapup_requested"
  | "wrapup_due_to_budget"
  | "profile_weakness_focus";

export interface InterviewConfig {
  skill?: string;
  skillFocuses?: string[];
  persona?: InterviewPersona;
  level: string;
  focus: string;
  mode: InterviewMode;
  timeBudget: string;
  outputStyle: OutputStyle;
  enableWebSearch?: boolean;
}

export interface ModelConfig {
  provider: string;
  model: string;
  apiKey?: string;
  baseUrl?: string;
}

export interface Conversation {
  id: string;
  title: string;
  status: string;
  pinned?: boolean;
  archived?: boolean;
  currentTaskId?: string;
  latestRunId?: string;
  latestRunStatus?: RunStatus;
  createdAt: string;
  updatedAt: string;
}

export interface Task {
  id: string;
  conversationId: string;
  title: string;
  prompt: string;
  artifactIds?: string[];
  config: InterviewConfig;
  modelConfig: ModelConfig;
  createdAt: string;
  updatedAt: string;
}

export interface Run {
  id: string;
  conversationId: string;
  taskId: string;
  artifactIds?: string[];
  status: RunStatus;
  phase: RunPhase;
  interviewState?: RunInterviewState;
  traceTree?: InterviewTraceTree;
  input: string;
  output?: string;
  lastError?: string;
  createdAt: string;
  updatedAt: string;
  completedAt?: string;
}

export interface InterviewRoundSnapshot {
  round: number;
  phase: InterviewPhase;
  difficulty: number;
  scenario?: string;
  adversarial?: boolean;
  pressure?: boolean;
  reason?: string;
  explanation?: string;
  weakSignals?: string[];
  strongSignals?: string[];
  weakSignalConfidence?: Record<string, number>;
  strongSignalConfidence?: Record<string, number>;
}

export interface NextStepDecision {
  keepTopic?: boolean;
  switchTopic?: boolean;
  escalatePressure?: boolean;
  triggerAdversarial?: boolean;
  increaseDifficulty?: boolean;
  recommendedFocus?: string[];
  reason?: DecisionReason;
  explanation?: string;
}

export interface RunInterviewState {
  phase: InterviewPhase;
  round: number;
  difficulty: number;
  weakSignals?: string[];
  strongSignals?: string[];
  lastScenario?: string;
  lastDecision?: NextStepDecision;
  history?: InterviewRoundSnapshot[];
}

export interface Message {
  id: string;
  conversationId: string;
  taskId: string;
  runId: string;
  role: "user" | "assistant";
  content: string;
  createdAt: string;
}

export type CopilotState = "stable" | "needs_structure" | "needs_specificity" | "stuck" | "anxious";

export interface CopilotFeedback {
  state: CopilotState;
  summary: string;
  triggers?: string[];
  suggestedMoves?: string[];
  confidence?: number;
}

export interface CopilotHint {
  title: string;
  summary: string;
  focus?: string;
  strategy?: string[];
  guardrails?: string[];
}

export interface CopilotAssistResponse {
  feedback: CopilotFeedback;
  hint: CopilotHint;
  events?: RunEvent[];
}

export interface RunEvent {
  id: string;
  conversationId: string;
  taskId: string;
  runId: string;
  type:
    | "run.created"
    | "run.started"
    | "message.delta"
    | "message.completed"
    | "tool.called"
    | "tool.completed"
    | "plan.generated"
    | "decision.generated"
    | "trace.span"
    | "clarify.requested"
    | "clarify.resumed"
    | "checkpoint.loaded"
    | "checkpoint.saved"
    | "persona.selected"
    | "interview_tree.generated"
    | "score.generated"
    | "profile.updated"
    | "review.generated"
    | "copilot.hint"
    | "copilot.feedback"
    | "run.completed"
    | "run.cancelled"
    | "run.failed"
    | "heartbeat";
  timestamp: string;
  payload: unknown;
}

export interface AnswerSignalSummary {
  weakSignals?: string[];
  strongSignals?: string[];
  weakSignalConfidence?: Record<string, number>;
  strongSignalConfidence?: Record<string, number>;
  tooGeneric?: boolean;
  hasTradeoff?: boolean;
  hasConcreteImplementation?: boolean;
}

export interface DimensionScore {
  name: string;
  score: number;
  maxScore?: number;
  rationale?: string;
}

export interface Scorecard {
  title: string;
  summary?: string;
  overallScore?: number;
  overallMaxScore?: number;
  anchors: string[];
  dimensionScores?: DimensionScore[];
  strengths?: string[];
  gaps?: string[];
  improvements?: string[];
  studyPlan?: string[];
}

export interface InterviewTraceNode {
  id: string;
  messageId?: string;
  parentId?: string;
  depth: number;
  kind: string;
  round?: number;
  phase?: InterviewPhase;
  difficulty?: number;
  scenario?: string;
  adversarial?: boolean;
  pressure?: boolean;
  question: string;
  answerSummary?: string;
  topic?: string;
  reason?: string;
  explanation?: string;
  profileHit?: boolean;
  focusHits?: string[];
  signal?: string;
  weakSignals?: string[];
  strongSignals?: string[];
}

export interface InterviewTraceTree {
  runId: string;
  persona: InterviewPersona;
  generatedAt: string;
  questionCount: number;
  nodes: InterviewTraceNode[];
}

export interface ProfileDimension {
  key: string;
  label: string;
  score: number;
  normalizedScore?: number;
  evidenceCount: number;
  summary?: string;
  lastUpdatedAt?: string;
  recentDelta?: number;
  trend?: ProfileTrendPoint[];
}

export interface PersonaStat {
  persona: InterviewPersona;
  count: number;
}

export interface ProfileTrendPoint {
  timestamp: string;
  score: number;
  normalizedScore: number;
}

export interface ProfileRadarPoint {
  key: string;
  label: string;
  normalizedScore: number;
}

export interface ProfileGrowthCurve {
  key: string;
  label: string;
  points?: ProfileTrendPoint[];
}

export interface CandidateProfile {
  id: string;
  interviewCount: number;
  lastSkill?: string;
  lastPersona?: InterviewPersona;
  updatedAt: string;
  dimensions?: ProfileDimension[];
  radar?: ProfileRadarPoint[];
  growthCurves?: ProfileGrowthCurve[];
  personaUsage?: PersonaStat[];
  stableStrengths?: string[];
  recurringGaps?: string[];
  recentChanges?: string[];
  recommendedFocus?: string[];
}

export type AgentRole = "interviewer" | "evaluator";

export interface AgentExecution {
  role: AgentRole;
  status: RunStatus;
  promptVersion?: string;
  startedAt: string;
  completedAt: string;
  sharedContextSummary?: string;
  inputSummary?: string;
  outputSummary?: string;
  error?: ErrorPayload;
}

export interface ErrorPayload {
  kind: "unknown" | "interview" | "tool" | "model";
  stage?: string;
  operation?: string;
  message: string;
  retryable?: boolean;
}

export interface ReviewSnapshot {
  runId: string;
  generatedAt: string;
  interviewState?: RunInterviewState;
  decision?: DecisionAudit;
  scorecard?: Scorecard;
  trace?: InterviewTraceTree;
  profile?: CandidateProfile;
  agents?: AgentExecution[];
  summary?: ReviewSummary;
}

export interface ReviewSummary {
  mode?: InterviewMode;
  persona?: InterviewPersona;
  currentPhase?: InterviewPhase;
  pressureRound?: number;
  adversarialRound?: number;
  wrapupRound?: number;
  mostCommonWeakSignal?: string;
  decisionReason?: DecisionReason;
  decisionExplanation?: string;
  recommendedFocus?: string[];
  historicalWeaknessesHit?: string[];
  newWeaknesses?: string[];
  resolvedWeaknesses?: string[];
}

export interface DecisionAudit {
  state: RunInterviewState;
  mode?: InterviewMode;
  promptVersion?: string;
  analysis: AnswerSignalSummary;
  profileFocus?: string[];
  decision: NextStepDecision;
}

export interface Artifact {
  id: string;
  conversationId: string;
  taskId?: string;
  runId?: string;
  name: string;
  contentType: string;
  size: number;
  storageKey: string;
  createdAt: string;
}

export interface ArtifactDocument {
  artifact: Artifact;
  content: string;
}

export interface SkillMetadata {
  name: string;
  description: string;
  version?: string;
  focusAreas?: string[];
  composedOf?: string[];
  capabilityBoundaries?: string[];
  installSource?: string;
  sourceUrl?: string;
  rating?: number;
  ratingCount?: number;
}

export interface SkillDocument {
  name: string;
  description: string;
  version?: string;
  focusAreas?: string[];
  composedOf?: string[];
  capabilityBoundaries?: string[];
  sampleQuestions?: string[];
  followUps?: string[];
  scenarios?: string[];
  adversarial?: string[];
  pressure?: string[];
  scoringAnchors?: string[];
  installSource?: string;
  sourceUrl?: string;
  rating?: number;
  ratingCount?: number;
  content: string;
}

export interface WebSearchResult {
  title: string;
  url: string;
  snippet: string;
}

export interface ConversationDetail {
  conversation: Conversation;
  tasks: Task[];
  runs: Run[];
}

export interface RunDetail {
  run: Run;
  messages: Message[];
  events: RunEvent[];
}
