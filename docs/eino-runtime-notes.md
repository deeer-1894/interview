# Eino Runtime Notes

Source baseline: `eino-source` at `1aee2895c7036a750d35165774a30ba7afede2ea` (`v0.8.12`, `origin/main`).

## Runtime Facts

- Use `eino-source` only for source study. Do not use older module cache source as design truth.
- `ChatModelAgent` is implemented on top of a ReAct graph: `Init -> ChatModel -> ToolNode -> ChatModel -> END`.
- `MaxIterations` defaults to 20 in `adk/react.go`; OfferBot should set it explicitly to 4 for one interview turn.
- `defaultGenModelInput` formats `Instruction` with `SessionValues` using FString when session values exist. OfferBot should provide a custom `GenModelInput` because resumes may contain braces, JSON, or code.
- `Runner.Run` creates a new run context, adds `SessionValues`, and returns an async iterator of `AgentEvent`.
- `Runner.Resume` requires a `CheckPointStore`; business session state must still live in the product store.
- `SessionValues` are per-run values, not the durable interview session.

## Middleware Facts

- New code should use `ChatModelAgentConfig.Handlers`, not legacy `AgentMiddleware`, for dynamic behavior.
- `ChatModelAgentMiddleware` hooks:
  - `BeforeAgent`: modify instruction, raw tools, and `ReturnDirectly` for the whole run.
  - `BeforeModelRewriteState`: rewrite persisted message state before each model call.
  - `AfterModelRewriteState`: rewrite persisted message state after each model call.
  - `WrapModel`: wrap one model call, useful for trace/cost/timeout observability.
  - `Wrap*ToolCall`: wrap one tool call, useful for permission, audit, errors, and truncation.
- In `v0.8.12`, handler wrapping order is consistent for model and tool paths: `handlers[0]` is the outermost wrapper.
- `adk.State` is deprecated for user extension. Use `ChatModelAgentState` in hooks and `SetRunLocalValue` / `GetRunLocalValue` for run-local data.
- `SendEvent` can emit custom events from middleware into the Runner event stream.

## Built-In Middleware Notes

- `skill.NewMiddleware` injects a skill system instruction and a `skill` tool.
- Skill backend scans only first-level `skills/*/SKILL.md`.
- Skill modes:
  - inline: return skill content as the current agent tool result.
  - fork: run a sub-agent without parent history.
  - fork_with_context: run a sub-agent with parent history.
- OfferBot v1 should use inline skill only.
- `patchtoolcalls.New` patches assistant tool calls that have no corresponding tool message; useful for interrupted or resumed long interviews.
- `plantask` is coding-task-oriented task management and should not drive interview turns.
- Filesystem middleware should not be enabled in v1 unless skills need references/scripts or uploaded materials need file reads.

## OfferBot Mapping

- `ChatModelAgent`: interviewer brain and tool selector.
- `Runner`: per-turn execution and SSE event source.
- Business store: durable users, interview sessions, messages, resume profiles, scorecards.
- `SessionValues`: small run-scoped identifiers such as `session_id`, `user_id`, `role`, `mode`, `round`.
- `customGenModelInput`: builds system behavior, turn context JSON, recent history, and latest candidate answer.
- `SessionContextMiddleware`: loads durable session/resume/messages and builds `TurnContext`.
- `ContextIsolationMiddleware`: exposes full active project context and only brief summaries for other projects.
- `QuestionQualityMiddleware`: checks single-question output, duplicate fingerprints, and forbidden facts after model output.
- `SkillMiddleware`: loads generic interview capability packs, not resume facts.
- `SafeToolMiddleware`: controls permissions and converts tool errors while preserving interrupt/resume errors.
- `TraceMiddleware`: wraps model and tool calls and emits trace events.

## Business Model Boundary

- Business models are intentionally independent from Eino ADK types.
- Durable interview state lives in `internal/interview/session`, `internal/interview/resume`, and `internal/interview/report`.
- Eino run state is used only for the active turn execution; it must not become the source of truth for sessions, messages, resume profiles, or scorecards.
- `TurnContext` is the bridge from business state into `ChatModelAgentState`.

## Store Boundary

- Latest ADK `CheckPointStore` is intentionally small: `Get(ctx, checkpointID)` and `Set(ctx, checkpointID, bytes)`.
- Runner persists checkpoint bytes before returning an interrupt event, so API code can safely expose `checkpointID` after receiving the interrupt.
- Business stores and ADK checkpoint storage stay separate: Mongo stores product state; Redis stores run locks, checkpoint id mapping, and ADK checkpoint bytes.
- Every business read is scoped by `userID`; this prevents cross-account and cross-session history leakage before middleware sees context.
- Memory store mirrors the same interfaces for local development and later E2E validation without requiring Mongo/Redis.

## Runtime Skeleton

- Project dependency is aligned to `github.com/cloudwego/eino v0.8.12`, matching the cloned source baseline.
- Runtime owns only ADK construction and per-turn execution; it does not write interview messages or reports.
- `RunTurn` always passes `WithSessionValues` and `WithCheckPointID`, so middleware can rely on `user_id` and `session_id` without parsing prompts.
- `ResumeTurn` can load the latest checkpoint id from `RunStore`, then uses ADK `Resume` or `ResumeWithParams`.
- `MaxIterations` is set to 4 by default to keep one product turn bounded.

## customGenModelInput

- Eino default `GenModelInput` uses FString when session values exist; OfferBot must avoid it because resumes may contain JSON, braces, or code.
- Current `BuildModelInput` never formats the instruction string. It appends system messages, structured `TurnContext` JSON, recent user/assistant history, then the current user input.
- The global instruction stays small and generic. Resume/project facts come from `TurnContext`, not hard-coded prompt branches.
- Tool/system messages from persisted history are not replayed into the model by default because tool call ids are not available from product messages yet.

## SessionContextMiddleware

- `BeforeAgent` is the right hook for loading durable business context because it can return a modified Go context before `GenModelInput` runs.
- Session context is built from `user_id` and `session_id` in ADK `SessionValues`; no prompt parsing is used.
- The middleware loads interview session, resume profile, active project, other project briefs, recent messages, and latest answer into `TurnContext`.
- It keeps current user input in `last_answer` via session values, so the API does not need to pre-persist the message before model execution.

## ContextIsolationMiddleware

- Context isolation is implemented as data shaping, not project-specific prompt text.
- The active project keeps full structured facts; other projects are reduced to name/domain briefs by `SessionContextMiddleware`.
- `ContextIsolationMiddleware` adds forbidden project name/domain facts and filters recent messages that mention another project without also mentioning the active project.
- Shared technologies such as Go, Redis, or MongoDB are not used as isolation keys because they are too broad and can create false positives.

## PatchToolCallsMiddleware

- Latest Eino `patchtoolcalls` scans `ChatModelAgentState.Messages` before each model call.
- It inserts a synthetic tool message when an assistant tool call lacks a matching tool result.
- OfferBot uses the built-in middleware directly; the only customization is a short Chinese patched tool result so resumed interviews continue without dangling tool-call protocol errors.

## SkillMiddleware

- Latest Eino skill middleware is a `ChatModelAgentMiddleware` that appends a skill tool during `BeforeAgent`.
- Filesystem backend scans first-level `*/SKILL.md`; OfferBot seeds an Eino in-memory filesystem to avoid granting broad local file access.
- Default Eino skill prompt is intentionally replaced with a short product-specific instruction and concise tool description.
- Six generic interview skills are available: backend system design, agent runtime/tools, Go concurrency, Python automation, data pipeline/storage, and content safety/risk.
- Skills contain evaluation methods only. They do not contain resume project facts.

## SkillRoutingMiddleware

- Skill routing writes `recommendedSkills` into `TurnContext`; it does not generate project-specific questions.
- Routing uses broad cues from role, mode, active project domain, tech stack, and claims.
- The recommended list is capped at three skills to avoid flooding the model with options.
- Project names are not used as route rules; only generic technology/domain cues are considered.

## SafeToolMiddleware

- Tool safety is implemented with ADK tool wrappers.
- The middleware enforces a tool whitelist and defaults to allowing `interview_skill`.
- Synchronous tool calls get a bounded timeout; streaming tools are allowed/rejected and errors are standardized, but the stream lifecycle itself remains owned by Eino.

## TraceMiddleware

- Trace uses `WrapModel` and tool wrappers, matching Eino handler extension points.
- It emits `CustomizedAction` events with type, tool/model name, call id, duration, and error.
- Trace events are for SSE/API observability; they are not fed back into prompt text.

## QuestionQualityMiddleware

- Question quality is checked as structured validation after model state rewrite.
- Current checks: multiple question marks, duplicate normalized question hash, and forbidden project facts.
- The middleware emits a `question_quality` custom event. It does not hard-code replacement questions.
- API orchestration can later decide whether to accept, retry, or mark the turn based on this event.

## Graph Tools

- Graph tools are read-only Eino tools backed by `TurnContext`.
- Current tools: `interview_context`, `active_project`, `recent_history`, and `scorecard_outline`.
- They return JSON strings and do not access arbitrary files or external systems.
- `GraphToolNames` feeds `SafeToolMiddleware` whitelist construction.

## Interrupt/Resume

- Tool-level interrupts should use `github.com/cloudwego/eino/components/tool.Interrupt` or `StatefulInterrupt`; Runner will checkpoint through ADK.
- `clarify` is a stateful tool: first execution interrupts with `ClarifyInfo`, resume returns candidate clarification text to the agent.
- `ClarifyInfo` and `ClarifyState` are registered for Eino/gob serialization.
- `SafeToolMiddleware` wraps errors with `%w`, preserving interrupt errors for ADK propagation.

## Current Runtime Design

Recommended handler order:

```go
Handlers: []adk.ChatModelAgentMiddleware{
    traceMiddleware,
    safeToolMiddleware,
    sessionContextMiddleware,
    contextIsolationMiddleware,
    patchToolCallsMiddleware,
    skillRoutingMiddleware,
    skillMiddleware,
    questionQualityMiddleware,
}
```

First handler is outermost in the latest `v0.8.12` model and tool wrapper paths.
