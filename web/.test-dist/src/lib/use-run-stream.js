import { startTransition, useEffect, useState } from "react";
import { getRun, subscribeRunEvents } from "./api.js";
function isTerminalRunStatus(status) {
    return status === "completed" || status === "failed" || status === "cancelled";
}
function mergeById(current, incoming) {
    const map = new Map();
    for (const item of current) {
        map.set(item.id, item);
    }
    for (const item of incoming) {
        map.set(item.id, item);
    }
    return Array.from(map.values());
}
function upsertStreamingAssistantMessage(current, runId, content) {
    const syntheticId = `streaming-${runId}`;
    const next = current.filter((message) => message.id !== syntheticId);
    if (!content.trim()) {
        return next;
    }
    return [
        ...next,
        {
            id: syntheticId,
            conversationId: "",
            taskId: "",
            runId,
            role: "assistant",
            content,
            createdAt: new Date().toISOString(),
        },
    ];
}
export function useRunStream(runId) {
    const [run, setRun] = useState(null);
    const [messages, setMessages] = useState([]);
    const [events, setEvents] = useState([]);
    const [isLoading, setIsLoading] = useState(false);
    const [error, setError] = useState("");
    function appendLocalMessage(message) {
        startTransition(() => {
            setMessages((current) => mergeById(current, [message]));
        });
    }
    function appendLocalEvents(incoming) {
        if (incoming.length === 0) {
            return;
        }
        startTransition(() => {
            setEvents((current) => mergeById(current, incoming));
        });
    }
    useEffect(() => {
        if (!runId) {
            setRun(null);
            setMessages([]);
            setEvents([]);
            setError("");
            return;
        }
        const activeRunID = runId;
        let cancelled = false;
        let subscribed = false;
        let unsubscribe = () => { };
        function closeSubscription() {
            if (!subscribed) {
                return;
            }
            unsubscribe();
            unsubscribe = () => { };
            subscribed = false;
        }
        function openSubscription() {
            if (subscribed || cancelled) {
                return;
            }
            unsubscribe = subscribeRunEvents(activeRunID, {
                onEvent: (event) => {
                    if (cancelled)
                        return;
                    startTransition(() => {
                        setEvents((current) => mergeById(current, [event]));
                        if (event.type === "message.delta") {
                            const payload = (event.payload ?? {});
                            setMessages((current) => upsertStreamingAssistantMessage(current, activeRunID, payload.content ?? ""));
                            return;
                        }
                        if (event.type === "message.completed") {
                            setMessages((current) => current.filter((message) => message.id !== `streaming-${activeRunID}`));
                        }
                    });
                    if (event.type === "clarify.requested" ||
                        event.type === "clarify.resumed" ||
                        event.type === "run.cancelled" ||
                        event.type === "run.completed" ||
                        event.type === "run.failed") {
                        void refresh({ silent: true });
                    }
                },
                onError: () => {
                    if (cancelled)
                        return;
                    void refreshAfterDisconnect();
                },
            });
            subscribed = true;
        }
        async function refresh(options) {
            if (!options?.silent) {
                setIsLoading(true);
            }
            try {
                const detail = await getRun(activeRunID);
                if (cancelled)
                    return;
                startTransition(() => {
                    setRun(detail.run);
                    setMessages(detail.messages);
                    setEvents(detail.events);
                });
                if (isTerminalRunStatus(detail.run.status)) {
                    closeSubscription();
                }
                else {
                    openSubscription();
                }
                setError("");
            }
            catch (err) {
                if (cancelled)
                    return;
                setError(err instanceof Error ? err.message : "加载运行详情失败");
            }
            finally {
                if (!cancelled && !options?.silent) {
                    setIsLoading(false);
                }
            }
        }
        async function refreshAfterDisconnect() {
            try {
                const detail = await getRun(activeRunID);
                if (cancelled)
                    return;
                startTransition(() => {
                    setRun(detail.run);
                    setMessages(detail.messages);
                    setEvents(detail.events);
                });
                if (isTerminalRunStatus(detail.run.status)) {
                    closeSubscription();
                    setError("");
                }
                else {
                    openSubscription();
                }
            }
            catch {
                if (cancelled)
                    return;
                setError("实时事件流重连中");
            }
        }
        void refresh();
        return () => {
            cancelled = true;
            closeSubscription();
        };
    }, [runId]);
    return {
        run,
        messages,
        events,
        isLoading,
        error,
        appendLocalMessage,
        appendLocalEvents,
    };
}
