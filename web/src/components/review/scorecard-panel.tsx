import type { ReactNode } from "react";

import { ArrowRight, CheckCircle2, Gauge, RotateCcw, Target } from "lucide-react";

import { Button } from "@/components/ui/button";
import type { OutputStyle, RunEvent, Scorecard } from "@/lib/types";

type ScorecardPanelProps = {
  scorecard: Scorecard | null;
  outputStyle: OutputStyle | undefined;
  onRetry?: () => void;
  retryDisabled: boolean;
  latestFailure: RunEvent | null;
};

export function ScorecardPanel({
  scorecard,
  outputStyle,
  onRetry,
  retryDisabled,
  latestFailure,
}: ScorecardPanelProps) {
  const anchors = scorecard?.anchors ?? [];
  const wantsStudyPlan = outputStyle === "interview_plus_score_and_study_plan";
  const strengths = scorecard?.strengths ?? [];
  const gaps = scorecard?.gaps ?? [];
  const improvements = scorecard?.improvements ?? [];
  const studyPlan = scorecard?.studyPlan ?? [];
  const summary = scorecard?.summary?.trim() ?? "";
  const dimensionScores = scorecard?.dimensionScores ?? [];
  const priorityFocus = improvements[0] ?? gaps[0] ?? anchors[0] ?? "";
  const averageScore =
    dimensionScores.length > 0
      ? dimensionScores.reduce((total, dimension) => total + dimension.score, 0) / dimensionScores.length
      : null;
  const averagePercentage = averageScore ? Math.round((averageScore / 5) * 100) : null;
  const overallTone = getScoreTone(averageScore);
  const overallLabel = averageScore ? getOverallLabel(averageScore) : "等待评分";
  const nextAction = improvements[0] ?? gaps[0] ?? "等评分生成后，这里会告诉你下一场最该优先补什么。";
  const standoutStrength = strengths[0] ?? anchors[0] ?? "等结构化评分生成后，这里会总结你最稳的一项表现。";

  return (
    <section className="panel-card">
      <p className="tech-label text-[0.68rem] text-[rgba(97,123,150,0.62)]">评分卡</p>
      <h2 className="mt-2 font-display text-2xl text-[rgb(72,91,114)]">{scorecard?.title ?? "等待评分结果"}</h2>
      <p className="mt-2 max-w-[64ch] text-sm leading-7 text-[rgba(115,137,161,0.8)]">
        {summary || "先看这里判断本场整体表现，再向下查看关键短板、建议和训练计划。"}
      </p>

      <div className="mt-5 grid gap-3 xl:grid-cols-[1.25fr_1fr_1fr]">
        <div className="rounded-[1.45rem] border border-[rgba(191,219,254,0.92)] bg-[linear-gradient(135deg,rgba(239,246,255,0.98),rgba(248,250,255,0.98))] px-5 py-5">
          <div className="flex items-start justify-between gap-4">
            <div>
              <p className="tech-label text-[0.64rem] text-[rgb(29,78,216)]">本场整体表现</p>
              <h3 className="mt-2 text-xl font-semibold text-[rgb(30,64,175)]">{overallLabel}</h3>
            </div>
            <div className="rounded-full border border-[rgba(191,219,254,0.92)] bg-white/80 p-2 text-[rgb(37,99,235)]">
              <Gauge className="h-4 w-4" />
            </div>
          </div>
          <div className="mt-4">
            <div className="h-2.5 overflow-hidden rounded-full bg-white/80">
              <div
                className={`h-full rounded-full transition-all ${overallTone.barClass}`}
                style={{ width: `${averagePercentage ?? 0}%` }}
              />
            </div>
            <div className="mt-3 flex items-end justify-between gap-4">
              <div>
                <p className="text-3xl font-semibold tracking-[-0.04em] text-[rgb(15,23,42)]">
                  {averageScore ? averageScore.toFixed(1) : "--"}
                  <span className="ml-1 text-base font-medium text-[rgba(71,85,105,0.72)]">/ 5</span>
                </p>
                <p className="mt-1 text-sm text-[rgba(71,85,105,0.78)]">
                  {averagePercentage ? `约领先 ${averagePercentage}% 的目标完成度` : "等待后端评分返回结构化结果"}
                </p>
              </div>
              <span className={`rounded-full px-3 py-1 text-xs font-medium ${overallTone.badgeClass}`}>{overallTone.label}</span>
            </div>
          </div>
        </div>

        <OverviewCallout
          icon={<Target className="h-4 w-4" />}
          label="当前最该先修"
          text={priorityFocus || "先完成一轮评分后，这里会给出最优先的修正方向。"}
          tone="warning"
        />
        <OverviewCallout
          icon={<CheckCircle2 className="h-4 w-4" />}
          label="你这次最稳的点"
          text={standoutStrength}
          tone="success"
        />
      </div>

      <div className="mt-4 rounded-[1.3rem] border border-[rgba(214,222,234,0.92)] bg-[rgba(248,250,252,0.94)] px-4 py-4">
        <div className="flex items-start gap-3">
          <div className="rounded-full border border-[rgba(191,219,254,0.92)] bg-[rgba(239,246,255,0.96)] p-2 text-[rgb(37,99,235)]">
            <ArrowRight className="h-4 w-4" />
          </div>
          <div>
            <p className="tech-label text-[0.64rem] text-[rgba(97,123,150,0.68)]">下一步最值得做</p>
            <p className="mt-2 text-sm leading-7 text-[rgb(51,65,85)]">{nextAction}</p>
          </div>
        </div>
      </div>

      {dimensionScores.length > 0 ? (
        <div className="mt-6">
          <p className="tech-label text-[0.68rem] text-[rgba(97,123,150,0.62)]">维度分数</p>
          <div className="mt-3 grid gap-3 sm:grid-cols-2">
            {dimensionScores.map((dimension) => {
              const tone = getScoreTone(dimension.score);
              const percentage = Math.round((dimension.score / 5) * 100);
              return (
                <div
                  key={`${dimension.name}-${dimension.score}`}
                  className="rounded-[1.35rem] border border-[rgba(214,222,234,0.92)] bg-[rgba(255,255,255,0.95)] px-4 py-4 shadow-[0_16px_42px_-34px_rgba(15,23,42,0.38)]"
                >
                  <div className="flex items-start justify-between gap-4">
                    <div>
                      <p className="text-sm font-semibold text-[rgb(51,65,85)]">{dimension.name}</p>
                      <p className="mt-1 text-xs uppercase tracking-[0.16em] text-[rgba(100,116,139,0.74)]">{tone.label}</p>
                    </div>
                    <span className={`rounded-full px-3 py-1 text-xs font-medium ${tone.badgeClass}`}>
                      {dimension.score}/5
                    </span>
                  </div>

                  <div className="mt-4">
                    <div className="h-2.5 overflow-hidden rounded-full bg-[rgba(226,232,240,0.72)]">
                      <div
                        className={`h-full rounded-full transition-all ${tone.barClass}`}
                        style={{ width: `${percentage}%` }}
                      />
                    </div>
                    <div className="mt-2 flex items-center justify-between text-xs text-[rgba(100,116,139,0.78)]">
                      <span>当前完成度</span>
                      <span>{percentage}%</span>
                    </div>
                  </div>

                  {dimension.rationale ? (
                    <p className="mt-4 text-sm leading-6 text-[rgba(71,85,105,0.84)]">{dimension.rationale}</p>
                  ) : null}
                </div>
              );
            })}
          </div>
        </div>
      ) : null}

      {anchors.length === 0 ? (
        <p className="mt-5 rounded-[1.3rem] border border-dashed border-[rgba(153,191,201,0.18)] px-4 py-5 text-sm leading-7 text-[rgba(115,137,161,0.78)]">
          运行到达评分阶段后，结构化评分会显示在这里。
        </p>
      ) : (
        <div className="mt-5 space-y-3">
          {anchors.map((anchor) => (
            <div
              key={anchor}
              className="panel-card-soft text-sm leading-7 text-[rgb(72,91,114)]"
            >
              {anchor}
            </div>
          ))}
        </div>
      )}

      <ScoreSection title="优势" items={strengths} tone="neutral" />
      <ScoreSection title="短板" items={gaps} tone="warning" />
      <ScoreSection title="改进建议" items={improvements} tone="success" />

      {wantsStudyPlan ? (
        <div className="mt-6">
          <p className="tech-label text-[0.68rem] text-[rgba(97,123,150,0.62)]">学习计划</p>
          {studyPlan.length === 0 ? (
            <p className="mt-3 rounded-[1.3rem] border border-dashed border-[rgba(153,191,201,0.18)] px-4 py-5 text-sm leading-7 text-[rgba(115,137,161,0.78)]">
              本次运行要求生成学习计划，但后端暂未返回结构化学习项。
            </p>
          ) : (
            <div className="mt-3 space-y-3">
              {studyPlan.map((item, index) => (
                <div
                  key={`${index}-${item}`}
                  className="rounded-[1.3rem] border border-[rgba(187,247,208,0.94)] bg-[rgba(240,253,244,0.94)] px-4 py-4 text-sm leading-7 text-[rgb(21,128,61)]"
                >
                  {item}
                </div>
              ))}
            </div>
          )}
        </div>
      ) : null}

      {latestFailure && onRetry ? (
        <div className="mt-6 rounded-[1.4rem] border border-[rgba(191,219,254,0.92)] bg-[rgba(239,246,255,0.96)] px-4 py-4">
          <p className="font-serif text-xs uppercase tracking-[0.28em] text-[rgb(0,102,255)]">恢复运行</p>
          <p className="mt-2 text-sm leading-7 text-[rgb(0,102,255)]">
            最近一次运行失败。修复阻塞问题后，可以基于当前任务配置重新启动一次运行。
          </p>
          <Button
            type="button"
            onClick={onRetry}
            disabled={retryDisabled}
            className="mt-4 rounded-full bg-[rgb(0,102,255)] px-4 text-primary-foreground hover:bg-[rgb(0,88,220)]"
          >
            <RotateCcw className="mr-2 h-4 w-4" />
            启动恢复运行
          </Button>
        </div>
      ) : null}
    </section>
  );
}

function OverviewCallout({
  icon,
  label,
  text,
  tone,
}: {
  icon: ReactNode;
  label: string;
  text: string;
  tone: "success" | "warning";
}) {
  const className =
    tone === "warning"
      ? "border-[rgba(253,230,138,0.92)] bg-[rgba(254,252,232,0.96)] text-[rgb(161,98,7)]"
      : "border-[rgba(187,247,208,0.94)] bg-[rgba(240,253,244,0.94)] text-[rgb(21,128,61)]";

  return (
    <div className={`rounded-[1.35rem] border px-4 py-4 ${className}`}>
      <div className="flex items-start gap-3">
        <div className="rounded-full border border-white/70 bg-white/70 p-2">{icon}</div>
        <div>
          <p className="tech-label text-[0.64rem] opacity-80">{label}</p>
          <p className="mt-2 text-sm leading-7">{text}</p>
        </div>
      </div>
    </div>
  );
}

function ScoreSection({
  title,
  items,
  tone,
}: {
  title: string;
  items: string[];
  tone: "neutral" | "warning" | "success";
}) {
  if (items.length === 0) {
    return null;
  }

  const toneClass =
    tone === "warning"
      ? "border-[rgba(253,230,138,0.92)] bg-[rgba(254,252,232,0.96)] text-[rgb(161,98,7)]"
      : tone === "success"
        ? "border-[rgba(187,247,208,0.94)] bg-[rgba(240,253,244,0.94)] text-[rgb(21,128,61)]"
        : "border-[rgba(153,191,201,0.16)] bg-[rgba(252,255,255,0.88)] text-[rgb(72,91,114)]";

  return (
    <div className="mt-6">
      <p className="tech-label text-[0.68rem] text-[rgba(97,123,150,0.62)]">{title}</p>
      <div className="mt-3 space-y-3">
        {items.map((item, index) => (
          <div key={`${title}-${index}-${item}`} className={`rounded-[1.3rem] border px-4 py-4 text-sm leading-7 ${toneClass}`}>
            {item}
          </div>
        ))}
      </div>
    </div>
  );
}

function getScoreTone(score: number | null) {
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

function getOverallLabel(score: number) {
  if (score >= 4.2) {
    return "这场发挥很稳，可以开始看更高阶的优化";
  }
  if (score >= 3.3) {
    return "整体合格，但还有几处会被继续深挖";
  }
  return "目前短板比较明确，建议先补关键表达和实现层细节";
}
