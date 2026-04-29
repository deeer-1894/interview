import React from "react";
import { createRoot } from "react-dom/client";
import ReactMarkdown from "react-markdown";
import rehypeHighlight from "rehype-highlight";
import remarkGfm from "remark-gfm";
import {
  Bot,
  CheckCircle2,
  ChevronsUpDown,
  CircleAlert,
  FileText,
  Loader2,
  LogIn,
  MessageSquarePlus,
  MoreHorizontal,
  Moon,
  PanelLeft,
  Plus,
  RefreshCw,
  Save,
  Search,
  Send,
  Sun,
  Trash2,
} from "lucide-react";

import { Avatar, AvatarFallback } from "./components/ui/avatar";
import { Button } from "./components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "./components/ui/dialog";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "./components/ui/dropdown-menu";
import { Input } from "./components/ui/input";
import { Label } from "./components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "./components/ui/select";
import { Separator } from "./components/ui/separator";
import { Textarea } from "./components/ui/textarea";
import { cn } from "./lib/utils";
import "./styles.css";

type Session = {
  id: string;
  role: string;
  level: string;
  mode: string;
  round: number;
  status: string;
  resumeProfileId: string;
  activeProjectId?: string;
  updatedAt: string;
};

type Message = {
  id: string;
  role: "user" | "assistant" | "tool" | "system";
  content: string;
  createdAt: string;
};

type StreamResult = {
  assistant?: string;
  checkpointId?: string;
};

type Profile = {
  id: string;
  rawText: string;
  summary?: string;
  skills?: string[];
  projects?: Array<{ id: string; name: string; domain?: string; summary?: string }>;
};

const apiBase =
  import.meta.env.VITE_API_BASE ?? `${window.location.protocol}//${window.location.hostname}:8080`;
const tokenKey = "offerbot_token";
const themeKey = "offerbot_theme";
const emptyAssistantText = "这轮没有收到模型的有效输出，请重新生成一次。";

type Theme = "dark" | "light";

function App() {
  const [token, setToken] = React.useState(localStorage.getItem(tokenKey) ?? "");
  const [email, setEmail] = React.useState("shiyi123@123.com");
  const [password, setPassword] = React.useState("shiyi123456");
  const [profile, setProfile] = React.useState<Profile | null>(null);
  const [resumeText, setResumeText] = React.useState("");
  const [role, setRole] = React.useState("");
  const [level, setLevel] = React.useState("中高级");
  const [mode, setMode] = React.useState("长面试");
  const [sessions, setSessions] = React.useState<Session[]>([]);
  const [activeSession, setActiveSession] = React.useState<Session | null>(null);
  const [messages, setMessages] = React.useState<Message[]>([]);
  const [answer, setAnswer] = React.useState("");
  const [busy, setBusy] = React.useState(false);
  const [streamingMessageId, setStreamingMessageId] = React.useState("");
  const [streamingHint, setStreamingHint] = React.useState("");
  const [notice, setNotice] = React.useState("");
  const [error, setError] = React.useState("");
  const [setupOpen, setSetupOpen] = React.useState(false);
  const [searchOpen, setSearchOpen] = React.useState(false);
  const [searchQuery, setSearchQuery] = React.useState("");
  const [sidebarCollapsed, setSidebarCollapsed] = React.useState(false);
  const messageListRef = React.useRef<HTMLDivElement>(null);
  const searchInputRef = React.useRef<HTMLInputElement>(null);
  const [theme, setTheme] = React.useState<Theme>(() => {
    const stored = localStorage.getItem(themeKey);
    return stored === "dark" ? "dark" : "light";
  });

  const resumeChars = resumeText.trim().length;
  const canCreateSession = Boolean(profile?.id && role.trim());
  const filteredSessions = React.useMemo(() => {
    const query = searchQuery.trim().toLowerCase();
    if (!query) return sessions;
    return sessions.filter((session) =>
      [session.role, session.level, session.mode, session.status]
        .filter(Boolean)
        .some((value) => value.toLowerCase().includes(query)),
    );
  }, [searchQuery, sessions]);

  React.useEffect(() => {
    if (!token) return;
    void loadBootstrap();
  }, [token]);

  React.useEffect(() => {
    document.documentElement.dataset.theme = theme;
    document.documentElement.classList.toggle("dark", theme === "dark");
    localStorage.setItem(themeKey, theme);
  }, [theme]);

  React.useEffect(() => {
    const node = messageListRef.current;
    if (!node) return;
    node.scrollTo({ top: node.scrollHeight, behavior: streamingMessageId ? "auto" : "smooth" });
  }, [messages, streamingMessageId]);

  React.useEffect(() => {
    if (!searchOpen) return;
    const timer = window.setTimeout(() => searchInputRef.current?.focus(), 0);
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape") setSearchOpen(false);
    };
    window.addEventListener("keydown", onKeyDown);
    return () => {
      window.clearTimeout(timer);
      window.removeEventListener("keydown", onKeyDown);
    };
  }, [searchOpen]);

  async function request<T>(path: string, init: RequestInit = {}): Promise<T> {
    const response = await fetch(`${apiBase}${path}`, {
      ...init,
      headers: {
        "content-type": "application/json",
        ...(token ? { authorization: `Bearer ${token}` } : {}),
        ...(init.headers ?? {}),
      },
    });
    const body = await response.json().catch(() => ({}));
    if (!response.ok) throw new Error(body.error ?? `HTTP ${response.status}`);
    return body as T;
  }

  async function login() {
    setBusy(true);
    setError("");
    try {
      const data = await fetch(`${apiBase}/api/login`, {
        method: "POST",
        headers: { "content-type": "application/json" },
        body: JSON.stringify({ email, password }),
      }).then(async (response) => {
        const body = await response.json();
        if (!response.ok) throw new Error(body.error ?? "登录失败");
        return body as { token: string };
      });
      localStorage.setItem(tokenKey, data.token);
      setToken(data.token);
    } catch (err) {
      setError(err instanceof Error ? err.message : "登录失败");
    } finally {
      setBusy(false);
    }
  }

  function logout() {
    localStorage.removeItem(tokenKey);
    setToken("");
    setProfile(null);
    setSessions([]);
    setActiveSession(null);
    setMessages([]);
  }

  async function loadBootstrap() {
    await Promise.all([loadProfile(), loadSessions()]);
  }

  async function loadProfile() {
    try {
      const data = await request<Profile>("/api/profile");
      setProfile(data);
      setResumeText(data.rawText ?? "");
    } catch {
      setProfile(null);
      setResumeText("");
    }
  }

  async function saveProfile() {
    if (!resumeText.trim()) {
      setError("请先粘贴候选人的真实简历内容");
      return;
    }
    setBusy(true);
    setError("");
    try {
      const saved = await request<Profile>("/api/profile", {
        method: "POST",
        body: JSON.stringify({ rawText: resumeText }),
      });
      setProfile(saved);
      setNotice("简历已保存");
    } catch (err) {
      setError(err instanceof Error ? err.message : "保存失败");
    } finally {
      setBusy(false);
    }
  }

  async function loadSessions() {
    try {
      const data = await request<{ sessions: Session[] }>("/api/sessions");
      const nextSessions = data.sessions ?? [];
      setSessions(nextSessions);
      if (!activeSession && nextSessions.length) {
        setActiveSession(nextSessions[0]);
        await loadMessages(nextSessions[0].id);
        return;
      }
      if (activeSession) {
        const updated = nextSessions.find((item) => item.id === activeSession.id);
        if (updated) setActiveSession(updated);
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : "加载会话失败");
    }
  }

  async function startSession() {
    if (!profile?.id) {
      setError("请先保存简历，再创建面试");
      return;
    }
    if (!role.trim()) {
      setError("请填写面试职位");
      return;
    }
    setBusy(true);
    setError("");
    setStreamingHint("正在准备面试");
    try {
      await streamNewSession();
    } catch (err) {
      setError(err instanceof Error ? err.message : "创建面试失败");
    } finally {
      setStreamingMessageId("");
      setStreamingHint("");
      setBusy(false);
    }
  }

  async function streamNewSession() {
    const assistantId = crypto.randomUUID();
    const response = await fetch(`${apiBase}/api/sessions/stream`, {
      method: "POST",
      headers: {
        "content-type": "application/json",
        authorization: `Bearer ${token}`,
      },
      body: JSON.stringify({
        resumeProfileId: profile?.id,
        role,
        level,
        mode,
      }),
    });
    if (!response.ok || !response.body) {
      const body = await response.json().catch(() => ({}));
      throw new Error(body.error ?? `HTTP ${response.status}`);
    }

    setSetupOpen(false);
    setMessages([{ id: assistantId, role: "assistant", content: "", createdAt: new Date().toISOString() }]);
    setStreamingMessageId(assistantId);

    const reader = response.body.getReader();
    const decoder = new TextDecoder();
    let buffer = "";
    let assistantText = "";
    for (;;) {
      const { value, done } = await reader.read();
      if (done) break;
      buffer += decoder.decode(value, { stream: true });
      const events = buffer.split("\n\n");
      buffer = events.pop() ?? "";
      for (const raw of events) {
        const parsed = parseSSE(raw);
        if (!parsed) continue;
        if (parsed.event === "session") {
          const session = JSON.parse(parsed.data) as Session;
          setActiveSession(session);
          setSessions((items) => [session, ...items.filter((item) => item.id !== session.id)]);
        }
        if (parsed.event === "delta") {
          const delta = JSON.parse(parsed.data) as string;
          assistantText += delta;
          appendAssistantDelta(assistantId, delta);
          setStreamingHint("正在生成");
        }
        if (parsed.event === "custom") {
          updateStreamingHint(parsed.data);
        }
        if (parsed.event === "error") {
          const payload = JSON.parse(parsed.data) as { error?: string };
          throw new Error(payload.error ?? "流式创建面试失败");
        }
        if (parsed.event === "done") {
          const result = JSON.parse(parsed.data) as StreamResult | Session;
          const finalText = replaceAssistantFromDone(assistantId, result);
          if (finalText) assistantText = finalText;
        }
      }
    }
    if (!assistantText.trim()) {
      replaceAssistantContent(assistantId, emptyAssistantText);
    }
    setNotice("新面试已创建");
    await loadSessions();
  }

  async function selectSession(session: Session) {
    setActiveSession(session);
    await loadMessages(session.id);
  }

  async function selectSessionFromSearch(session: Session) {
    setSearchOpen(false);
    setSearchQuery("");
    await selectSession(session);
  }

  async function loadMessages(sessionId: string) {
    const data = await request<{ messages: Message[] }>(`/api/sessions/${sessionId}/messages`);
    setMessages(data.messages ?? []);
  }

  async function sendAnswer() {
    if (!activeSession || !answer.trim()) return;
    const content = answer.trim();
    setAnswer("");
    setBusy(true);
    setError("");
    setStreamingHint("正在思考");
    const assistantId = crypto.randomUUID();
    setStreamingMessageId(assistantId);
    setMessages((items) => [
      ...items,
      { id: crypto.randomUUID(), role: "user", content, createdAt: new Date().toISOString() },
      { id: assistantId, role: "assistant", content: "", createdAt: new Date().toISOString() },
    ]);
    try {
      await streamAnswer(activeSession.id, content, assistantId);
      await loadSessions();
    } catch (err) {
      setError(err instanceof Error ? err.message : "发送失败");
      setMessages((items) => items.filter((item) => item.id !== assistantId || item.content.trim()));
    } finally {
      setStreamingMessageId("");
      setStreamingHint("");
      setBusy(false);
    }
  }

  async function streamAnswer(sessionId: string, content: string, assistantId: string) {
    const response = await fetch(`${apiBase}/api/sessions/${sessionId}/messages/stream`, {
      method: "POST",
      headers: {
        "content-type": "application/json",
        authorization: `Bearer ${token}`,
      },
      body: JSON.stringify({ content }),
    });
    if (!response.ok || !response.body) {
      const body = await response.json().catch(() => ({}));
      throw new Error(body.error ?? `HTTP ${response.status}`);
    }

    const reader = response.body.getReader();
    const decoder = new TextDecoder();
    let buffer = "";
    let assistantText = "";
    for (;;) {
      const { value, done } = await reader.read();
      if (done) break;
      buffer += decoder.decode(value, { stream: true });
      const events = buffer.split("\n\n");
      buffer = events.pop() ?? "";
      for (const raw of events) {
        const parsed = parseSSE(raw);
        if (!parsed) continue;
        if (parsed.event === "delta") {
          const delta = JSON.parse(parsed.data) as string;
          assistantText += delta;
          appendAssistantDelta(assistantId, delta);
          setStreamingHint("正在生成");
        }
        if (parsed.event === "custom") {
          updateStreamingHint(parsed.data);
        }
        if (parsed.event === "error") {
          const payload = JSON.parse(parsed.data) as { error?: string };
          throw new Error(payload.error ?? "流式输出失败");
        }
        if (parsed.event === "done") {
          const result = JSON.parse(parsed.data) as StreamResult;
          const finalText = replaceAssistantFromDone(assistantId, result);
          if (finalText) assistantText = finalText;
        }
      }
    }
    if (!assistantText.trim()) {
      replaceAssistantContent(assistantId, emptyAssistantText);
    }
  }

  function appendAssistantDelta(assistantId: string, delta: string) {
    if (!delta) return;
    setMessages((items) =>
      items.map((item) =>
        item.id === assistantId ? { ...item, content: item.content + delta } : item,
      ),
    );
  }

  function replaceAssistantFromDone(assistantId: string, result: StreamResult | Session) {
    if (!("assistant" in result) || !result.assistant) return "";
    setMessages((items) =>
      items.map((item) => {
        if (item.id !== assistantId) return item;
        if (item.content === result.assistant) return item;
        if (item.content && result.assistant?.startsWith(item.content)) {
          return { ...item, content: result.assistant };
        }
        return { ...item, content: item.content || result.assistant || "" };
      }),
    );
    return result.assistant;
  }

  function replaceAssistantContent(assistantId: string, content: string) {
    setMessages((items) =>
      items.map((item) => (item.id === assistantId ? { ...item, content } : item)),
    );
  }

  function updateStreamingHint(rawData: string) {
    const payload = JSON.parse(rawData) as { type?: string; name?: string; error?: string };
    if (payload.error) {
      setStreamingHint("生成遇到错误");
      return;
    }
    if (payload.type === "tool" || payload.name) {
      setStreamingHint("正在查看上下文");
      return;
    }
    if (payload.type === "model_stream_open") {
      setStreamingHint("正在思考");
    }
  }

  async function deleteSession(sessionId: string) {
    setBusy(true);
    setError("");
    try {
      await request<Record<string, never>>(`/api/sessions/${sessionId}`, { method: "DELETE" });
      const nextSessions = sessions.filter((item) => item.id !== sessionId);
      setSessions(nextSessions);
      if (activeSession?.id === sessionId) {
        const next = nextSessions[0] ?? null;
        setActiveSession(next);
        if (next) await loadMessages(next.id);
        else setMessages([]);
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : "删除失败");
    } finally {
      setBusy(false);
    }
  }

  function handleComposerKeyDown(event: React.KeyboardEvent<HTMLTextAreaElement>) {
    if (event.key !== "Enter" || event.shiftKey || event.nativeEvent.isComposing) return;
    event.preventDefault();
    void sendAnswer();
  }

  function displayMessage(content: string) {
    return content
      .replace(/[（(]\s*第\s*\d+\s*轮\s*[）)]/g, "")
      .replace(/第\s*\d+\s*轮/g, "")
      .trim();
  }

  function toggleTheme() {
    setTheme((value) => (value === "dark" ? "light" : "dark"));
  }

  if (!token) {
    return (
      <main className="grid min-h-dvh place-items-center bg-background px-6 text-foreground">
        <section className="grid w-full max-w-sm gap-5 rounded-2xl border bg-card p-6 text-card-foreground shadow-sm">
          <div className="flex items-center gap-3">
            <div className="grid size-9 place-items-center rounded-lg border bg-background">
              <Bot className="size-5" />
            </div>
            <div>
              <h1 className="text-lg font-semibold">OfferBot</h1>
              <p className="text-sm text-muted-foreground">AI 模拟面试</p>
            </div>
          </div>
          <div className="grid gap-2">
            <Label htmlFor="email">账号</Label>
            <Input id="email" value={email} onChange={(event) => setEmail(event.target.value)} />
          </div>
          <div className="grid gap-2">
            <Label htmlFor="password">密码</Label>
            <Input
              id="password"
              type="password"
              value={password}
              onChange={(event) => setPassword(event.target.value)}
            />
          </div>
          <Button onClick={login} disabled={busy}>
            {busy ? <Loader2 className="mr-2 size-4 animate-spin" /> : <LogIn className="mr-2 size-4" />}
            登录
          </Button>
          {error && <InlineNotice tone="error" text={error} />}
        </section>
      </main>
    );
  }

  return (
    <main
      className={cn(
        "grid h-dvh overflow-hidden bg-background text-foreground transition-[grid-template-columns] duration-200",
        sidebarCollapsed ? "grid-cols-[72px_minmax(0,1fr)]" : "grid-cols-[260px_minmax(0,1fr)]",
      )}
    >
      <aside className="grid min-h-0 grid-rows-[auto_auto_auto_1fr_auto] border-r bg-sidebar text-sidebar-foreground">
        <div className={cn("flex h-14 items-center px-3", sidebarCollapsed ? "justify-center" : "justify-between")}>
          <Button variant="ghost" size="icon" className="size-9 rounded-lg" aria-label="OfferBot">
            <Bot className="size-[18px]" />
          </Button>
          {!sidebarCollapsed && (
            <Button
              variant="ghost"
              size="icon"
              className="size-9 rounded-lg"
              aria-label="收起侧边栏"
              onClick={() => setSidebarCollapsed(true)}
            >
              <PanelLeft className="size-[18px]" />
            </Button>
          )}
        </div>

        <nav className={cn("grid gap-1 pb-4", sidebarCollapsed ? "px-2" : "px-3")}>
          <Button
            type="button"
            variant="ghost"
            className={cn("h-10 rounded-lg text-[15px]", sidebarCollapsed ? "justify-center px-0" : "justify-start gap-3 px-3")}
            onClick={() => setSetupOpen(true)}
            aria-label="新面试"
          >
            <MessageSquarePlus className="size-[18px]" />
            {!sidebarCollapsed && "新面试"}
          </Button>
          <Button
            type="button"
            variant="ghost"
            className={cn("h-10 rounded-lg text-[15px]", sidebarCollapsed ? "justify-center px-0" : "justify-start gap-3 px-3")}
            aria-label="搜索面试"
            onClick={() => setSearchOpen(true)}
          >
            <Search className="size-[18px]" />
            {!sidebarCollapsed && "搜索面试"}
          </Button>
        </nav>

        <div className={cn("px-5 pb-2 text-xs text-muted-foreground", sidebarCollapsed && "sr-only")}>最近</div>

        <div className={cn("min-h-0 overflow-y-auto pb-3", sidebarCollapsed ? "px-2" : "px-2")}>
          {sidebarCollapsed ? (
            <div className="grid gap-1">
              {sessions.slice(0, 12).map((session) => (
                <Button
                  key={session.id}
                  variant={activeSession?.id === session.id ? "secondary" : "ghost"}
                  size="icon"
                  className="size-10 rounded-lg"
                  title={session.role || "未命名职位"}
                  onClick={() => void selectSession(session)}
                >
                  <FileText className="size-[17px]" />
                </Button>
              ))}
            </div>
          ) : sessions.length ? (
            sessions.map((session) => (
              <SessionRow
                key={session.id}
                session={session}
                active={activeSession?.id === session.id}
                onSelect={() => void selectSession(session)}
                onDelete={() => void deleteSession(session.id)}
              />
            ))
          ) : (
            <div className="px-3 py-2 text-sm leading-6 text-muted-foreground">保存简历并填写职位后创建面试</div>
          )}
        </div>

        <div className="border-t p-2">
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button
                variant="ghost"
                className={cn(
                  "h-11 rounded-lg",
                  sidebarCollapsed ? "w-11 justify-center px-0" : "w-full justify-start gap-3 px-3",
                )}
              >
                <Avatar className="size-7">
                  <AvatarFallback className="bg-amber-500 text-xs font-semibold text-white">S</AvatarFallback>
                </Avatar>
                {!sidebarCollapsed && (
                  <>
                    <span className="min-w-0 flex-1 truncate text-left">shiyi123</span>
                    <ChevronsUpDown className="size-4 text-muted-foreground" />
                  </>
                )}
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end" className="w-52">
              {sidebarCollapsed && (
                <DropdownMenuItem onClick={() => setSidebarCollapsed(false)}>
                  <PanelLeft className="size-4" />
                  展开侧边栏
                </DropdownMenuItem>
              )}
              <DropdownMenuItem onClick={toggleTheme}>
                {theme === "dark" ? <Sun className="size-4" /> : <Moon className="size-4" />}
                {theme === "dark" ? "浅色模式" : "深色模式"}
              </DropdownMenuItem>
              <DropdownMenuSeparator />
              <DropdownMenuItem onClick={logout}>退出登录</DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        </div>
      </aside>

      <section className="grid min-w-0 grid-rows-[56px_1fr_auto] overflow-hidden">
        <header className="flex items-center justify-between border-b px-4">
          <div className="flex items-center gap-1">
            {sidebarCollapsed && (
              <Button
                variant="ghost"
                size="icon"
                className="size-9 rounded-lg"
                aria-label="展开侧边栏"
                onClick={() => setSidebarCollapsed(false)}
              >
                <PanelLeft className="size-[18px]" />
              </Button>
            )}
            <Button variant="ghost" className="gap-1 rounded-lg px-3 text-lg font-semibold">
            OfferBot
            <ChevronsUpDown className="size-4 text-muted-foreground" />
            </Button>
          </div>
          <div className="flex items-center gap-1">
            <Button variant="ghost" size="icon" className="size-9 rounded-lg" onClick={toggleTheme}>
              {theme === "dark" ? <Sun className="size-[18px]" /> : <Moon className="size-[18px]" />}
            </Button>
            <Button variant="ghost" size="icon" className="size-9 rounded-lg" onClick={() => setSetupOpen(true)}>
              <Plus className="size-[18px]" />
            </Button>
            <Button
              variant="ghost"
              size="icon"
              className="size-9 rounded-lg"
              disabled={!activeSession || busy}
              onClick={() => activeSession && loadMessages(activeSession.id)}
            >
              <RefreshCw className="size-[18px]" />
            </Button>
          </div>
        </header>

        <div ref={messageListRef} className="min-h-0 overflow-y-auto px-6 py-7">
          <div className="mx-auto flex w-full max-w-[720px] flex-col gap-7">
            {messages.map((message) => (
              <MessageItem
                key={message.id}
                message={message}
                content={displayMessage(message.content)}
                streaming={message.id === streamingMessageId}
                streamingHint={message.id === streamingMessageId ? streamingHint : ""}
              />
            ))}
            {!messages.length && (
              <div className="grid min-h-[55vh] place-items-center text-center">
                <div>
                  <h2 className="text-3xl font-semibold tracking-tight">准备练哪场面试？</h2>
                  <p className="mt-3 text-sm text-muted-foreground">点击右上角或侧边栏的 +，先粘贴简历并选择职位。</p>
                </div>
              </div>
            )}
          </div>
        </div>

        <div className="px-6 pb-5">
          <div className="mx-auto grid w-full max-w-[680px] grid-cols-[40px_minmax(0,1fr)_40px] items-center gap-2 rounded-[26px] border bg-background p-1.5 shadow-[0_12px_34px_rgba(0,0,0,0.08)]">
            <Button
              type="button"
              variant="ghost"
              size="icon"
              className="size-10 rounded-full"
              onClick={() => setSetupOpen(true)}
              aria-label="新面试"
            >
              <Plus className="size-5" />
            </Button>
            <Textarea
              value={answer}
              onChange={(event) => setAnswer(event.target.value)}
              onKeyDown={handleComposerKeyDown}
              placeholder="输入候选人的真实回答"
              disabled={!activeSession || busy}
              rows={1}
              className="h-10 !min-h-10 max-h-32 resize-none border-0 bg-transparent px-1 py-2 text-base shadow-none focus-visible:ring-0"
            />
            <Button
              type="button"
              size="icon"
              className="size-10 rounded-full"
              onClick={sendAnswer}
              disabled={!activeSession || !answer.trim() || busy}
            >
              {busy ? <Loader2 className="size-4 animate-spin" /> : <Send className="size-4" />}
            </Button>
          </div>
          {error && <div className="mx-auto mt-3 max-w-[680px]"><InlineNotice tone="error" text={error} /></div>}
        </div>
      </section>

      <Dialog open={setupOpen} onOpenChange={setSetupOpen}>
        <DialogContent className="sm:max-w-2xl">
          <DialogHeader>
            <DialogTitle>新面试</DialogTitle>
            <DialogDescription>粘贴简历，选择职位和面试模式。</DialogDescription>
          </DialogHeader>

          <div className="grid gap-5">
            <div className="grid gap-2">
              <div className="flex items-center justify-between">
                <Label htmlFor="resume">候选人简历</Label>
                <span className="inline-flex items-center gap-1 text-xs text-muted-foreground">
                  <FileText className="size-3.5" />
                  {resumeChars} 字
                </span>
              </div>
              <Textarea
                id="resume"
                value={resumeText}
                onChange={(event) => setResumeText(event.target.value)}
                placeholder="粘贴候选人真实简历。"
                className="min-h-52 resize-none"
              />
              <Button variant="secondary" onClick={saveProfile} disabled={busy || !resumeText.trim()} className="w-fit">
                <Save className="mr-2 size-4" />
                保存简历
              </Button>
            </div>

            <Separator />

            <div className="grid gap-4 sm:grid-cols-2">
              <div className="grid gap-2 sm:col-span-2">
                <Label htmlFor="role">职位</Label>
                <Input
                  id="role"
                  value={role}
                  onChange={(event) => setRole(event.target.value)}
                  placeholder="例如：后端工程师 / Agent 工程师"
                />
              </div>
              <div className="grid gap-2">
                <Label>级别</Label>
                <Select value={level} onValueChange={setLevel}>
                  <SelectTrigger>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    {["初级", "中级", "中高级", "高级", "专家"].map((item) => (
                      <SelectItem key={item} value={item}>
                        {item}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
              <div className="grid gap-2">
                <Label>模式</Label>
                <Select value={mode} onValueChange={setMode}>
                  <SelectTrigger>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    {["长面试", "项目深挖", "系统设计", "工程排障", "跨职位验证"].map((item) => (
                      <SelectItem key={item} value={item}>
                        {item}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
            </div>

            {(notice || error) && <InlineNotice tone={error ? "error" : "success"} text={error || notice} />}
          </div>

          <DialogFooter>
            <Button variant="outline" onClick={() => setSetupOpen(false)}>
              取消
            </Button>
            <Button onClick={startSession} disabled={busy || !canCreateSession}>
              {busy ? <Loader2 className="mr-2 size-4 animate-spin" /> : <Plus className="mr-2 size-4" />}
              开始面试
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {searchOpen && (
        <div className="fixed inset-0 z-50 bg-background/20 backdrop-blur-[1px]" onMouseDown={() => setSearchOpen(false)}>
          <section
            className="mx-auto mt-[8vh] flex max-h-[74vh] w-[min(720px,calc(100vw-32px))] flex-col overflow-hidden rounded-2xl border bg-popover text-popover-foreground shadow-[0_28px_90px_rgba(0,0,0,0.22)]"
            onMouseDown={(event) => event.stopPropagation()}
          >
            <div className="flex h-16 items-center gap-3 border-b px-5">
              <Search className="size-5 text-muted-foreground" />
              <Input
                ref={searchInputRef}
                value={searchQuery}
                onChange={(event) => setSearchQuery(event.target.value)}
                placeholder="搜索面试..."
                className="h-12 border-0 bg-transparent px-0 text-lg shadow-none focus-visible:ring-0"
              />
              <Button
                type="button"
                variant="ghost"
                size="icon"
                className="size-9 rounded-lg text-muted-foreground"
                onClick={() => setSearchOpen(false)}
              >
                <span className="text-2xl leading-none">×</span>
              </Button>
            </div>
            <div className="min-h-0 overflow-y-auto p-3">
              <button
                className="flex w-full items-center gap-3 rounded-xl px-3 py-3 text-left transition-colors hover:bg-accent"
                onClick={() => {
                  setSearchOpen(false);
                  setSetupOpen(true);
                }}
              >
                <MessageSquarePlus className="size-5" />
                <span className="text-sm font-medium">新面试</span>
              </button>
              <div className="px-3 pb-2 pt-5 text-xs text-muted-foreground">最近</div>
              {filteredSessions.length ? (
                filteredSessions.map((session) => (
                  <button
                    key={session.id}
                    className={cn(
                      "flex w-full items-center gap-3 rounded-xl px-3 py-3 text-left transition-colors hover:bg-accent",
                      activeSession?.id === session.id && "bg-accent",
                    )}
                    onClick={() => void selectSessionFromSearch(session)}
                  >
                    <FileText className="size-5 shrink-0 text-muted-foreground" />
                    <div className="min-w-0 flex-1">
                      <div className="truncate text-sm font-medium">{session.role || "未命名职位"}</div>
                      <div className="mt-0.5 truncate text-xs text-muted-foreground">
                        {[session.level, session.mode, session.status].filter(Boolean).join(" · ")}
                      </div>
                    </div>
                  </button>
                ))
              ) : (
                <div className="grid h-32 place-items-center text-sm text-muted-foreground">没有找到匹配的面试</div>
              )}
            </div>
          </section>
        </div>
      )}
    </main>
  );
}

function SessionRow({
  session,
  active,
  onSelect,
  onDelete,
}: {
  session: Session;
  active: boolean;
  onSelect: () => void;
  onDelete: () => void;
}) {
  return (
    <div className={cn("group flex h-9 items-center rounded-xl", active && "bg-sidebar-accent text-sidebar-accent-foreground")}>
      <button
        className="min-w-0 flex-1 truncate px-3 text-left text-sm leading-9 outline-none"
        title={`${session.role || "未命名职位"} · ${session.level} · ${session.mode}`}
        onClick={onSelect}
      >
        {session.role || "未命名职位"}
      </button>
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <Button
            variant="ghost"
            size="icon"
            className="mr-1 size-7 rounded-lg opacity-0 group-hover:opacity-100 data-[state=open]:opacity-100"
          >
            <MoreHorizontal className="size-4" />
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end">
          <DropdownMenuItem className="text-destructive focus:text-destructive" onClick={onDelete}>
            <Trash2 className="size-4" />
            删除
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>
    </div>
  );
}

function MessageItem({
  message,
  content,
  streaming,
  streamingHint,
}: {
  message: Message;
  content: string;
  streaming: boolean;
  streamingHint: string;
}) {
  if (message.role === "user") {
    return (
      <article className="flex justify-end">
        <div className="max-w-[78%] rounded-3xl bg-muted px-4 py-2.5 text-base leading-7 text-foreground">
          <MarkdownMessage content={content} />
        </div>
      </article>
    );
  }

  return (
    <article className="flex justify-start">
      <div className="min-w-0 max-w-full text-base leading-7">
        {!content && streaming ? (
          <ThinkingIndicator label={streamingHint || "正在思考"} />
        ) : (
          <div className="relative">
            <MarkdownMessage content={content} />
            {streaming && <span className="typing-caret" aria-hidden="true" />}
          </div>
        )}
      </div>
    </article>
  );
}

function ThinkingIndicator({ label }: { label: string }) {
  return (
    <div className="thinking-indicator" role="status" aria-live="polite">
      <span className="thinking-orb" aria-hidden="true" />
      <span>{label}</span>
      <span className="thinking-dot" aria-hidden="true" />
      <span className="thinking-dot" aria-hidden="true" />
      <span className="thinking-dot" aria-hidden="true" />
    </div>
  );
}

function InlineNotice({ tone, text }: { tone: "error" | "success"; text: string }) {
  return (
    <div
      className={cn(
        "flex items-center gap-2 rounded-xl border px-3 py-2 text-sm",
        tone === "error"
          ? "border-destructive/25 bg-destructive/10 text-destructive"
          : "border-emerald-500/20 bg-emerald-500/10 text-emerald-700 dark:text-emerald-300",
      )}
    >
      {tone === "error" ? <CircleAlert className="size-4" /> : <CheckCircle2 className="size-4" />}
      {text}
    </div>
  );
}

function parseSSE(raw: string) {
  let event = "message";
  const data: string[] = [];
  for (const line of raw.replace(/\r/g, "").split("\n")) {
    if (line.startsWith("event:")) event = line.slice(6).trim();
    if (line.startsWith("data:")) data.push(line.slice(5).trimStart());
  }
  if (!data.length) return null;
  return { event, data: data.join("\n") };
}

function MarkdownMessage({ content }: { content: string }) {
  return (
    <div className="markdown-body">
      <ReactMarkdown
        remarkPlugins={[remarkGfm]}
        rehypePlugins={[rehypeHighlight]}
        components={{
          a: ({ children, ...props }) => (
            <a {...props} target="_blank" rel="noreferrer">
              {children}
            </a>
          ),
        }}
      >
        {content}
      </ReactMarkdown>
    </div>
  );
}

createRoot(document.getElementById("root")!).render(<App />);
