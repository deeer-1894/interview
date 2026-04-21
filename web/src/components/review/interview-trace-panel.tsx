import { ChevronDown, ChevronRight, Download, Search } from "lucide-react";
import type { PointerEvent as ReactPointerEvent } from "react";
import { useEffect, useMemo, useRef, useState } from "react";

import { Input } from "@/components/ui/input";
import type { InterviewPhase, InterviewTraceNode, InterviewTraceTree } from "@/lib/types";

type TracePhaseTone = {
  frame: string;
  rail: string;
  badge: string;
  svgFill: string;
  svgStroke: string;
  svgRail: string;
};

type InterviewTracePanelProps = {
  trace: InterviewTraceTree | null;
  onFocusMessage: (messageID?: string) => void;
  getPersonaLabel: (persona?: string | null) => string;
  formatTraceNodeKind: (kind: string) => string;
  formatPhaseLabel: (phase?: string | null) => string;
  formatDecisionReasonLabel: (reason?: string | null) => string;
  formatTraceSignal: (signal: string) => string;
  traceSignalClass: (signal: string) => string;
  tracePhaseTone: (phase?: InterviewPhase | null) => TracePhaseTone;
};

export function InterviewTracePanel({
  trace,
  onFocusMessage,
  getPersonaLabel,
  formatTraceNodeKind,
  formatPhaseLabel,
  formatDecisionReasonLabel,
  formatTraceSignal,
  traceSignalClass,
  tracePhaseTone,
}: InterviewTracePanelProps) {
  const nodes = trace?.nodes ?? [];
  const rootNodes = useMemo(() => buildTraceForest(nodes), [nodes]);
  const [collapsedNodeIDs, setCollapsedNodeIDs] = useState<Record<string, boolean>>({});
  const [zoom, setZoom] = useState(1);
  const [searchQuery, setSearchQuery] = useState("");
  const panViewportRef = useRef<HTMLDivElement>(null);
  const dragStateRef = useRef<{ active: boolean; startX: number; startY: number; scrollLeft: number; scrollTop: number }>({
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
      return new Set<string>();
    }
    return new Set(
      nodes
        .filter((node) => traceNodeSearchText(node).includes(normalizedSearchQuery))
        .map((node) => node.id),
    );
  }, [nodes, normalizedSearchQuery]);
  const searchExpandedNodeIDs = useMemo(() => {
    if (matchedNodeIDs.size === 0) {
      return new Set<string>();
    }
    const ancestors = new Set<string>();
    const nodeByID = new Map(nodes.map((node) => [node.id, node] as const));
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

  function toggleNodeCollapse(nodeID: string) {
    setCollapsedNodeIDs((current) => ({
      ...current,
      [nodeID]: !current[nodeID],
    }));
  }

  function handlePanStart(event: ReactPointerEvent<HTMLDivElement>) {
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

  function handlePanMove(event: ReactPointerEvent<HTMLDivElement>) {
    const viewport = panViewportRef.current;
    const dragState = dragStateRef.current;
    if (!viewport || !dragState.active) {
      return;
    }
    viewport.scrollLeft = dragState.scrollLeft - (event.clientX - dragState.startX);
    viewport.scrollTop = dragState.scrollTop - (event.clientY - dragState.startY);
  }

  function handlePanEnd(event: ReactPointerEvent<HTMLDivElement>) {
    const viewport = panViewportRef.current;
    dragStateRef.current.active = false;
    if (viewport?.hasPointerCapture(event.pointerId)) {
      viewport.releasePointerCapture(event.pointerId);
    }
  }

  return (
    <section className="panel-card">
      <div className="flex items-start justify-between gap-4">
        <div>
          <p className="tech-label text-[0.68rem] text-[rgba(97,123,150,0.62)]">追问树</p>
          <h2 className="mt-2 font-display text-2xl text-[rgb(72,91,114)]">
            {nodes.length > 0 ? `${nodes.length} 个追问节点` : "等待追问链路"}
          </h2>
        </div>
        {trace ? (
          <div className="flex flex-wrap items-center justify-end gap-2">
            <div className="relative min-w-[220px] flex-1 sm:flex-none">
              <Search className="pointer-events-none absolute left-3 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-[rgba(100,116,139,0.7)]" />
              <Input
                value={searchQuery}
                onChange={(event) => setSearchQuery(event.target.value)}
                placeholder="搜索问题、摘要、主题"
                className="field-surface h-10 rounded-full pl-9 pr-3 text-sm shadow-none"
              />
            </div>
            {normalizedSearchQuery ? (
              <div className="chip-info px-3 py-1.5 text-[0.68rem] tracking-[0.18em]">
                命中 {matchedNodeIDs.size}
              </div>
            ) : null}
            <div className="chip-accent px-3 py-1.5 text-[0.68rem] tracking-[0.2em]">
              {getPersonaLabel(trace.persona)}
            </div>
            <button
              type="button"
              onClick={handleExportSVG}
              className="control-chip gap-1.5"
            >
              <Download className="h-3.5 w-3.5" />
              导出 SVG
            </button>
            <button
              type="button"
              onClick={() => setZoom((current) => Math.max(0.8, Number((current - 0.1).toFixed(2))))}
              className="control-chip"
            >
              缩小
            </button>
            <button
              type="button"
              onClick={() => setZoom(1)}
              className="control-chip"
            >
              重置
            </button>
            <button
              type="button"
              onClick={() => setZoom((current) => Math.min(1.35, Number((current + 0.1).toFixed(2))))}
              className="control-chip"
            >
              放大
            </button>
          </div>
        ) : null}
      </div>
      <p className="mt-3 max-w-[68ch] text-sm leading-7 text-[rgba(115,137,161,0.78)]">
        优先从首问往后看被持续深挖的节点，结合弱信号和决策原因，你能更快看清系统为什么继续追问。
      </p>

      {!trace || nodes.length === 0 ? (
        <p className="mt-5 rounded-[1.3rem] border border-dashed border-[rgba(153,191,201,0.18)] px-4 py-5 text-sm leading-7 text-[rgba(115,137,161,0.78)]">
          面试进行后，这里会生成一条可读的追问路径，帮助你看清首问、追问、切题和回答强弱信号。
        </p>
      ) : (
        <div
          ref={panViewportRef}
          className="mt-5 max-h-[58vh] cursor-grab overflow-auto pb-2 active:cursor-grabbing"
          onPointerDown={handlePanStart}
          onPointerMove={handlePanMove}
          onPointerUp={handlePanEnd}
          onPointerCancel={handlePanEnd}
        >
          <div className="min-w-[540px] origin-top-left space-y-4 transition-transform" style={{ transform: `scale(${zoom})`, width: `${100 / zoom}%` }}>
            {rootNodes.map((node) => (
              <TraceMindMapNode
                key={node.node.id}
                item={node}
                depth={0}
                collapsedNodeIDs={collapsedNodeIDs}
                matchedNodeIDs={matchedNodeIDs}
                searchExpandedNodeIDs={searchExpandedNodeIDs}
                onToggleCollapse={toggleNodeCollapse}
                onFocusMessage={onFocusMessage}
                formatTraceNodeKind={formatTraceNodeKind}
                formatPhaseLabel={formatPhaseLabel}
                formatDecisionReasonLabel={formatDecisionReasonLabel}
                formatTraceSignal={formatTraceSignal}
                traceSignalClass={traceSignalClass}
                tracePhaseTone={tracePhaseTone}
              />
            ))}
          </div>
        </div>
      )}
    </section>
  );
}

type TraceTreeItem = {
  node: InterviewTraceNode;
  children: TraceTreeItem[];
};

function TraceMindMapNode({
  item,
  depth,
  collapsedNodeIDs,
  matchedNodeIDs,
  searchExpandedNodeIDs,
  onToggleCollapse,
  onFocusMessage,
  formatTraceNodeKind,
  formatPhaseLabel,
  formatDecisionReasonLabel,
  formatTraceSignal,
  traceSignalClass,
  tracePhaseTone,
}: {
  item: TraceTreeItem;
  depth: number;
  collapsedNodeIDs: Record<string, boolean>;
  matchedNodeIDs: Set<string>;
  searchExpandedNodeIDs: Set<string>;
  onToggleCollapse: (nodeID: string) => void;
  onFocusMessage: (messageID?: string) => void;
  formatTraceNodeKind: (kind: string) => string;
  formatPhaseLabel: (phase?: string | null) => string;
  formatDecisionReasonLabel: (reason?: string | null) => string;
  formatTraceSignal: (signal: string) => string;
  traceSignalClass: (signal: string) => string;
  tracePhaseTone: (phase?: InterviewPhase | null) => TracePhaseTone;
}) {
  const { node, children } = item;
  const phaseTone = tracePhaseTone(node.phase);
  const matched = matchedNodeIDs.has(node.id);
  const forcedExpanded = searchExpandedNodeIDs.has(node.id);
  const collapsed = forcedExpanded ? false : Boolean(collapsedNodeIDs[node.id]);
  const shortQuestion = compactTraceText(node.question, 96);
  const shortAnswerSummary = compactTraceText(node.answerSummary, 120);
  const shortExplanation = compactTraceText(node.explanation, 120);

  return (
    <div className="relative">
      {depth > 0 ? (
        <div className="absolute left-0 top-0 h-full w-6">
          <div className="absolute left-3 top-0 h-full w-px bg-[rgba(203,213,225,0.92)]" />
          <div className="absolute left-3 top-7 h-px w-3 bg-[rgba(203,213,225,0.92)]" />
        </div>
      ) : null}
      <div className={depth > 0 ? "pl-8" : ""}>
        <div
          title={node.question}
          className={`group relative block w-full overflow-hidden rounded-[1.35rem] border px-4 py-4 text-left transition-colors ${
            matched ? "ring-2 ring-[rgba(37,99,235,0.22)] shadow-[0_18px_32px_rgba(37,99,235,0.12)]" : ""
          } ${phaseTone.frame}`}
        >
          <div className={`pointer-events-none absolute left-0 top-0 h-full w-1.5 ${phaseTone.rail}`} />
          <div className="flex flex-wrap items-center gap-2 pl-1">
            {children.length > 0 ? (
              <button
                type="button"
                onClick={(event) => {
                  event.stopPropagation();
                  onToggleCollapse(node.id);
                }}
                className="chip-neutral h-6 w-6 justify-center border-[rgba(255,255,255,0.78)] bg-[rgba(255,255,255,0.72)] p-0 text-[rgba(71,85,105,0.82)]"
                aria-label={collapsed ? "展开分支" : "折叠分支"}
              >
                {collapsed ? <ChevronRight className="h-3.5 w-3.5" /> : <ChevronDown className="h-3.5 w-3.5" />}
              </button>
            ) : null}
            <span className="chip-neutral px-2.5 py-1 text-[0.68rem] tracking-[0.2em]">
              {formatTraceNodeKind(node.kind)}
            </span>
            {node.phase ? (
              <span className={`rounded-full px-2.5 py-1 text-[0.68rem] uppercase tracking-[0.2em] ${phaseTone.badge}`}>
                {formatPhaseLabel(node.phase)}
              </span>
            ) : null}
            {matched ? (
              <span className="chip-info px-2.5 py-1 text-[0.68rem] tracking-[0.18em]">
                命中
              </span>
            ) : null}
            {node.topic ? (
              <span className="chip-neutral px-2.5 py-1 text-[0.68rem] tracking-[0.18em]">
                {node.topic}
              </span>
            ) : null}
            {node.reason ? (
              <span className="chip-info px-2.5 py-1 text-[0.68rem] tracking-[0.18em]">
                {formatDecisionReasonLabel(node.reason)}
              </span>
            ) : null}
            {node.profileHit ? (
              <span className="chip-warning px-2.5 py-1 text-[0.68rem] tracking-[0.18em]">
                命中历史弱项
              </span>
            ) : null}
            {node.signal ? (
              <span className={`rounded-full px-2.5 py-1 text-[0.68rem] uppercase tracking-[0.18em] ${traceSignalClass(node.signal)}`}>
                {formatTraceSignal(node.signal)}
              </span>
            ) : null}
            {typeof node.round === "number" && node.round > 0 ? (
              <span className="chip-neutral px-2.5 py-1 text-[0.68rem] tracking-[0.18em]">
                第 {node.round} 轮
              </span>
            ) : null}
          </div>
          <button
            type="button"
            onClick={() => onFocusMessage(node.messageId)}
            className="mt-3 block w-full pl-1 text-left text-sm leading-7 text-[rgb(31,41,55)]"
          >
            {shortQuestion}
          </button>
          {node.answerSummary ? (
            <div className="mt-3 rounded-[1rem] border border-[rgba(226,231,239,0.78)] bg-[rgba(255,255,255,0.76)] px-3 py-3 text-sm leading-6 text-[rgba(71,85,105,0.9)]" title={node.answerSummary}>
              <p className="tech-label text-[0.6rem] text-[rgba(100,116,139,0.7)]">回答摘要</p>
              <p className="mt-1.5">{shortAnswerSummary}</p>
            </div>
          ) : null}
          {node.explanation ? (
            <div
              className="mt-3 rounded-[1rem] border border-[rgba(191,219,254,0.82)] bg-[rgba(239,246,255,0.78)] px-3 py-3 text-sm leading-6 text-[rgb(30,64,175)]"
              title={node.explanation}
            >
              <p className="tech-label text-[0.6rem] text-[rgb(37,99,235)]">追问原因</p>
              <p className="mt-1.5">{shortExplanation}</p>
            </div>
          ) : null}
          {node.focusHits?.length ? (
            <div className="mt-3 flex flex-wrap gap-2">
              {node.focusHits.map((item) => (
                <span key={`${node.id}-${item}`} className="chip-warning px-2.5 py-1 text-[0.68rem] tracking-[0.18em]">
                  历史命中 · {item}
                </span>
              ))}
            </div>
          ) : null}
          {node.scenario || node.adversarial || node.pressure ? (
            <div className="mt-3 flex flex-wrap gap-2">
              {node.scenario ? (
                <span className="chip-neutral px-2.5 py-1 text-[0.68rem] tracking-[0.18em]">
                  场景
                </span>
              ) : null}
              {node.adversarial ? (
                <span className="chip-warning px-2.5 py-1 text-[0.68rem] tracking-[0.18em]">
                  反驳
                </span>
              ) : null}
              {node.pressure ? (
                <span className="chip-danger px-2.5 py-1 text-[0.68rem] tracking-[0.18em]">
                  压力
                </span>
              ) : null}
            </div>
          ) : null}
        </div>
      </div>
      {children.length > 0 && !collapsed ? (
        <div className="mt-3 space-y-3">
          {children.map((child) => (
            <TraceMindMapNode
              key={child.node.id}
              item={child}
              depth={depth + 1}
              collapsedNodeIDs={collapsedNodeIDs}
              matchedNodeIDs={matchedNodeIDs}
              searchExpandedNodeIDs={searchExpandedNodeIDs}
              onToggleCollapse={onToggleCollapse}
              onFocusMessage={onFocusMessage}
              formatTraceNodeKind={formatTraceNodeKind}
              formatPhaseLabel={formatPhaseLabel}
              formatDecisionReasonLabel={formatDecisionReasonLabel}
              formatTraceSignal={formatTraceSignal}
              traceSignalClass={traceSignalClass}
              tracePhaseTone={tracePhaseTone}
            />
          ))}
        </div>
      ) : null}
    </div>
  );
}

function buildTraceForest(nodes: InterviewTraceNode[]): TraceTreeItem[] {
  const items = new Map<string, TraceTreeItem>();
  nodes.forEach((node) => {
    items.set(node.id, { node, children: [] });
  });

  const roots: TraceTreeItem[] = [];
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

function compactTraceText(value?: string | null, maxLength = 96) {
  const normalized = (value ?? "").replace(/\s+/g, " ").trim();
  if (normalized.length <= maxLength) {
    return normalized;
  }
  return `${normalized.slice(0, maxLength).trimEnd()}…`;
}

function traceNodeSearchText(node: InterviewTraceNode) {
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

type TraceSvgLayoutNode = {
  node: InterviewTraceNode;
  x: number;
  y: number;
  width: number;
  height: number;
  centerY: number;
};

function buildTraceMindMapSVG(
  trace: InterviewTraceTree,
  roots: TraceTreeItem[],
  options: Pick<
    InterviewTracePanelProps,
    "getPersonaLabel" | "formatTraceNodeKind" | "formatPhaseLabel" | "formatDecisionReasonLabel" | "tracePhaseTone"
  >,
) {
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
      const meta = escapeSvgText(
        [
          options.formatTraceNodeKind(entry.node.kind),
          entry.node.phase ? options.formatPhaseLabel(entry.node.phase) : "",
          entry.node.topic ?? "",
          entry.node.reason ? options.formatDecisionReasonLabel(entry.node.reason) : "",
        ]
          .filter(Boolean)
          .join(" · "),
      );
      const summary = escapeSvgText(compactTraceText(entry.node.answerSummary, 72));
      const summaryY = entry.node.answerSummary ? entry.y + 78 : entry.y + 58;
      return `
        <g>
          <rect x="${entry.x}" y="${entry.y}" rx="18" ry="18" width="${entry.width}" height="${entry.height}" fill="${tone.svgFill}" stroke="${tone.svgStroke}" stroke-width="1.5" />
          <rect x="${entry.x}" y="${entry.y}" rx="18" ry="18" width="8" height="${entry.height}" fill="${tone.svgRail}" />
          <text x="${entry.x + 18}" y="${entry.y + 24}" font-size="11" font-family="Sora, 'Noto Sans SC', sans-serif" fill="rgba(71,85,105,0.82)">${meta}</text>
          <text x="${entry.x + 18}" y="${entry.y + 50}" font-size="14" font-weight="600" font-family="Sora, 'Noto Sans SC', sans-serif" fill="rgb(31,41,55)">${question}</text>
          ${
            entry.node.answerSummary
              ? `<text x="${entry.x + 18}" y="${summaryY}" font-size="12" font-family="'Noto Sans SC', sans-serif" fill="rgba(71,85,105,0.88)">${summary}</text>`
              : ""
          }
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

function layoutTraceForestForSVG(roots: TraceTreeItem[]) {
  const nodeWidth = 320;
  const nodeHeight = 92;
  const gapX = 72;
  const gapY = 24;
  const paddingTop = 48;
  const paddingLeft = 28;
  const nodes: TraceSvgLayoutNode[] = [];
  const lines: Array<{ fromX: number; fromY: number; toX: number; toY: number }> = [];
  let cursorY = paddingTop;
  let maxDepth = 0;

  function layoutNode(item: TraceTreeItem, depth: number, startY: number): { centerY: number; nextY: number; x: number; width: number } {
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

function escapeSvgText(value: string) {
  return value
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/\"/g, "&quot;")
    .replace(/'/g, "&apos;");
}
