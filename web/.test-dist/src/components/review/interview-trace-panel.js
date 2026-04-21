import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { ChevronDown, ChevronRight, Download, Search } from "lucide-react";
import { useEffect, useMemo, useRef, useState } from "react";
import { Input } from "../ui/input.js";
export function InterviewTracePanel({ trace, onFocusMessage, getPersonaLabel, formatTraceNodeKind, formatPhaseLabel, formatDecisionReasonLabel, formatTraceSignal, traceSignalClass, tracePhaseTone, }) {
    const nodes = trace?.nodes ?? [];
    const rootNodes = useMemo(() => buildTraceForest(nodes), [nodes]);
    const [collapsedNodeIDs, setCollapsedNodeIDs] = useState({});
    const [zoom, setZoom] = useState(1);
    const [searchQuery, setSearchQuery] = useState("");
    const panViewportRef = useRef(null);
    const dragStateRef = useRef({
        active: false,
        startX: 0,
        startY: 0,
        scrollLeft: 0,
        scrollTop: 0,
    });
    useEffect(() => {
        setCollapsedNodeIDs({});
        setZoom(1);
    }, [trace?.runId]);
    const normalizedSearchQuery = searchQuery.trim().toLowerCase();
    const matchedNodeIDs = useMemo(() => {
        if (!normalizedSearchQuery) {
            return new Set();
        }
        return new Set(nodes
            .filter((node) => traceNodeSearchText(node).includes(normalizedSearchQuery))
            .map((node) => node.id));
    }, [nodes, normalizedSearchQuery]);
    const searchExpandedNodeIDs = useMemo(() => {
        if (matchedNodeIDs.size === 0) {
            return new Set();
        }
        const ancestors = new Set();
        const nodeByID = new Map(nodes.map((node) => [node.id, node]));
        matchedNodeIDs.forEach((nodeID) => {
            let currentParent = nodeByID.get(nodeID)?.parentId;
            while (currentParent) {
                ancestors.add(currentParent);
                currentParent = nodeByID.get(currentParent)?.parentId;
            }
        });
        return ancestors;
    }, [matchedNodeIDs, nodes]);
    function handleExportSVG() {
        if (!trace || rootNodes.length === 0) {
            return;
        }
        const svgContent = buildTraceMindMapSVG(trace, rootNodes, {
            getPersonaLabel,
            formatTraceNodeKind,
            formatPhaseLabel,
            formatDecisionReasonLabel,
            tracePhaseTone,
        });
        const blob = new Blob([svgContent], { type: "image/svg+xml;charset=utf-8" });
        const blobURL = URL.createObjectURL(blob);
        const anchor = document.createElement("a");
        anchor.href = blobURL;
        anchor.download = `trace-${trace.runId}.svg`;
        anchor.click();
        URL.revokeObjectURL(blobURL);
    }
    function toggleNodeCollapse(nodeID) {
        setCollapsedNodeIDs((current) => ({
            ...current,
            [nodeID]: !current[nodeID],
        }));
    }
    function handlePanStart(event) {
        const viewport = panViewportRef.current;
        if (!viewport) {
            return;
        }
        dragStateRef.current = {
            active: true,
            startX: event.clientX,
            startY: event.clientY,
            scrollLeft: viewport.scrollLeft,
            scrollTop: viewport.scrollTop,
        };
        viewport.setPointerCapture(event.pointerId);
    }
    function handlePanMove(event) {
        const viewport = panViewportRef.current;
        const dragState = dragStateRef.current;
        if (!viewport || !dragState.active) {
            return;
        }
        viewport.scrollLeft = dragState.scrollLeft - (event.clientX - dragState.startX);
        viewport.scrollTop = dragState.scrollTop - (event.clientY - dragState.startY);
    }
    function handlePanEnd(event) {
        const viewport = panViewportRef.current;
        dragStateRef.current.active = false;
        if (viewport?.hasPointerCapture(event.pointerId)) {
            viewport.releasePointerCapture(event.pointerId);
        }
    }
    return (_jsxs("section", { className: "panel-card", children: [_jsxs("div", { className: "flex items-start justify-between gap-4", children: [_jsxs("div", { children: [_jsx("p", { className: "tech-label text-[0.68rem] text-[rgba(97,123,150,0.62)]", children: "\u8FFD\u95EE\u6811" }), _jsx("h2", { className: "mt-2 font-display text-2xl text-[rgb(72,91,114)]", children: nodes.length > 0 ? `${nodes.length} 个追问节点` : "等待追问链路" })] }), trace ? (_jsxs("div", { className: "flex flex-wrap items-center justify-end gap-2", children: [_jsxs("div", { className: "relative min-w-[220px] flex-1 sm:flex-none", children: [_jsx(Search, { className: "pointer-events-none absolute left-3 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-[rgba(100,116,139,0.7)]" }), _jsx(Input, { value: searchQuery, onChange: (event) => setSearchQuery(event.target.value), placeholder: "\u641C\u7D22\u95EE\u9898\u3001\u6458\u8981\u3001\u4E3B\u9898", className: "field-surface h-10 rounded-full pl-9 pr-3 text-sm shadow-none" })] }), normalizedSearchQuery ? (_jsxs("div", { className: "chip-info px-3 py-1.5 text-[0.68rem] tracking-[0.18em]", children: ["\u547D\u4E2D ", matchedNodeIDs.size] })) : null, _jsx("div", { className: "chip-accent px-3 py-1.5 text-[0.68rem] tracking-[0.2em]", children: getPersonaLabel(trace.persona) }), _jsxs("button", { type: "button", onClick: handleExportSVG, className: "control-chip gap-1.5", children: [_jsx(Download, { className: "h-3.5 w-3.5" }), "\u5BFC\u51FA SVG"] }), _jsx("button", { type: "button", onClick: () => setZoom((current) => Math.max(0.8, Number((current - 0.1).toFixed(2)))), className: "control-chip", children: "\u7F29\u5C0F" }), _jsx("button", { type: "button", onClick: () => setZoom(1), className: "control-chip", children: "\u91CD\u7F6E" }), _jsx("button", { type: "button", onClick: () => setZoom((current) => Math.min(1.35, Number((current + 0.1).toFixed(2)))), className: "control-chip", children: "\u653E\u5927" })] })) : null] }), _jsx("p", { className: "mt-3 max-w-[68ch] text-sm leading-7 text-[rgba(115,137,161,0.78)]", children: "\u4F18\u5148\u4ECE\u9996\u95EE\u5F80\u540E\u770B\u88AB\u6301\u7EED\u6DF1\u6316\u7684\u8282\u70B9\uFF0C\u7ED3\u5408\u5F31\u4FE1\u53F7\u548C\u51B3\u7B56\u539F\u56E0\uFF0C\u4F60\u80FD\u66F4\u5FEB\u770B\u6E05\u7CFB\u7EDF\u4E3A\u4EC0\u4E48\u7EE7\u7EED\u8FFD\u95EE\u3002" }), !trace || nodes.length === 0 ? (_jsx("p", { className: "mt-5 rounded-[1.3rem] border border-dashed border-[rgba(153,191,201,0.18)] px-4 py-5 text-sm leading-7 text-[rgba(115,137,161,0.78)]", children: "\u9762\u8BD5\u8FDB\u884C\u540E\uFF0C\u8FD9\u91CC\u4F1A\u751F\u6210\u4E00\u6761\u53EF\u8BFB\u7684\u8FFD\u95EE\u8DEF\u5F84\uFF0C\u5E2E\u52A9\u4F60\u770B\u6E05\u9996\u95EE\u3001\u8FFD\u95EE\u3001\u5207\u9898\u548C\u56DE\u7B54\u5F3A\u5F31\u4FE1\u53F7\u3002" })) : (_jsx("div", { ref: panViewportRef, className: "mt-5 max-h-[58vh] cursor-grab overflow-auto pb-2 active:cursor-grabbing", onPointerDown: handlePanStart, onPointerMove: handlePanMove, onPointerUp: handlePanEnd, onPointerCancel: handlePanEnd, children: _jsx("div", { className: "min-w-[540px] origin-top-left space-y-4 transition-transform", style: { transform: `scale(${zoom})`, width: `${100 / zoom}%` }, children: rootNodes.map((node) => (_jsx(TraceMindMapNode, { item: node, depth: 0, collapsedNodeIDs: collapsedNodeIDs, matchedNodeIDs: matchedNodeIDs, searchExpandedNodeIDs: searchExpandedNodeIDs, onToggleCollapse: toggleNodeCollapse, onFocusMessage: onFocusMessage, formatTraceNodeKind: formatTraceNodeKind, formatPhaseLabel: formatPhaseLabel, formatDecisionReasonLabel: formatDecisionReasonLabel, formatTraceSignal: formatTraceSignal, traceSignalClass: traceSignalClass, tracePhaseTone: tracePhaseTone }, node.node.id))) }) }))] }));
}
function TraceMindMapNode({ item, depth, collapsedNodeIDs, matchedNodeIDs, searchExpandedNodeIDs, onToggleCollapse, onFocusMessage, formatTraceNodeKind, formatPhaseLabel, formatDecisionReasonLabel, formatTraceSignal, traceSignalClass, tracePhaseTone, }) {
    const { node, children } = item;
    const phaseTone = tracePhaseTone(node.phase);
    const matched = matchedNodeIDs.has(node.id);
    const forcedExpanded = searchExpandedNodeIDs.has(node.id);
    const collapsed = forcedExpanded ? false : Boolean(collapsedNodeIDs[node.id]);
    const shortQuestion = compactTraceText(node.question, 96);
    const shortAnswerSummary = compactTraceText(node.answerSummary, 120);
    const shortExplanation = compactTraceText(node.explanation, 120);
    return (_jsxs("div", { className: "relative", children: [depth > 0 ? (_jsxs("div", { className: "absolute left-0 top-0 h-full w-6", children: [_jsx("div", { className: "absolute left-3 top-0 h-full w-px bg-[rgba(203,213,225,0.92)]" }), _jsx("div", { className: "absolute left-3 top-7 h-px w-3 bg-[rgba(203,213,225,0.92)]" })] })) : null, _jsx("div", { className: depth > 0 ? "pl-8" : "", children: _jsxs("div", { title: node.question, className: `group relative block w-full overflow-hidden rounded-[1.35rem] border px-4 py-4 text-left transition-colors ${matched ? "ring-2 ring-[rgba(37,99,235,0.22)] shadow-[0_18px_32px_rgba(37,99,235,0.12)]" : ""} ${phaseTone.frame}`, children: [_jsx("div", { className: `pointer-events-none absolute left-0 top-0 h-full w-1.5 ${phaseTone.rail}` }), _jsxs("div", { className: "flex flex-wrap items-center gap-2 pl-1", children: [children.length > 0 ? (_jsx("button", { type: "button", onClick: (event) => {
                                        event.stopPropagation();
                                        onToggleCollapse(node.id);
                                    }, className: "chip-neutral h-6 w-6 justify-center border-[rgba(255,255,255,0.78)] bg-[rgba(255,255,255,0.72)] p-0 text-[rgba(71,85,105,0.82)]", "aria-label": collapsed ? "展开分支" : "折叠分支", children: collapsed ? _jsx(ChevronRight, { className: "h-3.5 w-3.5" }) : _jsx(ChevronDown, { className: "h-3.5 w-3.5" }) })) : null, _jsx("span", { className: "chip-neutral px-2.5 py-1 text-[0.68rem] tracking-[0.2em]", children: formatTraceNodeKind(node.kind) }), node.phase ? (_jsx("span", { className: `rounded-full px-2.5 py-1 text-[0.68rem] uppercase tracking-[0.2em] ${phaseTone.badge}`, children: formatPhaseLabel(node.phase) })) : null, matched ? (_jsx("span", { className: "chip-info px-2.5 py-1 text-[0.68rem] tracking-[0.18em]", children: "\u547D\u4E2D" })) : null, node.topic ? (_jsx("span", { className: "chip-neutral px-2.5 py-1 text-[0.68rem] tracking-[0.18em]", children: node.topic })) : null, node.reason ? (_jsx("span", { className: "chip-info px-2.5 py-1 text-[0.68rem] tracking-[0.18em]", children: formatDecisionReasonLabel(node.reason) })) : null, node.profileHit ? (_jsx("span", { className: "chip-warning px-2.5 py-1 text-[0.68rem] tracking-[0.18em]", children: "\u547D\u4E2D\u5386\u53F2\u5F31\u9879" })) : null, node.signal ? (_jsx("span", { className: `rounded-full px-2.5 py-1 text-[0.68rem] uppercase tracking-[0.18em] ${traceSignalClass(node.signal)}`, children: formatTraceSignal(node.signal) })) : null, typeof node.round === "number" && node.round > 0 ? (_jsxs("span", { className: "chip-neutral px-2.5 py-1 text-[0.68rem] tracking-[0.18em]", children: ["\u7B2C ", node.round, " \u8F6E"] })) : null] }), _jsx("button", { type: "button", onClick: () => onFocusMessage(node.messageId), className: "mt-3 block w-full pl-1 text-left text-sm leading-7 text-[rgb(31,41,55)]", children: shortQuestion }), node.answerSummary ? (_jsxs("div", { className: "mt-3 rounded-[1rem] border border-[rgba(226,231,239,0.78)] bg-[rgba(255,255,255,0.76)] px-3 py-3 text-sm leading-6 text-[rgba(71,85,105,0.9)]", title: node.answerSummary, children: [_jsx("p", { className: "tech-label text-[0.6rem] text-[rgba(100,116,139,0.7)]", children: "\u56DE\u7B54\u6458\u8981" }), _jsx("p", { className: "mt-1.5", children: shortAnswerSummary })] })) : null, node.explanation ? (_jsxs("div", { className: "mt-3 rounded-[1rem] border border-[rgba(191,219,254,0.82)] bg-[rgba(239,246,255,0.78)] px-3 py-3 text-sm leading-6 text-[rgb(30,64,175)]", title: node.explanation, children: [_jsx("p", { className: "tech-label text-[0.6rem] text-[rgb(37,99,235)]", children: "\u8FFD\u95EE\u539F\u56E0" }), _jsx("p", { className: "mt-1.5", children: shortExplanation })] })) : null, node.focusHits?.length ? (_jsx("div", { className: "mt-3 flex flex-wrap gap-2", children: node.focusHits.map((item) => (_jsxs("span", { className: "chip-warning px-2.5 py-1 text-[0.68rem] tracking-[0.18em]", children: ["\u5386\u53F2\u547D\u4E2D \u00B7 ", item] }, `${node.id}-${item}`))) })) : null, node.scenario || node.adversarial || node.pressure ? (_jsxs("div", { className: "mt-3 flex flex-wrap gap-2", children: [node.scenario ? (_jsx("span", { className: "chip-neutral px-2.5 py-1 text-[0.68rem] tracking-[0.18em]", children: "\u573A\u666F" })) : null, node.adversarial ? (_jsx("span", { className: "chip-warning px-2.5 py-1 text-[0.68rem] tracking-[0.18em]", children: "\u53CD\u9A73" })) : null, node.pressure ? (_jsx("span", { className: "chip-danger px-2.5 py-1 text-[0.68rem] tracking-[0.18em]", children: "\u538B\u529B" })) : null] })) : null] }) }), children.length > 0 && !collapsed ? (_jsx("div", { className: "mt-3 space-y-3", children: children.map((child) => (_jsx(TraceMindMapNode, { item: child, depth: depth + 1, collapsedNodeIDs: collapsedNodeIDs, matchedNodeIDs: matchedNodeIDs, searchExpandedNodeIDs: searchExpandedNodeIDs, onToggleCollapse: onToggleCollapse, onFocusMessage: onFocusMessage, formatTraceNodeKind: formatTraceNodeKind, formatPhaseLabel: formatPhaseLabel, formatDecisionReasonLabel: formatDecisionReasonLabel, formatTraceSignal: formatTraceSignal, traceSignalClass: traceSignalClass, tracePhaseTone: tracePhaseTone }, child.node.id))) })) : null] }));
}
function buildTraceForest(nodes) {
    const items = new Map();
    nodes.forEach((node) => {
        items.set(node.id, { node, children: [] });
    });
    const roots = [];
    nodes.forEach((node) => {
        const current = items.get(node.id);
        if (!current) {
            return;
        }
        if (node.parentId && items.has(node.parentId)) {
            items.get(node.parentId)?.children.push(current);
            return;
        }
        roots.push(current);
    });
    return roots;
}
function compactTraceText(value, maxLength = 96) {
    const normalized = (value ?? "").replace(/\s+/g, " ").trim();
    if (normalized.length <= maxLength) {
        return normalized;
    }
    return `${normalized.slice(0, maxLength).trimEnd()}…`;
}
function traceNodeSearchText(node) {
    return [
        node.question,
        node.answerSummary,
        node.explanation,
        ...(node.focusHits ?? []),
        node.topic,
        node.reason ?? "",
        ...(node.weakSignals ?? []),
        ...(node.strongSignals ?? []),
        node.scenario ?? "",
    ]
        .join(" ")
        .toLowerCase();
}
function buildTraceMindMapSVG(trace, roots, options) {
    const layout = layoutTraceForestForSVG(roots);
    const width = Math.max(layout.width + 72, 960);
    const height = Math.max(layout.height + 64, 520);
    const lines = layout.lines
        .map((line) => `<path d="M ${line.fromX} ${line.fromY} C ${line.fromX + 28} ${line.fromY}, ${line.toX - 28} ${line.toY}, ${line.toX} ${line.toY}" fill="none" stroke="rgba(148,163,184,0.85)" stroke-width="1.5" />`)
        .join("");
    const nodes = layout.nodes
        .map((entry) => {
        const tone = options.tracePhaseTone(entry.node.phase);
        const question = escapeSvgText(compactTraceText(entry.node.question, 54));
        const meta = escapeSvgText([
            options.formatTraceNodeKind(entry.node.kind),
            entry.node.phase ? options.formatPhaseLabel(entry.node.phase) : "",
            entry.node.topic ?? "",
            entry.node.reason ? options.formatDecisionReasonLabel(entry.node.reason) : "",
        ]
            .filter(Boolean)
            .join(" · "));
        const summary = escapeSvgText(compactTraceText(entry.node.answerSummary, 72));
        const summaryY = entry.node.answerSummary ? entry.y + 78 : entry.y + 58;
        return `
        <g>
          <rect x="${entry.x}" y="${entry.y}" rx="18" ry="18" width="${entry.width}" height="${entry.height}" fill="${tone.svgFill}" stroke="${tone.svgStroke}" stroke-width="1.5" />
          <rect x="${entry.x}" y="${entry.y}" rx="18" ry="18" width="8" height="${entry.height}" fill="${tone.svgRail}" />
          <text x="${entry.x + 18}" y="${entry.y + 24}" font-size="11" font-family="Sora, 'Noto Sans SC', sans-serif" fill="rgba(71,85,105,0.82)">${meta}</text>
          <text x="${entry.x + 18}" y="${entry.y + 50}" font-size="14" font-weight="600" font-family="Sora, 'Noto Sans SC', sans-serif" fill="rgb(31,41,55)">${question}</text>
          ${entry.node.answerSummary
            ? `<text x="${entry.x + 18}" y="${summaryY}" font-size="12" font-family="'Noto Sans SC', sans-serif" fill="rgba(71,85,105,0.88)">${summary}</text>`
            : ""}
        </g>
      `;
    })
        .join("");
    return `<?xml version="1.0" encoding="UTF-8"?>
<svg xmlns="http://www.w3.org/2000/svg" width="${width}" height="${height}" viewBox="0 0 ${width} ${height}" fill="none">
  <rect width="${width}" height="${height}" rx="28" fill="rgb(248,251,255)" />
  <text x="28" y="34" font-size="14" font-family="Sora, 'Noto Sans SC', sans-serif" fill="rgba(71,85,105,0.76)">追问树 · ${escapeSvgText(options.getPersonaLabel(trace.persona))}</text>
  ${lines}
  ${nodes}
</svg>`;
}
function layoutTraceForestForSVG(roots) {
    const nodeWidth = 320;
    const nodeHeight = 92;
    const gapX = 72;
    const gapY = 24;
    const paddingTop = 48;
    const paddingLeft = 28;
    const nodes = [];
    const lines = [];
    let cursorY = paddingTop;
    let maxDepth = 0;
    function layoutNode(item, depth, startY) {
        maxDepth = Math.max(maxDepth, depth);
        const x = paddingLeft + depth * (nodeWidth + gapX);
        if (item.children.length === 0) {
            const centerY = startY + nodeHeight / 2;
            nodes.push({ node: item.node, x, y: startY, width: nodeWidth, height: nodeHeight, centerY });
            return { centerY, nextY: startY + nodeHeight + gapY, x, width: nodeWidth };
        }
        let childY = startY;
        const childLayouts = item.children.map((child) => {
            const result = layoutNode(child, depth + 1, childY);
            childY = result.nextY;
            return result;
        });
        const firstCenter = childLayouts[0].centerY;
        const lastCenter = childLayouts[childLayouts.length - 1].centerY;
        const centerY = (firstCenter + lastCenter) / 2;
        const y = centerY - nodeHeight / 2;
        nodes.push({ node: item.node, x, y, width: nodeWidth, height: nodeHeight, centerY });
        childLayouts.forEach((child) => {
            lines.push({
                fromX: x + nodeWidth,
                fromY: centerY,
                toX: child.x,
                toY: child.centerY,
            });
        });
        return { centerY, nextY: Math.max(childY, y + nodeHeight + gapY), x, width: nodeWidth };
    }
    roots.forEach((root) => {
        const result = layoutNode(root, 0, cursorY);
        cursorY = result.nextY + 8;
    });
    return {
        nodes,
        lines,
        width: paddingLeft * 2 + (maxDepth + 1) * nodeWidth + maxDepth * gapX,
        height: cursorY + 24,
    };
}
function escapeSvgText(value) {
    return value
        .replace(/&/g, "&amp;")
        .replace(/</g, "&lt;")
        .replace(/>/g, "&gt;")
        .replace(/\"/g, "&quot;")
        .replace(/'/g, "&apos;");
}
