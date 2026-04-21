import type {
  Artifact,
  ArtifactDocument,
  CopilotAssistResponse,
  Conversation,
  ConversationDetail,
  CandidateProfile,
  InterviewConfig,
  ModelConfig,
  ReviewSnapshot,
  Run,
  RunDetail,
  RunEvent,
  SkillDocument,
  SkillMetadata,
  Task,
} from "@/lib/types";

interface ErrorPayload {
  error?: string;
}

async function parseJSON<T>(response: Response, fallbackMessage: string): Promise<T> {
  if (!response.ok) {
    const error = (await response.json().catch(() => null)) as ErrorPayload | null;
    throw new Error(error?.error ?? fallbackMessage);
  }
  return (await response.json()) as T;
}

export async function listConversations(): Promise<Conversation[]> {
  const response = await fetch("/api/conversations");
  return parseJSON<Conversation[]>(response, "获取工作区列表失败");
}

export async function createConversation(title: string): Promise<Conversation> {
  const response = await fetch("/api/conversations", {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify({ title }),
  });
  return parseJSON<Conversation>(response, "创建工作区失败");
}

export async function getConversation(id: string): Promise<ConversationDetail> {
  const response = await fetch(`/api/conversations/${id}`);
  return parseJSON<ConversationDetail>(response, "加载工作区详情失败");
}

export async function getCandidateProfile(): Promise<CandidateProfile> {
  const response = await fetch("/api/profile");
  const payload = await parseJSON<{ profile: CandidateProfile }>(response, "加载候选人画像失败");
  return payload.profile;
}

export async function updateConversation(
  id: string,
  payload: { title?: string; pinned?: boolean; archived?: boolean }
): Promise<Conversation> {
  const response = await fetch(`/api/conversations/${id}`, {
    method: "PATCH",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify(payload),
  });
  return parseJSON<Conversation>(response, "更新工作区失败");
}

export async function deleteConversation(id: string): Promise<Conversation> {
  const response = await fetch(`/api/conversations/${id}`, {
    method: "DELETE",
  });
  return parseJSON<Conversation>(response, "删除工作区失败");
}

export async function createTask(payload: {
  conversationId: string;
  title: string;
  prompt: string;
  artifactIds?: string[];
  config: InterviewConfig;
  modelConfig: ModelConfig;
}): Promise<Task> {
  const response = await fetch("/api/tasks", {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify(payload),
  });
  return parseJSON<Task>(response, "创建任务失败");
}

export async function createRun(payload: { taskId: string; prompt?: string; artifactIds?: string[] }): Promise<Run> {
  const response = await fetch("/api/runs", {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify(payload),
  });
  return parseJSON<Run>(response, "创建运行失败");
}

export async function resumeRun(
  runId: string,
  payload: { message: string; config: InterviewConfig; artifactIds: string[] }
): Promise<Run> {
  const response = await fetch(`/api/runs/${runId}/resume`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify(payload),
  });
  return parseJSON<Run>(response, "恢复运行失败");
}

export async function cancelRun(runId: string): Promise<Run> {
  const response = await fetch(`/api/runs/${runId}/cancel`, {
    method: "POST",
  });
  return parseJSON<Run>(response, "取消运行失败");
}

export async function requestCopilotHint(runId: string): Promise<CopilotAssistResponse> {
  const response = await fetch(`/api/runs/${runId}/copilot`, {
    method: "POST",
  });
  return parseJSON<CopilotAssistResponse>(response, "获取 Copilot 提示失败");
}

export async function getRun(runId: string): Promise<RunDetail> {
  const response = await fetch(`/api/runs/${runId}`);
  return parseJSON<RunDetail>(response, "加载运行详情失败");
}

export async function getRunReview(runId: string): Promise<ReviewSnapshot> {
  const response = await fetch(`/api/runs/${runId}/review`);
  const payload = await parseJSON<{ review: ReviewSnapshot }>(response, "加载复盘快照失败");
  return payload.review;
}

export async function listArtifacts(conversationId: string): Promise<Artifact[]> {
  const response = await fetch(`/api/files?conversationId=${encodeURIComponent(conversationId)}`);
  const payload = await parseJSON<{ artifacts: Artifact[] }>(response, "获取材料列表失败");
  return payload.artifacts;
}

export async function listSkills(): Promise<SkillMetadata[]> {
  const response = await fetch("/api/skills");
  const payload = await parseJSON<{ skills: SkillMetadata[] }>(response, "获取技能列表失败");
  return payload.skills;
}

export async function getSkill(name: string): Promise<SkillDocument> {
  const response = await fetch(`/api/skills/${encodeURIComponent(name)}`);
  const payload = await parseJSON<{ skill: SkillDocument }>(response, "获取技能详情失败");
  return payload.skill;
}

export async function createSkill(payload: SkillDocument): Promise<SkillMetadata> {
  const response = await fetch("/api/skills", {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify(payload),
  });
  return parseJSON<SkillMetadata>(response, "创建技能失败");
}

export async function updateSkill(name: string, payload: SkillDocument): Promise<SkillMetadata> {
  const response = await fetch(`/api/skills/${encodeURIComponent(name)}`, {
    method: "PUT",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify(payload),
  });
  return parseJSON<SkillMetadata>(response, "更新技能失败");
}

export async function uploadSkill(file: File): Promise<SkillMetadata> {
  const form = new FormData();
  form.append("file", file);
  const response = await fetch("/api/skills", {
    method: "POST",
    body: form,
  });
  return parseJSON<SkillMetadata>(response, "上传技能失败");
}

export async function uploadArtifact(payload: {
  conversationId: string;
  taskId?: string;
  runId?: string;
  file: File;
}): Promise<Artifact> {
  const form = new FormData();
  form.append("conversationId", payload.conversationId);
  if (payload.taskId) {
    form.append("taskId", payload.taskId);
  }
  if (payload.runId) {
    form.append("runId", payload.runId);
  }
  form.append("file", payload.file);

  const response = await fetch("/api/files", {
    method: "POST",
    body: form,
  });
  return parseJSON<Artifact>(response, "上传材料失败");
}

export async function getArtifactContent(id: string): Promise<ArtifactDocument> {
  const response = await fetch(`/api/files/${encodeURIComponent(id)}?content=1`);
  return parseJSON<ArtifactDocument>(response, "加载材料内容失败");
}

export async function createTextArtifact(payload: {
  conversationId: string;
  taskId?: string;
  runId?: string;
  name: string;
  contentType?: string;
  content: string;
}): Promise<Artifact> {
  const response = await fetch("/api/files", {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify(payload),
  });
  return parseJSON<Artifact>(response, "创建材料失败");
}

export async function updateArtifact(payload: {
  id: string;
  taskId?: string;
  runId?: string;
  name: string;
  contentType?: string;
  content: string;
}): Promise<Artifact> {
  const response = await fetch(`/api/files/${encodeURIComponent(payload.id)}`, {
    method: "PUT",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify(payload),
  });
  return parseJSON<Artifact>(response, "更新材料失败");
}

export async function deleteArtifact(id: string): Promise<void> {
  const response = await fetch(`/api/files/${encodeURIComponent(id)}`, {
    method: "DELETE",
  });
  await parseJSON<{ status: string }>(response, "删除材料失败");
}

export function subscribeRunEvents(
  runId: string,
  handlers: {
    onEvent: (event: RunEvent) => void;
    onError?: () => void;
  }
): () => void {
  const source = new EventSource(`/api/runs/${runId}/events`);
  source.addEventListener("event", (message) => {
    try {
      const event = JSON.parse((message as MessageEvent<string>).data) as RunEvent;
      handlers.onEvent(event);
    } catch {
      handlers.onError?.();
    }
  });
  source.onerror = () => {
    handlers.onError?.();
  };
  return () => {
    source.close();
  };
}
