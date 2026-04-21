import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
export function CandidateProfilePanel({ profile, reviewSummary, getPersonaLabel, }) {
    const dimensions = profile?.dimensions ?? [];
    const radar = profile?.radar ?? [];
    const growthCurves = profile?.growthCurves ?? [];
    const personaUsage = profile?.personaUsage ?? [];
    const stableStrengths = profile?.stableStrengths ?? [];
    const recurringGaps = profile?.recurringGaps ?? [];
    const recommendedFocus = profile?.recommendedFocus ?? [];
    const recentChanges = profile?.recentChanges ?? [];
    return (_jsxs("section", { className: "panel-card", children: [_jsxs("div", { className: "flex items-start justify-between gap-4", children: [_jsxs("div", { children: [_jsx("p", { className: "tech-label text-[0.68rem] text-[rgba(97,123,150,0.62)]", children: "\u8DE8\u4F1A\u8BDD\u753B\u50CF" }), _jsx("h2", { className: "mt-2 font-display text-2xl text-[rgb(72,91,114)]", children: profile?.interviewCount ? `累计 ${profile.interviewCount} 场面试` : "等待画像沉淀" })] }), profile?.lastPersona ? (_jsxs("div", { className: "chip-info px-3 py-1.5 text-[0.68rem] tracking-[0.2em]", children: ["\u6700\u8FD1\u4EBA\u683C \u00B7 ", getPersonaLabel(profile.lastPersona)] })) : null] }), _jsx("p", { className: "mt-3 max-w-[64ch] text-sm leading-7 text-[rgba(115,137,161,0.78)]", children: "\u8FD9\u4E9B\u8DE8\u4F1A\u8BDD\u7ED3\u8BBA\u4F1A\u9010\u6E10\u5F71\u54CD\u4E0B\u4E00\u573A\u9762\u8BD5\u7684\u8FFD\u95EE\u91CD\u70B9\uFF0C\u8BA9\u8BAD\u7EC3\u8D8A\u6765\u8D8A\u9488\u5BF9\u4F60\u7684\u771F\u5B9E\u8584\u5F31\u9879\u3002" }), !profile || profile.interviewCount === 0 ? (_jsx("p", { className: "mt-5 rounded-[1.3rem] border border-dashed border-[rgba(153,191,201,0.18)] px-4 py-5 text-sm leading-7 text-[rgba(115,137,161,0.78)]", children: "\u8BC4\u5206\u5B8C\u6210\u540E\uFF0C\u8FD9\u91CC\u4F1A\u8DE8\u4F1A\u8BDD\u7D2F\u8BA1\u4F18\u52BF\u3001\u77ED\u677F\u3001\u4EBA\u683C\u4F7F\u7528\u504F\u597D\u548C\u5EFA\u8BAE\u8BAD\u7EC3\u91CD\u70B9\u3002" })) : (_jsxs("div", { className: "mt-5 space-y-5", children: [_jsxs("div", { className: "grid gap-3 md:grid-cols-2", children: [_jsx(ProfileFactCard, { label: "\u6700\u8FD1\u6280\u80FD", value: profile.lastSkill || "默认 interview" }), _jsx(ProfileFactCard, { label: "\u6700\u8FD1\u66F4\u65B0", value: profile.updatedAt ? new Date(profile.updatedAt).toLocaleString() : "刚刚" })] }), _jsxs("div", { children: [_jsx("p", { className: "tech-label text-[0.68rem] text-[rgba(97,123,150,0.62)]", children: "\u80FD\u529B\u7EF4\u5EA6" }), dimensions.length === 0 ? (_jsx("p", { className: "mt-3 text-sm leading-7 text-[rgba(115,137,161,0.78)]", children: "\u5F53\u524D\u8FD8\u6CA1\u6709\u8DB3\u591F\u8BC1\u636E\u5F62\u6210\u7EF4\u5EA6\u753B\u50CF\u3002" })) : (_jsx("div", { className: "mt-3 space-y-3", children: dimensions.map((dimension) => (_jsx(ProfileDimensionRow, { dimension: dimension }, dimension.key))) }))] }), radar.length > 0 ? (_jsxs("div", { children: [_jsx("p", { className: "tech-label text-[0.68rem] text-[rgba(97,123,150,0.62)]", children: "\u96F7\u8FBE\u56FE\u5E95\u7A3F" }), _jsx("div", { className: "mt-3 grid gap-3 sm:grid-cols-2 xl:grid-cols-3", children: radar.map((point) => (_jsxs("div", { className: "panel-card-soft", children: [_jsxs("div", { className: "flex items-center justify-between gap-3", children: [_jsx("p", { className: "text-sm font-medium text-[rgb(31,41,55)]", children: point.label }), _jsxs("span", { className: "chip-info px-2.5 py-1 text-[0.68rem] tracking-[0.18em]", children: [point.normalizedScore, "/100"] })] }), _jsx("div", { className: "mt-3 h-2 overflow-hidden rounded-full bg-[rgba(226,232,240,0.96)]", children: _jsx("div", { className: "h-full rounded-full bg-[linear-gradient(90deg,rgba(14,165,233,0.86),rgba(125,211,252,0.9))]", style: { width: `${point.normalizedScore}%` } }) })] }, point.key))) })] })) : null, growthCurves.length > 0 ? (_jsxs("div", { children: [_jsx("p", { className: "tech-label text-[0.68rem] text-[rgba(97,123,150,0.62)]", children: "\u8DE8\u573A\u6B21\u8D8B\u52BF" }), _jsx("div", { className: "mt-3 space-y-3", children: growthCurves.map((curve) => (_jsx(GrowthCurveRow, { label: curve.label, points: curve.points ?? [] }, curve.key))) })] })) : null, personaUsage.length > 0 ? (_jsxs("div", { children: [_jsx("p", { className: "tech-label text-[0.68rem] text-[rgba(97,123,150,0.62)]", children: "\u4EBA\u683C\u4F7F\u7528\u504F\u597D" }), _jsx("div", { className: "mt-3 flex flex-wrap gap-2", children: personaUsage.map((entry) => (_jsxs("span", { className: "chip-info px-3 py-1.5 text-sm normal-case tracking-[0.04em]", children: [getPersonaLabel(entry.persona), " \u00B7 ", entry.count] }, entry.persona))) })] })) : null, _jsx(ProfileBulletSection, { title: "\u7A33\u5B9A\u4F18\u52BF", items: stableStrengths, tone: "neutral" }), _jsx(ProfileBulletSection, { title: "\u91CD\u590D\u77ED\u677F", items: recurringGaps, tone: "warning" }), _jsx(ProfileBulletSection, { title: "\u672C\u573A\u518D\u6B21\u547D\u4E2D\u7684\u5386\u53F2\u5F31\u9879", items: reviewSummary?.historicalWeaknessesHit ?? [], tone: "warning" }), _jsx(ProfileBulletSection, { title: "\u672C\u573A\u65B0\u589E\u5F31\u9879", items: reviewSummary?.newWeaknesses ?? [], tone: "warning" }), _jsx(ProfileBulletSection, { title: "\u672C\u573A\u4FEE\u6B63\u9879", items: reviewSummary?.resolvedWeaknesses ?? [], tone: "success" }), _jsx(ProfileBulletSection, { title: "\u63A8\u8350\u8BAD\u7EC3\u91CD\u70B9", items: recommendedFocus, tone: "accent" }), _jsx(ProfileBulletSection, { title: "\u6700\u8FD1\u53D8\u5316", items: recentChanges, tone: "neutral" })] }))] }));
}
function ProfileFactCard({ label, value }) {
    return (_jsxs("div", { className: "panel-card-soft", children: [_jsx("p", { className: "tech-label text-[0.64rem] text-[rgba(97,123,150,0.62)]", children: label }), _jsx("p", { className: "mt-2 text-sm leading-6 text-[rgb(31,41,55)]", children: value })] }));
}
function ProfileDimensionRow({ dimension }) {
    const score = Math.max(-6, Math.min(6, dimension.score));
    const normalized = typeof dimension.normalizedScore === "number" ? dimension.normalizedScore : Math.round(((score + 6) / 12) * 100);
    const toneClass = normalized >= 67
        ? "bg-[linear-gradient(90deg,rgba(59,130,246,0.9),rgba(147,197,253,0.92))]"
        : normalized <= 40
            ? "bg-[linear-gradient(90deg,rgba(96,165,250,0.56),rgba(191,219,254,0.84))]"
            : "bg-[linear-gradient(90deg,rgba(191,219,254,0.7),rgba(219,234,254,0.92))]";
    const trendPoints = dimension.trend ?? [];
    const deltaLabel = typeof dimension.recentDelta === "number" && dimension.recentDelta !== 0
        ? dimension.recentDelta > 0
            ? `本场 +${dimension.recentDelta}`
            : `本场 ${dimension.recentDelta}`
        : "本场持平";
    return (_jsxs("div", { className: "panel-card-soft", children: [_jsxs("div", { className: "flex items-start justify-between gap-3", children: [_jsxs("div", { children: [_jsx("p", { className: "text-sm font-medium text-[rgb(31,41,55)]", children: dimension.label }), dimension.summary ? _jsx("p", { className: "mt-1 text-xs leading-6 text-[rgba(100,116,139,0.82)]", children: dimension.summary }) : null] }), _jsxs("div", { className: "flex flex-wrap items-center justify-end gap-2", children: [_jsxs("div", { className: "chip-info px-2.5 py-1 text-[0.68rem] tracking-[0.18em]", children: [normalized, "/100"] }), _jsxs("div", { className: "chip-neutral px-2.5 py-1 text-[0.68rem] tracking-[0.2em]", children: ["\u8BC1\u636E ", dimension.evidenceCount] })] })] }), _jsx("div", { className: "mt-3 h-2.5 overflow-hidden rounded-full bg-[rgba(226,232,240,0.96)]", children: _jsx("div", { className: `h-full rounded-full ${toneClass}`, style: { width: `${normalized}%` } }) }), _jsxs("div", { className: "mt-3 flex flex-wrap items-center justify-between gap-3 text-xs leading-6 text-[rgba(100,116,139,0.82)]", children: [_jsx("span", { children: deltaLabel }), _jsx("span", { children: dimension.lastUpdatedAt ? `最近命中 ${new Date(dimension.lastUpdatedAt).toLocaleString()}` : "等待更多样本" })] }), trendPoints.length > 1 ? _jsx(TrendStrip, { points: trendPoints }) : null] }));
}
function GrowthCurveRow({ label, points, }) {
    if (!points || points.length === 0) {
        return null;
    }
    const latest = points[points.length - 1];
    return (_jsxs("div", { className: "panel-card-soft", children: [_jsxs("div", { className: "flex items-center justify-between gap-3", children: [_jsx("p", { className: "text-sm font-medium text-[rgb(31,41,55)]", children: label }), _jsxs("span", { className: "chip-info px-2.5 py-1 text-[0.68rem] tracking-[0.18em]", children: [latest.normalizedScore, "/100"] })] }), _jsx(TrendStrip, { points: points, compact: true })] }));
}
function TrendStrip({ points, compact = false, }) {
    if (points.length === 0) {
        return null;
    }
    return (_jsx("div", { className: compact ? "mt-3" : "mt-3 rounded-[1rem] bg-[rgba(248,250,252,0.82)] px-3 py-3", children: _jsx("div", { className: "flex items-end gap-1.5", children: points.map((point, index) => (_jsx("div", { className: "flex-1 rounded-full bg-[linear-gradient(180deg,rgba(14,165,233,0.84),rgba(191,219,254,0.92))]", style: { height: `${Math.max(10, Math.round(point.normalizedScore * 0.44))}px` }, title: `${new Date(point.timestamp).toLocaleString()} · ${point.normalizedScore}/100` }, `${point.timestamp}-${index}`))) }) }));
}
function ProfileBulletSection({ title, items, tone, }) {
    if (items.length === 0) {
        return null;
    }
    const toneClass = tone === "warning"
        ? "border-[rgba(253,230,138,0.92)] bg-[rgba(254,252,232,0.96)] text-[rgb(161,98,7)]"
        : tone === "success"
            ? "border-[rgba(187,247,208,0.96)] bg-[rgba(240,253,244,0.96)] text-[rgb(21,128,61)]"
            : tone === "accent"
                ? "border-[rgba(219,234,254,0.96)] bg-[rgba(248,250,255,0.96)] text-[rgb(29,78,216)]"
                : "border-[rgba(153,191,201,0.16)] bg-[rgba(252,255,255,0.9)] text-[rgb(31,41,55)]";
    return (_jsxs("div", { children: [_jsx("p", { className: "tech-label text-[0.68rem] text-[rgba(97,123,150,0.62)]", children: title }), _jsx("div", { className: "mt-3 space-y-2.5", children: items.map((item, index) => (_jsx("div", { className: `rounded-[1.15rem] border px-4 py-3 text-sm leading-7 ${toneClass}`, children: item }, `${title}-${index}-${item}`))) })] }));
}
