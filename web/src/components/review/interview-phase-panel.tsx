import type { InterviewTraceTree, Run } from "@/lib/types";

type InterviewPhasePanelProps = {
  run: Run | null;
  trace: InterviewTraceTree | null;
  onFocusMessage: (messageID?: string) => void;
  formatPhaseLabel: (phase?: string | null) => string;
  formatDecisionReasonLabel: (reason?: string | null) => string;
};

export function InterviewPhasePanel({
  run,
  trace,
  onFocusMessage,
  formatPhaseLabel,
  formatDecisionReasonLabel,
}: InterviewPhasePanelProps) {
  const history = run?.interviewState?.history ?? [];
  const nodes = trace?.nodes ?? [];
  const firstAdversarialRound = history.find((entry) => entry.adversarial)?.round;
  const firstPressureRound = history.find((entry) => entry.pressure)?.round;
  const firstWrapupRound = history.find((entry) => entry.phase === "wrapup")?.round;
  const historicalHitCount = nodes.filter((node) => node.profileHit).length;
  const phaseSummary = firstPressureRound
    ? `系统在第 ${firstPressureRound} 轮开始明显上强度。`
    : firstAdversarialRound
      ? `系统在第 ${firstAdversarialRound} 轮开始进入反驳深挖。`
      : history.length > 0
        ? "当前还处于相对温和的推进阶段。"
        : "等待阶段数据。";

  return (
    <section className="panel-card">
      <div className="flex items-start justify-between gap-4">
        <div>
          <p className="tech-label text-[0.68rem] text-[rgba(97,123,150,0.62)]">阶段视图</p>
          <h2 className="mt-2 font-display text-2xl text-[rgb(72,91,114)]">
            {history.length > 0 ? `${history.length} 轮节奏变化` : "等待阶段数据"}
          </h2>
        </div>
        {run?.interviewState?.phase ? (
          <div className="chip-info px-3 py-1.5 text-[0.68rem] tracking-[0.2em]">
            当前 · {formatPhaseLabel(run.interviewState.phase)}
          </div>
        ) : null}
      </div>
      <p className="mt-3 max-w-[64ch] text-sm leading-7 text-[rgba(115,137,161,0.78)]">{phaseSummary}</p>

      {history.length === 0 ? (
        <p className="mt-5 rounded-[1.3rem] border border-dashed border-[rgba(153,191,201,0.18)] px-4 py-5 text-sm leading-7 text-[rgba(115,137,161,0.78)]">
          面试推进后，这里会显示每轮属于哪个阶段、什么时候进入压力追问，以及哪些轮次暴露了薄弱信号。
        </p>
      ) : (
        <div className="mt-5 space-y-3">
          <div className="flex flex-wrap gap-2">
            {typeof firstAdversarialRound === "number" ? (
              <span className="chip-warning px-3 py-1.5 text-xs normal-case tracking-[0.04em]">
                第 {firstAdversarialRound} 轮进入 adversarial
              </span>
            ) : null}
            {typeof firstPressureRound === "number" ? (
              <span className="chip-danger px-3 py-1.5 text-xs normal-case tracking-[0.04em]">
                第 {firstPressureRound} 轮进入 stress
              </span>
            ) : null}
            {typeof firstWrapupRound === "number" ? (
              <span className="chip-accent px-3 py-1.5 text-xs normal-case tracking-[0.04em]">
                第 {firstWrapupRound} 轮进入 wrapup
              </span>
            ) : null}
            {historicalHitCount > 0 ? (
              <span className="chip-warning px-3 py-1.5 text-xs normal-case tracking-[0.04em]">
                历史弱项命中 {historicalHitCount} 次
              </span>
            ) : null}
          </div>
          {history.map((entry, index) => {
            const node = nodes.find((item) => item.round === entry.round) ?? null;
            return (
              <button
                key={`${entry.round}-${entry.phase}-${index}`}
                type="button"
                onClick={() => onFocusMessage(node?.messageId)}
                className="panel-card-soft block w-full text-left transition-colors hover:border-[rgba(59,130,246,0.18)] hover:bg-[rgba(248,250,255,0.98)]"
              >
                <div className="flex flex-wrap items-center gap-2">
                  <span className="chip-neutral px-2.5 py-1 text-[0.68rem] tracking-[0.2em]">
                    第 {entry.round} 轮
                  </span>
                  <span className="chip-info px-2.5 py-1 text-[0.68rem] tracking-[0.2em]">
                    {formatPhaseLabel(entry.phase)}
                  </span>
                  <span className="chip-neutral px-2.5 py-1 text-[0.68rem] tracking-[0.2em]">
                    压力 {entry.difficulty}
                  </span>
                  {entry.adversarial ? (
                    <span className="chip-warning px-2.5 py-1 text-[0.68rem] tracking-[0.2em]">
                      反驳
                    </span>
                  ) : null}
                  {entry.pressure ? (
                    <span className="chip-danger px-2.5 py-1 text-[0.68rem] tracking-[0.2em]">
                      压力
                    </span>
                  ) : null}
                </div>
                {entry.scenario ? <p className="mt-3 text-sm leading-7 text-[rgb(31,41,55)]">场景：{entry.scenario}</p> : null}
                {entry.reason ? (
                  <p className="mt-2 text-sm leading-7 text-[rgb(37,99,235)]">决策原因：{formatDecisionReasonLabel(entry.reason)}</p>
                ) : null}
                {entry.explanation ? (
                  <p className="mt-2 text-sm leading-7 text-[rgba(72,91,114,0.82)]">{entry.explanation}</p>
                ) : null}
                {node?.question ? <p className="mt-2 text-sm leading-7 text-[rgba(71,85,105,0.9)]">问题：{node.question}</p> : null}
                {node?.focusHits?.length ? (
                  <div className="mt-3 flex flex-wrap gap-2">
                    {node.focusHits.map((item) => (
                      <span key={`${entry.round}-${item}`} className="chip-warning px-2.5 py-1 text-[0.68rem] tracking-[0.18em]">
                        历史命中 · {item}
                      </span>
                    ))}
                  </div>
                ) : null}
                {entry.weakSignals?.length || entry.strongSignals?.length ? (
                  <div className="mt-3 flex flex-wrap gap-2">
                    {entry.weakSignals?.map((signal) => (
                      <span key={`weak-${entry.round}-${signal}`} className="chip-danger px-2.5 py-1 text-[0.68rem] tracking-[0.18em]">
                        {signal}
                      </span>
                    ))}
                    {entry.strongSignals?.map((signal) => (
                      <span key={`strong-${entry.round}-${signal}`} className="chip-success px-2.5 py-1 text-[0.68rem] tracking-[0.18em]">
                        {signal}
                      </span>
                    ))}
                  </div>
                ) : null}
              </button>
            );
          })}
        </div>
      )}
    </section>
  );
}
