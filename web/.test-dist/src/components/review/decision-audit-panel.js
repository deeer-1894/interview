import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { ArrowRight, Bot, CheckCircle2, CircleOff, GitBranch, Lightbulb, ShieldAlert, } from "lucide-react";
export function DecisionAuditPanel({ audit, replay, trace, onFocusMessage, formatInterviewModeLabel, formatInterviewPhaseLabel, formatDecisionReasonLabel, formatSignalLabel, }) {
    const weakSignals = audit?.analysis?.weakSignals ?? [];
    const strongSignals = audit?.analysis?.strongSignals ?? [];
    const weakSignalConfidence = audit?.analysis?.weakSignalConfidence ?? {};
    const strongSignalConfidence = audit?.analysis?.strongSignalConfidence ?? {};
    const profileFocus = audit?.profileFocus ?? [];
    const recommendedFocus = audit?.decision?.recommendedFocus ?? [];
    const history = audit?.state?.history ?? [];
    const traceNodes = trace?.nodes ?? [];
    const primaryAction = getPrimaryAction(audit);
    const preparationHint = recommendedFocus[0] ??
        weakSignals[0] ??
        profileFocus[0] ??
        "等更多追问发生后，这里会提示下一问最可能继续盯住什么。";
    return (_jsxs("section", { className: "panel-card", children: [_jsxs("div", { className: "flex items-start justify-between gap-4", children: [_jsxs("div", { children: [_jsx("p", { className: "tech-label text-[0.68rem] text-[rgba(97,123,150,0.62)]", children: "\u7B56\u7565\u5224\u65AD" }), _jsx("h2", { className: "mt-2 font-display text-2xl text-[rgb(72,91,114)]", children: audit ? "系统为什么会这样继续追问" : "等待策略数据" })] }), audit?.decision?.reason ? (_jsx("span", { className: "chip-info px-3 py-1.5 text-[0.68rem] tracking-[0.2em]", children: formatDecisionReasonLabel(audit.decision.reason) })) : null] }), _jsx("p", { className: "mt-3 max-w-[68ch] text-sm leading-7 text-[rgba(115,137,161,0.78)]", children: "\u8FD9\u91CC\u4E0D\u53EA\u662F\u7ED9\u4F60\u770B\u7CFB\u7EDF\u65E5\u5FD7\uFF0C\u800C\u662F\u628A\u540E\u7AEF\u7684\u4FE1\u53F7\u5224\u65AD\u7FFB\u8BD1\u6210\u4E00\u5957\u4EBA\u8BDD\u8BF4\u660E\uFF0C\u8BA9\u4F60\u77E5\u9053\u7CFB\u7EDF\u521A\u521A\u770B\u5230\u4E86\u4EC0\u4E48\u3001\u63A5\u4E0B\u6765\u51C6\u5907\u600E\u4E48\u95EE\u3001\u4F60\u8BE5\u600E\u4E48\u51C6\u5907\u4E0B\u4E00\u7B54\u3002" }), !audit ? (_jsx("p", { className: "mt-5 rounded-[1.3rem] border border-dashed border-[rgba(153,191,201,0.18)] px-4 py-5 text-sm leading-7 text-[rgba(115,137,161,0.78)]", children: "\u9762\u8BD5\u8FDB\u5165\u8FFD\u95EE\u540E\uFF0C\u8FD9\u91CC\u4F1A\u5C55\u793A\u6700\u65B0\u4E00\u8F6E\u7684\u7B56\u7565\u8F93\u5165\u3001\u547D\u4E2D\u4FE1\u53F7\u548C\u4E0B\u4E00\u6B65\u52A8\u4F5C\u3002" })) : (_jsxs("div", { className: "mt-5 space-y-5", children: [_jsxs("div", { className: "grid gap-3 xl:grid-cols-[1.2fr_0.95fr_0.95fr]", children: [_jsx(SummaryCallout, { icon: _jsx(Bot, { className: "h-4 w-4" }), label: "\u7CFB\u7EDF\u521A\u521A\u7684\u5224\u65AD", title: audit.decision?.explanation ?? "正在整理这一轮为什么这么问", text: audit.decision?.reason
                                    ? `这次主要因为「${formatDecisionReasonLabel(audit.decision.reason)}」而调整追问策略。`
                                    : "等这轮追问结束后，这里会说明判断依据。", tone: "primary" }), _jsx(SummaryCallout, { icon: _jsx(GitBranch, { className: "h-4 w-4" }), label: "\u4E0B\u4E00\u95EE\u5927\u6982\u7387\u4F1A\u600E\u4E48\u8D70", title: primaryAction.title, text: primaryAction.description, tone: "neutral" }), _jsx(SummaryCallout, { icon: _jsx(Lightbulb, { className: "h-4 w-4" }), label: "\u4F60\u73B0\u5728\u66F4\u8BE5\u51C6\u5907\u4EC0\u4E48", title: formatSignalLabel(preparationHint), text: "\u4F18\u5148\u56F4\u7ED5\u8FD9\u4E2A\u70B9\u8865\u7ED3\u8BBA\u3001\u4F9D\u636E\u548C\u4F8B\u5B50\uFF0C\u4F1A\u66F4\u5BB9\u6613\u63A5\u4F4F\u4E0B\u4E00\u8F6E\u8FFD\u95EE\u3002", tone: "accent" })] }), _jsxs("div", { className: "grid gap-3 md:grid-cols-2 xl:grid-cols-4", children: [_jsx(AuditFactCard, { label: "\u6A21\u5F0F", value: formatInterviewModeLabel(audit.mode) }), _jsx(AuditFactCard, { label: "\u9636\u6BB5", value: formatInterviewPhaseLabel(audit.state?.phase) }), _jsx(AuditFactCard, { label: "\u8F6E\u6B21", value: typeof audit.state?.round === "number" ? `第 ${audit.state.round} 轮` : "未知" }), _jsx(AuditFactCard, { label: "\u538B\u529B / Prompt", value: `${typeof audit.state?.difficulty === "number" ? audit.state.difficulty : "未知"} · ${audit.promptVersion ?? "legacy"}` })] }), _jsx(AuditActionGrid, { audit: audit }), _jsx(AuditChipSection, { title: "\u7CFB\u7EDF\u89C9\u5F97\u8FD8\u6CA1\u8BB2\u900F\u7684\u70B9", items: weakSignals, tone: "warning", confidence: weakSignalConfidence, formatSignalLabel: formatSignalLabel }), _jsx(AuditChipSection, { title: "\u7CFB\u7EDF\u5DF2\u7ECF\u542C\u5230\u7684\u5F3A\u9879", items: strongSignals, tone: "success", confidence: strongSignalConfidence, formatSignalLabel: formatSignalLabel }), _jsx(AuditChipSection, { title: "\u5386\u53F2\u753B\u50CF\u6B63\u5728\u63D0\u9192\u7684\u70B9", items: profileFocus, tone: "neutral", formatSignalLabel: formatSignalLabel }), _jsx(AuditChipSection, { title: "\u4E0B\u4E00\u95EE\u53EF\u80FD\u7EE7\u7EED\u76EF\u4F4F", items: recommendedFocus, tone: "accent", formatSignalLabel: formatSignalLabel }), _jsxs("div", { className: "grid gap-3 md:grid-cols-3", children: [_jsx(AuditBooleanCard, { label: "\u56DE\u7B54\u662F\u4E0D\u662F\u504F\u6CDB", active: Boolean(audit.analysis?.tooGeneric), tone: "warning", positiveText: "\u7CFB\u7EDF\u89C9\u5F97\u8FD9\u8F6E\u6709\u70B9\u6CDB", negativeText: "\u8FD9\u8F6E\u8868\u8FBE\u4E0D\u7B97\u7A7A\u6CDB" }), _jsx(AuditBooleanCard, { label: "\u6709\u6CA1\u6709\u8BB2 tradeoff", active: Boolean(audit.analysis?.hasTradeoff), tone: "success", positiveText: "\u5DF2\u7ECF\u8BB2\u5230\u53D6\u820D\u5224\u65AD", negativeText: "\u8FD8\u6CA1\u660E\u663E\u542C\u5230\u53D6\u820D" }), _jsx(AuditBooleanCard, { label: "\u6709\u6CA1\u6709\u843D\u5230\u5B9E\u73B0\u7EC6\u8282", active: Boolean(audit.analysis?.hasConcreteImplementation), tone: "success", positiveText: "\u5DF2\u7ECF\u89E6\u8FBE\u5B9E\u73B0\u5C42", negativeText: "\u8FD8\u9700\u8981\u66F4\u5177\u4F53\u5B9E\u73B0" })] }), history.length > 0 ? (_jsxs("div", { children: [_jsx("p", { className: "tech-label text-[0.68rem] text-[rgba(97,123,150,0.62)]", children: "\u8DE8\u8F6E\u7B56\u7565\u53D8\u5316" }), _jsx("div", { className: "mt-3 space-y-3", children: history.map((entry, index) => {
                                    const node = traceNodes.find((item) => item.round === entry.round) ?? null;
                                    return (_jsxs("button", { type: "button", onClick: () => onFocusMessage(node?.messageId), className: "panel-card-soft block w-full text-left transition-colors hover:border-[rgba(59,130,246,0.18)] hover:bg-[rgba(248,250,255,0.98)]", children: [_jsxs("div", { className: "flex flex-wrap items-center gap-2", children: [_jsxs("span", { className: "chip-neutral px-2.5 py-1 text-[0.68rem] tracking-[0.18em]", children: ["\u7B2C ", entry.round, " \u8F6E"] }), _jsx("span", { className: "chip-info px-2.5 py-1 text-[0.68rem] tracking-[0.18em]", children: formatInterviewPhaseLabel(entry.phase) }), _jsxs("span", { className: "chip-neutral px-2.5 py-1 text-[0.68rem] tracking-[0.18em]", children: ["\u538B\u529B ", entry.difficulty] }), entry.reason ? (_jsx("span", { className: "chip-warning px-2.5 py-1 text-[0.68rem] tracking-[0.18em]", children: formatDecisionReasonLabel(entry.reason) })) : null] }), entry.explanation ? (_jsx("p", { className: "mt-3 text-sm leading-7 text-[rgba(72,91,114,0.86)]", children: entry.explanation })) : null, node?.question ? (_jsxs("p", { className: "mt-2 text-sm leading-7 text-[rgba(71,85,105,0.88)]", children: ["\u95EE\u9898\uFF1A", node.question] })) : null, entry.weakSignals?.length || entry.strongSignals?.length ? (_jsxs("div", { className: "mt-3 flex flex-wrap gap-2", children: [entry.weakSignals?.map((signal) => (_jsx(SignalChip, { label: formatSignalLabel(signal), confidence: entry.weakSignalConfidence?.[signal], className: "chip-danger px-2.5 py-1 text-[0.68rem] tracking-[0.12em]" }, `strategy-weak-${entry.round}-${signal}`))), entry.strongSignals?.map((signal) => (_jsx(SignalChip, { label: formatSignalLabel(signal), confidence: entry.strongSignalConfidence?.[signal], className: "chip-success px-2.5 py-1 text-[0.68rem] tracking-[0.12em]" }, `strategy-strong-${entry.round}-${signal}`)))] })) : null] }, `${entry.round}-${entry.phase}-${index}`));
                                }) })] })) : null, replay?.steps?.length ? (_jsxs("div", { children: [_jsx("p", { className: "tech-label text-[0.68rem] text-[rgba(97,123,150,0.62)]", children: "\u6309 Run \u56DE\u653E\u51B3\u7B56\u8DEF\u5F84" }), _jsx("div", { className: "mt-3 space-y-3", children: replay.steps.map((step, index) => (_jsxs("div", { className: "rounded-[1.15rem] border border-[rgba(214,222,234,0.92)] bg-[rgba(248,250,255,0.92)] px-4 py-4", children: [_jsxs("div", { className: "flex flex-wrap items-center gap-2", children: [_jsxs("span", { className: "chip-neutral px-2.5 py-1 text-[0.68rem] tracking-[0.18em]", children: ["#", index + 1] }), _jsx("span", { className: "chip-info px-2.5 py-1 text-[0.68rem] tracking-[0.18em]", children: step.kind }), _jsx("span", { className: "chip-neutral px-2.5 py-1 text-[0.68rem] tracking-[0.18em]", children: step.name }), step.decisionReason ? (_jsx("span", { className: "chip-warning px-2.5 py-1 text-[0.68rem] tracking-[0.18em]", children: formatDecisionReasonLabel(step.decisionReason) })) : null, step.status ? (_jsx("span", { className: "chip-neutral px-2.5 py-1 text-[0.68rem] tracking-[0.18em]", children: step.status })) : null] }), step.summary ? _jsx("p", { className: "mt-3 text-sm leading-7 text-[rgba(72,91,114,0.86)]", children: step.summary }) : null, step.decisionExplanation ? (_jsx("p", { className: "mt-2 text-sm leading-7 text-[rgba(71,85,105,0.84)]", children: step.decisionExplanation })) : null, (step.requires?.length ?? 0) > 0 ? (_jsx("div", { className: "mt-3 flex flex-wrap gap-2", children: step.requires?.map((item) => (_jsxs("span", { className: "chip-muted px-2.5 py-1 text-[0.68rem] tracking-[0.12em]", children: ["\u4F9D\u8D56 \u00B7 ", item] }, `${step.id}-${item}`))) })) : null, step.promptSummary ? (_jsxs("p", { className: "mt-3 text-xs leading-6 text-[rgba(100,116,139,0.76)]", children: ["\u8F93\u5165\u6458\u8981\uFF1A", step.promptSummary] })) : null] }, step.id))) })] })) : null, replay ? (_jsxs("div", { className: "rounded-[1.35rem] border border-[rgba(214,222,234,0.92)] bg-[rgba(248,250,252,0.94)] px-4 py-4", children: [_jsxs("div", { className: "flex items-start gap-3", children: [_jsx("div", { className: "rounded-full border border-[rgba(226,232,240,0.92)] bg-white/80 p-2 text-[rgba(71,85,105,0.78)]", children: _jsx(ShieldAlert, { className: "h-4 w-4" }) }), _jsxs("div", { children: [_jsx("p", { className: "tech-label text-[0.68rem] text-[rgba(97,123,150,0.62)]", children: "\u9AD8\u7EA7\u6392\u969C / \u672C\u5730\u8C03\u8BD5\u590D\u73B0" }), _jsx("p", { className: "mt-2 text-sm leading-7 text-[rgba(71,85,105,0.84)]", children: "\u8FD9\u90E8\u5206\u66F4\u504F\u5F00\u53D1\u6392\u969C\uFF0C\u7528\u6765\u590D\u73B0\u8FD9\u6B21\u51B3\u7B56\u8F93\u5165\uFF0C\u4E0D\u5F71\u54CD\u6B63\u5E38\u590D\u76D8\u9605\u8BFB\u3002" })] })] }), _jsxs("div", { className: "mt-3 grid gap-3 md:grid-cols-2", children: [_jsx(AuditFactCard, { label: "\u64CD\u4F5C", value: replay.reproduction.operation || "DecideNextStep" }), _jsx(AuditFactCard, { label: "Prompt \u7248\u672C", value: replay.input.promptVersion ?? "legacy" }), _jsx(AuditFactCard, { label: "\u6280\u80FD", value: replay.input.skill ?? "auto" }), _jsx(AuditFactCard, { label: "\u4E3B\u6A21\u5F0F", value: formatInterviewModeLabel(replay.reproduction.config.mode) })] }), replay.input.promptSummary ? (_jsxs("p", { className: "mt-3 text-sm leading-7 text-[rgba(71,85,105,0.84)]", children: ["Prompt \u6458\u8981\uFF1A", replay.input.promptSummary] })) : null, replay.input.latestAnswerSummary ? (_jsxs("p", { className: "mt-2 text-sm leading-7 text-[rgba(71,85,105,0.84)]", children: ["\u6700\u8FD1\u56DE\u7B54\u6458\u8981\uFF1A", replay.input.latestAnswerSummary] })) : null, _jsxs("div", { className: "mt-3 flex flex-wrap gap-2", children: [(replay.input.skillFocuses ?? []).map((focus) => (_jsxs("span", { className: "chip-accent px-3 py-1.5 text-xs normal-case tracking-[0.04em]", children: ["skill focus \u00B7 ", formatSignalLabel(focus)] }, `skill-${focus}`))), (replay.input.profileFocus ?? []).map((focus) => (_jsxs("span", { className: "chip-warning px-3 py-1.5 text-xs normal-case tracking-[0.04em]", children: ["profile focus \u00B7 ", formatSignalLabel(focus)] }, `profile-${focus}`)))] }), _jsx("pre", { className: "mt-4 overflow-x-auto rounded-[1rem] border border-[rgba(214,222,234,0.92)] bg-[rgba(255,255,255,0.9)] px-4 py-4 text-xs leading-6 text-[rgba(100,116,139,0.9)]", children: JSON.stringify(replay.reproduction, null, 2) })] })) : null] }))] }));
}
function SummaryCallout({ icon, label, title, text, tone, }) {
    const toneClass = tone === "primary"
        ? "border-[rgba(191,219,254,0.92)] bg-[linear-gradient(135deg,rgba(239,246,255,0.98),rgba(248,250,255,0.98))]"
        : tone === "accent"
            ? "border-[rgba(216,180,254,0.4)] bg-[linear-gradient(135deg,rgba(250,245,255,0.98),rgba(255,255,255,0.98))]"
            : "border-[rgba(214,222,234,0.92)] bg-[rgba(248,250,252,0.94)]";
    return (_jsx("div", { className: `rounded-[1.35rem] border px-4 py-4 ${toneClass}`, children: _jsxs("div", { className: "flex items-start gap-3", children: [_jsx("div", { className: "rounded-full border border-white/80 bg-white/80 p-2 text-[rgb(37,99,235)]", children: icon }), _jsxs("div", { children: [_jsx("p", { className: "tech-label text-[0.64rem] text-[rgba(97,123,150,0.72)]", children: label }), _jsx("h3", { className: "mt-2 text-base font-semibold leading-7 text-[rgb(51,65,85)]", children: title }), _jsx("p", { className: "mt-2 text-sm leading-7 text-[rgba(71,85,105,0.84)]", children: text })] })] }) }));
}
function AuditFactCard({ label, value }) {
    return (_jsxs("div", { className: "panel-card-soft", children: [_jsx("p", { className: "tech-label text-[0.64rem] text-[rgba(97,123,150,0.62)]", children: label }), _jsx("p", { className: "mt-2 text-sm leading-6 text-[rgb(31,41,55)]", children: value })] }));
}
function AuditBooleanCard({ label, active, tone, positiveText, negativeText, }) {
    const className = active
        ? tone === "warning"
            ? "border-[rgba(253,230,138,0.92)] bg-[rgba(254,252,232,0.96)] text-[rgb(161,98,7)]"
            : "border-[rgba(187,247,208,0.94)] bg-[rgba(240,253,244,0.94)] text-[rgb(21,128,61)]"
        : "border-[rgba(226,231,239,0.88)] bg-[rgba(248,250,252,0.92)] text-[rgba(100,116,139,0.78)]";
    return (_jsxs("div", { className: `rounded-[1.2rem] border px-4 py-4 text-sm ${className}`, children: [_jsx("p", { className: "tech-label text-[0.64rem] opacity-80", children: label }), _jsxs("div", { className: "mt-2 flex items-start gap-2 leading-6", children: [active ? (tone === "warning" ? _jsx(ShieldAlert, { className: "mt-0.5 h-4 w-4 shrink-0" }) : _jsx(CheckCircle2, { className: "mt-0.5 h-4 w-4 shrink-0" })) : (_jsx(CircleOff, { className: "mt-0.5 h-4 w-4 shrink-0 opacity-70" })), _jsx("p", { children: active ? positiveText : negativeText })] })] }));
}
function AuditActionGrid({ audit }) {
    const actions = [
        {
            label: "继续当前主题",
            helper: "系统觉得这个点还值得继续往下挖。",
            active: Boolean(audit.decision?.keepTopic),
        },
        {
            label: "切换下个主题",
            helper: "系统准备转去别的能力点确认广度。",
            active: Boolean(audit.decision?.switchTopic),
        },
        {
            label: "提高压力",
            helper: "系统会用更严格、更快的追问方式继续问。",
            active: Boolean(audit.decision?.escalatePressure),
        },
        {
            label: "触发反驳",
            helper: "系统会主动挑战你的方案，确认你是否站得住。",
            active: Boolean(audit.decision?.triggerAdversarial),
        },
        {
            label: "提高难度",
            helper: "系统会把问题拉到更复杂场景或更高要求。",
            active: Boolean(audit.decision?.increaseDifficulty),
        },
    ];
    return (_jsxs("div", { children: [_jsx("p", { className: "tech-label text-[0.68rem] text-[rgba(97,123,150,0.62)]", children: "\u4E0B\u4E00\u6B65\u52A8\u4F5C" }), _jsx("div", { className: "mt-3 grid gap-3 md:grid-cols-2 xl:grid-cols-3", children: actions.map((action) => (_jsx("div", { className: `rounded-[1.2rem] border px-4 py-4 text-sm ${action.active
                        ? "border-[rgba(191,219,254,0.92)] bg-[rgba(239,246,255,0.96)] text-[rgb(29,78,216)]"
                        : "border-[rgba(226,231,239,0.88)] bg-[rgba(248,250,252,0.92)] text-[rgba(100,116,139,0.78)]"}`, children: _jsxs("div", { className: "flex items-start gap-2", children: [action.active ? _jsx(ArrowRight, { className: "mt-0.5 h-4 w-4 shrink-0" }) : _jsx(CircleOff, { className: "mt-0.5 h-4 w-4 shrink-0 opacity-70" }), _jsxs("div", { children: [_jsx("p", { className: "leading-6", children: action.label }), _jsx("p", { className: "mt-1 text-xs leading-5 opacity-80", children: action.helper })] })] }) }, action.label))) })] }));
}
function AuditChipSection({ title, items, tone, confidence, formatSignalLabel, }) {
    if (items.length === 0) {
        return null;
    }
    const chipClass = tone === "warning"
        ? "chip-warning"
        : tone === "success"
            ? "chip-success"
            : tone === "accent"
                ? "chip-accent"
                : "chip-neutral";
    return (_jsxs("div", { children: [_jsx("p", { className: "tech-label text-[0.68rem] text-[rgba(97,123,150,0.62)]", children: title }), _jsx("div", { className: "mt-3 flex flex-wrap gap-2", children: items.map((item) => (_jsx(SignalChip, { label: formatSignalLabel(item), confidence: confidence?.[item], className: `${chipClass} px-3 py-1.5 text-xs normal-case tracking-[0.04em]` }, `${title}-${item}`))) })] }));
}
function SignalChip({ label, confidence, className, }) {
    return (_jsxs("span", { className: `${className} inline-flex items-center gap-2`, children: [_jsx("span", { children: label }), typeof confidence === "number" ? (_jsx("span", { className: "rounded-full bg-[rgba(255,255,255,0.72)] px-2 py-0.5 text-[0.64rem] font-medium tracking-[0.04em] text-[rgba(15,23,42,0.78)]", children: formatConfidence(confidence) })) : null] }));
}
function formatConfidence(value) {
    const percentage = Math.round(Math.max(0, Math.min(1, value)) * 100);
    return `${percentage}%`;
}
function getPrimaryAction(audit) {
    if (!audit?.decision) {
        return {
            title: "等这一轮结束后再判断",
            description: "后端会结合这轮回答、阶段和历史画像，决定是继续追问、切题还是加压。",
        };
    }
    if (audit.decision.triggerAdversarial) {
        return {
            title: "系统准备主动反驳你的方案",
            description: "说明系统想确认你的判断能不能扛住质疑，下一答最好补上边界、取舍和兜底方案。",
        };
    }
    if (audit.decision.escalatePressure) {
        return {
            title: "系统会开始更快、更严地追问",
            description: "通常发生在压力测试或收尾阶段，你需要更短、更准地给出判断和依据。",
        };
    }
    if (audit.decision.switchTopic) {
        return {
            title: "系统准备切到下一个主题点",
            description: "当前点可能已经覆盖得差不多了，下一问更像在确认能力广度。",
        };
    }
    if (audit.decision.increaseDifficulty) {
        return {
            title: "系统会把题目抬到更复杂场景",
            description: "下一问大概率会加约束、加边界条件或提高系统规模要求。",
        };
    }
    return {
        title: "系统会继续顺着当前点往下追",
        description: "这说明系统觉得你还没把这个能力点讲透，下一答最好更具体、更贴近实现。",
    };
}
