import {
  ArrowUpRight,
  Check,
  ChevronDownCircle,
  ChevronDown,
  CircleDashed,
  Copy,
  Download,
  Ellipsis,
  FolderKanban,
  FilePenLine,
  FileUp,
  Flame,
  Hourglass,
  LoaderCircle,
  LockKeyhole,
  MessageSquareQuote,
  Plus,
  Radar,
  PencilLine,
  RotateCcw,
  Sparkles,
  Archive,
  ChevronLeft,
  ChevronRight,
  Pin,
  Trash2,
  Upload,
  Waypoints,
  X,
} from "lucide-react";
import type { ChangeEvent, Dispatch, ReactNode, SetStateAction } from "react";
import { lazy, Suspense, useDeferredValue, useEffect, useMemo, useRef, useState } from "react";

import { Button } from "@/components/ui/button";
import { SessionContextModal } from "@/components/session/session-context-modal";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Textarea } from "@/components/ui/textarea";
import { ConversationSection } from "@/components/workbench/conversation-section";
import { EmptyRunState } from "@/components/workbench/empty-run-state";
import { RunComposerModal } from "@/components/workbench/run-composer-modal";
import { cancelRun, createConversation, createRun, createTask, createSkill, createTextArtifact, deleteArtifact, deleteConversation, getArtifactContent, getConversation, getRunReview, getSkill, listArtifacts, listConversations, listSkills, requestCopilotHint, resumeRun, updateArtifact, updateConversation, updateSkill, uploadArtifact, uploadSkill } from "@/lib/api";
import type { Artifact, ArtifactDocument, CandidateProfile, Conversation, CopilotFeedback, CopilotHint, DecisionAudit, DecisionReason, InterviewConfig, InterviewMode, InterviewPersona, InterviewPhase, InterviewTraceNode, InterviewTraceTree, Message, ModelConfig, OutputStyle, ReviewSnapshot, Run, RunEvent, Scorecard, SkillDocument, SkillMetadata, Task } from "@/lib/types";
import { useRunStream } from "@/lib/use-run-stream";

const MarkdownBubbleContent = lazy(async () => {
  const module = await import("@/components/chat/markdown-bubble-content");
  return { default: module.MarkdownBubbleContent };
});

const defaultConfig: InterviewConfig = {
  skill: "",
  skillFocuses: [],
  persona: "rigorous",
  level: "中级",
  focus: "generalist",
  mode: "standard",
  timeBudget: "25 分钟",
  outputStyle: "interview_plus_score",
  enableWebSearch: false,
};

const defaultModelConfig: ModelConfig = {
  provider: "",
  model: "",
  apiKey: "",
  baseUrl: "",
};

const defaultSkillDocument: SkillDocument = {
  name: "",
  description: "",
  version: "",
  focusAreas: [],
  composedOf: [],
  capabilityBoundaries: [],
  installSource: "local",
  sourceUrl: "",
  rating: 0,
  ratingCount: 0,
  content: "",
};

const levelOptions = [
  { value: "初级", label: "初级" },
  { value: "中级", label: "中级" },
  { value: "高级", label: "高级" },
  { value: "专家", label: "专家" },
];

const timeBudgetOptions = [
  { value: "15 分钟", label: "15 分钟" },
  { value: "25 分钟", label: "25 分钟" },
  { value: "45 分钟", label: "45 分钟" },
  { value: "60 分钟", label: "60 分钟" },
];

const personaOptions: Array<{ value: InterviewPersona; label: string; hint: string }> = [
  { value: "rigorous", label: "严格拷打型", hint: "压强更高，追问更快" },
  { value: "calm", label: "冷静专业型", hint: "专业克制，稳定深入" },
  { value: "supportive", label: "启发引导型", hint: "允许恢复，但保持技术标准" },
  { value: "manager", label: "业务负责人型", hint: "更关注业务影响和执行判断" },
];

const interviewModeOptions: Array<{ value: InterviewConfig["mode"]; label: string; hint: string }> = [
  { value: "standard", label: "标准面试", hint: "平衡深挖、压力和节奏控制" },
  { value: "stress", label: "压力面试", hint: "更早进入压力与对抗阶段" },
  { value: "weakness_focused", label: "查漏补缺", hint: "围绕历史薄弱项持续深挖" },
  { value: "system_design", label: "系统设计专项", hint: "优先架构、可靠性与可观测性" },
  { value: "resume_deep_dive", label: "简历深挖", hint: "围绕经历、项目和 ownership 追问" },
];

type ArtifactEditorDraft = {
  id?: string;
  name: string;
  contentType: string;
  content: string;
};

type MiniTraceTreeNode = InterviewTraceNode & {
  children: MiniTraceTreeNode[];
};

const defaultArtifactDraft: ArtifactEditorDraft = {
  name: "",
  contentType: "text/markdown",
  content: "",
};

const wrapupAssistantCues = [
  "面试到此结束",
  "面试到这里结束",
  "本场面试到这里结束",
  "感谢你的参与",
  "最终评分",
  "综合得分",
  "学习计划",
  "总评：",
];

const explicitWrapupRequestCues = [
  "请结束",
  "现在结束",
  "到这里结束",
  "结束这场面试",
  "结束本场面试",
  "结束面试",
  "面试到这里结束",
  "面试到此结束",
  "不要继续追问",
  "停止追问",
  "wrap up",
  "end the interview",
  "finish the interview",
];

function normalizeWrapupText(content: string) {
  return content.trim().toLowerCase().replace(/\s+/g, " ");
}

function isWrapupAssistantMessage(content: string) {
  const normalized = normalizeWrapupText(content);
  return wrapupAssistantCues.some((cue) => normalized.includes(normalizeWrapupText(cue)));
}

function isExplicitWrapupRequestMessage(content: string) {
  const normalized = normalizeWrapupText(content);
  return explicitWrapupRequestCues.some((cue) => normalized.includes(normalizeWrapupText(cue)));
}

function filterConversationMessagesForDisplay(
  messages: Message[],
  run: Run | null,
  reviewSnapshot: ReviewSnapshot | null,
) {
  const decisionReason = reviewSnapshot?.summary?.decisionReason ?? run?.interviewState?.lastDecision?.reason;
  const shouldSuppressRogueFollowups =
    decisionReason === "wrapup_requested" ||
    run?.phase === "evaluating" ||
    run?.phase === "completed" ||
    run?.interviewState?.phase === "wrapup";

  if (!shouldSuppressRogueFollowups) {
    return messages;
  }

  const visible: Message[] = [];
  let wrapupRequested = false;

  for (const message of messages) {
    const content = message.content?.trim() ?? "";
    if (!content) {
      visible.push(message);
      continue;
    }

    if (message.role === "user" && isExplicitWrapupRequestMessage(content)) {
      wrapupRequested = true;
      visible.push(message);
      continue;
    }

    if (wrapupRequested && message.role === "assistant") {
      if (isWrapupAssistantMessage(content)) {
        visible.push(message);
      }
      continue;
    }

    visible.push(message);
  }

  return visible;
}

export default function App() {
  const [workspaceTitle, setWorkspaceTitle] = useState("系统面试工作台");
  const [prompt, setPrompt] = useState("请模拟一场 Go agent 开发岗位的技术面试，并在最后给出结构化评分。");
  const [config, setConfig] = useState<InterviewConfig>(defaultConfig);
  const [modelConfig, setModelConfig] = useState<ModelConfig>(defaultModelConfig);
  const [conversations, setConversations] = useState<Conversation[]>([]);
  const [selectedConversationId, setSelectedConversationId] = useState<string | null>(null);
  const [selectedTask, setSelectedTask] = useState<Task | null>(null);
  const [selectedConversationRuns, setSelectedConversationRuns] = useState<Run[]>([]);
  const [selectedRunId, setSelectedRunId] = useState<string | null>(null);
  const [selectedArtifactIDs, setSelectedArtifactIDs] = useState<string[]>([]);
  const [artifacts, setArtifacts] = useState<Artifact[]>([]);
  const [skills, setSkills] = useState<SkillMetadata[]>([]);
  const [skillEditorOpen, setSkillEditorOpen] = useState(false);
  const [skillEditorMode, setSkillEditorMode] = useState<"create" | "edit">("create");
  const [skillDraft, setSkillDraft] = useState<SkillDocument>(defaultSkillDocument);
  const [skillBusy, setSkillBusy] = useState(false);
  const [artifactEditorOpen, setArtifactEditorOpen] = useState(false);
  const [artifactEditorMode, setArtifactEditorMode] = useState<"create" | "edit">("create");
  const [artifactDraft, setArtifactDraft] = useState<ArtifactEditorDraft>(defaultArtifactDraft);
  const [artifactBusy, setArtifactBusy] = useState(false);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [copilotBusy, setCopilotBusy] = useState(false);
  const [copiedMessageId, setCopiedMessageId] = useState<string | null>(null);
  const [error, setError] = useState("");
  const [showScrollToBottom, setShowScrollToBottom] = useState(false);
  const [showWorkspaceSidebar, setShowWorkspaceSidebar] = useState(true);
  const [runComposerOpen, setRunComposerOpen] = useState(false);
  const [showRawTelemetry, setShowRawTelemetry] = useState(false);
  const [activeReplyContextPanel, setActiveReplyContextPanel] = useState<"persona" | "skill" | "materials" | null>(null);
  const [openConversationMenuId, setOpenConversationMenuId] = useState<string | null>(null);
  const [editingConversationId, setEditingConversationId] = useState<string | null>(null);
  const [editingConversationTitle, setEditingConversationTitle] = useState("");
  const [highlightedMessageId, setHighlightedMessageId] = useState<string | null>(null);
  const [reviewSnapshot, setReviewSnapshot] = useState<ReviewSnapshot | null>(null);
  const [reviewLoading, setReviewLoading] = useState(false);

  const { run, messages, events, isLoading: runLoading, error: runError, appendLocalMessage, appendLocalEvents } = useRunStream(selectedRunId);
  const messageViewportRef = useRef<HTMLDivElement>(null);
  const messagesEndRef = useRef<HTMLDivElement>(null);
  const messageNodeRefs = useRef<Record<string, HTMLDivElement | null>>({});
  const shouldAutoScrollRef = useRef(true);
  const followRunOutputRef = useRef(false);
  const previousLastMessageRef = useRef<{ id: string; content: string } | null>(null);
  const selectedConversationIdRef = useRef<string | null>(null);
  const selectedRunIdRef = useRef<string | null>(null);
  const hasAutoSelectedInitialConversationRef = useRef(false);

  useEffect(() => {
    void refreshConversations();
    void refreshSkills();
  }, []);

  useEffect(() => {
    const lastMessage = messages[messages.length - 1];
    const previousLastMessage = previousLastMessageRef.current;
    previousLastMessageRef.current = lastMessage
      ? {
          id: lastMessage.id,
          content: lastMessage.content,
        }
      : null;

    if (!lastMessage || !shouldAutoScrollRef.current || !followRunOutputRef.current) {
      return;
    }

    const appendedAssistantMessage =
      lastMessage.role === "assistant" &&
      (!previousLastMessage || previousLastMessage.id !== lastMessage.id);
    const updatedStreamingAssistantMessage =
      lastMessage.role === "assistant" &&
      previousLastMessage?.id === lastMessage.id &&
      previousLastMessage.content !== lastMessage.content;

    if (!appendedAssistantMessage && !updatedStreamingAssistantMessage) {
      return;
    }

    messagesEndRef.current?.scrollIntoView({
      behavior: updatedStreamingAssistantMessage ? "auto" : "smooth",
    });
  }, [messages]);

  useEffect(() => {
    if (!selectedConversationId) return;
    void loadConversation(selectedConversationId);
    void refreshArtifacts(selectedConversationId);
  }, [selectedConversationId]);

  useEffect(() => {
    selectedConversationIdRef.current = selectedConversationId;
  }, [selectedConversationId]);

  useEffect(() => {
    selectedRunIdRef.current = selectedRunId;
  }, [selectedRunId]);

  useEffect(() => {
    previousLastMessageRef.current = null;
    followRunOutputRef.current = false;
    shouldAutoScrollRef.current = false;
    setShowScrollToBottom(false);
  }, [selectedRunId]);

  useEffect(() => {
    if (runError) {
      setError(runError);
    }
  }, [runError]);

  useEffect(() => {
    if (!copiedMessageId) return;
    const timer = window.setTimeout(() => setCopiedMessageId(null), 1600);
    return () => window.clearTimeout(timer);
  }, [copiedMessageId]);

  useEffect(() => {
    if (!highlightedMessageId) {
      return;
    }
    const timer = window.setTimeout(() => setHighlightedMessageId(null), 2200);
    return () => window.clearTimeout(timer);
  }, [highlightedMessageId]);

  useEffect(() => {
    function handleWindowClick() {
      setOpenConversationMenuId(null);
    }
    window.addEventListener("click", handleWindowClick);
    return () => window.removeEventListener("click", handleWindowClick);
  }, []);

  const latestClarify = useMemo(() => {
    return [...events]
      .reverse()
      .find((event) => event.type === "clarify.requested");
  }, [events]);
  const deferredEvents = useDeferredValue(events);

  const timelineEvents = useMemo(() => {
    return deferredEvents.filter((event) => event.type !== "heartbeat");
  }, [deferredEvents]);

  const telemetryHighlights = useMemo(() => {
    const priority = new Set([
      "run.started",
      "clarify.requested",
      "clarify.resumed",
      "checkpoint.loaded",
      "checkpoint.saved",
      "tool.called",
      "tool.completed",
      "copilot.feedback",
      "copilot.hint",
      "score.generated",
      "decision.generated",
      "review.generated",
      "run.failed",
      "run.cancelled",
      "run.completed",
    ]);
    return timelineEvents.filter((event) => priority.has(event.type));
  }, [timelineEvents]);

  const activeClarify = useMemo(() => {
    if (run?.status !== "waiting_clarify") {
      return null;
    }
    return latestClarify ?? null;
  }, [latestClarify, run?.status]);

  const latestCheckpoint = useMemo(() => {
    return [...events]
      .reverse()
      .find((event) => event.type === "checkpoint.saved" || event.type === "checkpoint.loaded");
  }, [events]);

  const latestFailure = useMemo(() => {
    return [...events].reverse().find((event) => event.type === "run.failed") ?? null;
  }, [events]);

  const latestCopilotFeedback = useMemo(() => {
    return (([...events].reverse().find((event) => event.type === "copilot.feedback")?.payload ?? null) as CopilotFeedback | null) ?? null;
  }, [events]);

  const latestCopilotHint = useMemo(() => {
    return (([...events].reverse().find((event) => event.type === "copilot.hint")?.payload ?? null) as CopilotHint | null) ?? null;
  }, [events]);

  const copilotFeed = useMemo(() => {
    return events.filter((event) => event.type === "copilot.feedback" || event.type === "copilot.hint").slice(-4);
  }, [events]);

  const currentModeLabel = formatInterviewModeLabel(selectedTask?.config.mode ?? config.mode);
  const currentPersonaLabel = getPersonaMeta(selectedTask?.config.persona ?? config.persona).label;
  const currentPhaseLabel = formatInterviewPhaseLabel(run?.interviewState?.phase ?? run?.phase);
  const currentTimeBudget = selectedTask?.config.timeBudget ?? config.timeBudget;
  const currentRunStatusLabel = activeClarify ? "等待澄清" : formatRunStatus(run?.status ?? "created");

  const isStreamingAssistant = useMemo(
    () => messages.some((message) => message.id.startsWith("streaming-") && message.role === "assistant"),
    [messages]
  );
  const visibleMessages = useMemo(
    () => filterConversationMessagesForDisplay(messages, run, reviewSnapshot),
    [messages, reviewSnapshot, run]
  );
  const latestAssistantMessage = useMemo(
    () => [...visibleMessages].reverse().find((message) => message.role === "assistant") ?? null,
    [visibleMessages]
  );
  const isGeneratingFinalReview = useMemo(() => {
    if (run?.phase === "evaluating") {
      return true;
    }
    if (!(run?.status === "running" || run?.status === "resuming")) {
      return false;
    }
    const content = latestAssistantMessage?.content?.trim() ?? "";
    if (!content) {
      return false;
    }
    return isWrapupAssistantMessage(content);
  }, [latestAssistantMessage?.content, run?.phase, run?.status]);
  const reviewSignalVersion = useMemo(
    () =>
      events.filter((event) =>
        event.type === "score.generated" ||
        event.type === "review.generated" ||
        event.type === "profile.updated" ||
        event.type === "run.completed"
      ).length,
    [events]
  );
  const shouldShowInlineReview = Boolean(
    selectedRunId &&
      (run?.phase === "evaluating" ||
        run?.status === "completed" ||
        isGeneratingFinalReview ||
        reviewSnapshot?.scorecard ||
        reviewSnapshot?.profile ||
        reviewSnapshot?.trace)
  );

  useEffect(() => {
    if (!selectedRunId) {
      setReviewSnapshot(null);
      setReviewLoading(false);
      return;
    }
    const activeRunId = selectedRunId;

    const hasReviewSignals = reviewSignalVersion > 0;
    const needsReview =
      run?.phase === "evaluating" ||
      run?.status === "completed" ||
      isGeneratingFinalReview ||
      hasReviewSignals;
    if (!needsReview) {
      setReviewSnapshot(null);
      setReviewLoading(false);
      return;
    }

    let cancelled = false;
    async function loadReview() {
      setReviewLoading(true);
      try {
        const next = await getRunReview(activeRunId);
        if (!cancelled) {
          setReviewSnapshot(next);
        }
      } catch {
        if (!cancelled) {
          setReviewSnapshot((current) => current);
        }
      } finally {
        if (!cancelled) {
          setReviewLoading(false);
        }
      }
    }

    void loadReview();
    return () => {
      cancelled = true;
    };
  }, [selectedRunId, run?.phase, run?.status, isGeneratingFinalReview, reviewSignalVersion]);

  const activeConversations = useMemo(
    () =>
      conversations
        .filter((conversation) => !conversation.archived)
        .slice()
        .sort(compareConversationOrder),
    [conversations]
  );

  const archivedConversations = useMemo(
    () =>
      conversations
        .filter((conversation) => conversation.archived)
        .slice()
        .sort(compareConversationOrder),
    [conversations]
  );

  useEffect(() => {
    if (hasAutoSelectedInitialConversationRef.current) {
      return;
    }
    if (selectedConversationId) {
      hasAutoSelectedInitialConversationRef.current = true;
      return;
    }
    const candidate =
      conversations
        .filter((conversation) => !conversation.archived)
        .slice()
        .sort((left, right) => new Date(right.updatedAt).getTime() - new Date(left.updatedAt).getTime())[0] ?? null;
    if (!candidate) {
      return;
    }
    hasAutoSelectedInitialConversationRef.current = true;
    setSelectedConversationId(candidate.id);
  }, [conversations, selectedConversationId]);

  function handleMessageViewportScroll() {
    const viewport = messageViewportRef.current;
    if (!viewport) {
      shouldAutoScrollRef.current = false;
      followRunOutputRef.current = false;
      setShowScrollToBottom(false);
      return;
    }
    const distanceFromBottom = viewport.scrollHeight - viewport.scrollTop - viewport.clientHeight;
    shouldAutoScrollRef.current = distanceFromBottom < 72;
    if (distanceFromBottom >= 72) {
      followRunOutputRef.current = false;
    }
    setShowScrollToBottom(distanceFromBottom >= 72);
  }

  function scrollMessagesToBottom() {
    shouldAutoScrollRef.current = true;
    followRunOutputRef.current = true;
    setShowScrollToBottom(false);
    messagesEndRef.current?.scrollIntoView({ behavior: "smooth" });
  }

  async function handleCopyMessage(message: Message) {
    try {
      await navigator.clipboard.writeText(message.content);
      setCopiedMessageId(message.id);
    } catch (err) {
      setError(err instanceof Error ? err.message : "复制消息失败");
    }
  }

  async function handleCopyConversationTitle(title: string) {
    try {
      await navigator.clipboard.writeText(title);
      setError("");
    } catch (err) {
      setError(err instanceof Error ? err.message : "复制标题失败");
    }
  }

  function focusMessage(messageID?: string) {
    if (!messageID) {
      return;
    }
    const node = messageNodeRefs.current[messageID];
    if (!node) {
      return;
    }
    setHighlightedMessageId(messageID);
    shouldAutoScrollRef.current = false;
    followRunOutputRef.current = false;
    node.scrollIntoView({ behavior: "smooth", block: "center" });
  }

  function startRenameConversation(conversation: Conversation) {
    setEditingConversationId(conversation.id);
    setEditingConversationTitle(conversation.title);
    setOpenConversationMenuId(null);
  }

  function cancelRenameConversation() {
    setEditingConversationId(null);
    setEditingConversationTitle("");
  }

  async function submitRenameConversation(conversation: Conversation) {
    const nextTitle = editingConversationTitle.trim();
    if (!nextTitle || nextTitle === conversation.title) {
      cancelRenameConversation();
      return;
    }
    try {
      await updateConversation(conversation.id, { title: nextTitle });
      if (selectedConversationId === conversation.id) {
        setWorkspaceTitle(nextTitle);
      }
      await refreshConversations();
      cancelRenameConversation();
    } catch (err) {
      setError(err instanceof Error ? err.message : "重命名工作区失败");
    }
  }

  async function handleToggleConversationPin(conversation: Conversation) {
    try {
      await updateConversation(conversation.id, { pinned: !conversation.pinned });
      await refreshConversations();
      setOpenConversationMenuId(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : "更新置顶状态失败");
    }
  }

  async function handleToggleConversationArchive(conversation: Conversation) {
    try {
      await updateConversation(conversation.id, { archived: !conversation.archived });
      await refreshConversations();
      if (selectedConversationId === conversation.id) {
        await loadConversation(conversation.id);
      }
      setOpenConversationMenuId(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : "更新归档状态失败");
    }
  }

  async function handleDeleteConversation(conversation: Conversation) {
    if (!window.confirm(`确认删除工作区“${conversation.title}”吗？`)) {
      return;
    }
    try {
      await deleteConversation(conversation.id);
      if (selectedConversationId === conversation.id) {
        handleNewWorkspace();
      }
      await refreshConversations();
      setOpenConversationMenuId(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : "删除工作区失败");
    }
  }

  async function refreshConversations() {
    try {
      const next = await listConversations();
      setConversations(next);
    } catch (err) {
      setError(err instanceof Error ? err.message : "加载工作区失败");
    }
  }

  async function loadConversation(conversationId: string) {
    try {
      const detail = await getConversation(conversationId);
      if (selectedConversationIdRef.current !== conversationId) {
        return;
      }

      const preservedRunId = selectedRunIdRef.current;
      const preservedRun =
        preservedRunId && detail.runs.some((entry) => entry.id === preservedRunId)
          ? detail.runs.find((entry) => entry.id === preservedRunId) ?? null
          : null;
      const nextRun =
        preservedRun ??
        detail.runs.find((entry) => entry.id === detail.conversation.latestRunId) ??
        detail.runs[0] ??
        null;
      const nextTask =
        (nextRun ? detail.tasks.find((task) => task.id === nextRun.taskId) : null) ??
        detail.tasks[0] ??
        null;

      setWorkspaceTitle(detail.conversation.title);
      setSelectedConversationRuns(detail.runs);
      setSelectedTask(nextTask);
      setSelectedRunId(nextRun?.id ?? null);
      setSelectedArtifactIDs(nextTask?.artifactIds ?? []);
      setError("");
    } catch (err) {
      setError(err instanceof Error ? err.message : "加载工作区详情失败");
    }
  }

  async function refreshArtifacts(conversationId: string) {
    try {
      const next = await listArtifacts(conversationId);
      setArtifacts(next);
    } catch (err) {
      setError(err instanceof Error ? err.message : "加载材料失败");
    }
  }

  async function refreshSkills() {
    try {
      const next = await listSkills();
      setSkills(next);
    } catch (err) {
      setError(err instanceof Error ? err.message : "加载技能列表失败");
    }
  }

  async function openCreateSkill() {
    setSkillEditorMode("create");
    setSkillDraft(defaultSkillDocument);
    setSkillEditorOpen(true);
  }

  async function openEditSkill(name: string) {
    setSkillBusy(true);
    setError("");
    try {
      const detail = await getSkill(name);
      setSkillEditorMode("edit");
      setSkillDraft(detail);
      setSkillEditorOpen(true);
    } catch (err) {
      setError(err instanceof Error ? err.message : "加载技能详情失败");
    } finally {
      setSkillBusy(false);
    }
  }

  async function handleSaveSkill(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!skillDraft.name.trim() || !skillDraft.description.trim() || !skillDraft.content.trim()) {
      setError("技能名称、描述和指令不能为空");
      return;
    }

    setSkillBusy(true);
    setError("");
    try {
      const saved =
        skillEditorMode === "create"
          ? await createSkill(skillDraft)
          : await updateSkill(skillDraft.name, skillDraft);
      await refreshSkills();
      setConfig((current) => ({ ...current, skill: saved.name }));
      setSkillEditorOpen(false);
      setSkillDraft(defaultSkillDocument);
    } catch (err) {
      setError(err instanceof Error ? err.message : "保存技能失败");
    } finally {
      setSkillBusy(false);
    }
  }

  async function handleSkillUpload(event: ChangeEvent<HTMLInputElement>) {
    const file = event.target.files?.[0];
    event.target.value = "";
    if (!file) {
      return;
    }

    setSkillBusy(true);
    setError("");
    try {
      const saved = await uploadSkill(file);
      await refreshSkills();
      setConfig((current) => ({ ...current, skill: saved.name }));
    } catch (err) {
      setError(err instanceof Error ? err.message : "上传技能失败");
    } finally {
      setSkillBusy(false);
    }
  }

  async function launchInterview() {
    setIsSubmitting(true);
    setError("");

    try {
      const launchTitle = buildWorkspaceTitleFromPrompt(prompt) || workspaceTitle || "新的面试工作区";
      const selectedConversation =
        selectedConversationId !== null
          ? conversations.find((item) => item.id === selectedConversationId) ?? null
          : null;
      const conversation = selectedConversation ?? (await createConversation(launchTitle));
      if (conversation.title !== launchTitle) {
        await updateConversation(conversation.id, { title: launchTitle });
        conversation.title = launchTitle;
      }

      const task = await createTask({
        conversationId: conversation.id,
        title: launchTitle,
        prompt,
        artifactIds: selectedArtifactIDs,
        config,
        modelConfig,
      });
      const nextRun = await createRun({
        taskId: task.id,
        prompt,
        artifactIds: selectedArtifactIDs,
      });

      setSelectedConversationId(conversation.id);
      setWorkspaceTitle(launchTitle);
      setSelectedTask(task);
      setSelectedRunId(nextRun.id);
      followRunOutputRef.current = true;
      shouldAutoScrollRef.current = true;
      appendLocalMessage({
        id: `local-user-${nextRun.id}-${Date.now()}`,
        conversationId: conversation.id,
        taskId: task.id,
        runId: nextRun.id,
        role: "user",
        content: prompt,
        createdAt: new Date().toISOString(),
      });
      await refreshConversations();
      await refreshArtifacts(conversation.id);
    } catch (err) {
      setError(err instanceof Error ? err.message : "启动面试失败");
    } finally {
      setIsSubmitting(false);
    }
  }

  async function handleLaunchInterview(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    await launchInterview();
  }

  async function handleReply(reply: string) {
    if (!selectedRunId || !reply.trim() || isSubmitting) {
      return false;
    }

    setIsSubmitting(true);
    setError("");

    try {
      const nextReply = reply.trim();
      followRunOutputRef.current = true;
      shouldAutoScrollRef.current = true;
      appendLocalMessage({
        id: `local-user-${selectedRunId}-${Date.now()}`,
        conversationId: run?.conversationId ?? "",
        taskId: run?.taskId ?? "",
        runId: selectedRunId,
        role: "user",
        content: nextReply,
        createdAt: new Date().toISOString(),
      });
      await resumeRun(selectedRunId, {
        message: nextReply,
        config,
        artifactIds: selectedArtifactIDs,
      });
      await refreshConversations();
      return true;
    } catch (err) {
      setError(err instanceof Error ? err.message : "发送回复失败");
      return false;
    } finally {
      setIsSubmitting(false);
    }
  }

  async function handleRetryRun() {
    if (!selectedTask || isSubmitting) {
      return;
    }

    setIsSubmitting(true);
    setError("");

    try {
      const nextRun = await createRun({
        taskId: selectedTask.id,
        prompt: selectedTask.prompt,
        artifactIds: selectedTask.artifactIds,
      });
      setSelectedRunId(nextRun.id);
      followRunOutputRef.current = true;
      shouldAutoScrollRef.current = true;
      await refreshConversations();
      if (selectedConversationId) {
        await refreshArtifacts(selectedConversationId);
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : "启动恢复运行失败");
    } finally {
      setIsSubmitting(false);
    }
  }

  async function handleCancelRun() {
    if (!selectedRunId || isSubmitting) {
      return;
    }

    setIsSubmitting(true);
    setError("");

    try {
      await cancelRun(selectedRunId);
      await refreshConversations();
    } catch (err) {
      setError(err instanceof Error ? err.message : "取消运行失败");
    } finally {
      setIsSubmitting(false);
    }
  }

  async function handleRequestCopilot() {
    if (!selectedRunId || copilotBusy || isSubmitting) {
      return;
    }

    setCopilotBusy(true);
    setError("");
    try {
      const result = await requestCopilotHint(selectedRunId);
      appendLocalEvents(result.events ?? []);
    } catch (err) {
      setError(err instanceof Error ? err.message : "获取 Copilot 提示失败");
    } finally {
      setCopilotBusy(false);
    }
  }

  async function handleArtifactUpload(event: ChangeEvent<HTMLInputElement>) {
    const file = event.target.files?.[0];
    if (!file || !selectedConversationId) {
      return;
    }

    setIsSubmitting(true);
    setError("");
    try {
      const artifact = await uploadArtifact({
        conversationId: selectedConversationId,
        taskId: selectedTask?.id,
        runId: selectedRunId ?? undefined,
        file,
      });
      await refreshArtifacts(selectedConversationId);
      setSelectedArtifactIDs((current) => (current.includes(artifact.id) ? current : [...current, artifact.id]));
    } catch (err) {
      setError(err instanceof Error ? err.message : "上传材料失败");
    } finally {
      event.target.value = "";
      setIsSubmitting(false);
    }
  }

  function openCreateArtifact() {
    setArtifactEditorMode("create");
    setArtifactDraft(defaultArtifactDraft);
    setArtifactEditorOpen(true);
  }

  async function openEditArtifact(artifact: Artifact) {
    if (!isEditableTextArtifact(artifact)) {
      window.open(`/api/files/${artifact.id}?download=1`, "_blank", "noopener,noreferrer");
      return;
    }

    setArtifactBusy(true);
    setError("");
    try {
      const detail: ArtifactDocument = await getArtifactContent(artifact.id);
      setArtifactEditorMode("edit");
      setArtifactDraft({
        id: detail.artifact.id,
        name: detail.artifact.name,
        contentType: detail.artifact.contentType,
        content: detail.content,
      });
      setArtifactEditorOpen(true);
    } catch (err) {
      setError(err instanceof Error ? err.message : "加载材料内容失败");
    } finally {
      setArtifactBusy(false);
    }
  }

  async function handleSaveArtifact(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!selectedConversationId) {
      setError("请先创建或选择一个工作区");
      return;
    }
    if (!artifactDraft.name.trim()) {
      setError("材料名称不能为空");
      return;
    }

    setArtifactBusy(true);
    setError("");
    try {
      const saved =
        artifactEditorMode === "create"
          ? await createTextArtifact({
              conversationId: selectedConversationId,
              taskId: selectedTask?.id,
              runId: selectedRunId ?? undefined,
              name: artifactDraft.name,
              contentType: artifactDraft.contentType,
              content: artifactDraft.content,
            })
          : await updateArtifact({
              id: artifactDraft.id ?? "",
              taskId: selectedTask?.id,
              runId: selectedRunId ?? undefined,
              name: artifactDraft.name,
              contentType: artifactDraft.contentType,
              content: artifactDraft.content,
            });
      await refreshArtifacts(selectedConversationId);
      setSelectedArtifactIDs((current) => (current.includes(saved.id) ? current : [...current, saved.id]));
      setArtifactEditorOpen(false);
      setArtifactDraft(defaultArtifactDraft);
    } catch (err) {
      setError(err instanceof Error ? err.message : "保存材料失败");
    } finally {
      setArtifactBusy(false);
    }
  }

  async function handleDeleteArtifact(artifact: Artifact) {
    if (!selectedConversationId || artifactBusy) {
      return;
    }
    if (!window.confirm(`确认删除材料“${artifact.name}”吗？`)) {
      return;
    }

    setArtifactBusy(true);
    setError("");
    try {
      await deleteArtifact(artifact.id);
      await refreshArtifacts(selectedConversationId);
      setSelectedArtifactIDs((current) => current.filter((id) => id !== artifact.id));
      if (artifactDraft.id === artifact.id) {
        setArtifactEditorOpen(false);
        setArtifactDraft(defaultArtifactDraft);
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : "删除材料失败");
    } finally {
      setArtifactBusy(false);
    }
  }

  function handleNewWorkspace() {
    setSelectedConversationId(null);
    setSelectedTask(null);
    setSelectedConversationRuns([]);
    setSelectedRunId(null);
    setSelectedArtifactIDs([]);
    setArtifacts([]);
    setError("");
    setActiveReplyContextPanel(null);
    setShowRawTelemetry(false);
    setOpenConversationMenuId(null);
    followRunOutputRef.current = false;
    shouldAutoScrollRef.current = false;
    setShowScrollToBottom(false);
  }

  function handleSelectConversation(conversationId: string) {
    followRunOutputRef.current = false;
    shouldAutoScrollRef.current = false;
    setShowScrollToBottom(false);
    setSelectedConversationId(conversationId);
    setOpenConversationMenuId(null);
  }

  function handleOpenWorkspaceHome() {
    handleNewWorkspace();
  }

  const layoutColumns = useMemo(
    () => (showWorkspaceSidebar ? "lg:grid-cols-[292px_minmax(0,1fr)]" : "lg:grid-cols-[minmax(0,1fr)]"),
    [showWorkspaceSidebar]
  );
  const launchWorkspaceTitle = useMemo(
    () => buildWorkspaceTitleFromPrompt(prompt) || workspaceTitle || "新的面试工作区",
    [prompt, workspaceTitle],
  );

  function toggleReplyPanel(target: "persona" | "skill" | "materials") {
    setActiveReplyContextPanel((current) => (current === target ? null : target));
  }

  return (
    <div className="flex h-screen flex-col overflow-hidden bg-background text-foreground">
      <div className="pointer-events-none fixed inset-0 bg-[radial-gradient(circle_at_50%_14%,rgba(111,142,255,0.08),transparent_24%),radial-gradient(circle_at_54%_22%,rgba(74,222,128,0.06),transparent_16%),radial-gradient(circle_at_92%_4%,rgba(14,165,233,0.05),transparent_18%)]" />

      <main className={`relative grid min-h-0 flex-1 gap-4 overflow-hidden px-3 py-3 lg:px-4 ${layoutColumns}`}>
        {showWorkspaceSidebar ? (
        <section className="paper-panel flex h-full min-h-0 flex-col rounded-[1.45rem] border border-[rgba(223,228,238,0.68)] bg-[rgba(255,255,255,0.9)] p-4 shadow-[0_12px_28px_rgba(15,23,42,0.05)]">
          <div className="flex items-center justify-between gap-3">
            <button
              type="button"
              onClick={handleOpenWorkspaceHome}
              className="inline-flex h-11 w-11 items-center justify-center rounded-[1.05rem] border border-[rgba(223,228,238,0.8)] bg-[rgba(255,255,255,0.94)] text-[rgb(17,24,39)] transition hover:border-[rgba(191,219,254,0.96)] hover:text-[rgb(37,99,235)]"
              aria-label="返回会话首页"
              title="返回会话首页"
            >
              <WorkspaceGlyph />
            </button>
            <div className="flex items-center gap-1.5">
              <button
                type="button"
                aria-label="开始一场面试"
                title="开始一场面试"
                onClick={() => setRunComposerOpen(true)}
                className="inline-flex h-8 w-8 items-center justify-center rounded-[0.95rem] border border-[rgba(223,228,238,0.8)] bg-[rgba(255,255,255,0.94)] text-[rgb(37,99,235)] transition hover:border-[rgba(191,219,254,0.96)] hover:bg-[rgba(239,246,255,0.96)]"
              >
                <Plus className="h-4 w-4" />
              </button>
              <button
                type="button"
                aria-label="隐藏工作区"
                onClick={() => setShowWorkspaceSidebar(false)}
                className="inline-flex h-8 w-8 items-center justify-center rounded-[0.95rem] border border-[rgba(223,228,238,0.8)] bg-[rgba(255,255,255,0.94)] text-[rgba(107,114,128,0.76)] transition hover:border-[rgba(191,219,254,0.96)] hover:text-[rgb(37,99,235)]"
              >
                <ChevronLeft className="h-4 w-4" />
              </button>
            </div>
          </div>

          <div className="mt-4 min-h-0 flex-1 space-y-4 overflow-y-auto overscroll-y-contain pr-1 [scrollbar-gutter:stable]">
            {conversations.length === 0 ? (
              <div className="rounded-[1.2rem] border border-dashed border-[rgba(222,227,236,0.92)] bg-[rgba(255,255,255,0.9)] px-4 py-6 text-center">
                <p className="font-serif text-lg text-[rgb(17,24,39)]">还没有会话</p>
                <p className="mt-2 text-sm leading-6 text-[rgba(107,114,128,0.82)]">
                  发起第一场模拟面试后，这里会自动沉淀会话、运行状态和复盘结果。
                </p>
                <Button
                  type="button"
                  onClick={() => setRunComposerOpen(true)}
                  className="mt-4 h-10 rounded-full bg-[rgb(37,99,235)] px-4 text-primary-foreground hover:bg-[rgb(29,78,216)]"
                >
                  <CircleDashed className="mr-2 h-4 w-4" />
                  开始一场面试
                </Button>
              </div>
            ) : (
              <>
                <ConversationSection
                  title="最近"
                  conversations={activeConversations}
                  selectedConversationId={selectedConversationId}
                  openConversationMenuId={openConversationMenuId}
                  editingConversationId={editingConversationId}
                  editingConversationTitle={editingConversationTitle}
                  onEditingConversationTitleChange={setEditingConversationTitle}
                  onSelectConversation={handleSelectConversation}
                  onToggleMenu={setOpenConversationMenuId}
                  onCopyConversationTitle={handleCopyConversationTitle}
                  onStartRenameConversation={startRenameConversation}
                  onCancelRenameConversation={cancelRenameConversation}
                  onSubmitRenameConversation={submitRenameConversation}
                  onToggleConversationPin={handleToggleConversationPin}
                  onToggleConversationArchive={handleToggleConversationArchive}
                  onDeleteConversation={handleDeleteConversation}
                  describeStatus={describeConversationStatus}
                />
                {archivedConversations.length > 0 ? (
                  <ConversationSection
                    title="已归档"
                    conversations={archivedConversations}
                    selectedConversationId={selectedConversationId}
                    openConversationMenuId={openConversationMenuId}
                    editingConversationId={editingConversationId}
                    editingConversationTitle={editingConversationTitle}
                    onEditingConversationTitleChange={setEditingConversationTitle}
                    onSelectConversation={handleSelectConversation}
                    onToggleMenu={setOpenConversationMenuId}
                    onCopyConversationTitle={handleCopyConversationTitle}
                    onStartRenameConversation={startRenameConversation}
                    onCancelRenameConversation={cancelRenameConversation}
                    onSubmitRenameConversation={submitRenameConversation}
                    onToggleConversationPin={handleToggleConversationPin}
                    onToggleConversationArchive={handleToggleConversationArchive}
                    onDeleteConversation={handleDeleteConversation}
                    describeStatus={describeConversationStatus}
                  />
                ) : null}
              </>
            )}
          </div>
        </section>
        ) : null}

        {!showWorkspaceSidebar ? (
          <button
            type="button"
            aria-label="显示会话列表"
            onClick={() => setShowWorkspaceSidebar(true)}
            className="fixed left-4 top-4 z-30 inline-flex h-11 w-11 items-center justify-center rounded-full border border-[rgba(223,228,238,0.8)] bg-[rgba(255,255,255,0.96)] text-[rgba(71,85,105,0.78)] shadow-[0_12px_24px_rgba(15,23,42,0.08)] transition hover:border-[rgba(191,219,254,0.96)] hover:text-[rgb(37,99,235)]"
          >
            <WorkspaceGlyph />
          </button>
        ) : null}

        <section className="flex h-full min-h-0 flex-1 flex-col overflow-hidden bg-transparent">
          <div className="relative min-h-0 flex-1">
            <div
              ref={messageViewportRef}
              onScroll={handleMessageViewportScroll}
              className="h-full overflow-y-auto overscroll-y-contain px-5 py-3 [scrollbar-gutter:stable]"
            >
              {messages.length === 0 && !runLoading ? (
                <EmptyRunState
                  hasRun={Boolean(selectedRunId)}
                  prompt={prompt}
                  isSubmitting={isSubmitting}
                  onPromptChange={setPrompt}
                  onLaunch={() => void launchInterview()}
                  onOpenComposer={() => setRunComposerOpen(true)}
                  onApplyPrompt={setPrompt}
                />
              ) : (
                <div className="space-y-5">
                  {visibleMessages.map((message) => (
                    <MessageBubble
                      key={message.id}
                      message={message}
                      highlighted={highlightedMessageId === message.id}
                      messageRef={(node) => {
                        messageNodeRefs.current[message.id] = node;
                      }}
                      copied={copiedMessageId === message.id}
                      onCopy={() => void handleCopyMessage(message)}
                    />
                  ))}
                  {(run?.status === "running" || run?.status === "resuming") && !isStreamingAssistant && (
                    <ThinkingBubble label={isGeneratingFinalReview ? "正在生成最终评分与总结..." : "正在思考中"} />
                  )}
                  {shouldShowInlineReview ? (
                    <InlineReviewDigest
                      run={run}
                      reviewSnapshot={reviewSnapshot}
                      loading={reviewLoading}
                      isGeneratingFinalReview={isGeneratingFinalReview}
                    />
                  ) : null}
                  {runLoading && (
                    <div className="flex items-center gap-3 rounded-[1.4rem] border border-[rgba(226,231,239,0.96)] bg-[rgba(255,255,255,0.94)] px-4 py-3 text-sm text-[rgba(107,114,128,0.82)]">
                      <LoaderCircle className="h-4 w-4 animate-spin" />
                      正在同步运行状态...
                    </div>
                  )}
                  {activeClarify && (
                    <ClarifyPanel
                      event={activeClarify}
                      latestCheckpoint={latestCheckpoint}
                    />
                  )}
                  <div ref={messagesEndRef} />
                </div>
              )}
            </div>
            {showScrollToBottom && (
              <button
                type="button"
                onClick={scrollMessagesToBottom}
                className="absolute bottom-5 right-6 inline-flex items-center gap-2 rounded-full border border-[rgba(226,231,239,0.96)] bg-[rgba(255,255,255,0.96)] px-4 py-2 text-sm text-[rgb(55,65,81)] shadow-[0_10px_24px_rgba(30,41,59,0.08)] transition-all hover:-translate-y-0.5 hover:border-[rgba(59,130,246,0.18)] hover:text-[rgb(37,99,235)]"
              >
                <ChevronDownCircle className="h-4 w-4" />
                回到底部
              </button>
            )}
          </div>

          {(error || run?.lastError) && (
            <div className="mx-6 mb-4 rounded-[1.4rem] border border-[rgba(122,176,188,0.24)] bg-[rgba(236,248,250,0.94)] px-4 py-3 text-sm text-[rgb(52,110,124)]">
              {error || run?.lastError}
            </div>
          )}

          <div className="mt-auto shrink-0 px-6 pb-3 pt-2">
            <ReplyComposer
              selectedRunId={selectedRunId}
              isSubmitting={isSubmitting}
              copilotBusy={copilotBusy}
              latestCopilotFeedback={latestCopilotFeedback}
              latestCopilotHint={latestCopilotHint}
              runStatus={run?.status}
              activeClarify={Boolean(activeClarify)}
              latestFailure={latestFailure}
              currentPersonaLabel={currentPersonaLabel}
              sessionConfigActive={Boolean(activeReplyContextPanel)}
              webSearchEnabled={Boolean(config.enableWebSearch)}
              runCanCancel={Boolean(selectedRunId && (run?.status === "running" || run?.status === "resuming" || run?.status === "waiting_clarify"))}
              onOpenSessionConfig={() => toggleReplyPanel("persona")}
              onToggleWebSearch={() =>
                setConfig((current) => ({
                  ...current,
                  enableWebSearch: !current.enableWebSearch,
                }))
              }
              onCancelRun={() => void handleCancelRun()}
              onRequestCopilot={handleRequestCopilot}
              onSubmitMessage={handleReply}
            />
          </div>
        </section>
      </main>

      {runComposerOpen ? (
        <RunComposerModal
          workspaceTitle={launchWorkspaceTitle}
          prompt={prompt}
          config={config}
          modelConfig={modelConfig}
          isSubmitting={isSubmitting}
          onClose={() => setRunComposerOpen(false)}
          onPromptChange={setPrompt}
          onConfigChange={setConfig}
          onModelConfigChange={setModelConfig}
          levelOptions={levelOptions}
          timeBudgetOptions={timeBudgetOptions}
          onSubmit={async (event) => {
            await handleLaunchInterview(event);
            setRunComposerOpen(false);
          }}
        />
      ) : null}

      {activeReplyContextPanel ? (
        <SessionContextModal
          activePanel={activeReplyContextPanel}
          config={config}
          currentModeLabel={currentModeLabel}
          currentPhaseLabel={currentPhaseLabel}
          currentRunStatusLabel={currentRunStatusLabel}
          selectedArtifactIDs={selectedArtifactIDs}
          artifacts={artifacts}
          skills={skills}
          skillBusy={skillBusy}
          isSubmitting={isSubmitting}
          selectedConversationId={selectedConversationId}
          onClose={() => setActiveReplyContextPanel(null)}
          onActivePanelChange={setActiveReplyContextPanel}
          onConfigChange={setConfig}
          onSelectedArtifactIDsChange={setSelectedArtifactIDs}
          onSkillUpload={handleSkillUpload}
          onArtifactUpload={handleArtifactUpload}
          onOpenCreateSkill={openCreateSkill}
          onOpenCreateArtifact={openCreateArtifact}
          personaOptions={personaOptions}
          getPersonaMeta={getPersonaMeta}
          formatInterviewModeLabel={formatInterviewModeLabel}
        />
      ) : null}

      {skillEditorOpen ? (
        <SkillEditorModal
          draft={skillDraft}
          busy={skillBusy}
          mode={skillEditorMode}
          onChange={setSkillDraft}
          onClose={() => {
            if (skillBusy) return;
            setSkillEditorOpen(false);
          }}
          onSubmit={handleSaveSkill}
        />
      ) : null}

      {artifactEditorOpen ? (
        <ArtifactEditorModal
          draft={artifactDraft}
          busy={artifactBusy}
          mode={artifactEditorMode}
          onChange={setArtifactDraft}
          onClose={() => {
            if (artifactBusy) return;
            setArtifactEditorOpen(false);
          }}
          onSubmit={handleSaveArtifact}
        />
      ) : null}
    </div>
  );
}

function WorkspaceGlyph() {
  return (
    <span className="relative block h-6 w-6">
      <span className="absolute inset-0 rounded-[0.9rem] bg-[radial-gradient(circle_at_30%_24%,rgba(255,255,255,0.98),rgba(219,234,254,0.9)_40%,rgba(239,246,255,0.56)_68%,transparent)]" />
      <span className="absolute left-[2px] top-[4px] h-[12px] w-[12px] rounded-[0.45rem] border border-[rgba(59,130,246,0.34)] bg-[rgba(255,255,255,0.96)] shadow-[0_4px_10px_rgba(59,130,246,0.1)]" />
      <span className="absolute left-[5px] top-[7px] h-[1.5px] w-[6px] rounded-full bg-[rgba(59,130,246,0.62)]" />
      <span className="absolute left-[5px] top-[10px] h-[1.5px] w-[4px] rounded-full bg-[rgba(148,163,184,0.72)]" />
      <span className="absolute right-[2px] top-[6px] h-[8px] w-[8px] rounded-full border border-[rgba(14,116,144,0.46)] bg-[rgba(236,254,255,0.98)]" />
      <span className="absolute bottom-[2px] left-[5px] h-[8px] w-[8px] rounded-full border border-[rgba(30,41,59,0.42)] bg-[rgba(255,255,255,0.96)]" />
      <span className="absolute bottom-[3px] right-[4px] h-[10px] w-[10px] rounded-[0.55rem] border border-[rgba(125,211,252,0.52)] bg-[rgba(239,246,255,0.98)]" />
      <span className="absolute right-[8px] top-[12px] h-[1.5px] w-[4px] rounded-full bg-[rgba(59,130,246,0.48)]" />
      <span className="absolute left-[12px] top-[10px] h-[1.5px] w-[4px] rotate-[22deg] bg-[rgba(59,130,246,0.34)]" />
      <span className="absolute left-[11px] top-[14px] h-[1.5px] w-[5px] rotate-[-28deg] bg-[rgba(14,116,144,0.3)]" />
    </span>
  );
}

function Field({ label, children }: { label: string; children: ReactNode }) {
  return (
    <div className="space-y-2">
      <Label className="tech-label text-[0.64rem] text-[rgba(97,123,150,0.66)]">{label}</Label>
      {children}
    </div>
  );
}


function ContextPill({
  icon,
  label,
  value,
  active,
  onClick,
}: {
  icon: ReactNode;
  label: string;
  value: string;
  active: boolean;
  onClick: () => void;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      title={`${label}：${value}`}
      aria-label={`${label}：${value}`}
      className={`inline-flex min-h-[2.8rem] items-center gap-2 rounded-full border px-3 py-1.5 text-left transition-all ${
        active
          ? "border-[rgba(8,47,73,0.14)] bg-[rgba(240,249,255,0.96)] text-[rgb(8,47,73)] shadow-[0_10px_18px_rgba(8,47,73,0.08)]"
          : "border-[rgba(153,191,201,0.18)] bg-[rgba(255,255,255,0.9)] text-[rgb(72,91,114)] hover:border-[rgba(8,47,73,0.14)] hover:text-[rgb(8,47,73)]"
      }`}
    >
      <span className={`flex h-7 w-7 items-center justify-center rounded-full ${active ? "bg-[rgba(8,47,73,0.08)]" : "bg-[rgba(118,177,189,0.1)]"}`}>
        {icon}
      </span>
      <span className="min-w-0">
        <span className="block text-[0.64rem] uppercase tracking-[0.14em] text-[rgba(71,85,105,0.62)]">{label}</span>
        <span className="block max-w-[9rem] truncate text-[0.86rem] font-medium leading-5">{value}</span>
      </span>
    </button>
  );
}

function ReplyComposer({
  selectedRunId,
  isSubmitting,
  copilotBusy,
  latestCopilotFeedback,
  latestCopilotHint,
  runStatus,
  activeClarify,
  latestFailure,
  currentPersonaLabel,
  sessionConfigActive,
  webSearchEnabled,
  runCanCancel,
  onOpenSessionConfig,
  onToggleWebSearch,
  onCancelRun,
  onRequestCopilot,
  onSubmitMessage,
}: {
  selectedRunId: string | null;
  isSubmitting: boolean;
  copilotBusy: boolean;
  latestCopilotFeedback: CopilotFeedback | null;
  latestCopilotHint: CopilotHint | null;
  runStatus: string | undefined;
  activeClarify: boolean;
  latestFailure: RunEvent | null;
  currentPersonaLabel: string;
  sessionConfigActive: boolean;
  webSearchEnabled: boolean;
  runCanCancel: boolean;
  onOpenSessionConfig: () => void;
  onToggleWebSearch: () => void;
  onCancelRun: () => void;
  onRequestCopilot: () => void;
  onSubmitMessage: (message: string) => Promise<boolean>;
}) {
  const [draft, setDraft] = useState("");
  const [copilotPanelOpen, setCopilotPanelOpen] = useState(false);
  const composerTextareaRef = useRef<HTMLTextAreaElement | null>(null);

  function syncComposerHeight() {
    const node = composerTextareaRef.current;
    if (!node) {
      return;
    }
    node.style.height = "0px";
    node.style.height = `${Math.min(164, Math.max(44, node.scrollHeight))}px`;
  }

  useEffect(() => {
    setDraft("");
    setCopilotPanelOpen(false);
  }, [selectedRunId]);

  useEffect(() => {
    syncComposerHeight();
  }, [draft, selectedRunId]);

  async function handleSubmit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const submitted = await onSubmitMessage(draft);
    if (submitted) {
      setDraft("");
    }
  }

  const inputDisabled = !selectedRunId || isSubmitting || runStatus === "running" || runStatus === "resuming";
  const copilotDisabled =
    !selectedRunId ||
    copilotBusy ||
    isSubmitting ||
    runStatus === "running" ||
    runStatus === "resuming" ||
    runStatus === "failed" ||
    runStatus === "cancelled";
  const helperText = activeClarify
    ? "先直接补充当前缺失信息，提交后会从检查点继续。"
    : runStatus === "completed"
      ? "这一场已经结束，可以补一句追问，或直接在上方查看本场总结。"
      : inputDisabled
        ? "系统正在推进当前轮次，暂时不接收新的回复。"
        : "建议一条消息只回答一个核心问题，按“结论 -> 依据 -> 取舍”组织会更稳。";
  const keyboardHint = inputDisabled ? "等待系统返回下一步" : "Ctrl / Cmd + Enter 发送";
  const compactCopilotSummary =
    latestCopilotHint?.summary?.trim() ||
    latestCopilotFeedback?.summary?.trim() ||
    "";
  const compactCopilotTitle =
    latestCopilotHint?.title?.trim() ||
    (latestCopilotFeedback ? `提示：${getCopilotStateMeta(latestCopilotFeedback.state).label}` : "");
  const compactCopilotSteps = (latestCopilotHint?.strategy?.slice(0, 2) ?? latestCopilotFeedback?.suggestedMoves?.slice(0, 2) ?? []).filter(Boolean);
  const hasCopilotContent = Boolean(compactCopilotSummary || compactCopilotSteps.length);

  function handleCopilotPrimaryAction() {
    if (copilotBusy) {
      return;
    }
    if (copilotPanelOpen) {
      setCopilotPanelOpen(false);
      return;
    }
    setCopilotPanelOpen(true);
    if (!hasCopilotContent) {
      void onRequestCopilot();
    }
  }

  function handleRefreshCopilot() {
    setCopilotPanelOpen(true);
    void onRequestCopilot();
  }

  const copilotPrimaryLabel = copilotBusy ? "提示生成中" : copilotPanelOpen ? "收起提示" : hasCopilotContent ? "查看提示" : "给我提示";

  return (
    <form onSubmit={handleSubmit} className="relative">
      <div className="rounded-[1rem] border border-[rgba(214,222,234,0.92)] bg-[rgba(255,255,255,0.96)] p-2 shadow-[0_8px_18px_rgba(15,23,42,0.04)]">
        {selectedRunId ? (
          <div className="mb-2 flex flex-wrap items-center gap-1.5">
            <CompactActionButton
              icon={<FolderKanban className="h-3.5 w-3.5" />}
              label="会话设置"
              title={`会话设置：人格 ${currentPersonaLabel}、技能和材料`}
              active={sessionConfigActive}
              onClick={onOpenSessionConfig}
            />
            <CompactActionButton
              icon={<Radar className="h-3.5 w-3.5" />}
              label={webSearchEnabled ? "联网" : "本地"}
              title={webSearchEnabled ? "联网已开启" : "联网已关闭"}
              active={webSearchEnabled}
              onClick={onToggleWebSearch}
            />
            {runCanCancel ? (
              <CompactActionButton
                icon={<Flame className="h-3.5 w-3.5" />}
                label="取消"
                title="取消当前运行"
                tone="danger"
                onClick={onCancelRun}
              />
            ) : null}
          </div>
        ) : null}

        {selectedRunId && copilotPanelOpen ? (
          <div className="mb-2 rounded-[0.85rem] border border-[rgba(191,219,254,0.92)] bg-[rgba(239,246,255,0.82)] px-3 py-2">
            <div className="flex items-start justify-between gap-3">
              <div className="min-w-0">
                <div className="flex items-center gap-2">
                  <span className="inline-flex h-6 w-6 items-center justify-center rounded-full bg-[rgba(255,255,255,0.84)] text-[rgb(37,99,235)]">
                    {copilotBusy ? <LoaderCircle className="h-3.5 w-3.5 animate-spin" /> : <Sparkles className="h-3.5 w-3.5" />}
                  </span>
                  <p className="text-[12px] font-medium uppercase tracking-[0.16em] text-[rgba(37,99,235,0.78)]">
                    {copilotBusy ? "提示生成中" : compactCopilotTitle || "Copilot 提示"}
                  </p>
                </div>
                <p className="mt-1.5 text-[13px] leading-5 text-[rgba(30,64,175,0.88)]">
                  {copilotBusy ? "正在根据当前轮次和后端信号整理更短的回答建议..." : compactCopilotSummary}
                </p>
              </div>
              <div className="flex shrink-0 items-center gap-1.5">
                <button
                  type="button"
                  onClick={handleRefreshCopilot}
                  disabled={copilotBusy}
                  className="inline-flex h-8 items-center gap-1.5 rounded-full border border-[rgba(191,219,254,0.96)] bg-[rgba(255,255,255,0.88)] px-3 text-[12px] font-medium text-[rgb(37,99,235)] transition hover:bg-[rgba(255,255,255,0.96)] disabled:cursor-not-allowed disabled:opacity-70"
                >
                  <Sparkles className="h-3.5 w-3.5" />
                  再来一条
                </button>
                <button
                  type="button"
                  onClick={() => setCopilotPanelOpen(false)}
                  className="inline-flex h-8 items-center gap-1.5 rounded-full border border-[rgba(226,231,239,0.96)] bg-[rgba(255,255,255,0.78)] px-3 text-[12px] font-medium text-[rgba(71,85,105,0.84)] transition hover:bg-[rgba(255,255,255,0.96)]"
                >
                  收起
                </button>
              </div>
            </div>
            {compactCopilotSteps.length > 0 ? (
              <div className="mt-2 flex flex-wrap gap-1.5">
                {compactCopilotSteps.map((item) => (
                  <span
                    key={item}
                    className="rounded-full border border-[rgba(191,219,254,0.88)] bg-[rgba(255,255,255,0.88)] px-2.5 py-1 text-[11px] text-[rgba(30,64,175,0.84)]"
                  >
                    {item}
                  </span>
                ))}
              </div>
            ) : null}
          </div>
        ) : null}

        <div className="rounded-[0.9rem] border border-[rgba(226,231,239,0.96)] bg-[rgba(255,255,255,0.98)] px-3 py-2.5">
          <textarea
            ref={composerTextareaRef}
            id="reply"
            value={draft}
            rows={1}
            onChange={(event) => {
              setDraft(event.target.value);
              syncComposerHeight();
            }}
            onKeyDown={(event) => {
              if ((event.metaKey || event.ctrlKey) && event.key === "Enter") {
                event.preventDefault();
                void handleSubmit(event as unknown as React.FormEvent<HTMLFormElement>);
              }
            }}
            placeholder={activeClarify ? "请直接补充当前澄清问题需要的信息..." : "直接回答当前问题。更推荐：先给结论，再给依据，最后补 tradeoff / 例子。"}
            disabled={inputDisabled}
            className="block w-full resize-none overflow-y-auto border-0 bg-transparent px-0 py-0 text-[14px] leading-6 text-[rgb(15,23,42)] outline-none placeholder:text-[rgba(148,163,184,0.95)]"
          />

          <div className="mt-2 flex flex-wrap items-center justify-between gap-3 border-t border-[rgba(226,231,239,0.82)] pt-2">
            <div className="min-w-0">
              <p className="text-[12px] leading-5 text-[rgba(71,85,105,0.8)]">{helperText}</p>
              <p className="mt-1 text-[10px] uppercase tracking-[0.16em] text-[rgba(100,116,139,0.68)]">{keyboardHint}</p>
            </div>

            <div className="flex flex-wrap items-center gap-2">
              <Button
                type="button"
                variant="outline"
                disabled={copilotDisabled && !copilotPanelOpen}
                onClick={handleCopilotPrimaryAction}
                className="h-10 rounded-full border-[rgba(191,219,254,0.96)] bg-[rgba(239,246,255,0.92)] px-3.5 text-[rgb(37,99,235)] hover:bg-[rgba(219,234,254,0.96)]"
              >
                {copilotBusy ? <LoaderCircle className="mr-2 h-4 w-4 animate-spin" /> : <Sparkles className="mr-2 h-4 w-4" />}
                {copilotPrimaryLabel}
              </Button>
              <Button
                type="submit"
                disabled={!selectedRunId || !draft.trim() || inputDisabled}
                className="h-10 rounded-full bg-[rgb(37,99,235)] px-4.5 text-white hover:bg-[rgb(29,78,216)]"
              >
                {isSubmitting ? <LoaderCircle className="mr-2 h-4 w-4 animate-spin" /> : null}
                发送回答
              </Button>
            </div>
          </div>
        </div>
      </div>
      <div className="mt-3">
        <RunGuidance runStatus={runStatus} activeClarify={activeClarify} latestFailure={latestFailure} />
      </div>
    </form>
  );
}

function CompactActionButton({
  icon,
  label,
  title,
  active = false,
  tone = "default",
  onClick,
}: {
  icon: ReactNode;
  label: string;
  title: string;
  active?: boolean;
  tone?: "default" | "danger";
  onClick: () => void;
}) {
  const toneClass =
    tone === "danger"
      ? "border-[rgba(254,205,211,0.94)] bg-[rgba(255,241,242,0.96)] text-[rgb(190,24,93)] hover:bg-[rgba(255,228,230,0.98)]"
      : active
        ? "border-[rgba(191,219,254,0.96)] bg-[rgba(239,246,255,0.96)] text-[rgb(37,99,235)] hover:bg-[rgba(219,234,254,0.96)]"
        : "border-[rgba(226,231,239,0.96)] bg-[rgba(255,255,255,0.92)] text-[rgba(71,85,105,0.84)] hover:bg-[rgba(248,250,252,0.98)]";

  return (
    <button
      type="button"
      title={title}
      aria-label={title}
      onClick={onClick}
      className={`inline-flex h-8 items-center gap-1.5 rounded-full border px-2.5 text-[12px] transition ${toneClass}`}
    >
      {icon}
      <span className="font-medium">{label}</span>
    </button>
  );
}

function RunOperatorDeck({
  run,
  reviewSnapshot,
  decisionAudit,
  feedback,
  hint,
  latestFailure,
  activeClarify,
  hasReplay,
  currentPhaseLabel,
  currentRunStatusLabel,
  onOpenReview,
  onRequestCopilot,
  onOpenReplay,
  formatSignalLabel,
  formatDecisionReasonLabel,
}: {
  run: Run | null;
  reviewSnapshot: ReviewSnapshot | null;
  decisionAudit: DecisionAudit | null;
  feedback: CopilotFeedback | null;
  hint: CopilotHint | null;
  latestFailure: RunEvent | null;
  activeClarify: boolean;
  hasReplay: boolean;
  currentPhaseLabel: string;
  currentRunStatusLabel: string;
  onOpenReview: () => void;
  onRequestCopilot: () => void;
  onOpenReplay: () => void;
  formatSignalLabel: (signal?: string | null) => string;
  formatDecisionReasonLabel: (reason?: string | null) => string;
}) {
  const brief = buildOperatorBrief({
    run,
    reviewSnapshot,
    decisionAudit,
    feedback,
    hint,
    latestFailure,
    activeClarify,
    currentPhaseLabel,
    currentRunStatusLabel,
    formatSignalLabel,
    formatDecisionReasonLabel,
  });

  return (
    <div className="grid gap-3 xl:grid-cols-[minmax(0,1.08fr)_minmax(320px,0.92fr)]">
      <section className="overflow-hidden rounded-[1.65rem] border border-[rgba(214,222,234,0.92)] bg-[linear-gradient(140deg,rgba(255,255,255,0.98),rgba(245,248,255,0.96))] shadow-[0_16px_36px_rgba(15,23,42,0.05)]">
        <div className={`h-1.5 w-full ${brief.railClass}`} />
        <div className="px-5 py-5">
          <div className="flex flex-wrap items-start justify-between gap-3">
            <div>
              <p className="tech-label text-[0.66rem] text-[rgba(100,116,139,0.72)]">面试驾驶舱</p>
              <h3 className="mt-2 font-display text-[1.35rem] leading-tight text-[rgb(15,23,42)]">{brief.headline}</h3>
              <p className="mt-2 max-w-[72ch] text-sm leading-7 text-[rgba(71,85,105,0.86)]">{brief.summary}</p>
            </div>
            <div className={`rounded-full border px-3 py-1.5 text-[0.72rem] uppercase tracking-[0.18em] ${brief.badgeClass}`}>
              {brief.statusLabel}
            </div>
          </div>

          <div className="mt-4 grid gap-3 md:grid-cols-3">
            <OperatorFactCard label="系统判断" value={brief.systemJudgement} tone="neutral" />
            <OperatorFactCard label="当前更该做" value={brief.primaryAction} tone={brief.actionTone} />
            <OperatorFactCard label="别急着做" value={brief.guardrailTitle} tone="warning" />
          </div>

          {brief.focusLabels.length > 0 ? (
            <div className="mt-4 flex flex-wrap gap-2">
              {brief.focusLabels.map((label) => (
                <span
                  key={label}
                  className="rounded-full border border-[rgba(191,219,254,0.94)] bg-[rgba(239,246,255,0.92)] px-3 py-1.5 text-[0.76rem] text-[rgb(29,78,216)]"
                >
                  {label}
                </span>
              ))}
            </div>
          ) : null}
        </div>
      </section>

      <section className="rounded-[1.65rem] border border-[rgba(214,222,234,0.92)] bg-[rgba(255,255,255,0.96)] px-5 py-5 shadow-[0_16px_36px_rgba(15,23,42,0.05)]">
        <p className="tech-label text-[0.66rem] text-[rgba(100,116,139,0.72)]">下一步怎么做</p>
        <h3 className="mt-2 font-display text-[1.1rem] text-[rgb(15,23,42)]">把后端信号翻译成人话操作</h3>
        <div className="mt-4 space-y-3">
          {brief.checklist.map((item, index) => (
            <div
              key={`${index}-${item}`}
              className="rounded-[1.15rem] border border-[rgba(226,231,239,0.92)] bg-[rgba(248,250,252,0.92)] px-4 py-3"
            >
              <p className="text-xs uppercase tracking-[0.18em] text-[rgba(100,116,139,0.7)]">Step {index + 1}</p>
              <p className="mt-1 text-sm leading-7 text-[rgb(51,65,85)]">{item}</p>
            </div>
          ))}
        </div>

        <div className="mt-4 flex flex-wrap gap-2">
          <Button
            type="button"
            variant="outline"
            onClick={onOpenReview}
            className="h-10 rounded-full border-[rgba(214,222,234,0.96)] bg-[rgba(255,255,255,0.98)] px-4 text-[rgb(55,65,81)] hover:bg-[rgba(248,250,252,0.98)]"
          >
            <Sparkles className="mr-2 h-4 w-4" />
            查看复盘
          </Button>
          <Button
            type="button"
            variant="outline"
            onClick={onRequestCopilot}
            className="h-10 rounded-full border-[rgba(191,219,254,0.96)] bg-[rgba(239,246,255,0.96)] px-4 text-[rgb(37,99,235)] hover:bg-[rgba(219,234,254,0.96)]"
          >
            <MessageSquareQuote className="mr-2 h-4 w-4" />
            请求提示
          </Button>
          {hasReplay ? (
            <Button
              type="button"
              variant="outline"
              onClick={onOpenReplay}
              className="h-10 rounded-full border-[rgba(191,219,254,0.96)] bg-[rgba(255,255,255,0.98)] px-4 text-[rgb(37,99,235)] hover:bg-[rgba(239,246,255,0.96)]"
            >
              <Waypoints className="mr-2 h-4 w-4" />
              打开回放
            </Button>
          ) : null}
        </div>
      </section>
    </div>
  );
}

function OperatorFactCard({
  label,
  value,
  tone,
}: {
  label: string;
  value: string;
  tone: "neutral" | "warning" | "success" | "info";
}) {
  const toneClass =
    tone === "warning"
      ? "border-[rgba(253,230,138,0.92)] bg-[rgba(254,252,232,0.96)] text-[rgb(161,98,7)]"
      : tone === "success"
        ? "border-[rgba(187,247,208,0.92)] bg-[rgba(240,253,244,0.96)] text-[rgb(21,128,61)]"
        : tone === "info"
          ? "border-[rgba(191,219,254,0.92)] bg-[rgba(239,246,255,0.96)] text-[rgb(29,78,216)]"
          : "border-[rgba(226,231,239,0.92)] bg-[rgba(248,250,252,0.92)] text-[rgb(51,65,85)]";

  return (
    <div className={`rounded-[1.2rem] border px-4 py-4 ${toneClass}`}>
      <p className="text-[0.7rem] uppercase tracking-[0.18em] opacity-70">{label}</p>
      <p className="mt-2 text-sm leading-7">{value}</p>
    </div>
  );
}

function MessageBubble({
  message,
  highlighted,
  messageRef,
  copied,
  onCopy,
}: {
  message: Message;
  highlighted: boolean;
  messageRef?: (node: HTMLDivElement | null) => void;
  copied: boolean;
  onCopy: () => void;
}) {
  const isUser = message.role === "user";
  const isStreaming = message.id.startsWith("streaming-");
  return (
    <div
      ref={messageRef}
      data-message-id={message.id}
      className={`flex scroll-mt-24 rounded-[1.4rem] transition-all duration-500 ${
        highlighted ? "bg-[rgba(219,234,254,0.38)] ring-1 ring-[rgba(59,130,246,0.18)]" : ""
      } ${isUser ? "justify-end" : "justify-start"}`}
    >
      <div className={`${isUser ? "max-w-[78%]" : "w-full max-w-[54rem]"}`}>
        <div
          className={`px-5 py-4 ${
            isUser
              ? "rounded-[1.65rem] bg-[linear-gradient(135deg,rgba(37,99,235,0.96),rgba(96,165,250,0.94))] text-white shadow-[0_14px_28px_rgba(37,99,235,0.14)]"
              : "rounded-[1.15rem] bg-transparent text-[rgb(31,41,55)]"
          }`}
        >
          <div className="flex justify-end">
            {isStreaming && !isUser && (
              <span className="rounded-full bg-[rgba(219,234,254,0.9)] px-2 py-1 text-[0.62rem] uppercase tracking-[0.18em] text-[rgb(37,99,235)]">
                正在生成
              </span>
            )}
          </div>
          <div className={`text-sm leading-7 ${isStreaming && !isUser ? "mt-2" : ""} ${isUser ? "whitespace-pre-wrap" : ""}`}>
            {isUser ? (
              <p className="whitespace-pre-wrap">{message.content}</p>
            ) : (
              <Suspense fallback={<p className="whitespace-pre-wrap leading-7">{message.content}</p>}>
                <MarkdownBubbleContent content={message.content} showCursor={isStreaming} />
              </Suspense>
            )}
          </div>
        </div>
        <div className={`mt-1 flex ${isUser ? "justify-end" : "justify-start"}`}>
          <button
            type="button"
            onClick={onCopy}
            className="inline-flex items-center gap-2 rounded-full px-3 py-1.5 text-[0.72rem] text-[rgba(107,114,128,0.76)] transition-colors hover:bg-[rgba(255,255,255,0.82)] hover:text-[rgb(37,99,235)]"
          >
            {copied ? <Check className="h-3.5 w-3.5" /> : <Copy className="h-3.5 w-3.5" />}
            {copied ? "已复制" : "复制"}
          </button>
        </div>
      </div>
    </div>
  );
}

function ThinkingBubble({ label = "正在思考中" }: { label?: string }) {
  return (
    <div className="flex justify-start">
      <div className="w-full max-w-[54rem] rounded-[1.15rem] bg-transparent px-5 py-4 text-[rgb(31,41,55)]">
        <div className="flex items-center gap-3 text-sm text-[rgba(107,114,128,0.84)]">
          <span className="inline-flex gap-1">
            <span className="h-2 w-2 animate-bounce rounded-full bg-[rgb(37,99,235)] [animation-delay:-0.2s]" />
            <span className="h-2 w-2 animate-bounce rounded-full bg-[rgba(37,99,235,0.72)] [animation-delay:-0.1s]" />
            <span className="h-2 w-2 animate-bounce rounded-full bg-[rgba(37,99,235,0.48)]" />
          </span>
          {label}
        </div>
      </div>
    </div>
  );
}

function InlineReviewDigest({
  run,
  reviewSnapshot,
  loading,
  isGeneratingFinalReview,
}: {
  run: Run | null;
  reviewSnapshot: ReviewSnapshot | null;
  loading: boolean;
  isGeneratingFinalReview: boolean;
}) {
  const scorecard = reviewSnapshot?.scorecard ?? null;
  const profile = reviewSnapshot?.profile ?? null;
  const trace = reviewSnapshot?.trace ?? null;
  const overallScore = scorecard?.overallScore;
  const overallMaxScore = scorecard?.overallMaxScore ?? 100;
  const strengths = (scorecard?.strengths ?? []).slice(0, 2);
  const gaps = (scorecard?.gaps ?? []).slice(0, 2);
  const profileFocus = [ ...(profile?.recommendedFocus ?? []), ...(profile?.recurringGaps ?? []) ].slice(0, 3);
  const stageSummary = summarizeTracePhases(trace);
  const [expandedSection, setExpandedSection] = useState<"trace" | "profile" | null>(null);
  const canExpandTrace = Boolean(trace && (trace.nodes?.length ?? 0) > 0);
  const canExpandProfile = Boolean(profile && ((profile.radar?.length ?? 0) > 0 || (profile.dimensions?.length ?? 0) > 0));
  const title = run?.status === "completed" ? "本场结果" : isGeneratingFinalReview ? "最终评分生成中" : "结果整理中";
  const subtitle =
    scorecard?.summary?.trim() ||
    reviewSnapshot?.summary?.decisionExplanation?.trim() ||
    (isGeneratingFinalReview ? "评分、画像和追问链路正在汇总，结果会直接补到这里。" : "本场结构化结果会随着后端产物返回逐步补齐。");

  return (
    <div className="rounded-[1.5rem] border border-[rgba(214,222,234,0.92)] bg-[linear-gradient(145deg,rgba(255,255,255,0.98),rgba(246,249,255,0.96))] px-5 py-5 shadow-[0_18px_38px_rgba(15,23,42,0.06)]">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div>
          <p className="tech-label text-[0.66rem] text-[rgba(97,123,150,0.68)]">结构化结果</p>
          <h3 className="mt-2 font-display text-[1.2rem] text-[rgb(15,23,42)]">{title}</h3>
          <p className="mt-2 max-w-[72ch] text-sm leading-7 text-[rgba(71,85,105,0.84)]">{subtitle}</p>
        </div>
        <div className="rounded-full border border-[rgba(191,219,254,0.96)] bg-[rgba(239,246,255,0.96)] px-4 py-2 text-sm font-medium text-[rgb(37,99,235)]">
          {overallScore != null ? `${overallScore}/${overallMaxScore}` : loading ? "生成中" : "等待结果"}
        </div>
      </div>

      <div className="mt-4 grid gap-3 lg:grid-cols-2">
        <DigestFactCard
          label="评分"
          title={overallScore != null ? `总分 ${overallScore}/${overallMaxScore}` : "总分待生成"}
          lines={[
            scorecard?.dimensionScores?.slice(0, 3).map((dimension) => {
              const maxScore = dimension.maxScore ?? 10;
              return `${dimension.name} ${dimension.score}/${maxScore}`;
            })?.join(" · ") || (loading ? "正在计算维度分数" : "结构化评分返回后会显示维度分数"),
          ]}
          tone="score"
        />
        <DigestFactCard
          label="阶段"
          title={stageSummary || "阶段链路待补齐"}
          lines={[
            reviewSnapshot?.summary?.decisionExplanation?.trim() || "系统会把阶段推进、收尾原因和策略说明汇总到这里。",
          ]}
          tone="phase"
        />
        <DigestFactCard
          label="训练重点"
          title={profileFocus[0] || gaps[0] || "训练重点待生成"}
          lines={
            profileFocus.length > 0
              ? profileFocus
              : gaps.length > 0
                ? gaps
                : [loading ? "正在汇总最该优先补的方向" : "评分与画像返回后，这里会显示最该优先补的方向"]
          }
          tone="warning"
        />
      </div>

      {(strengths.length > 0 || gaps.length > 0) ? (
        <div className="mt-4 grid gap-3 lg:grid-cols-2">
          <DigestListSection
            label="亮点"
            items={strengths.length > 0 ? strengths : ["评分卡返回后，这里会显示本场最稳的表现。"]}
            tone="success"
          />
          <DigestListSection
            label="待补强"
            items={gaps.length > 0 ? gaps : ["评分卡返回后，这里会显示最该优先补的短板。"]}
            tone="warning"
          />
        </div>
      ) : null}

      {(canExpandTrace || canExpandProfile) ? (
        <div className="mt-4 rounded-[1.3rem] border border-[rgba(214,222,234,0.92)] bg-[rgba(255,255,255,0.82)] px-4 py-4">
          <div className="flex flex-wrap items-center gap-2">
            {canExpandTrace ? (
              <InlineToggleChip
                active={expandedSection === "trace"}
                icon={<Waypoints className="h-3.5 w-3.5" />}
                label={expandedSection === "trace" ? "收起追问树" : "展开追问树"}
                onClick={() => setExpandedSection((current) => current === "trace" ? null : "trace")}
              />
            ) : null}
            {canExpandProfile ? (
              <InlineToggleChip
                active={expandedSection === "profile"}
                icon={<Radar className="h-3.5 w-3.5" />}
                label={expandedSection === "profile" ? "收起画像雷达" : "展开画像雷达"}
                onClick={() => setExpandedSection((current) => current === "profile" ? null : "profile")}
              />
            ) : null}
          </div>

          {expandedSection === "trace" && trace ? (
            <div className="mt-4">
              <MiniTraceTree trace={trace} />
            </div>
          ) : null}

          {expandedSection === "profile" && profile ? (
            <div className="mt-4">
              <MiniProfileRadar profile={profile} />
            </div>
          ) : null}
        </div>
      ) : null}
    </div>
  );
}

function InlineToggleChip({
  active,
  icon,
  label,
  onClick,
}: {
  active: boolean;
  icon: ReactNode;
  label: string;
  onClick: () => void;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      className={`inline-flex items-center gap-2 rounded-full border px-3 py-1.5 text-sm transition ${
        active
          ? "border-[rgba(191,219,254,0.96)] bg-[rgba(239,246,255,0.96)] text-[rgb(29,78,216)]"
          : "border-[rgba(226,231,239,0.96)] bg-[rgba(248,250,252,0.96)] text-[rgba(71,85,105,0.86)] hover:border-[rgba(191,219,254,0.96)] hover:text-[rgb(37,99,235)]"
      }`}
    >
      {icon}
      {label}
      {active ? <ChevronDown className="h-3.5 w-3.5" /> : <ChevronRight className="h-3.5 w-3.5" />}
    </button>
  );
}

function DigestFactCard({
  label,
  title,
  lines,
  tone,
}: {
  label: string;
  title: string;
  lines: string[];
  tone: "score" | "phase" | "trace" | "warning";
}) {
  const toneClass =
    tone === "score"
      ? "border-[rgba(191,219,254,0.92)] bg-[rgba(239,246,255,0.9)]"
      : tone === "phase"
        ? "border-[rgba(196,181,253,0.88)] bg-[rgba(245,243,255,0.9)]"
        : tone === "warning"
          ? "border-[rgba(254,240,138,0.88)] bg-[rgba(254,252,232,0.9)]"
          : "border-[rgba(214,222,234,0.92)] bg-[rgba(248,250,252,0.9)]";

  return (
    <section className={`rounded-[1.25rem] border px-4 py-4 ${toneClass}`}>
      <p className="text-[0.68rem] uppercase tracking-[0.18em] text-[rgba(71,85,105,0.68)]">{label}</p>
      <h4 className="mt-2 text-base font-semibold text-[rgb(15,23,42)]">{title}</h4>
      <div className="mt-3 space-y-2">
        {lines.map((line) => (
          <p key={`${label}-${line}`} className="text-sm leading-7 text-[rgba(51,65,85,0.86)]">
            {line}
          </p>
        ))}
      </div>
    </section>
  );
}

function DigestListSection({
  label,
  items,
  tone,
}: {
  label: string;
  items: string[];
  tone: "success" | "warning";
}) {
  const toneClass =
    tone === "success"
      ? "border-[rgba(187,247,208,0.92)] bg-[rgba(240,253,244,0.9)]"
      : "border-[rgba(254,240,138,0.9)] bg-[rgba(254,252,232,0.92)]";

  return (
    <section className={`rounded-[1.25rem] border px-4 py-4 ${toneClass}`}>
      <p className="text-[0.68rem] uppercase tracking-[0.18em] text-[rgba(71,85,105,0.68)]">{label}</p>
      <div className="mt-3 space-y-2">
        {items.map((item) => (
          <p key={`${label}-${item}`} className="text-sm leading-7 text-[rgba(51,65,85,0.86)]">
            {item}
          </p>
        ))}
      </div>
    </section>
  );
}

function MiniTraceTree({ trace }: { trace: InterviewTraceTree }) {
  const tree = buildMiniTraceForest(trace.nodes ?? []);

  if (tree.length === 0) {
    return (
      <p className="rounded-[1.1rem] border border-dashed border-[rgba(203,213,225,0.9)] px-4 py-4 text-sm text-[rgba(100,116,139,0.82)]">
        追问树还在生成中。
      </p>
    );
  }

  return (
    <div>
      <div className="flex items-center justify-between gap-3">
        <div>
          <p className="text-sm font-semibold text-[rgb(15,23,42)]">小型追问树视图</p>
          <p className="mt-1 text-sm leading-7 text-[rgba(71,85,105,0.8)]">展开后可以直接看到首问如何被持续深挖、切题和收束。</p>
        </div>
        <div className="rounded-full border border-[rgba(254,240,138,0.92)] bg-[rgba(254,252,232,0.94)] px-3 py-1.5 text-[0.72rem] uppercase tracking-[0.18em] text-[rgb(161,98,7)]">
          {trace.questionCount} 个节点
        </div>
      </div>
      <div className="mt-4 space-y-3">
        {tree.map((node) => (
          <MiniTraceNodeCard key={node.id} node={node} depth={0} />
        ))}
      </div>
    </div>
  );
}

function MiniTraceNodeCard({
  node,
  depth,
}: {
  node: MiniTraceTreeNode;
  depth: number;
}) {
  const [expanded, setExpanded] = useState(depth < 1);
  const hasChildren = node.children.length > 0;
  const phaseLabel = node.phase ? formatInterviewPhaseLabel(node.phase) : "未知阶段";
  const metaLine = [
    typeof node.round === "number" ? `R${node.round}` : null,
    phaseLabel,
    node.reason ? formatDecisionReasonLabel(node.reason) : null,
  ].filter(Boolean).join(" · ");

  return (
    <div
      className="rounded-[1.15rem] border border-[rgba(214,222,234,0.92)] bg-[rgba(248,250,252,0.88)] px-4 py-4"
      style={{ marginLeft: `${depth * 18}px` }}
    >
      <div className="flex items-start gap-3">
        {hasChildren ? (
          <button
            type="button"
            onClick={() => setExpanded((current) => !current)}
            className="mt-0.5 inline-flex h-7 w-7 shrink-0 items-center justify-center rounded-full border border-[rgba(191,219,254,0.92)] bg-[rgba(239,246,255,0.96)] text-[rgb(37,99,235)]"
          >
            {expanded ? <ChevronDown className="h-3.5 w-3.5" /> : <ChevronRight className="h-3.5 w-3.5" />}
          </button>
        ) : (
          <span className="mt-2 inline-flex h-3 w-3 shrink-0 rounded-full bg-[rgba(59,130,246,0.76)]" />
        )}
        <div className="min-w-0 flex-1">
          <p className="text-[0.72rem] uppercase tracking-[0.18em] text-[rgba(100,116,139,0.72)]">{metaLine}</p>
          <p className="mt-2 text-sm font-medium leading-7 text-[rgb(15,23,42)]">{node.question}</p>
          {node.answerSummary ? (
            <p className="mt-2 text-sm leading-7 text-[rgba(71,85,105,0.84)]">{compactLine(node.answerSummary, 180)}</p>
          ) : null}
        </div>
      </div>

      {expanded && hasChildren ? (
        <div className="mt-3 space-y-3">
          {node.children.map((child) => (
            <MiniTraceNodeCard key={child.id} node={child} depth={depth + 1} />
          ))}
        </div>
      ) : null}
    </div>
  );
}

function MiniProfileRadar({ profile }: { profile: CandidateProfile }) {
  const points = profile.radar ?? [];
  const dimensions = profile.dimensions ?? [];
  const focus = (profile.recommendedFocus ?? []).slice(0, 3);

  if (points.length === 0 && dimensions.length === 0) {
    return (
      <p className="rounded-[1.1rem] border border-dashed border-[rgba(203,213,225,0.9)] px-4 py-4 text-sm text-[rgba(100,116,139,0.82)]">
        画像雷达还在生成中。
      </p>
    );
  }

  const radarPoints = points.length > 0
    ? points
    : dimensions.slice(0, 6).map((dimension) => ({
        key: dimension.key,
        label: dimension.label,
        normalizedScore: typeof dimension.normalizedScore === "number" ? dimension.normalizedScore : Math.max(0, Math.min(100, Math.round(((dimension.score + 6) / 12) * 100))),
      }));

  const size = 280;
  const center = size / 2;
  const radius = 96;
  const levels = [25, 50, 75, 100];
  const polygon = radarPolygonPoints(radarPoints, center, radius).join(" ");

  return (
    <div className="grid gap-4 lg:grid-cols-[320px_minmax(0,1fr)]">
      <div className="rounded-[1.2rem] border border-[rgba(187,247,208,0.92)] bg-[rgba(240,253,244,0.92)] px-4 py-4">
        <p className="text-sm font-semibold text-[rgb(15,23,42)]">画像雷达图</p>
        <p className="mt-2 text-sm leading-7 text-[rgba(71,85,105,0.82)]">这张雷达图会把本场沉淀到画像里的关键能力维度压缩成一张图，更适合一眼看强弱分布。</p>
        <svg viewBox={`0 0 ${size} ${size}`} className="mx-auto mt-4 h-[280px] w-[280px]">
          {levels.map((level) => (
            <polygon
              key={level}
              points={radarRingPoints(radarPoints.length, center, radius * (level / 100)).join(" ")}
              fill="none"
              stroke="rgba(148,163,184,0.26)"
              strokeWidth="1"
            />
          ))}
          {radarPoints.map((point, index) => {
            const angle = (Math.PI * 2 * index) / radarPoints.length - Math.PI / 2;
            const x = center + Math.cos(angle) * radius;
            const y = center + Math.sin(angle) * radius;
            const labelX = center + Math.cos(angle) * (radius + 26);
            const labelY = center + Math.sin(angle) * (radius + 26);
            return (
              <g key={point.key}>
                <line x1={center} y1={center} x2={x} y2={y} stroke="rgba(148,163,184,0.3)" strokeWidth="1" />
                <text
                  x={labelX}
                  y={labelY}
                  textAnchor={labelX >= center ? "start" : "end"}
                  dominantBaseline="middle"
                  fontSize="12"
                  fill="rgba(71,85,105,0.82)"
                >
                  {point.label}
                </text>
              </g>
            );
          })}
          <polygon points={polygon} fill="rgba(37,99,235,0.18)" stroke="rgba(37,99,235,0.82)" strokeWidth="2" />
          {radarPoints.map((point, index) => {
            const angle = (Math.PI * 2 * index) / radarPoints.length - Math.PI / 2;
            const distance = radius * (point.normalizedScore / 100);
            const x = center + Math.cos(angle) * distance;
            const y = center + Math.sin(angle) * distance;
            return <circle key={`${point.key}-dot`} cx={x} cy={y} r="4" fill="rgba(37,99,235,0.94)" />;
          })}
        </svg>
      </div>

      <div className="space-y-3">
        <div className="grid gap-3 sm:grid-cols-2">
          {radarPoints.map((point) => (
            <div key={point.key} className="rounded-[1.1rem] border border-[rgba(214,222,234,0.92)] bg-[rgba(255,255,255,0.88)] px-4 py-4">
              <div className="flex items-center justify-between gap-3">
                <p className="text-sm font-medium text-[rgb(31,41,55)]">{point.label}</p>
                <span className="rounded-full border border-[rgba(191,219,254,0.92)] bg-[rgba(239,246,255,0.96)] px-2.5 py-1 text-[0.72rem] text-[rgb(37,99,235)]">
                  {point.normalizedScore}/100
                </span>
              </div>
              <div className="mt-3 h-2.5 overflow-hidden rounded-full bg-[rgba(226,232,240,0.72)]">
                <div
                  className="h-full rounded-full bg-[linear-gradient(90deg,rgba(37,99,235,0.92),rgba(125,211,252,0.92))]"
                  style={{ width: `${point.normalizedScore}%` }}
                />
              </div>
            </div>
          ))}
        </div>

        {focus.length > 0 ? (
          <div className="rounded-[1.1rem] border border-[rgba(254,240,138,0.9)] bg-[rgba(254,252,232,0.92)] px-4 py-4">
            <p className="text-[0.72rem] uppercase tracking-[0.18em] text-[rgb(161,98,7)]">推荐训练重点</p>
            <div className="mt-3 space-y-2">
              {focus.map((item) => (
                <p key={item} className="text-sm leading-7 text-[rgb(133,77,14)]">{item}</p>
              ))}
            </div>
          </div>
        ) : null}
      </div>
    </div>
  );
}

function TimelineEventCard({ event, compact = false }: { event: RunEvent; compact?: boolean }) {
  const meta = getEventMeta(event);

  return (
    <div className={`rounded-[1.4rem] border ${compact ? "px-4 py-3" : "px-4 py-4"} ${meta.frameClass}`}>
      <div className="flex items-center justify-between gap-3">
        <div className="flex items-center gap-3">
          <div className={`flex h-9 w-9 items-center justify-center rounded-full ${meta.iconClass}`}>
            <meta.icon className="h-4 w-4" />
          </div>
          <div>
            <p className="font-display text-base text-[rgb(72,91,114)]">{meta.label}</p>
            <p className="text-xs text-[rgba(115,137,161,0.78)]">{meta.description}</p>
          </div>
        </div>
        <p className="text-[0.68rem] uppercase tracking-[0.22em] text-[rgba(122,144,168,0.72)]">
          {new Date(event.timestamp).toLocaleTimeString()}
        </p>
      </div>
      {!compact ? (
        <pre className="mt-3 whitespace-pre-wrap break-words font-body text-xs leading-6 text-[rgba(122,144,168,0.8)]">
          {JSON.stringify(event.payload, null, 2)}
        </pre>
      ) : null}
    </div>
  );
}

function ClarifyPanel({ event, latestCheckpoint }: { event: RunEvent; latestCheckpoint: RunEvent | undefined }) {
  const payload = event.payload as { question?: string; field?: string } | null;
  const checkpointPayload = latestCheckpoint?.payload as { resumeCount?: number } | undefined;

  return (
    <div className="rounded-[1.7rem] border border-[rgba(153,191,201,0.2)] bg-[linear-gradient(135deg,rgba(247,253,255,0.98),rgba(236,248,250,0.96))] px-5 py-5 shadow-[0_24px_46px_rgba(111,177,190,0.12)]">
      <div className="flex items-start justify-between gap-4">
        <div className="flex items-start gap-4">
          <div className="flex h-11 w-11 items-center justify-center rounded-full bg-[rgba(118,177,189,0.12)] text-[rgb(58,124,139)]">
            <LockKeyhole className="h-5 w-5" />
          </div>
          <div>
            <p className="tech-label text-[0.64rem] text-[rgba(115,137,161,0.76)]">澄清关卡</p>
            <h3 className="mt-2 font-display text-xl text-[rgb(72,91,114)]">执行因缺少输入而暂停</h3>
            <p className="mt-3 text-sm leading-7 text-[rgba(115,137,161,0.84)]">
              {payload?.question ?? "运行时需要补充信息后才能继续。"}
            </p>
          </div>
        </div>
        <div className="rounded-full bg-[rgba(118,177,189,0.12)] px-3 py-2 text-[0.68rem] uppercase tracking-[0.22em] text-[rgba(115,137,161,0.78)]">
          字段：{payload?.field ?? "未知"}
        </div>
      </div>
      {latestCheckpoint && (
        <div className="mt-4 flex items-center gap-2 text-xs uppercase tracking-[0.24em] text-[rgba(115,137,161,0.76)]">
          <RotateCcw className="h-3.5 w-3.5" />
          检查点已就绪
          <span className="h-1 w-1 rounded-full bg-[rgba(115,137,161,0.44)]" />
          恢复次数：{checkpointPayload?.resumeCount ?? 0}
        </div>
      )}
    </div>
  );
}

function RunGuidance({
  runStatus,
  activeClarify,
  latestFailure,
}: {
  runStatus: string | undefined;
  activeClarify: boolean;
  latestFailure: RunEvent | null;
}) {
  if (activeClarify) {
    return (
      <p className="flex items-center gap-2 text-sm text-[rgba(58,124,139,0.86)]">
        <Hourglass className="h-4 w-4" />
        提交澄清回复后，将从最新检查点继续执行。
      </p>
    );
  }

  if (runStatus === "running" || runStatus === "resuming") {
    return (
      <p className="flex items-center gap-2 text-sm text-[rgba(96,134,145,0.74)]">
        <LoaderCircle className="h-4 w-4 animate-spin" />
        运行时正在执行中。在新的澄清请求或终态事件到来前，输入会保持锁定。
      </p>
    );
  }

  if (latestFailure) {
    const payload = latestFailure.payload as { error?: string } | null;
    return (
      <p className="flex items-center gap-2 text-sm text-[rgb(0,102,255)]">
        <Flame className="h-4 w-4" />
        最近一次失败：{payload?.error ?? "未知错误"}
      </p>
    );
  }

  if (runStatus === "cancelled") {
    return (
      <p className="flex items-center gap-2 text-sm text-[rgb(0,102,255)]">
        <Flame className="h-4 w-4" />
        这次运行已取消。如果你想基于当前任务配置重试，请启动一次恢复运行。
      </p>
    );
  }

  return (
    <p className="flex items-center gap-2 text-sm text-[rgba(115,137,161,0.78)]">
      <Waypoints className="h-4 w-4" />
      当运行到达新的交互边界后，可在这里继续追加后续对话。
    </p>
  );
}

function CopilotPanel({
  feedback,
  hint,
  events,
  busy,
  onRequestHint,
}: {
  feedback: CopilotFeedback | null;
  hint: CopilotHint | null;
  events: RunEvent[];
  busy: boolean;
  onRequestHint: () => void;
}) {
  const hasData = Boolean(feedback || hint || events.length);
  const stateMeta = getCopilotStateMeta(feedback?.state);
  const confidenceLabel = typeof feedback?.confidence === "number" ? `${Math.round(feedback.confidence * 100)}%` : null;
  const strategy = hint?.strategy?.length ? hint.strategy : buildFallbackCopilotStrategy(feedback);
  const guardrails = hint?.guardrails?.length ? hint.guardrails : buildFallbackCopilotGuardrails(feedback);

  return (
    <div className="rounded-[1.55rem] border border-[rgba(222,227,236,0.92)] bg-[linear-gradient(135deg,rgba(255,255,255,0.98),rgba(247,250,255,0.96))] px-5 py-5 shadow-[0_12px_28px_rgba(15,23,42,0.05)]">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <p className="tech-label text-[0.64rem] text-[rgba(97,123,150,0.66)]">答题副驾驶</p>
          <h3 className="mt-1 font-display text-lg text-[rgb(17,24,39)]">实时答题辅助</h3>
          <p className="mt-1 text-sm leading-6 text-[rgba(100,116,139,0.84)]">
            像 ChatGPT / Claude 的侧边提醒一样，它只负责把“该怎么答”说清楚，不直接替你作答。
          </p>
        </div>
        <Button
          type="button"
          variant="outline"
          disabled={busy}
          onClick={onRequestHint}
          className="h-10 rounded-full border-[rgba(191,219,254,0.96)] bg-[rgba(239,246,255,0.96)] px-4 text-[rgb(37,99,235)] hover:bg-[rgba(219,234,254,0.96)]"
        >
          {busy ? <LoaderCircle className="mr-2 h-4 w-4 animate-spin" /> : <Sparkles className="mr-2 h-4 w-4" />}
          {busy ? "提示生成中" : "请求 Copilot"}
        </Button>
      </div>

      {!hasData ? (
        <div className="mt-4 rounded-[1.25rem] border border-dashed border-[rgba(222,227,236,0.92)] px-4 py-4">
          <p className="text-sm leading-7 text-[rgba(107,114,128,0.82)]">
            当你卡住、紧张，或不确定这轮该先补 tradeoff 还是实现细节时，可以随时请求提示。
          </p>
          <div className="mt-3 flex flex-wrap gap-2">
            {["先给结论", "补一个例子", "再收 tradeoff"].map((item) => (
              <span
                key={item}
                className="rounded-full border border-[rgba(226,231,239,0.96)] bg-[rgba(248,250,252,0.98)] px-3 py-1 text-[0.76rem] text-[rgba(71,85,105,0.82)]"
              >
                {item}
              </span>
            ))}
          </div>
        </div>
      ) : (
        <div className="mt-4 grid gap-3 xl:grid-cols-[minmax(0,0.78fr)_minmax(0,1.22fr)]">
          <div className="space-y-3">
            <div className={`rounded-[1.25rem] border px-4 py-4 ${stateMeta.frameClass}`}>
              <div className="flex items-center justify-between gap-3">
                <div>
                  <p className="tech-label text-[0.64rem] opacity-70">当前状态</p>
                  <p className="mt-2 font-display text-lg">{stateMeta.label}</p>
                </div>
                {confidenceLabel ? (
                  <span className="rounded-full bg-[rgba(255,255,255,0.72)] px-3 py-1 text-[0.72rem] uppercase tracking-[0.16em]">
                    {confidenceLabel}
                  </span>
                ) : null}
              </div>
              <p className="mt-3 text-sm leading-7">{feedback?.summary ?? "等待反馈。"}</p>
              {feedback?.triggers?.length ? (
                <div className="mt-3 flex flex-wrap gap-2">
                  {feedback.triggers.map((item) => (
                    <span
                      key={item}
                      className="rounded-full border border-[rgba(255,255,255,0.7)] bg-[rgba(255,255,255,0.68)] px-3 py-1 text-[0.72rem]"
                    >
                      {item}
                    </span>
                  ))}
                </div>
              ) : null}
            </div>

            <div className="rounded-[1.25rem] border border-[rgba(214,222,234,0.9)] bg-[rgba(248,250,255,0.92)] px-4 py-4">
              <p className="tech-label text-[0.64rem] text-[rgba(97,123,150,0.66)]">建议动作</p>
              <div className="mt-3 space-y-2">
                {(feedback?.suggestedMoves?.length ? feedback.suggestedMoves : buildFallbackSuggestedMoves(feedback)).map((item, index) => (
                  <p key={`${item}-${index}`} className="text-sm leading-6 text-[rgba(71,85,105,0.86)]">
                    {index + 1}. {item}
                  </p>
                ))}
              </div>
            </div>
          </div>

          <div className="rounded-[1.25rem] border border-[rgba(191,219,254,0.92)] bg-[rgba(239,246,255,0.92)] px-4 py-4">
            <p className="tech-label text-[0.64rem] text-[rgba(37,99,235,0.72)]">建议答法</p>
            <p className="mt-2 font-medium text-[rgb(29,78,216)]">{hint?.title ?? stateMeta.title}</p>
            <p className="mt-2 text-sm leading-7 text-[rgba(30,64,175,0.84)]">{hint?.summary ?? "Copilot 会围绕当前问题给出更容易落地的回答路径。"}</p>

            <div className="mt-4 grid gap-3 lg:grid-cols-2">
              <div className="rounded-[1rem] border border-[rgba(191,219,254,0.82)] bg-[rgba(255,255,255,0.78)] px-4 py-4">
                <p className="text-xs uppercase tracking-[0.18em] text-[rgba(37,99,235,0.74)]">推荐结构</p>
                <div className="mt-3 space-y-2">
                  {strategy.map((item, index) => (
                    <p key={`${item}-${index}`} className="text-sm leading-6 text-[rgba(30,64,175,0.86)]">
                      {index + 1}. {item}
                    </p>
                  ))}
                </div>
              </div>

              <div className="rounded-[1rem] border border-[rgba(219,234,254,0.92)] bg-[rgba(255,255,255,0.78)] px-4 py-4">
                <p className="text-xs uppercase tracking-[0.18em] text-[rgba(37,99,235,0.74)]">回答 guardrail</p>
                <div className="mt-3 flex flex-wrap gap-2">
                  {guardrails.map((item) => (
                    <span
                      key={item}
                      className="rounded-full border border-[rgba(191,219,254,0.88)] bg-[rgba(255,255,255,0.88)] px-3 py-1 text-[0.74rem] text-[rgba(30,64,175,0.86)]"
                    >
                      {item}
                    </span>
                  ))}
                </div>
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

function buildOperatorBrief({
  run,
  reviewSnapshot,
  decisionAudit,
  feedback,
  hint,
  latestFailure,
  activeClarify,
  currentPhaseLabel,
  currentRunStatusLabel,
  formatSignalLabel,
  formatDecisionReasonLabel,
}: {
  run: Run | null;
  reviewSnapshot: ReviewSnapshot | null;
  decisionAudit: DecisionAudit | null;
  feedback: CopilotFeedback | null;
  hint: CopilotHint | null;
  latestFailure: RunEvent | null;
  activeClarify: boolean;
  currentPhaseLabel: string;
  currentRunStatusLabel: string;
  formatSignalLabel: (signal?: string | null) => string;
  formatDecisionReasonLabel: (reason?: string | null) => string;
}) {
  const summary = reviewSnapshot?.summary ?? null;
  const decisionReason = decisionAudit?.decision?.reason ?? summary?.decisionReason ?? run?.interviewState?.lastDecision?.reason;
  const decisionExplanation =
    decisionAudit?.decision?.explanation ??
    summary?.decisionExplanation ??
    run?.interviewState?.lastDecision?.explanation ??
    feedback?.summary ??
    "系统会根据上一轮的质量信号决定下一轮追问方向。";
  const focusSignals = [
    ...(summary?.recommendedFocus ?? []),
    ...(summary?.historicalWeaknessesHit ?? []),
    ...(decisionAudit?.decision?.recommendedFocus ?? []),
    ...(decisionAudit?.analysis?.weakSignals ?? []),
  ].filter(Boolean);
  const focusLabels = Array.from(new Set(focusSignals)).slice(0, 4).map((item) => formatSignalLabel(item));
  const checklist = hint?.strategy?.length
    ? hint.strategy.slice(0, 3)
    : buildChecklistFromSignals(decisionReason, decisionAudit, feedback, formatSignalLabel);

  if (activeClarify) {
    return {
      headline: "系统在等你补齐关键信息",
      summary: "当前不是继续自由发挥的时机，先把澄清问题补完整，运行才会从检查点继续。",
      systemJudgement: "缺少必要输入，流程被显式暂停。",
      primaryAction: "直接回答澄清点，不要扩写无关背景。",
      guardrailTitle: "别先开启新话题。",
      checklist,
      focusLabels,
      statusLabel: currentRunStatusLabel,
      badgeClass: "border-[rgba(191,219,254,0.92)] bg-[rgba(239,246,255,0.96)] text-[rgb(29,78,216)]",
      railClass: "bg-[linear-gradient(90deg,rgba(59,130,246,0.94),rgba(125,211,252,0.92))]",
      actionTone: "info" as const,
    };
  }

  if (latestFailure) {
    const payload = (latestFailure.payload ?? null) as { error?: string } | null;
    return {
      headline: "这一轮被阻塞了，先看失败点再继续",
      summary: payload?.error?.trim() || "运行过程中出现错误，建议先查看复盘或恢复运行入口。",
      systemJudgement: "系统已停止继续追问。",
      primaryAction: "优先确认报错，再决定是重试还是重新回答。",
      guardrailTitle: "别忽略错误直接继续发消息。",
      checklist,
      focusLabels,
      statusLabel: currentRunStatusLabel,
      badgeClass: "border-[rgba(254,202,202,0.94)] bg-[rgba(254,242,242,0.98)] text-[rgb(185,28,28)]",
      railClass: "bg-[linear-gradient(90deg,rgba(244,63,94,0.9),rgba(251,113,133,0.9))]",
      actionTone: "warning" as const,
    };
  }

  if (run?.status === "completed") {
    return {
      headline: "这一场已经结束，现在更适合复盘而不是继续硬答",
      summary: decisionExplanation,
      systemJudgement: decisionReason ? `最终收束原因：${formatDecisionReasonLabel(decisionReason)}` : "评分、追问链路和画像已经聚合完成。",
      primaryAction: "先看评分卡和策略，再决定下一轮训练重点。",
      guardrailTitle: "别只看分数，忽略导致分数的信号。",
      checklist,
      focusLabels,
      statusLabel: "可复盘",
      badgeClass: "border-[rgba(187,247,208,0.92)] bg-[rgba(240,253,244,0.96)] text-[rgb(21,128,61)]",
      railClass: "bg-[linear-gradient(90deg,rgba(34,197,94,0.88),rgba(134,239,172,0.92))]",
      actionTone: "success" as const,
    };
  }

  if (run?.status === "running" || run?.status === "resuming") {
    return {
      headline: `系统正在推进${currentPhaseLabel}阶段，先等这一轮生成完成`,
      summary: decisionExplanation,
      systemJudgement: decisionReason ? `最近决策：${formatDecisionReasonLabel(decisionReason)}` : "系统正在根据上一轮回答选择下一问。",
      primaryAction: "先观察最新问题，等输入解锁后再给一条更聚焦的回答。",
      guardrailTitle: "别在锁定期间连续补发多条消息。",
      checklist,
      focusLabels,
      statusLabel: currentRunStatusLabel,
      badgeClass: "border-[rgba(186,230,253,0.92)] bg-[rgba(236,254,255,0.96)] text-[rgb(14,116,144)]",
      railClass: "bg-[linear-gradient(90deg,rgba(14,165,233,0.9),rgba(56,189,248,0.92))]",
      actionTone: "info" as const,
    };
  }

  return {
    headline: `现在处于${currentPhaseLabel}阶段，建议按信号补最短板`,
    summary: decisionExplanation,
    systemJudgement: decisionReason ? `当前追问方向：${formatDecisionReasonLabel(decisionReason)}` : "系统会继续围绕最近一轮弱信号追问。",
    primaryAction: checklist[0] ?? "先给结论，再补一个更具体的工程证据。",
    guardrailTitle: "别继续铺抽象背景或重复上一轮表述。",
    checklist,
    focusLabels,
    statusLabel: currentRunStatusLabel,
    badgeClass: "border-[rgba(191,219,254,0.92)] bg-[rgba(239,246,255,0.96)] text-[rgb(29,78,216)]",
    railClass: "bg-[linear-gradient(90deg,rgba(59,130,246,0.94),rgba(99,102,241,0.9))]",
    actionTone: "info" as const,
  };
}

function buildChecklistFromSignals(
  decisionReason: string | undefined,
  decisionAudit: DecisionAudit | null,
  feedback: CopilotFeedback | null,
  formatSignalLabel: (signal?: string | null) => string,
) {
  const weakSignals = decisionAudit?.analysis?.weakSignals ?? [];

  if (feedback?.suggestedMoves?.length) {
    return feedback.suggestedMoves.slice(0, 3);
  }

  if (decisionReason === "missing_tradeoff" || weakSignals.includes("missing_tradeoff")) {
    return [
      "先明确你的主结论，不要再铺背景。",
      "补一个你会怎么权衡成本、复杂度和稳定性的 tradeoff。",
      "最后用一个具体场景解释为什么你会这么选。",
    ];
  }

  if (decisionReason === "lack_implementation_detail" || weakSignals.includes("missing_implementation_detail")) {
    return [
      "直接落到模块、数据流或关键接口，不要只讲原则。",
      "说明异常处理、边界条件和失败回退怎么做。",
      "最后用一句话收束为什么这个实现能满足题目目标。",
    ];
  }

  if (decisionReason === "weak_signal_timeout" || weakSignals.includes("missing_timeout_detail")) {
    return [
      "先说 timeout / retry / cancellation 的整体策略。",
      "再补超时阈值、重试退避和失败兜底的细节。",
      "最后说明这样设计对稳定性和体验的影响。",
    ];
  }

  if (decisionReason === "weak_signal_observability" || weakSignals.includes("missing_observability_detail")) {
    return [
      "先回答你会看哪些日志、指标和 trace。",
      "再补告警阈值、排障路径和定位动作。",
      "最后说明你如何用这些信号发现并修复问题。",
    ];
  }

  return [
    "先给一个明确判断，避免继续绕概念。",
    `围绕“${formatSignalLabel(weakSignals[0])}”补一条更具体的工程细节。`,
    "最后用一个例子或真实取舍把答案收住。",
  ];
}

function getCopilotStateMeta(state?: CopilotFeedback["state"]) {
  switch (state) {
    case "stuck":
      return {
        label: "卡住了",
        title: "先打破停顿，再组织答案",
        frameClass: "border-[rgba(253,230,138,0.92)] bg-[rgba(254,252,232,0.96)] text-[rgb(161,98,7)]",
      };
    case "anxious":
      return {
        label: "有点紧张",
        title: "先收束句子，再补细节",
        frameClass: "border-[rgba(251,207,232,0.92)] bg-[rgba(253,242,248,0.96)] text-[rgb(190,24,93)]",
      };
    case "needs_structure":
      return {
        label: "结构还不够稳",
        title: "先把答案骨架搭出来",
        frameClass: "border-[rgba(191,219,254,0.92)] bg-[rgba(239,246,255,0.96)] text-[rgb(29,78,216)]",
      };
    case "needs_specificity":
      return {
        label: "还不够具体",
        title: "把概念压成实现细节",
        frameClass: "border-[rgba(186,230,253,0.92)] bg-[rgba(236,254,255,0.96)] text-[rgb(14,116,144)]",
      };
    default:
      return {
        label: "状态稳定",
        title: "继续把答案收得更锋利",
        frameClass: "border-[rgba(187,247,208,0.92)] bg-[rgba(240,253,244,0.96)] text-[rgb(21,128,61)]",
      };
  }
}

function buildFallbackSuggestedMoves(feedback: CopilotFeedback | null) {
  switch (feedback?.state) {
    case "stuck":
      return ["先说结论", "补一个最关键细节", "最后补 tradeoff"];
    case "anxious":
      return ["放慢一句话长度", "先答主问题", "不要同时展开太多分支"];
    case "needs_specificity":
      return ["补模块或数据流", "补边界条件", "补一个例子"];
    case "needs_structure":
      return ["按结论 / 依据 / 取舍组织", "一段只讲一个重点", "最后一句收束"];
    default:
      return ["先给判断", "再给证据", "最后补风险或 tradeoff"];
  }
}

function buildFallbackCopilotStrategy(feedback: CopilotFeedback | null) {
  switch (feedback?.state) {
    case "stuck":
      return ["先重复题目的核心目标", "直接给一个判断", "再补一条最有把握的依据"];
    case "needs_specificity":
      return ["先说你会怎么做", "再说具体模块或步骤", "最后补为什么这样做"];
    case "needs_structure":
      return ["第一句先下结论", "第二句补实现路径", "第三句补 tradeoff 或风险"];
    default:
      return ["先给结论", "再给关键实现或证据", "最后给 tradeoff / 边界条件"];
  }
}

function buildFallbackCopilotGuardrails(feedback: CopilotFeedback | null) {
  if (feedback?.state === "anxious") {
    return ["不要一句话塞太多点", "不要还没回答就先道歉", "不要先展开背景故事"];
  }
  return ["不要直接背标准答案", "不要只讲概念不落实现", "不要忽略 tradeoff 和边界"];
}

function buildWorkspaceTitleFromPrompt(prompt: string) {
  const firstLine = prompt
    .split(/\n+/)
    .map((line) => line.trim())
    .find(Boolean);

  if (!firstLine) {
    return "新的面试工作区";
  }

  const normalized = firstLine.replace(/\s+/g, " ").trim();
  const company = detectCompanyName(normalized);
  const track = detectInterviewTrack(normalized);

  const title = [company, track, "面试"].filter(Boolean).join(" ");
  if (title) {
    return title.length > 24 ? `${title.slice(0, 24).trimEnd()}…` : title;
  }

  const fallback = normalized
    .replace(/^(请|帮我)?\s*(模拟|来一场|做一场|开启|进行)?\s*(一场)?/u, "")
    .replace(/[，。！？].*$/u, "")
    .replace(/(岗位|职位)?的?技术面试/u, "面试")
    .replace(/技术面试/u, "面试")
    .trim();

  if (!fallback) {
    return "新的面试工作区";
  }

  return fallback.length > 24 ? `${fallback.slice(0, 24).trimEnd()}…` : fallback;
}

function detectCompanyName(prompt: string) {
  const companyRules: Array<{ pattern: RegExp; label: string }> = [
    { pattern: /(字节跳动|字节)/iu, label: "字节" },
    { pattern: /(阿里巴巴|阿里云|阿里)/iu, label: "阿里" },
    { pattern: /(腾讯)/iu, label: "腾讯" },
    { pattern: /(美团)/iu, label: "美团" },
    { pattern: /(快手)/iu, label: "快手" },
    { pattern: /(小红书)/iu, label: "小红书" },
    { pattern: /(拼多多|PDD)/iu, label: "拼多多" },
    { pattern: /(百度)/iu, label: "百度" },
    { pattern: /(京东|JD)/iu, label: "京东" },
    { pattern: /(蚂蚁|蚂蚁集团)/iu, label: "蚂蚁" },
    { pattern: /(OpenAI)/iu, label: "OpenAI" },
    { pattern: /(Anthropic)/iu, label: "Anthropic" },
  ];

  return companyRules.find((rule) => rule.pattern.test(prompt))?.label ?? "";
}

function detectInterviewTrack(prompt: string) {
  const hasGo = /\bgo\b|golang|Go/iu.test(prompt);
  const hasPython = /\bpython\b|Python/iu.test(prompt);
  const hasJava = /\bjava\b|Java/iu.test(prompt);
  const hasFrontend = /(前端|frontend|react|next\.js|nextjs|vue)/iu.test(prompt);
  const hasAgent = /(agent|智能体|tool calling|workflow|orchestration|编排)/iu.test(prompt);
  const hasBackend = /(后端|backend|微服务|服务端)/iu.test(prompt);
  const hasSystem = /(系统设计|架构设计|system design|架构)/iu.test(prompt);

  if (hasGo && hasAgent) return "Go Agent";
  if (hasPython && hasAgent) return "Python Agent";
  if (hasJava && hasAgent) return "Java Agent";
  if (hasAgent) return "Agent";
  if (hasGo) return "Go";
  if (hasPython) return "Python";
  if (hasJava) return "Java";
  if (hasFrontend) return "前端";
  if (hasBackend) return "后端";
  if (hasSystem) return "系统设计";
  return "";
}

function SkillLibraryPanel({
  skills,
  selectedSkill,
  isBusy,
  embedded = false,
  onSelectSkill,
  onCreate,
  onUpload,
}: {
  skills: SkillMetadata[];
  selectedSkill: string;
  isBusy: boolean;
  embedded?: boolean;
  onSelectSkill: (value: string) => void;
  onCreate: () => void;
  onUpload: (event: ChangeEvent<HTMLInputElement>) => void;
}) {
  const normalizedSelectedSkill = typeof selectedSkill === "string" ? selectedSkill.trim() : "";
  const activeSkill = skills.find((skill) => {
    const skillName = typeof skill.name === "string" ? skill.name.trim() : "";
    return skillName === normalizedSelectedSkill;
  });
  const skillOptions = skills.map((skill) => ({
    value: typeof skill.name === "string" && skill.name.trim() ? skill.name.trim() : "__invalid__",
    label: typeof skill.name === "string" && skill.name.trim() ? skill.name.trim() : "未命名技能",
  }));

  return (
    <section className={embedded ? "" : "paper-panel editorial-frame rounded-[2rem] border border-[rgba(153,191,201,0.18)] p-5 shadow-paper"}>
      <div className="flex items-center justify-between gap-3">
        <div>
          <p className="tech-label text-[0.68rem] text-[rgba(97,123,150,0.62)]">{embedded ? "技能管理" : "技能库"}</p>
          <h2 className={`font-display text-[rgb(72,91,114)] ${embedded ? "mt-1 text-xl" : "mt-2 text-2xl"}`}>项目技能</h2>
        </div>
        <div className="flex items-center gap-2">
          <label
            className={`inline-flex h-10 items-center rounded-full px-4 text-sm ${isBusy ? "cursor-default bg-[rgba(146,190,200,0.12)] text-[rgba(96,134,145,0.48)]" : "cursor-pointer bg-[rgba(118,177,189,0.1)] text-[rgb(36,95,110)]"}`}
          >
            <FileUp className="mr-2 h-4 w-4" />
            上传
            <input type="file" className="hidden" disabled={isBusy} accept=".zip,.skill,.md" onChange={onUpload} />
          </label>
          <Button
            type="button"
            disabled={isBusy}
            onClick={onCreate}
            className="h-10 rounded-full bg-[rgb(74,156,175)] px-4 text-primary-foreground hover:bg-[rgb(61,132,150)]"
          >
            <Plus className="mr-2 h-4 w-4" />
            新建
          </Button>
        </div>
      </div>

	        <div className={`${embedded ? "mt-4" : "mt-5"} space-y-4`}>
	          <div className="rounded-[1.4rem] border border-[rgba(153,191,201,0.16)] bg-[rgba(250,254,255,0.9)] px-4 py-4">
	            <p className="tech-label text-[0.64rem] text-[rgba(97,123,150,0.62)]">当前运行技能</p>
          <div className="relative mt-3">
            <select
              value={normalizedSelectedSkill || "__auto__"}
              onChange={(event) => onSelectSkill(event.target.value === "__auto__" ? "" : event.target.value)}
              className="h-12 w-full appearance-none rounded-[1rem] border border-[rgba(153,191,201,0.18)] bg-[rgba(255,255,255,0.95)] px-4 pr-10 text-[rgb(72,91,114)] outline-none transition-colors focus:border-[rgba(0,102,255,0.28)]"
            >
              <option value="__auto__">自动选择</option>
              {skillOptions.filter((skill) => skill.value !== "__invalid__").map((skill) => (
                <option key={skill.value} value={skill.value}>
                  {skill.label}
                </option>
              ))}
            </select>
            <ChevronDown className="pointer-events-none absolute right-4 top-1/2 h-4 w-4 -translate-y-1/2 text-[rgba(97,123,150,0.72)]" />
          </div>
	          <p className="mt-3 text-xs leading-6 text-[rgba(115,137,161,0.78)]">
	            {normalizedSelectedSkill ? activeSkill?.description ?? normalizedSelectedSkill : "未指定时会使用默认 interview skill。"}
	          </p>
	          {activeSkill ? (
	            <div className="mt-3 flex flex-wrap gap-2">
	              {activeSkill.version ? <span className="chip-accent px-3 py-1 text-[11px] normal-case tracking-[0.04em]">v{activeSkill.version}</span> : null}
	              {activeSkill.installSource ? <span className="chip-muted px-3 py-1 text-[11px] normal-case tracking-[0.04em]">{activeSkill.installSource}</span> : null}
	              {(activeSkill.composedOf?.length ?? 0) > 0 ? (
	                <span className="chip-warning px-3 py-1 text-[11px] normal-case tracking-[0.04em]">组合 {activeSkill.composedOf?.length}</span>
	              ) : null}
	            </div>
	          ) : null}
	          </div>

        {skills.length === 0 ? (
          <p className="rounded-[1.3rem] border border-dashed border-[rgba(153,191,201,0.18)] px-4 py-5 text-sm leading-7 text-[rgba(115,137,161,0.78)]">
            还没有可用技能。你可以上传 zip/.skill 包，或者直接手工创建一个 SKILL.md。
          </p>
        ) : null}
      </div>
    </section>
  );
}

function SkillEditorModal({
  draft,
  busy,
  mode,
  onChange,
  onClose,
  onSubmit,
}: {
  draft: SkillDocument;
  busy: boolean;
  mode: "create" | "edit";
  onChange: (value: SkillDocument) => void;
  onClose: () => void;
  onSubmit: (event: React.FormEvent<HTMLFormElement>) => void;
}) {
  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-[rgba(228,241,248,0.48)] px-4 py-6 backdrop-blur-sm">
      <div className="w-full max-w-3xl rounded-[2rem] border border-[rgba(153,191,201,0.18)] bg-[rgb(250,254,255)] p-6 shadow-[0_30px_80px_rgba(111,177,190,0.18)]">
        <div className="flex items-start justify-between gap-4">
          <div>
            <p className="tech-label text-[0.68rem] text-[rgba(97,123,150,0.62)]">技能编辑器</p>
            <h2 className="mt-2 font-display text-3xl text-[rgb(72,91,114)]">{mode === "create" ? "创建技能" : "编辑技能"}</h2>
          </div>
          <button
            type="button"
            onClick={onClose}
            disabled={busy}
            className="inline-flex h-10 w-10 items-center justify-center rounded-full border border-[rgba(153,191,201,0.16)] text-[rgba(115,137,161,0.78)] transition-colors hover:bg-[rgba(118,177,189,0.08)]"
          >
            <X className="h-4 w-4" />
          </button>
        </div>

        <form onSubmit={onSubmit} className="mt-6 space-y-4">
          <div className="rounded-[1.6rem] border border-dashed border-[rgba(153,191,201,0.18)] bg-[rgba(252,255,255,0.9)] px-5 py-5 text-center text-sm leading-7 text-[rgba(115,137,161,0.78)]">
            上传 zip/.skill/SKILL.md 可自动导入；也可以直接在这里编辑 frontmatter 之外的技能正文。
          </div>

          <Field label="技能名称">
            <Input
              value={draft.name}
              onChange={(event) => onChange({ ...draft, name: event.target.value })}
              placeholder="例如：codemap"
              disabled={busy}
              className="rounded-[1rem] border-[rgba(153,191,201,0.18)] bg-[rgba(255,255,255,0.94)]"
            />
          </Field>

	          <Field label="描述">
	            <Textarea
              value={draft.description}
              onChange={(event) => onChange({ ...draft, description: event.target.value })}
              disabled={busy}
              className="min-h-[96px] rounded-[1.3rem] border-[rgba(153,191,201,0.18)] bg-[rgba(255,255,255,0.94)]"
	            />
	          </Field>

	          <div className="grid gap-4 md:grid-cols-2">
	            <Field label="版本">
	              <Input
	                value={draft.version ?? ""}
	                onChange={(event) => onChange({ ...draft, version: event.target.value })}
	                placeholder="例如：1.0.0"
	                disabled={busy}
	                className="rounded-[1rem] border-[rgba(153,191,201,0.18)] bg-[rgba(255,255,255,0.94)]"
	              />
	            </Field>
	            <Field label="安装来源">
	              <Input
	                value={draft.installSource ?? ""}
	                onChange={(event) => onChange({ ...draft, installSource: event.target.value })}
	                placeholder="例如：local / marketplace / imported"
	                disabled={busy}
	                className="rounded-[1rem] border-[rgba(153,191,201,0.18)] bg-[rgba(255,255,255,0.94)]"
	              />
	            </Field>
	          </div>

	          <Field label="来源链接">
	            <Input
	              value={draft.sourceUrl ?? ""}
	              onChange={(event) => onChange({ ...draft, sourceUrl: event.target.value })}
	              placeholder="https://..."
	              disabled={busy}
	              className="rounded-[1rem] border-[rgba(153,191,201,0.18)] bg-[rgba(255,255,255,0.94)]"
	            />
	          </Field>

	          <div className="grid gap-4 md:grid-cols-2">
	            <Field label="组合技能">
	              <Textarea
	                value={joinListValues(draft.composedOf)}
	                onChange={(event) => onChange({ ...draft, composedOf: parseListInput(event.target.value) })}
	                placeholder={"例如：\ngo-agent\nobservability-deep-dive"}
	                disabled={busy}
	                className="min-h-[96px] rounded-[1.3rem] border-[rgba(153,191,201,0.18)] bg-[rgba(255,255,255,0.94)]"
	              />
	            </Field>
	            <Field label="可选 Focus">
	              <Textarea
	                value={joinListValues(draft.focusAreas)}
	                onChange={(event) => onChange({ ...draft, focusAreas: parseListInput(event.target.value) })}
	                placeholder={"例如：\nobservability\nreliability\nownership"}
	                disabled={busy}
	                className="min-h-[96px] rounded-[1.3rem] border-[rgba(153,191,201,0.18)] bg-[rgba(255,255,255,0.94)]"
	              />
	            </Field>
	          </div>

	          <div className="grid gap-4 md:grid-cols-2">
	            <Field label="能力边界">
	              <Textarea
	                value={joinListValues(draft.capabilityBoundaries)}
	                onChange={(event) => onChange({ ...draft, capabilityBoundaries: parseListInput(event.target.value) })}
	                placeholder={"例如：\n只负责并发与可观测性\n不覆盖前端系统设计"}
	                disabled={busy}
	                className="min-h-[96px] rounded-[1.3rem] border-[rgba(153,191,201,0.18)] bg-[rgba(255,255,255,0.94)]"
	              />
	            </Field>
	            <Field label="评分信息">
	              <div className="grid gap-3 sm:grid-cols-2">
	                <Input
	                  type="number"
	                  min="0"
	                  max="5"
	                  step="0.1"
	                  value={draft.rating ?? 0}
	                  onChange={(event) => onChange({ ...draft, rating: Number(event.target.value) || 0 })}
	                  disabled={busy}
	                  className="rounded-[1rem] border-[rgba(153,191,201,0.18)] bg-[rgba(255,255,255,0.94)]"
	                />
	                <Input
	                  type="number"
	                  min="0"
	                  step="1"
	                  value={draft.ratingCount ?? 0}
	                  onChange={(event) => onChange({ ...draft, ratingCount: Number(event.target.value) || 0 })}
	                  disabled={busy}
	                  className="rounded-[1rem] border-[rgba(153,191,201,0.18)] bg-[rgba(255,255,255,0.94)]"
	                />
	              </div>
	              <p className="mt-2 text-xs leading-6 text-[rgba(115,137,161,0.72)]">左侧为评分，右侧为评分人数，先作为市场化字段预留。</p>
	            </Field>
	          </div>

	          <Field label="指令">
	            <Textarea
              value={draft.content}
              onChange={(event) => onChange({ ...draft, content: event.target.value })}
              placeholder={"# 使用场景\n# 输出解释\n# 示例"}
              disabled={busy}
              className="min-h-[280px] rounded-[1.3rem] border-[rgba(153,191,201,0.18)] bg-[rgba(255,255,255,0.94)]"
            />
          </Field>

          <div className="flex items-center justify-end gap-3 pt-2">
            <Button
              type="button"
              variant="outline"
              disabled={busy}
              onClick={onClose}
              className="h-11 rounded-full border-[rgba(153,191,201,0.18)] bg-transparent px-5 text-[rgb(72,91,114)]"
            >
              取消
            </Button>
            <Button
              type="submit"
              disabled={busy}
              className="h-11 rounded-full bg-[rgb(0,102,255)] px-5 text-primary-foreground hover:bg-[rgb(0,88,220)]"
            >
              {busy ? <LoaderCircle className="h-4 w-4 animate-spin" /> : mode === "create" ? "创建" : "保存"}
            </Button>
          </div>
        </form>
      </div>
    </div>
  );
}

function ArtifactPanel({
  artifacts,
  canUpload,
  isBusy,
  embedded = false,
  onCreate,
  onEdit,
  onDelete,
  onUpload,
}: {
  artifacts: Artifact[];
  canUpload: boolean;
  isBusy: boolean;
  embedded?: boolean;
  onCreate: () => void;
  onEdit: (artifact: Artifact) => void;
  onDelete: (artifact: Artifact) => void;
  onUpload: (event: ChangeEvent<HTMLInputElement>) => void;
}) {
  return (
    <section className={embedded ? "" : "paper-panel editorial-frame rounded-[2rem] border border-[rgba(153,191,201,0.18)] p-5 shadow-paper"}>
      <div className="flex items-center justify-between gap-3">
        <div>
          <p className="tech-label text-[0.68rem] text-[rgba(97,123,150,0.62)]">{embedded ? "文件管理" : "材料"}</p>
          <h2 className={`font-display text-[rgb(72,91,114)] ${embedded ? "mt-1 text-xl" : "mt-2 text-2xl"}`}>工作区文件</h2>
        </div>
        <div className="flex items-center gap-2">
          <label
            className={`inline-flex h-11 items-center rounded-full px-4 text-sm ${canUpload ? "cursor-pointer bg-[rgb(74,156,175)] text-primary-foreground" : "cursor-default bg-[rgba(146,190,200,0.12)] text-[rgba(96,134,145,0.48)]"}`}
          >
            <Upload className="mr-2 h-4 w-4" />
            上传
            <input type="file" className="hidden" disabled={!canUpload || isBusy} onChange={onUpload} />
          </label>
          <Button
            type="button"
            disabled={!canUpload || isBusy}
            onClick={onCreate}
            className="h-11 rounded-full bg-[rgba(118,177,189,0.1)] px-4 text-[rgb(36,95,110)] hover:bg-[rgba(118,177,189,0.16)]"
          >
            <FilePenLine className="mr-2 h-4 w-4" />
            新建
          </Button>
        </div>
      </div>

      <div className={`${embedded ? "mt-4" : "mt-5"} space-y-3`}>
        {artifacts.length === 0 ? (
          <p className="rounded-[1.3rem] border border-dashed border-[rgba(153,191,201,0.18)] px-4 py-5 text-sm leading-7 text-[rgba(115,137,161,0.78)]">
            将简历、岗位描述或补充说明上传到当前工作区，也可以直接新建一份文本材料。
          </p>
        ) : (
          artifacts.map((artifact) => {
            const editable = isEditableTextArtifact(artifact);

            return (
              <div
                key={artifact.id}
                className="flex items-center justify-between gap-3 rounded-[1.3rem] border border-[rgba(153,191,201,0.16)] bg-[rgba(252,255,255,0.88)] px-4 py-4"
              >
                <button type="button" onClick={() => onEdit(artifact)} className="min-w-0 flex-1 text-left">
                  <p className="truncate text-sm font-medium text-[rgb(72,91,114)]">{artifact.name}</p>
                  <p className="mt-1 text-xs uppercase tracking-[0.22em] text-[rgba(122,144,168,0.74)]">
                    {formatBytes(artifact.size)} · {new Date(artifact.createdAt).toLocaleDateString()} · {editable ? "可编辑" : "下载查看"}
                  </p>
                </button>
                <div className="flex items-center gap-2">
                  {editable ? (
                    <Button
                      type="button"
                      variant="outline"
                      onClick={() => onEdit(artifact)}
                      className="h-9 rounded-full border-[rgba(153,191,201,0.18)] bg-transparent px-3 text-[rgb(36,95,110)]"
                    >
                      <FilePenLine className="mr-2 h-4 w-4" />
                      编辑
                    </Button>
                  ) : null}
                  <Button
                    type="button"
                    variant="outline"
                    onClick={() => onDelete(artifact)}
                    disabled={isBusy}
                    className="h-9 rounded-full border-[rgba(153,191,201,0.18)] bg-[rgba(236,248,250,0.92)] px-3 text-[rgb(63,118,131)] hover:bg-[rgba(228,244,247,0.96)]"
                  >
                    删除
                  </Button>
                  <a
                    href={`/api/files/${artifact.id}?download=1`}
                    className="inline-flex h-9 items-center rounded-full border border-[rgba(153,191,201,0.18)] px-3 text-sm text-[rgb(36,95,110)]"
                  >
                    <Download className="mr-2 h-4 w-4" />
                    下载
                  </a>
                </div>
              </div>
            );
          })
        )}
      </div>
    </section>
  );
}

function ArtifactEditorModal({
  draft,
  busy,
  mode,
  onChange,
  onClose,
  onSubmit,
}: {
  draft: ArtifactEditorDraft;
  busy: boolean;
  mode: "create" | "edit";
  onChange: (value: ArtifactEditorDraft) => void;
  onClose: () => void;
  onSubmit: (event: React.FormEvent<HTMLFormElement>) => void;
}) {
  const inferredContentType = inferArtifactContentType(draft.name, draft.contentType);

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-[rgba(228,241,248,0.48)] px-4 py-6 backdrop-blur-sm">
      <div className="w-full max-w-3xl rounded-[2rem] border border-[rgba(153,191,201,0.18)] bg-[rgb(250,254,255)] p-6 shadow-[0_30px_80px_rgba(111,177,190,0.18)]">
        <div className="flex items-start justify-between gap-4">
          <div>
            <p className="tech-label text-[0.68rem] text-[rgba(97,123,150,0.62)]">材料编辑器</p>
            <h2 className="mt-2 font-display text-3xl text-[rgb(72,91,114)]">{mode === "create" ? "创建材料" : "编辑材料"}</h2>
          </div>
          <button
            type="button"
            onClick={onClose}
            disabled={busy}
            className="inline-flex h-10 w-10 items-center justify-center rounded-full border border-[rgba(153,191,201,0.16)] text-[rgba(115,137,161,0.78)] transition-colors hover:bg-[rgba(118,177,189,0.08)]"
          >
            <X className="h-4 w-4" />
          </button>
        </div>

        <form onSubmit={onSubmit} className="mt-6 space-y-4">
          <div className="rounded-[1.6rem] border border-dashed border-[rgba(153,191,201,0.18)] bg-[rgba(252,255,255,0.9)] px-5 py-5 text-sm leading-7 text-[rgba(115,137,161,0.78)]">
            适合直接整理岗位描述、面试补充说明、评审标准或候选人简历摘要。二进制文件仍然通过上传方式管理。
          </div>

          <div className="grid gap-4 md:grid-cols-[1.3fr_0.7fr]">
            <Field label="材料名称">
              <Input
                value={draft.name}
                onChange={(event) => onChange({ ...draft, name: event.target.value })}
                placeholder="例如：backend-job-description.md"
                disabled={busy}
                className="rounded-[1rem] border-[rgba(153,191,201,0.18)] bg-[rgba(255,255,255,0.94)]"
              />
            </Field>

            <Field label="内容类型">
              <Input
                value={draft.contentType}
                onChange={(event) => onChange({ ...draft, contentType: event.target.value })}
                placeholder="text/markdown"
                disabled={busy}
                className="rounded-[1rem] border-[rgba(153,191,201,0.18)] bg-[rgba(255,255,255,0.94)]"
              />
            </Field>
          </div>

          <div className="rounded-[1.3rem] border border-[rgba(153,191,201,0.16)] bg-[rgba(252,255,255,0.9)] px-4 py-4 text-sm leading-7 text-[rgba(115,137,161,0.78)]">
            <p className="tech-label text-[0.64rem] text-[rgba(97,123,150,0.62)]">预览</p>
            <p className="mt-2 text-[rgb(72,91,114)]">{typeof draft.name === "string" && draft.name.trim() ? draft.name.trim() : "untitled.md"}</p>
            <p className="text-xs uppercase tracking-[0.22em] text-[rgba(122,144,168,0.74)]">{inferredContentType}</p>
          </div>

          <Field label="正文">
            <Textarea
              value={draft.content}
              onChange={(event) => onChange({ ...draft, content: event.target.value })}
              placeholder={"# 岗位背景\n\n# 候选人简历要点\n\n# 补充说明"}
              disabled={busy}
              className="min-h-[320px] rounded-[1.3rem] border-[rgba(153,191,201,0.18)] bg-[rgba(255,255,255,0.94)]"
            />
          </Field>

          <div className="flex items-center justify-end gap-3 pt-2">
            <Button
              type="button"
              variant="outline"
              disabled={busy}
              onClick={onClose}
              className="h-11 rounded-full border-[rgba(153,191,201,0.18)] bg-transparent px-5 text-[rgb(72,91,114)]"
            >
              取消
            </Button>
            <Button
              type="submit"
              disabled={busy}
              className="h-11 rounded-full bg-[rgb(0,102,255)] px-5 text-primary-foreground hover:bg-[rgb(0,88,220)]"
            >
              {busy ? <LoaderCircle className="h-4 w-4 animate-spin" /> : mode === "create" ? "创建" : "保存"}
            </Button>
          </div>
        </form>
      </div>
    </div>
  );
}

function getEventMeta(event: RunEvent) {
  switch (event.type) {
    case "persona.selected": {
      const payload = (event.payload ?? null) as { persona?: InterviewPersona } | null;
      return {
        label: "人格已选定",
        description: `本轮面试官人格：${getPersonaMeta(payload?.persona).label}`,
        icon: MessageSquareQuote,
        iconClass: "bg-[rgba(219,234,254,0.96)] text-[rgb(37,99,235)]",
        frameClass: "border-[rgba(191,219,254,0.92)] bg-[rgba(239,246,255,0.96)]",
      };
    }
    case "clarify.requested":
      return {
        label: "请求澄清",
        description: "运行已暂停，正在等待用户补充信息。",
        icon: LockKeyhole,
        iconClass: "bg-[rgba(118,177,189,0.14)] text-[rgb(58,124,139)]",
        frameClass: "border-[rgba(153,191,201,0.18)] bg-[rgba(240,249,251,0.92)]",
      };
    case "clarify.resumed":
      return {
        label: "澄清恢复",
        description: "已接收用户输入，恢复流程开始执行。",
        icon: RotateCcw,
        iconClass: "bg-[rgba(111,177,190,0.14)] text-[rgb(58,124,139)]",
        frameClass: "border-[rgba(153,191,201,0.18)] bg-[rgba(240,249,251,0.92)]",
      };
    case "checkpoint.saved":
      return {
        label: "已保存检查点",
        description: "短期恢复状态已持久化。",
        icon: CircleDashed,
        iconClass: "bg-[rgba(68,85,107,0.14)] text-[rgb(68,85,107)]",
        frameClass: "border-[rgba(68,85,107,0.18)] bg-[rgba(244,247,251,0.88)]",
      };
    case "checkpoint.loaded":
      return {
        label: "已加载检查点",
        description: "已从持久化状态重建运行上下文。",
        icon: RotateCcw,
        iconClass: "bg-[rgba(74,99,151,0.14)] text-[rgb(74,99,151)]",
        frameClass: "border-[rgba(74,99,151,0.18)] bg-[rgba(241,245,255,0.9)]",
      };
    case "tool.called": {
      const payload = (event.payload ?? null) as { tool?: string; query?: string } | null;
      return {
        label: "工具调用开始",
        description: payload?.tool === "web.search" ? `web.search：${payload.query ?? "查询"}` : payload?.tool ?? "运行时已开始调用工具。",
        icon: CircleDashed,
        iconClass: "bg-[rgba(68,85,107,0.14)] text-[rgb(68,85,107)]",
        frameClass: "border-[rgba(68,85,107,0.18)] bg-[rgba(244,247,251,0.88)]",
      };
    }
    case "tool.completed": {
      const payload = (event.payload ?? null) as { tool?: string; status?: string; count?: number } | null;
      const isError = payload?.status === "error";
      return {
        label: isError ? "工具调用失败" : "工具调用完成",
        description:
          payload?.tool === "web.search"
            ? `web.search 返回 ${payload?.count ?? 0} 条结果`
            : payload?.tool ?? "运行时工具调用已结束。",
        icon: isError ? Flame : Waypoints,
        iconClass: isError
          ? "bg-[rgba(0,102,255,0.14)] text-[rgb(0,102,255)]"
          : "bg-[rgba(111,177,190,0.14)] text-[rgb(58,124,139)]",
        frameClass: isError
          ? "border-[rgba(153,191,201,0.18)] bg-[rgba(233,244,255,0.94)]"
          : "border-[rgba(153,191,201,0.18)] bg-[rgba(240,249,251,0.92)]",
      };
    }
    case "plan.generated":
      return {
        label: "计划已生成",
        description: "控制面已为本次运行生成执行计划。",
        icon: Radar,
        iconClass: "bg-[rgba(74,99,151,0.14)] text-[rgb(74,99,151)]",
        frameClass: "border-[rgba(74,99,151,0.18)] bg-[rgba(241,245,255,0.9)]",
      };
    case "interview_tree.generated": {
      const payload = (event.payload ?? null) as InterviewTraceTree | null;
      return {
        label: "追问树已更新",
        description: `当前已累计 ${payload?.questionCount ?? 0} 个问题节点。`,
        icon: Waypoints,
        iconClass: "bg-[rgba(219,234,254,0.96)] text-[rgb(37,99,235)]",
        frameClass: "border-[rgba(191,219,254,0.92)] bg-[rgba(239,246,255,0.96)]",
      };
    }
    case "profile.updated": {
      const payload = (event.payload ?? null) as CandidateProfile | null;
      return {
        label: "画像已更新",
        description: `累计面试 ${payload?.interviewCount ?? 0} 场，画像已重新归纳。`,
        icon: Radar,
        iconClass: "bg-[rgba(241,245,249,0.96)] text-[rgb(71,85,105)]",
        frameClass: "border-[rgba(203,213,225,0.92)] bg-[rgba(248,250,252,0.96)]",
      };
    }
    case "copilot.feedback": {
      const payload = (event.payload ?? null) as CopilotFeedback | null;
      return {
        label: "Copilot 反馈",
        description: payload?.summary?.trim() || "当前答题状态已重新评估。",
        icon: Sparkles,
        iconClass: "bg-[rgba(254,249,195,0.96)] text-[rgb(161,98,7)]",
        frameClass: "border-[rgba(253,224,71,0.42)] bg-[rgba(254,252,232,0.96)]",
      };
    }
    case "copilot.hint": {
      const payload = (event.payload ?? null) as CopilotHint | null;
      return {
        label: "Copilot 提示",
        description: payload?.title?.trim() || "已生成新的答题提示。",
        icon: Sparkles,
        iconClass: "bg-[rgba(254,249,195,0.96)] text-[rgb(161,98,7)]",
        frameClass: "border-[rgba(253,224,71,0.42)] bg-[rgba(255,251,235,0.96)]",
      };
    }
    case "decision.generated": {
      const payload = (event.payload ?? null) as { decision?: { explanation?: string } | null } | null;
      return {
        label: "决策已更新",
        description: payload?.decision?.explanation?.trim() || "下一轮追问策略已经重新计算。",
        icon: Waypoints,
        iconClass: "bg-[rgba(219,234,254,0.96)] text-[rgb(37,99,235)]",
        frameClass: "border-[rgba(191,219,254,0.92)] bg-[rgba(239,246,255,0.96)]",
      };
    }
    case "review.generated":
      return {
        label: "复盘已聚合",
        description: "评分、追问链路与画像快照已聚合完成。",
        icon: Radar,
        iconClass: "bg-[rgba(219,234,254,0.96)] text-[rgb(37,99,235)]",
        frameClass: "border-[rgba(191,219,254,0.92)] bg-[rgba(239,246,255,0.96)]",
      };
    case "trace.span": {
      const payload = (event.payload ?? null) as { scope?: string; name?: string; phase?: string; status?: string } | null;
      return {
        label: "链路追踪",
        description: `${payload?.scope ?? "runtime"}:${payload?.name ?? "unknown"} ${payload?.phase ?? "tick"} ${payload?.status ?? ""}`.trim(),
        icon: Radar,
        iconClass: "bg-[rgba(118,177,189,0.12)] text-[rgba(96,134,145,0.8)]",
        frameClass: "border-[rgba(153,191,201,0.16)] bg-[rgba(252,255,255,0.88)]",
      };
    }
    case "run.failed":
      return {
        label: "运行失败",
        description: "执行到达错误终点。",
        icon: Flame,
        iconClass: "bg-[rgba(0,102,255,0.14)] text-[rgb(0,102,255)]",
        frameClass: "border-[rgba(153,191,201,0.18)] bg-[rgba(233,244,255,0.94)]",
      };
    case "run.cancelled":
      return {
        label: "运行已取消",
        description: "执行已被主动停止。",
        icon: Flame,
        iconClass: "bg-[rgba(0,102,255,0.14)] text-[rgb(0,102,255)]",
        frameClass: "border-[rgba(153,191,201,0.18)] bg-[rgba(233,244,255,0.94)]",
      };
    case "run.completed":
      return {
        label: "运行完成",
        description: "执行已成功结束。",
        icon: Waypoints,
        iconClass: "bg-[rgba(111,177,190,0.14)] text-[rgb(58,124,139)]",
        frameClass: "border-[rgba(153,191,201,0.18)] bg-[rgba(240,249,251,0.92)]",
      };
    default:
      return {
        label: event.type,
        description: "结构化运行时遥测事件。",
        icon: Radar,
        iconClass: "bg-[rgba(118,177,189,0.12)] text-[rgba(96,134,145,0.8)]",
        frameClass: "border-[rgba(153,191,201,0.16)] bg-[rgba(252,255,255,0.88)]",
      };
  }
}

function getPersonaMeta(persona?: InterviewPersona | null) {
  const value = persona ?? "rigorous";
  return personaOptions.find((option) => option.value === value) ?? personaOptions[0];
}

function formatTraceNodeKind(kind: string) {
  switch (kind) {
    case "opening":
      return "首问";
    case "topic_shift":
      return "切题";
    default:
      return "追问";
  }
}

function formatTraceSignal(signal: string) {
  switch (signal) {
    case "strong":
      return "回答扎实";
    case "weak":
      return "需要深挖";
    case "unclear":
      return "表达模糊";
    default:
      return "待判断";
  }
}

function formatInterviewModeLabel(mode?: InterviewMode | null) {
  switch (mode) {
    case "standard":
      return "标准面试";
    case "stress":
      return "压力面试";
    case "weakness_focused":
      return "查漏补缺";
    case "system_design":
      return "系统设计专项";
    case "resume_deep_dive":
      return "简历深挖";
    default:
      return "标准面试";
  }
}

function formatInterviewPhaseLabel(phase?: string | null) {
  switch (phase) {
    case "warmup":
      return "热身";
    case "probe":
      return "深挖";
    case "adversarial":
      return "反驳";
    case "stress":
      return "压力";
    case "wrapup":
      return "收尾";
    default:
      return "阶段";
  }
}

function formatDecisionReasonLabel(reason?: DecisionReason | string | null) {
  switch (reason) {
    case "missing_tradeoff":
      return "缺少 tradeoff";
    case "lack_implementation_detail":
      return "实现细节不足";
    case "weak_signal_timeout":
      return "超时控制偏弱";
    case "weak_signal_observability":
      return "可观测性偏弱";
    case "pressure_test":
      return "进入压力测试";
    case "topic_switch":
      return "切换主题";
    case "confidence_confirm":
      return "确认强项";
    case "wrapup_due_to_budget":
      return "预算到达收尾";
    case "profile_weakness_focus":
      return "围绕历史弱项";
    default:
      return "继续追问";
  }
}

function conversationDecisionTone(reason?: DecisionReason | string | null): "neutral" | "info" | "warning" | "success" {
  switch (reason) {
    case "pressure_test":
    case "profile_weakness_focus":
    case "missing_tradeoff":
    case "lack_implementation_detail":
    case "weak_signal_timeout":
    case "weak_signal_observability":
    case "wrapup_due_to_budget":
      return "warning";
    case "confidence_confirm":
      return "success";
    case "topic_switch":
      return "info";
    default:
      return "neutral";
  }
}

function formatSignalLabel(signal?: string | null) {
  switch (signal) {
    case "too_generic":
      return "回答偏泛";
    case "missing_tradeoff":
      return "缺少 tradeoff";
    case "missing_implementation_detail":
      return "缺少实现细节";
    case "missing_timeout_detail":
      return "缺少超时细节";
    case "missing_observability_detail":
      return "缺少可观测性细节";
    case "avoids_core_question":
      return "回避核心问题";
    case "partial_answer":
      return "回答不完整";
    case "concept_without_plan":
      return "只讲概念没给方案";
    case "lacks_example_or_evidence":
      return "缺少例子或证据";
    case "tradeoff_reasoning":
      return "tradeoff 分析";
    case "implementation_detail":
      return "实现细节";
    case "timeout_control":
      return "超时控制";
    case "observability":
      return "可观测性";
    default:
      return signal ?? "未标记";
  }
}

function traceSignalClass(signal: string) {
  switch (signal) {
    case "strong":
      return "bg-[rgba(219,234,254,0.96)] text-[rgb(37,99,235)]";
    case "weak":
      return "bg-[rgba(239,246,255,0.96)] text-[rgb(29,78,216)]";
    case "unclear":
      return "bg-[rgba(241,245,249,0.96)] text-[rgba(71,85,105,0.86)]";
    default:
      return "bg-[rgba(241,245,249,0.96)] text-[rgba(100,116,139,0.84)]";
  }
}

function tracePhaseTone(phase?: InterviewPhase | null): {
  frame: string;
  rail: string;
  badge: string;
  svgFill: string;
  svgStroke: string;
  svgRail: string;
} {
  switch (phase) {
    case "warmup":
      return {
        frame:
          "border-[rgba(191,219,254,0.88)] bg-[linear-gradient(180deg,rgba(239,246,255,0.98),rgba(248,250,252,0.98))] hover:border-[rgba(147,197,253,0.96)]",
        rail: "bg-[linear-gradient(180deg,rgba(59,130,246,0.78),rgba(147,197,253,0.52))]",
        badge: "bg-[rgba(219,234,254,0.96)] text-[rgb(37,99,235)]",
        svgFill: "rgb(239,246,255)",
        svgStroke: "rgb(147,197,253)",
        svgRail: "rgb(59,130,246)",
      };
    case "probe":
      return {
        frame:
          "border-[rgba(167,243,208,0.9)] bg-[linear-gradient(180deg,rgba(236,253,245,0.98),rgba(248,250,252,0.98))] hover:border-[rgba(110,231,183,0.94)]",
        rail: "bg-[linear-gradient(180deg,rgba(16,185,129,0.82),rgba(110,231,183,0.52))]",
        badge: "bg-[rgba(209,250,229,0.96)] text-[rgb(5,150,105)]",
        svgFill: "rgb(236,253,245)",
        svgStroke: "rgb(110,231,183)",
        svgRail: "rgb(16,185,129)",
      };
    case "adversarial":
      return {
        frame:
          "border-[rgba(253,186,116,0.88)] bg-[linear-gradient(180deg,rgba(255,247,237,0.98),rgba(255,251,235,0.98))] hover:border-[rgba(251,146,60,0.92)]",
        rail: "bg-[linear-gradient(180deg,rgba(249,115,22,0.84),rgba(253,186,116,0.56))]",
        badge: "bg-[rgba(255,237,213,0.96)] text-[rgb(194,65,12)]",
        svgFill: "rgb(255,247,237)",
        svgStroke: "rgb(251,146,60)",
        svgRail: "rgb(249,115,22)",
      };
    case "stress":
      return {
        frame:
          "border-[rgba(252,165,165,0.9)] bg-[linear-gradient(180deg,rgba(254,242,242,0.98),rgba(255,251,235,0.98))] hover:border-[rgba(248,113,113,0.94)]",
        rail: "bg-[linear-gradient(180deg,rgba(239,68,68,0.84),rgba(252,165,165,0.56))]",
        badge: "bg-[rgba(254,226,226,0.96)] text-[rgb(185,28,28)]",
        svgFill: "rgb(254,242,242)",
        svgStroke: "rgb(248,113,113)",
        svgRail: "rgb(239,68,68)",
      };
    case "wrapup":
      return {
        frame:
          "border-[rgba(216,180,254,0.88)] bg-[linear-gradient(180deg,rgba(250,245,255,0.98),rgba(248,250,252,0.98))] hover:border-[rgba(192,132,252,0.92)]",
        rail: "bg-[linear-gradient(180deg,rgba(168,85,247,0.82),rgba(216,180,254,0.56))]",
        badge: "bg-[rgba(243,232,255,0.96)] text-[rgb(126,34,206)]",
        svgFill: "rgb(250,245,255)",
        svgStroke: "rgb(192,132,252)",
        svgRail: "rgb(168,85,247)",
      };
    default:
      return {
        frame:
          "border-[rgba(203,213,225,0.82)] bg-[rgba(252,255,255,0.94)] hover:border-[rgba(59,130,246,0.18)] hover:bg-[rgba(248,250,255,0.98)]",
        rail: "bg-[linear-gradient(180deg,rgba(148,163,184,0.72),rgba(203,213,225,0.42))]",
        badge: "bg-[rgba(241,245,249,0.96)] text-[rgba(71,85,105,0.86)]",
        svgFill: "rgb(252,255,255)",
        svgStroke: "rgb(203,213,225)",
        svgRail: "rgb(148,163,184)",
      };
  }
}

function formatBytes(size: number): string {
  if (size < 1024) {
    return `${size} B`;
  }
  if (size < 1024 * 1024) {
    return `${(size / 1024).toFixed(1)} KB`;
  }
  return `${(size / (1024 * 1024)).toFixed(1)} MB`;
}

function isEditableTextArtifact(artifact: Artifact): boolean {
  const contentType = artifact.contentType.trim().toLowerCase();
  if (contentType.startsWith("text/")) {
    return true;
  }

  if (["application/json", "application/ld+json", "application/xml", "application/yaml", "application/x-yaml"].includes(contentType)) {
    return true;
  }

  return [".md", ".txt", ".json", ".yaml", ".yml", ".xml", ".csv", ".tsv"].some((ext) => artifact.name.toLowerCase().endsWith(ext));
}

function inferArtifactContentType(name: string, declared: string): string {
  const trimmed = declared.trim().toLowerCase();
  if (trimmed) {
    return trimmed;
  }

  const lowerName = name.trim().toLowerCase();
  if (lowerName.endsWith(".md")) return "text/markdown";
  if (lowerName.endsWith(".txt")) return "text/plain";
  if (lowerName.endsWith(".json")) return "application/json";
  if (lowerName.endsWith(".yaml") || lowerName.endsWith(".yml")) return "application/yaml";
  if (lowerName.endsWith(".xml")) return "application/xml";
  if (lowerName.endsWith(".csv")) return "text/csv";
  if (lowerName.endsWith(".tsv")) return "text/tab-separated-values";
  return "text/markdown";
}

function compactLine(value: string, limit: number): string {
  const trimmed = value.trim();
  if (!trimmed || trimmed.length <= limit) {
    return trimmed;
  }
  return `${trimmed.slice(0, Math.max(0, limit - 1)).trimEnd()}…`;
}

function buildMiniTraceForest(nodes: InterviewTraceNode[]): MiniTraceTreeNode[] {
  const sorted = [...nodes]
    .filter((node) => typeof node.round === "number" && node.question.trim())
    .sort((left, right) => {
      const roundDiff = (left.round ?? 0) - (right.round ?? 0);
      if (roundDiff !== 0) {
        return roundDiff;
      }
      return left.depth - right.depth;
    });

  const map = new Map<string, MiniTraceTreeNode>();
  for (const node of sorted) {
    map.set(node.id, { ...node, children: [] });
  }

  const roots: MiniTraceTreeNode[] = [];
  for (const node of sorted) {
    const current = map.get(node.id);
    if (!current) {
      continue;
    }
    const parent = node.parentId ? map.get(node.parentId) : null;
    if (parent) {
      parent.children.push(current);
      continue;
    }
    roots.push(current);
  }

  return roots;
}

function radarRingPoints(axisCount: number, center: number, radius: number): string[] {
  if (axisCount <= 0) {
    return [];
  }
  return Array.from({ length: axisCount }, (_, index) => {
    const angle = (Math.PI * 2 * index) / axisCount - Math.PI / 2;
    const x = center + Math.cos(angle) * radius;
    const y = center + Math.sin(angle) * radius;
    return `${x},${y}`;
  });
}

function radarPolygonPoints(
  points: Array<{ normalizedScore: number }>,
  center: number,
  radius: number,
): string[] {
  if (points.length === 0) {
    return [];
  }
  return points.map((point, index) => {
    const angle = (Math.PI * 2 * index) / points.length - Math.PI / 2;
    const distance = radius * (Math.max(0, Math.min(100, point.normalizedScore)) / 100);
    const x = center + Math.cos(angle) * distance;
    const y = center + Math.sin(angle) * distance;
    return `${x},${y}`;
  });
}

function summarizeTracePhases(trace: InterviewTraceTree | null): string {
  const phases = Array.from(
    new Set(
      (trace?.nodes ?? [])
        .map((node) => node.phase)
        .filter((phase): phase is InterviewPhase => Boolean(phase))
    )
  );
  if (phases.length === 0) {
    return "";
  }
  return phases.map((phase) => formatInterviewPhaseLabel(phase)).join(" -> ");
}

function formatRunStatus(status: string): string {
  switch (status) {
    case "created":
      return "已创建";
    case "running":
      return "运行中";
    case "waiting_clarify":
      return "等待澄清";
    case "resuming":
      return "恢复中";
    case "completed":
      return "已完成";
    case "failed":
      return "已失败";
    case "cancelled":
      return "已取消";
    default:
      return status;
  }
}

function formatConversationStatus(conversation: Conversation): string {
  if (conversation.archived || conversation.status === "archived") {
    return "已归档";
  }
  if (conversation.latestRunStatus) {
    return formatRunStatus(conversation.latestRunStatus);
  }
  switch (conversation.status) {
    case "active":
      return "待开始";
    case "archived":
      return "已归档";
    default:
      return conversation.status;
  }
}

function conversationStatusPriority(status?: string | null): number {
  switch (status) {
    case "waiting_clarify":
      return 0;
    case "running":
    case "resuming":
      return 1;
    case "failed":
    case "cancelled":
      return 2;
    case "completed":
      return 3;
    case "created":
      return 4;
    default:
      return 5;
  }
}

function compareConversationOrder(left: Conversation, right: Conversation): number {
  if (Boolean(left.pinned) !== Boolean(right.pinned)) {
    return left.pinned ? -1 : 1;
  }

  const statusOrder = conversationStatusPriority(left.latestRunStatus) - conversationStatusPriority(right.latestRunStatus);
  if (statusOrder !== 0) {
    return statusOrder;
  }

  const leftUpdatedAt = new Date(left.updatedAt).getTime();
  const rightUpdatedAt = new Date(right.updatedAt).getTime();
  if (leftUpdatedAt !== rightUpdatedAt) {
    return rightUpdatedAt - leftUpdatedAt;
  }

  return left.title.localeCompare(right.title, "zh-CN");
}

function parseListInput(value: string): string[] {
  return value
    .split(/\r?\n|,/)
    .map((item) => item.trim())
    .filter(Boolean);
}

function joinListValues(values?: string[]): string {
  return (values ?? []).join("\n");
}

function describeConversationStatus(conversation: Conversation) {
  if (conversation.archived || conversation.status === "archived") {
    return {
      label: "已归档",
      tone: "archived" as const,
      detail: "工作区已从主列表收起",
    };
  }

  switch (conversation.latestRunStatus) {
    case "running":
    case "resuming":
      return {
        label: formatRunStatus(conversation.latestRunStatus),
        tone: "running" as const,
        detail: "训练正在推进中",
      };
    case "waiting_clarify":
      return {
        label: formatRunStatus(conversation.latestRunStatus),
        tone: "waiting" as const,
        detail: "等待你的补充回答",
      };
    case "failed":
    case "cancelled":
      return {
        label: formatRunStatus(conversation.latestRunStatus),
        tone: "failed" as const,
        detail: "可回到会话继续恢复",
      };
    case "completed":
      return {
        label: formatRunStatus(conversation.latestRunStatus),
        tone: "done" as const,
        highlights: [{ label: "结果已入会话", tone: "success" as const }],
        detail: "已生成本场训练结果",
      };
    default:
      return {
        label: formatConversationStatus(conversation),
        tone: "idle" as const,
        detail: "等待开始下一场训练",
      };
  }
}
