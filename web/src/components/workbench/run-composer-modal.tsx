import { CircleDashed, LoaderCircle } from "lucide-react";
import type { Dispatch, ReactNode, SetStateAction } from "react";
import { useEffect, useRef, useState } from "react";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Textarea } from "@/components/ui/textarea";
import type { InterviewConfig, ModelConfig, OutputStyle } from "@/lib/types";

type RunComposerModalProps = {
  workspaceTitle: string;
  prompt: string;
  config: InterviewConfig;
  modelConfig: ModelConfig;
  isSubmitting: boolean;
  onClose: () => void;
  onPromptChange: (value: string) => void;
  onConfigChange: Dispatch<SetStateAction<InterviewConfig>>;
  onModelConfigChange: Dispatch<SetStateAction<ModelConfig>>;
  onSubmit: (event: React.FormEvent<HTMLFormElement>) => Promise<void>;
  levelOptions: Array<{ value: string; label: string }>;
  timeBudgetOptions: Array<{ value: string; label: string }>;
};

export function RunComposerModal({
  workspaceTitle,
  prompt,
  config,
  modelConfig,
  isSubmitting,
  onClose,
  onPromptChange,
  onConfigChange,
  onModelConfigChange,
  onSubmit,
  levelOptions,
  timeBudgetOptions,
}: RunComposerModalProps) {
  const [showAdvancedModelConfig, setShowAdvancedModelConfig] = useState(false);

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-[rgba(15,23,42,0.16)] px-4 py-6 backdrop-blur-[2px]" onClick={onClose}>
      <div
        className="modal-surface modal-shell flex max-h-[82vh] w-full max-w-[860px] flex-col overflow-hidden"
        onClick={(event) => event.stopPropagation()}
      >
        <div className="flex items-start justify-between gap-4">
          <div>
            <p className="tech-label text-[0.68rem] text-[rgba(97,123,150,0.62)]">新建运行</p>
            <h2 className="mt-2 font-display text-[1.45rem] text-[rgb(17,24,39)]">配置并启动一场新的面试</h2>
            <p className="mt-2 max-w-[48ch] text-sm leading-7 text-[rgba(115,137,161,0.82)]">
              从这里定义训练方式、输出样式和时间预算。启动后，系统会持续追问，并在结束后聚合评分、追问链路和画像结论。
            </p>
          </div>
          <Button
            type="button"
            variant="outline"
            onClick={onClose}
            className="field-surface h-9 rounded-full px-4 text-[rgb(55,65,81)]"
          >
            收起
          </Button>
        </div>

        <form onSubmit={(event) => void onSubmit(event)} className="mt-4 min-h-0 flex-1 overflow-y-auto pr-1">
          <div className="grid gap-4 lg:grid-cols-[1.05fr_0.95fr]">
            <div className="grid gap-4">
              <Field label="工作区标题">
                <div className="surface-secondary rounded-[1rem] px-4 py-3 text-[rgb(72,91,114)]">
                  <p className="truncate text-base">{workspaceTitle}</p>
                  <p className="mt-1 text-xs leading-5 text-[rgba(107,114,128,0.72)]">将根据当前提示词自动生成，可在工作区列表中随时重命名。</p>
                </div>
              </Field>

              <Field label="任务提示词">
                <Textarea
                  value={prompt}
                  onChange={(event) => onPromptChange(event.target.value)}
                  className="field-surface min-h-[132px] rounded-[1.15rem]"
                />
              </Field>

              <div className="grid gap-3 md:grid-cols-2">
                <Field label="模式">
                  <Select
                    value={config.mode}
                    onValueChange={(value) => onConfigChange((current) => ({ ...current, mode: value as InterviewConfig["mode"] }))}
                  >
                    <SelectTrigger className="field-surface rounded-[1rem]">
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="standard">标准面试</SelectItem>
                      <SelectItem value="stress">压力面试</SelectItem>
                      <SelectItem value="weakness_focused">查漏补缺</SelectItem>
                      <SelectItem value="system_design">系统设计专项</SelectItem>
                      <SelectItem value="resume_deep_dive">简历深挖</SelectItem>
                    </SelectContent>
                  </Select>
                </Field>

                <Field label="输出样式">
                  <Select
                    value={config.outputStyle}
                    onValueChange={(value) => onConfigChange((current) => ({ ...current, outputStyle: value as OutputStyle }))}
                  >
                    <SelectTrigger className="field-surface rounded-[1rem]">
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="interview_only">仅面试</SelectItem>
                      <SelectItem value="interview_plus_score">面试 + 评分</SelectItem>
                      <SelectItem value="interview_plus_score_and_study_plan">面试 + 评分 + 学习计划</SelectItem>
                    </SelectContent>
                  </Select>
                </Field>
              </div>
            </div>

            <div className="grid gap-4">
              <div className="grid grid-cols-2 gap-3">
                <WheelPickerField
                  label="级别"
                  value={config.level}
                  options={levelOptions}
                  onChange={(value) => onConfigChange((current) => ({ ...current, level: value }))}
                />
                <WheelPickerField
                  label="时间预算"
                  value={config.timeBudget}
                  options={timeBudgetOptions}
                  onChange={(value) => onConfigChange((current) => ({ ...current, timeBudget: value }))}
                />
              </div>

              <div className="surface-secondary rounded-[1.1rem] p-4">
                <p className="tech-label text-[0.64rem] text-[rgba(97,123,150,0.62)]">本场将得到</p>
                <div className="mt-3 space-y-2">
                  <div className="rounded-[1rem] border border-[rgba(191,219,254,0.9)] bg-[rgba(239,246,255,0.92)] px-3 py-3 text-sm leading-6 text-[rgb(30,64,175)]">
                    结构化评分卡与关键短板
                  </div>
                  <div className="rounded-[1rem] border border-[rgba(186,230,253,0.9)] bg-[rgba(236,254,255,0.92)] px-3 py-3 text-sm leading-6 text-[rgb(14,116,144)]">
                    追问树、阶段变化和过程解释
                  </div>
                </div>
              </div>

              <div className="surface-secondary rounded-[1.1rem] p-4">
                <button
                  type="button"
                  onClick={() => setShowAdvancedModelConfig((current) => !current)}
                  className="flex w-full items-center justify-between text-left"
                >
                  <div>
                    <p className="tech-label text-[0.64rem] text-[rgba(97,123,150,0.62)]">模型高级设置</p>
                    <p className="mt-1 text-xs leading-6 text-[rgba(115,137,161,0.78)]">
                      默认使用后端环境变量；只有在需要临时覆盖 provider、模型或 base URL 时再展开。
                    </p>
                  </div>
                  <span className="control-chip h-8 px-3 text-xs normal-case tracking-[0.04em]">
                    {showAdvancedModelConfig ? "收起" : "展开"}
                  </span>
                </button>
                {showAdvancedModelConfig ? (
                  <>
                    <div className="mt-3 grid gap-3 md:grid-cols-2">
                      <Field label="模型提供方">
                        <Input
                          value={modelConfig.provider}
                          onChange={(event) => onModelConfigChange((current) => ({ ...current, provider: event.target.value }))}
                          className="field-surface rounded-[1rem]"
                        />
                      </Field>
                      <Field label="模型名称">
                        <Input
                          value={modelConfig.model}
                          onChange={(event) => onModelConfigChange((current) => ({ ...current, model: event.target.value }))}
                          className="field-surface rounded-[1rem]"
                        />
                      </Field>
                    </div>
                    <div className="mt-3 grid gap-3">
                      <Field label="API 密钥">
                        <Input
                          type="password"
                          value={modelConfig.apiKey ?? ""}
                          onChange={(event) => onModelConfigChange((current) => ({ ...current, apiKey: event.target.value }))}
                          className="field-surface rounded-[1rem]"
                        />
                      </Field>
                      <Field label="基础地址">
                        <Input
                          value={modelConfig.baseUrl ?? ""}
                          onChange={(event) => onModelConfigChange((current) => ({ ...current, baseUrl: event.target.value }))}
                          className="field-surface rounded-[1rem]"
                        />
                      </Field>
                    </div>
                  </>
                ) : null}
              </div>

              <Button type="submit" disabled={isSubmitting} className="h-11 w-full rounded-full bg-[rgb(0,102,255)] text-primary-foreground shadow-[0_18px_40px_rgba(0,102,255,0.2)] hover:bg-[rgb(0,88,220)]">
                {isSubmitting ? (
                  <>
                    <LoaderCircle className="mr-2 h-4 w-4 animate-spin" />
                    正在启动...
                  </>
                ) : (
                  <>
                    <CircleDashed className="mr-2 h-4 w-4" />
                    启动运行
                  </>
                )}
              </Button>
            </div>
          </div>
        </form>
      </div>
    </div>
  );
}

function Field({ label, children }: { label: string; children: ReactNode }) {
  return (
    <div className="space-y-2">
      <Label className="tech-label text-[0.64rem] text-[rgba(97,123,150,0.66)]">{label}</Label>
      {children}
    </div>
  );
}

function WheelPickerField({
  label,
  value,
  options,
  onChange,
}: {
  label: string;
  value: string;
  options: Array<{ value: string; label: string }>;
  onChange: (value: string) => void;
}) {
  const containerRef = useRef<HTMLDivElement>(null);
  const itemRefs = useRef<Record<string, HTMLButtonElement | null>>({});

  useEffect(() => {
    const container = containerRef.current;
    const selected = itemRefs.current[value];
    if (!container || !selected) {
      return;
    }

    const targetTop = selected.offsetTop - (container.clientHeight - selected.clientHeight) / 2;
    container.scrollTo({ top: Math.max(targetTop, 0), behavior: "smooth" });
  }, [value]);

  function handleScroll() {
    const container = containerRef.current;
    if (!container) {
      return;
    }

    const center = container.scrollTop + container.clientHeight / 2;
    let nearest: { value: string; distance: number } | null = null;

    for (const option of options) {
      const node = itemRefs.current[option.value];
      if (!node) continue;
      const nodeCenter = node.offsetTop + node.clientHeight / 2;
      const distance = Math.abs(center - nodeCenter);
      if (!nearest || distance < nearest.distance) {
        nearest = { value: option.value, distance };
      }
    }

    if (nearest && nearest.value !== value) {
      onChange(nearest.value);
    }
  }

  return (
    <Field label={label}>
      <div className="surface-soft relative overflow-hidden rounded-[1.2rem] p-2 shadow-[inset_0_1px_0_rgba(255,255,255,0.84)]">
        <div className="pointer-events-none absolute inset-x-2 top-1/2 z-10 h-[3rem] -translate-y-1/2 rounded-[1rem] border border-[rgba(105,183,196,0.2)] bg-[rgba(255,255,255,0.78)] shadow-[0_8px_18px_rgba(109,177,189,0.12)]" />
        <div className="pointer-events-none absolute inset-x-0 top-0 z-10 h-8 bg-[linear-gradient(180deg,rgba(245,252,253,0.96),rgba(245,252,253,0))]" />
        <div className="pointer-events-none absolute inset-x-0 bottom-0 z-10 h-8 bg-[linear-gradient(0deg,rgba(245,252,253,0.96),rgba(245,252,253,0))]" />
        <div
          ref={containerRef}
          onScroll={handleScroll}
          className="h-32 snap-y snap-mandatory overflow-y-auto py-8 [scrollbar-width:none] [-ms-overflow-style:none] [&::-webkit-scrollbar]:hidden"
        >
          {options.map((option) => {
            const active = option.value === value;

            return (
              <button
                key={option.value}
                type="button"
                ref={(node) => {
                  itemRefs.current[option.value] = node;
                }}
                onClick={() => onChange(option.value)}
                className={`flex h-[3rem] w-full snap-center items-center justify-between rounded-[1rem] border px-3 text-left transition-all duration-300 ${
                  active
                    ? "border-[rgba(105,183,196,0.22)] bg-[linear-gradient(145deg,rgba(255,255,255,0.98),rgba(238,250,252,0.98))] text-[rgb(24,91,108)]"
                    : "border-transparent bg-transparent text-[rgba(77,117,128,0.74)] hover:text-[rgb(36,95,110)]"
                }`}
              >
                <p className={`text-[0.95rem] ${active ? "font-medium" : ""}`}>{option.label}</p>
                <p className="text-[0.62rem] uppercase tracking-[0.18em]">{active ? "当前" : ""}</p>
              </button>
            );
          })}
        </div>
      </div>
    </Field>
  );
}
