async function parseJSON(response, fallbackMessage) {
    if (!response.ok) {
        const error = (await response.json().catch(() => null));
        throw new Error(error?.error ?? fallbackMessage);
    }
    return (await response.json());
}
export async function listConversations() {
    const response = await fetch("/api/conversations");
    return parseJSON(response, "获取工作区列表失败");
}
export async function createConversation(title) {
    const response = await fetch("/api/conversations", {
        method: "POST",
        headers: {
            "Content-Type": "application/json",
        },
        body: JSON.stringify({ title }),
    });
    return parseJSON(response, "创建工作区失败");
}
export async function getConversation(id) {
    const response = await fetch(`/api/conversations/${id}`);
    return parseJSON(response, "加载工作区详情失败");
}
export async function getCandidateProfile() {
    const response = await fetch("/api/profile");
    const payload = await parseJSON(response, "加载候选人画像失败");
    return payload.profile;
}
export async function updateConversation(id, payload) {
    const response = await fetch(`/api/conversations/${id}`, {
        method: "PATCH",
        headers: {
            "Content-Type": "application/json",
        },
        body: JSON.stringify(payload),
    });
    return parseJSON(response, "更新工作区失败");
}
export async function deleteConversation(id) {
    const response = await fetch(`/api/conversations/${id}`, {
        method: "DELETE",
    });
    return parseJSON(response, "删除工作区失败");
}
export async function createTask(payload) {
    const response = await fetch("/api/tasks", {
        method: "POST",
        headers: {
            "Content-Type": "application/json",
        },
        body: JSON.stringify(payload),
    });
    return parseJSON(response, "创建任务失败");
}
export async function createRun(payload) {
    const response = await fetch("/api/runs", {
        method: "POST",
        headers: {
            "Content-Type": "application/json",
        },
        body: JSON.stringify(payload),
    });
    return parseJSON(response, "创建运行失败");
}
export async function resumeRun(runId, payload) {
    const response = await fetch(`/api/runs/${runId}/resume`, {
        method: "POST",
        headers: {
            "Content-Type": "application/json",
        },
        body: JSON.stringify(payload),
    });
    return parseJSON(response, "恢复运行失败");
}
export async function cancelRun(runId) {
    const response = await fetch(`/api/runs/${runId}/cancel`, {
        method: "POST",
    });
    return parseJSON(response, "取消运行失败");
}
export async function requestCopilotHint(runId) {
    const response = await fetch(`/api/runs/${runId}/copilot`, {
        method: "POST",
    });
    return parseJSON(response, "获取 Copilot 提示失败");
}
export async function getRun(runId) {
    const response = await fetch(`/api/runs/${runId}`);
    return parseJSON(response, "加载运行详情失败");
}
export async function getRunReview(runId) {
    const response = await fetch(`/api/runs/${runId}/review`);
    const payload = await parseJSON(response, "加载复盘快照失败");
    return payload.review;
}
export async function listArtifacts(conversationId) {
    const response = await fetch(`/api/files?conversationId=${encodeURIComponent(conversationId)}`);
    const payload = await parseJSON(response, "获取材料列表失败");
    return payload.artifacts;
}
export async function listSkills() {
    const response = await fetch("/api/skills");
    const payload = await parseJSON(response, "获取技能列表失败");
    return payload.skills;
}
export async function getSkill(name) {
    const response = await fetch(`/api/skills/${encodeURIComponent(name)}`);
    const payload = await parseJSON(response, "获取技能详情失败");
    return payload.skill;
}
export async function createSkill(payload) {
    const response = await fetch("/api/skills", {
        method: "POST",
        headers: {
            "Content-Type": "application/json",
        },
        body: JSON.stringify(payload),
    });
    return parseJSON(response, "创建技能失败");
}
export async function updateSkill(name, payload) {
    const response = await fetch(`/api/skills/${encodeURIComponent(name)}`, {
        method: "PUT",
        headers: {
            "Content-Type": "application/json",
        },
        body: JSON.stringify(payload),
    });
    return parseJSON(response, "更新技能失败");
}
export async function uploadSkill(file) {
    const form = new FormData();
    form.append("file", file);
    const response = await fetch("/api/skills", {
        method: "POST",
        body: form,
    });
    return parseJSON(response, "上传技能失败");
}
export async function uploadArtifact(payload) {
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
    return parseJSON(response, "上传材料失败");
}
export async function getArtifactContent(id) {
    const response = await fetch(`/api/files/${encodeURIComponent(id)}?content=1`);
    return parseJSON(response, "加载材料内容失败");
}
export async function createTextArtifact(payload) {
    const response = await fetch("/api/files", {
        method: "POST",
        headers: {
            "Content-Type": "application/json",
        },
        body: JSON.stringify(payload),
    });
    return parseJSON(response, "创建材料失败");
}
export async function updateArtifact(payload) {
    const response = await fetch(`/api/files/${encodeURIComponent(payload.id)}`, {
        method: "PUT",
        headers: {
            "Content-Type": "application/json",
        },
        body: JSON.stringify(payload),
    });
    return parseJSON(response, "更新材料失败");
}
export async function deleteArtifact(id) {
    const response = await fetch(`/api/files/${encodeURIComponent(id)}`, {
        method: "DELETE",
    });
    await parseJSON(response, "删除材料失败");
}
export function subscribeRunEvents(runId, handlers) {
    const source = new EventSource(`/api/runs/${runId}/events`);
    source.addEventListener("event", (message) => {
        try {
            const event = JSON.parse(message.data);
            handlers.onEvent(event);
        }
        catch {
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
