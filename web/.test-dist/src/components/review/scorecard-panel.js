import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { ArrowRight, CheckCircle2, Gauge, RotateCcw, Target } from "lucide-react";
import { Button } from "../ui/button.js";
export function ScorecardPanel({ scorecard, outputStyle, onRetry, retryDisabled, latestFailure, }) {
    const anchors = scorecard?.anchors ?? [];
    const wantsStudyPlan = outputStyle === "interview_plus_score_and_study_plan";
    const strengths = scorecard?.strengths ?? [];
    const gaps = scorecard?.gaps ?? [];
    const improvements = scorecard?.improvements ?? [];
    const studyPlan = scorecard?.studyPlan ?? [];
    const summary = scorecard?.summary?.trim() ?? "";
    const dimensionScores = scorecard?.dimensionScores ?? [];
    const priorityFocus = improvements[0] ?? gaps[0] ?? anchors[0] ?? "";
    const averageScore = dimensionScores.length > 0
        ? dimensionScores.reduce((total, dimension) => total + dimension.score, 0) / dimensionScores.length
        : null;
    const averagePercentage = averageScore ? Math.round((averageScore / 5) * 100) : null;
    const overallTone = getScoreTone(averageScore);
    const overallLabel = averageScore ? getOverallLabel(averageScore) : "等待评分";
    const nextAction = improvements[0] ?? gaps[0] ?? "等评分生成后，这里会告诉你下一场最该优先补什么。";
    const standoutStrength = strengths[0] ?? anchors[0] ?? "等结构化评分生成后，这里会总结你最稳的一项表现。";
    return (_jsxs("section", { className: "panel-card", children: [_jsx("p", { className: "tech-label text-[0.68rem] text-[rgba(97,123,150,0.62)]", children: "\u8BC4\u5206\u5361" }), _jsx("h2", { className: "mt-2 font-display text-2xl text-[rgb(72,91,114)]", children: scorecard?.title ?? "等待评分结果" }), _jsx("p", { className: "mt-2 max-w-[64ch] text-sm leading-7 text-[rgba(115,137,161,0.8)]", children: summary || "先看这里判断本场整体表现，再向下查看关键短板、建议和训练计划。" }), _jsxs("div", { className: "mt-5 grid gap-3 xl:grid-cols-[1.25fr_1fr_1fr]", children: [_jsxs("div", { className: "rounded-[1.45rem] border border-[rgba(191,219,254,0.92)] bg-[linear-gradient(135deg,rgba(239,246,255,0.98),rgba(248,250,255,0.98))] px-5 py-5", children: [_jsxs("div", { className: "flex items-start justify-between gap-4", children: [_jsxs("div", { children: [_jsx("p", { className: "tech-label text-[0.64rem] text-[rgb(29,78,216)]", children: "\u672C\u573A\u6574\u4F53\u8868\u73B0" }), _jsx("h3", { className: "mt-2 text-xl font-semibold text-[rgb(30,64,175)]", children: overallLabel })] }), _jsx("div", { className: "rounded-full border border-[rgba(191,219,254,0.92)] bg-white/80 p-2 text-[rgb(37,99,235)]", children: _jsx(Gauge, { className: "h-4 w-4" }) })] }), _jsxs("div", { className: "mt-4", children: [_jsx("div", { className: "h-2.5 overflow-hidden rounded-full bg-white/80", children: _jsx("div", { className: `h-full rounded-full transition-all ${overallTone.barClass}`, style: { width: `${averagePercentage ?? 0}%` } }) }), _jsxs("div", { className: "mt-3 flex items-end justify-between gap-4", children: [_jsxs("div", { children: [_jsxs("p", { className: "text-3xl font-semibold tracking-[-0.04em] text-[rgb(15,23,42)]", children: [averageScore ? averageScore.toFixed(1) : "--", _jsx("span", { className: "ml-1 text-base font-medium text-[rgba(71,85,105,0.72)]", children: "/ 5" })] }), _jsx("p", { className: "mt-1 text-sm text-[rgba(71,85,105,0.78)]", children: averagePercentage ? `约领先 ${averagePercentage}% 的目标完成度` : "等待后端评分返回结构化结果" })] }), _jsx("span", { className: `rounded-full px-3 py-1 text-xs font-medium ${overallTone.badgeClass}`, children: overallTone.label })] })] })] }), _jsx(OverviewCallout, { icon: _jsx(Target, { className: "h-4 w-4" }), label: "\u5F53\u524D\u6700\u8BE5\u5148\u4FEE", text: priorityFocus || "先完成一轮评分后，这里会给出最优先的修正方向。", tone: "warning" }), _jsx(OverviewCallout, { icon: _jsx(CheckCircle2, { className: "h-4 w-4" }), label: "\u4F60\u8FD9\u6B21\u6700\u7A33\u7684\u70B9", text: standoutStrength, tone: "success" })] }), _jsx("div", { className: "mt-4 rounded-[1.3rem] border border-[rgba(214,222,234,0.92)] bg-[rgba(248,250,252,0.94)] px-4 py-4", children: _jsxs("div", { className: "flex items-start gap-3", children: [_jsx("div", { className: "rounded-full border border-[rgba(191,219,254,0.92)] bg-[rgba(239,246,255,0.96)] p-2 text-[rgb(37,99,235)]", children: _jsx(ArrowRight, { className: "h-4 w-4" }) }), _jsxs("div", { children: [_jsx("p", { className: "tech-label text-[0.64rem] text-[rgba(97,123,150,0.68)]", children: "\u4E0B\u4E00\u6B65\u6700\u503C\u5F97\u505A" }), _jsx("p", { className: "mt-2 text-sm leading-7 text-[rgb(51,65,85)]", children: nextAction })] })] }) }), dimensionScores.length > 0 ? (_jsxs("div", { className: "mt-6", children: [_jsx("p", { className: "tech-label text-[0.68rem] text-[rgba(97,123,150,0.62)]", children: "\u7EF4\u5EA6\u5206\u6570" }), _jsx("div", { className: "mt-3 grid gap-3 sm:grid-cols-2", children: dimensionScores.map((dimension) => {
                            const tone = getScoreTone(dimension.score);
                            const percentage = Math.round((dimension.score / 5) * 100);
                            return (_jsxs("div", { className: "rounded-[1.35rem] border border-[rgba(214,222,234,0.92)] bg-[rgba(255,255,255,0.95)] px-4 py-4 shadow-[0_16px_42px_-34px_rgba(15,23,42,0.38)]", children: [_jsxs("div", { className: "flex items-start justify-between gap-4", children: [_jsxs("div", { children: [_jsx("p", { className: "text-sm font-semibold text-[rgb(51,65,85)]", children: dimension.name }), _jsx("p", { className: "mt-1 text-xs uppercase tracking-[0.16em] text-[rgba(100,116,139,0.74)]", children: tone.label })] }), _jsxs("span", { className: `rounded-full px-3 py-1 text-xs font-medium ${tone.badgeClass}`, children: [dimension.score, "/5"] })] }), _jsxs("div", { className: "mt-4", children: [_jsx("div", { className: "h-2.5 overflow-hidden rounded-full bg-[rgba(226,232,240,0.72)]", children: _jsx("div", { className: `h-full rounded-full transition-all ${tone.barClass}`, style: { width: `${percentage}%` } }) }), _jsxs("div", { className: "mt-2 flex items-center justify-between text-xs text-[rgba(100,116,139,0.78)]", children: [_jsx("span", { children: "\u5F53\u524D\u5B8C\u6210\u5EA6" }), _jsxs("span", { children: [percentage, "%"] })] })] }), dimension.rationale ? (_jsx("p", { className: "mt-4 text-sm leading-6 text-[rgba(71,85,105,0.84)]", children: dimension.rationale })) : null] }, `${dimension.name}-${dimension.score}`));
                        }) })] })) : null, anchors.length === 0 ? (_jsx("p", { className: "mt-5 rounded-[1.3rem] border border-dashed border-[rgba(153,191,201,0.18)] px-4 py-5 text-sm leading-7 text-[rgba(115,137,161,0.78)]", children: "\u8FD0\u884C\u5230\u8FBE\u8BC4\u5206\u9636\u6BB5\u540E\uFF0C\u7ED3\u6784\u5316\u8BC4\u5206\u4F1A\u663E\u793A\u5728\u8FD9\u91CC\u3002" })) : (_jsx("div", { className: "mt-5 space-y-3", children: anchors.map((anchor) => (_jsx("div", { className: "panel-card-soft text-sm leading-7 text-[rgb(72,91,114)]", children: anchor }, anchor))) })), _jsx(ScoreSection, { title: "\u4F18\u52BF", items: strengths, tone: "neutral" }), _jsx(ScoreSection, { title: "\u77ED\u677F", items: gaps, tone: "warning" }), _jsx(ScoreSection, { title: "\u6539\u8FDB\u5EFA\u8BAE", items: improvements, tone: "success" }), wantsStudyPlan ? (_jsxs("div", { className: "mt-6", children: [_jsx("p", { className: "tech-label text-[0.68rem] text-[rgba(97,123,150,0.62)]", children: "\u5B66\u4E60\u8BA1\u5212" }), studyPlan.length === 0 ? (_jsx("p", { className: "mt-3 rounded-[1.3rem] border border-dashed border-[rgba(153,191,201,0.18)] px-4 py-5 text-sm leading-7 text-[rgba(115,137,161,0.78)]", children: "\u672C\u6B21\u8FD0\u884C\u8981\u6C42\u751F\u6210\u5B66\u4E60\u8BA1\u5212\uFF0C\u4F46\u540E\u7AEF\u6682\u672A\u8FD4\u56DE\u7ED3\u6784\u5316\u5B66\u4E60\u9879\u3002" })) : (_jsx("div", { className: "mt-3 space-y-3", children: studyPlan.map((item, index) => (_jsx("div", { className: "rounded-[1.3rem] border border-[rgba(187,247,208,0.94)] bg-[rgba(240,253,244,0.94)] px-4 py-4 text-sm leading-7 text-[rgb(21,128,61)]", children: item }, `${index}-${item}`))) }))] })) : null, latestFailure && onRetry ? (_jsxs("div", { className: "mt-6 rounded-[1.4rem] border border-[rgba(191,219,254,0.92)] bg-[rgba(239,246,255,0.96)] px-4 py-4", children: [_jsx("p", { className: "font-serif text-xs uppercase tracking-[0.28em] text-[rgb(0,102,255)]", children: "\u6062\u590D\u8FD0\u884C" }), _jsx("p", { className: "mt-2 text-sm leading-7 text-[rgb(0,102,255)]", children: "\u6700\u8FD1\u4E00\u6B21\u8FD0\u884C\u5931\u8D25\u3002\u4FEE\u590D\u963B\u585E\u95EE\u9898\u540E\uFF0C\u53EF\u4EE5\u57FA\u4E8E\u5F53\u524D\u4EFB\u52A1\u914D\u7F6E\u91CD\u65B0\u542F\u52A8\u4E00\u6B21\u8FD0\u884C\u3002" }), _jsxs(Button, { type: "button", onClick: onRetry, disabled: retryDisabled, className: "mt-4 rounded-full bg-[rgb(0,102,255)] px-4 text-primary-foreground hover:bg-[rgb(0,88,220)]", children: [_jsx(RotateCcw, { className: "mr-2 h-4 w-4" }), "\u542F\u52A8\u6062\u590D\u8FD0\u884C"] })] })) : null] }));
}
function OverviewCallout({ icon, label, text, tone, }) {
    const className = tone === "warning"
        ? "border-[rgba(253,230,138,0.92)] bg-[rgba(254,252,232,0.96)] text-[rgb(161,98,7)]"
        : "border-[rgba(187,247,208,0.94)] bg-[rgba(240,253,244,0.94)] text-[rgb(21,128,61)]";
    return (_jsx("div", { className: `rounded-[1.35rem] border px-4 py-4 ${className}`, children: _jsxs("div", { className: "flex items-start gap-3", children: [_jsx("div", { className: "rounded-full border border-white/70 bg-white/70 p-2", children: icon }), _jsxs("div", { children: [_jsx("p", { className: "tech-label text-[0.64rem] opacity-80", children: label }), _jsx("p", { className: "mt-2 text-sm leading-7", children: text })] })] }) }));
}
function ScoreSection({ title, items, tone, }) {
    if (items.length === 0) {
        return null;
    }
    const toneClass = tone === "warning"
        ? "border-[rgba(253,230,138,0.92)] bg-[rgba(254,252,232,0.96)] text-[rgb(161,98,7)]"
        : tone === "success"
            ? "border-[rgba(187,247,208,0.94)] bg-[rgba(240,253,244,0.94)] text-[rgb(21,128,61)]"
            : "border-[rgba(153,191,201,0.16)] bg-[rgba(252,255,255,0.88)] text-[rgb(72,91,114)]";
    return (_jsxs("div", { className: "mt-6", children: [_jsx("p", { className: "tech-label text-[0.68rem] text-[rgba(97,123,150,0.62)]", children: title }), _jsx("div", { className: "mt-3 space-y-3", children: items.map((item, index) => (_jsx("div", { className: `rounded-[1.3rem] border px-4 py-4 text-sm leading-7 ${toneClass}`, children: item }, `${title}-${index}-${item}`))) })] }));
}
function getScoreTone(score) {
    if (typeof score !== "number") {
        return {
            label: "等待评分",
            badgeClass: "bg-[rgba(241,245,249,0.94)] text-[rgba(100,116,139,0.88)]",
            barClass: "bg-[rgba(148,163,184,0.72)]",
        };
    }
    if (score >= 4) {
        return {
            label: "表现稳",
            badgeClass: "bg-[rgba(220,252,231,0.96)] text-[rgb(21,128,61)]",
            barClass: "bg-[linear-gradient(90deg,rgba(22,163,74,0.88),rgba(74,222,128,0.92))]",
        };
    }
    if (score >= 3) {
        return {
            label: "还可以再稳",
            badgeClass: "bg-[rgba(254,249,195,0.96)] text-[rgb(161,98,7)]",
            barClass: "bg-[linear-gradient(90deg,rgba(245,158,11,0.9),rgba(250,204,21,0.9))]",
        };
    }
    return {
        label: "需要重点补",
        badgeClass: "bg-[rgba(254,226,226,0.96)] text-[rgb(185,28,28)]",
        barClass: "bg-[linear-gradient(90deg,rgba(239,68,68,0.9),rgba(251,113,133,0.92))]",
    };
}
function getOverallLabel(score) {
    if (score >= 4.2) {
        return "这场发挥很稳，可以开始看更高阶的优化";
    }
    if (score >= 3.3) {
        return "整体合格，但还有几处会被继续深挖";
    }
    return "目前短板比较明确，建议先补关键表达和实现层细节";
}
