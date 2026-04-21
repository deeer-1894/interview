import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { CircleDashed, Flame, Hourglass, Radar, Sparkles, Waypoints } from "lucide-react";
export function RunStatusBar({ status, statusLabel, modeLabel, personaLabel, phaseLabel, timeBudget, hasReviewData, }) {
    const statusTone = status === "failed" || status === "cancelled"
        ? "border-[rgba(254,205,211,0.92)] bg-[rgba(255,241,242,0.96)] text-[rgb(190,24,93)]"
        : status === "waiting_clarify"
            ? "border-[rgba(191,219,254,0.92)] bg-[rgba(239,246,255,0.96)] text-[rgb(29,78,216)]"
            : status === "running" || status === "resuming"
                ? "border-[rgba(186,230,253,0.92)] bg-[rgba(236,254,255,0.96)] text-[rgb(14,116,144)]"
                : "border-[rgba(187,247,208,0.92)] bg-[rgba(240,253,244,0.96)] text-[rgb(21,128,61)]";
    const StatusIcon = status === "failed" || status === "cancelled"
        ? Flame
        : status === "waiting_clarify"
            ? Hourglass
            : status === "running" || status === "resuming"
                ? CircleDashed
                : Waypoints;
    return (_jsxs("div", { className: "status-strip rounded-[1.4rem] border px-4 py-3", children: [_jsxs("div", { className: "flex flex-wrap items-start justify-between gap-3", children: [_jsxs("div", { children: [_jsx("p", { className: "tech-label text-[0.64rem] text-[rgba(71,85,105,0.58)]", children: "\u5F53\u524D\u8BAD\u7EC3" }), _jsxs("h2", { className: "mt-1 font-display text-[1.2rem] text-[rgb(15,23,42)]", children: [modeLabel, " \u00B7 ", phaseLabel] }), _jsxs("p", { className: "mt-1 text-sm text-[rgba(51,65,85,0.74)]", children: [personaLabel, " \u00B7 \u65F6\u95F4\u9884\u7B97 ", timeBudget, hasReviewData ? " · 已累计复盘结果" : ""] })] }), _jsxs("div", { className: `status-badge gap-2 px-3 py-1.5 text-[0.76rem] ${statusTone}`, children: [_jsx(StatusIcon, { className: `h-3.5 w-3.5 ${status === "running" || status === "resuming" ? "animate-spin" : ""}` }), statusLabel] })] }), _jsxs("div", { className: "mt-3 flex flex-wrap gap-2", children: [_jsx(StatusPill, { icon: _jsx(Radar, { className: "h-3.5 w-3.5" }), label: "\u6A21\u5F0F", value: modeLabel }), _jsx(StatusPill, { icon: _jsx(Sparkles, { className: "h-3.5 w-3.5" }), label: "\u4EBA\u683C", value: personaLabel }), _jsx(StatusPill, { icon: _jsx(Waypoints, { className: "h-3.5 w-3.5" }), label: "\u9636\u6BB5", value: phaseLabel }), _jsx(StatusPill, { icon: _jsx(Hourglass, { className: "h-3.5 w-3.5" }), label: "\u9884\u7B97", value: timeBudget })] })] }));
}
function StatusPill({ icon, label, value }) {
    return (_jsxs("div", { className: "status-badge gap-2 border-[rgba(191,219,254,0.72)] bg-[rgba(255,255,255,0.82)] px-3 py-1.5 text-sm text-[rgb(30,41,59)]", children: [_jsx("span", { className: "inline-flex h-6 w-6 items-center justify-center rounded-full bg-[rgba(239,246,255,0.96)] text-[rgb(29,78,216)]", children: icon }), _jsx("span", { className: "text-[rgba(71,85,105,0.72)]", children: label }), _jsx("span", { className: "font-medium text-[rgb(15,23,42)]", children: value })] }));
}
