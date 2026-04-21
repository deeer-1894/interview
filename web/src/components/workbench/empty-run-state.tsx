import { ArrowUpRight, Radar, Sparkles, Waypoints } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Textarea } from "@/components/ui/textarea";

const promptExamples = [
  {
    label: "Go Agent",
    prompt: "请模拟一场 Go agent 开发岗位的技术面试，并在最后给出结构化评分。",
  },
  {
    label: "系统设计",
    prompt: "请模拟一场系统设计专项面试，重点考察高并发、可观测性和故障恢复能力。",
  },
  {
    label: "简历深挖",
    prompt: "请围绕我的后端项目经历模拟一场简历深挖面试，持续追问 tradeoff、实现细节和 ownership。",
  },
];

type EmptyRunStateProps = {
  hasRun: boolean;
  prompt: string;
  isSubmitting: boolean;
  onPromptChange: (value: string) => void;
  onLaunch: () => void;
  onOpenComposer: () => void;
  onApplyPrompt: (value: string) => void;
};

export function EmptyRunState({
  hasRun,
  prompt,
  isSubmitting,
  onPromptChange,
  onLaunch,
  onOpenComposer,
  onApplyPrompt,
}: EmptyRunStateProps) {
  const launchDisabled = isSubmitting || !prompt.trim();

  return (
    <div className="flex h-full min-h-[520px] items-center justify-center py-6">
      <div className="hero-panel w-full max-w-[72rem] overflow-hidden rounded-[2rem] border border-[rgba(191,219,254,0.62)] shadow-[0_28px_80px_rgba(15,23,42,0.08)]">
        <div className="grid gap-6 p-5 lg:grid-cols-[1.1fr_0.9fr] lg:p-7">
          <section className="rounded-[1.75rem] border border-[rgba(255,255,255,0.72)] bg-[rgba(255,255,255,0.88)] p-5 shadow-[inset_0_1px_0_rgba(255,255,255,0.9)] lg:p-6">
            <div className="inline-flex items-center gap-2 rounded-full border border-[rgba(191,219,254,0.96)] bg-[rgba(239,246,255,0.94)] px-3 py-1.5 text-[0.72rem] uppercase tracking-[0.18em] text-[rgb(29,78,216)]">
              <Sparkles className="h-3.5 w-3.5" />
              {hasRun ? "重新组织下一场训练" : "开始一场真实感更强的模拟面试"}
            </div>

            <h2 className="mt-5 max-w-[13ch] font-display text-[2.35rem] leading-[1.02] tracking-[-0.03em] text-[rgb(15,23,42)] lg:text-[3rem]">
              持续追问、阶段升级、结构化复盘。
            </h2>
            <p className="mt-4 max-w-[60ch] text-[0.98rem] leading-8 text-[rgba(51,65,85,0.82)]">
              输入岗位、公司或训练方向后就能直接开始。系统会保留上下文、追问链路和复盘结果，不只是陪你聊一段对话。
            </p>

            <div className="mt-6">
              <p className="tech-label text-[0.66rem] text-[rgba(71,85,105,0.62)]">快速开始</p>
              <Textarea
                value={prompt}
                onChange={(event) => onPromptChange(event.target.value)}
                placeholder="例如：请模拟一场字节跳动 Go agent 开发岗位的技术面试，并在最后给出结构化评分。"
                className="mt-3 min-h-[148px] rounded-[1.4rem] border-[rgba(191,219,254,0.72)] bg-[rgba(255,255,255,0.96)] text-[rgb(30,41,59)] shadow-[0_18px_40px_rgba(148,163,184,0.08)]"
              />
            </div>

            <div className="mt-4 flex flex-wrap gap-2">
              {promptExamples.map((example) => (
                <button
                  key={example.label}
                  type="button"
                  onClick={() => onApplyPrompt(example.prompt)}
                  className="inline-flex items-center rounded-full border border-[rgba(191,219,254,0.94)] bg-[rgba(248,250,255,0.92)] px-3 py-1.5 text-sm text-[rgb(30,64,175)] transition hover:border-[rgba(96,165,250,0.94)] hover:bg-[rgba(239,246,255,0.98)]"
                >
                  {example.label}
                </button>
              ))}
            </div>

            <div className="mt-6 flex flex-wrap gap-3">
              <Button
                type="button"
                onClick={onLaunch}
                disabled={launchDisabled}
                className="h-12 rounded-full bg-[rgb(8,47,73)] px-6 text-white shadow-[0_18px_40px_rgba(8,47,73,0.18)] hover:bg-[rgb(12,74,110)]"
              >
                <ArrowUpRight className="mr-2 h-4 w-4" />
                {isSubmitting ? "正在启动..." : "直接开始面试"}
              </Button>
              <Button
                type="button"
                variant="outline"
                onClick={onOpenComposer}
                className="h-12 rounded-full border-[rgba(191,219,254,0.94)] bg-[rgba(255,255,255,0.84)] px-5 text-[rgb(30,41,59)]"
              >
                高级配置
              </Button>
            </div>
          </section>

          <section className="grid gap-3">
            <div className="preview-tile">
              <div className="flex items-start justify-between gap-3">
                <div>
                  <p className="tech-label text-[0.64rem] text-[rgba(71,85,105,0.6)]">结果交付</p>
                  <h3 className="mt-2 font-display text-[1.3rem] text-[rgb(15,23,42)]">结构化评分卡</h3>
                </div>
                <Radar className="h-5 w-5 text-[rgb(37,99,235)]" />
              </div>
              <p className="mt-3 text-sm leading-7 text-[rgba(51,65,85,0.78)]">
                总评、关键短板、改进建议和学习计划会在一处聚合，不用再从对话里手动摘结论。
              </p>
            </div>

            <div className="preview-tile">
              <div className="flex items-start justify-between gap-3">
                <div>
                  <p className="tech-label text-[0.64rem] text-[rgba(71,85,105,0.6)]">过程可解释</p>
                  <h3 className="mt-2 font-display text-[1.3rem] text-[rgb(15,23,42)]">追问树与阶段视图</h3>
                </div>
                <Waypoints className="h-5 w-5 text-[rgb(8,145,178)]" />
              </div>
              <p className="mt-3 text-sm leading-7 text-[rgba(51,65,85,0.78)]">
                看到系统为什么继续深挖、在哪一轮进入压力、以及整场面试节奏是如何升级的。
              </p>
            </div>

            <div className="preview-tile">
              <div className="flex items-start justify-between gap-3">
                <div>
                  <p className="tech-label text-[0.64rem] text-[rgba(71,85,105,0.6)]">长期积累</p>
                  <h3 className="mt-2 font-display text-[1.3rem] text-[rgb(15,23,42)]">跨会话画像</h3>
                </div>
                <Sparkles className="h-5 w-5 text-[rgb(14,116,144)]" />
              </div>
              <p className="mt-3 text-sm leading-7 text-[rgba(51,65,85,0.78)]">
                系统会累计你的优势、重复短板和建议训练重点，让下一场面试更有针对性。
              </p>
            </div>
          </section>
        </div>
      </div>
    </div>
  );
}
