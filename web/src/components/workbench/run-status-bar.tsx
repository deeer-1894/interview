import type { ReactNode } from "react";
import { CircleDashed, Flame, Hourglass, Radar, Sparkles, Waypoints } from "lucide-react";

type RunStatusBarProps = {
  status?: string;
  statusLabel: string;
  modeLabel: string;
  personaLabel: string;
  phaseLabel: string;
  timeBudget: string;
  hasReviewData: boolean;
};

export function RunStatusBar({
  status,
  statusLabel,
  modeLabel,
  personaLabel,
  phaseLabel,
  timeBudget,
  hasReviewData,
}: RunStatusBarProps) {
  const statusTone =
    status === "failed" || status === "cancelled"
      ? "border-[rgba(254,205,211,0.92)] bg-[rgba(255,241,242,0.96)] text-[rgb(190,24,93)]"
      : status === "waiting_clarify"
        ? "border-[rgba(191,219,254,0.92)] bg-[rgba(239,246,255,0.96)] text-[rgb(29,78,216)]"
        : status === "running" || status === "resuming"
          ? "border-[rgba(186,230,253,0.92)] bg-[rgba(236,254,255,0.96)] text-[rgb(14,116,144)]"
          : "border-[rgba(187,247,208,0.92)] bg-[rgba(240,253,244,0.96)] text-[rgb(21,128,61)]";

  const StatusIcon =
    status === "failed" || status === "cancelled"
      ? Flame
      : status === "waiting_clarify"
        ? Hourglass
        : status === "running" || status === "resuming"
          ? CircleDashed
          : Waypoints;

  return (
    <div className="status-strip rounded-[1.4rem] border px-4 py-3">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div>
          <p className="tech-label text-[0.64rem] text-[rgba(71,85,105,0.58)]">当前训练</p>
          <h2 className="mt-1 font-display text-[1.2rem] text-[rgb(15,23,42)]">
            {modeLabel} · {phaseLabel}
          </h2>
          <p className="mt-1 text-sm text-[rgba(51,65,85,0.74)]">
            {personaLabel} · 时间预算 {timeBudget}
            {hasReviewData ? " · 已累计复盘结果" : ""}
          </p>
        </div>

        <div className={`status-badge gap-2 px-3 py-1.5 text-[0.76rem] ${statusTone}`}>
          <StatusIcon className={`h-3.5 w-3.5 ${status === "running" || status === "resuming" ? "animate-spin" : ""}`} />
          {statusLabel}
        </div>
      </div>

      <div className="mt-3 flex flex-wrap gap-2">
        <StatusPill icon={<Radar className="h-3.5 w-3.5" />} label="模式" value={modeLabel} />
        <StatusPill icon={<Sparkles className="h-3.5 w-3.5" />} label="人格" value={personaLabel} />
        <StatusPill icon={<Waypoints className="h-3.5 w-3.5" />} label="阶段" value={phaseLabel} />
        <StatusPill icon={<Hourglass className="h-3.5 w-3.5" />} label="预算" value={timeBudget} />
      </div>
    </div>
  );
}

function StatusPill({ icon, label, value }: { icon: ReactNode; label: string; value: string }) {
  return (
    <div className="status-badge gap-2 border-[rgba(191,219,254,0.72)] bg-[rgba(255,255,255,0.82)] px-3 py-1.5 text-sm text-[rgb(30,41,59)]">
      <span className="inline-flex h-6 w-6 items-center justify-center rounded-full bg-[rgba(239,246,255,0.96)] text-[rgb(29,78,216)]">
        {icon}
      </span>
      <span className="text-[rgba(71,85,105,0.72)]">{label}</span>
      <span className="font-medium text-[rgb(15,23,42)]">{value}</span>
    </div>
  );
}
