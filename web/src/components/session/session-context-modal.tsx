import { ChevronDown, FilePenLine, FileUp, Plus, Upload, X } from "lucide-react";
import type { ChangeEvent, Dispatch, SetStateAction } from "react";

import { Button } from "@/components/ui/button";
import type { Artifact, InterviewConfig, InterviewMode, InterviewPersona, SkillMetadata } from "@/lib/types";

type SessionPanel = "persona" | "skill" | "materials";

type SessionContextModalProps = {
  activePanel: SessionPanel;
  config: InterviewConfig;
  currentModeLabel: string;
  currentPhaseLabel: string;
  currentRunStatusLabel: string;
  selectedArtifactIDs: string[];
  artifacts: Artifact[];
  skills: SkillMetadata[];
  skillBusy: boolean;
  isSubmitting: boolean;
  selectedConversationId: string | null;
  onClose: () => void;
  onActivePanelChange: Dispatch<SetStateAction<SessionPanel | null>>;
  onConfigChange: Dispatch<SetStateAction<InterviewConfig>>;
  onSelectedArtifactIDsChange: Dispatch<SetStateAction<string[]>>;
  onSkillUpload: (event: ChangeEvent<HTMLInputElement>) => Promise<void>;
  onArtifactUpload: (event: ChangeEvent<HTMLInputElement>) => Promise<void>;
  onOpenCreateSkill: () => void;
  onOpenCreateArtifact: () => void;
  personaOptions: Array<{ value: InterviewPersona; label: string; hint: string }>;
  getPersonaMeta: (persona?: InterviewPersona | null) => { value: InterviewPersona; label: string; hint: string };
  formatInterviewModeLabel: (mode?: InterviewMode | null) => string;
};

export function SessionContextModal({
  activePanel,
  config,
  currentModeLabel,
  currentPhaseLabel,
  currentRunStatusLabel,
  selectedArtifactIDs,
  artifacts,
  skills,
  skillBusy,
  isSubmitting,
  selectedConversationId,
  onClose,
  onActivePanelChange,
  onConfigChange,
  onSelectedArtifactIDsChange,
  onSkillUpload,
  onArtifactUpload,
  onOpenCreateSkill,
  onOpenCreateArtifact,
  personaOptions,
  getPersonaMeta,
}: SessionContextModalProps) {
  const normalizedSkill = config.skill?.trim() ? config.skill.trim() : "";
  const activeSkill = skills.find((skill) => (skill.name?.trim() ?? "") === normalizedSkill) ?? null;
  const selectedSkillFocuses = config.skillFocuses ?? [];
  const panels = [
    { key: "persona" as const, label: "人格" },
    { key: "skill" as const, label: "技能" },
    { key: "materials" as const, label: "材料" },
  ];

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-[rgba(15,23,42,0.16)] px-4 py-6 backdrop-blur-[2px]" onClick={onClose}>
      <div
        className="modal-shell modal-surface flex h-[82vh] max-h-[860px] min-h-[620px] w-full max-w-[820px] flex-col overflow-hidden"
        onClick={(event) => event.stopPropagation()}
      >
        <div className="relative overflow-hidden rounded-[1.35rem] border border-[rgba(223,228,238,0.92)] bg-[rgba(255,255,255,0.94)] px-4 py-2.5">
          <div className="relative flex items-start justify-between gap-3">
            <div className="min-w-0 flex-1">
              <p className="tech-label text-[0.64rem] text-[rgba(97,123,150,0.58)]">会话上下文</p>
              <h2 className="mt-0.5 font-display text-[1.14rem] leading-[1.05] text-[rgb(17,24,39)]">调整当前会话设置</h2>
              <p className="mt-1 max-w-[48ch] line-clamp-1 text-[11px] leading-4 text-[rgba(71,85,105,0.68)]">
                人格、技能和材料都在这里管理，结果和评分已经直接放回会话主路径。
              </p>
              <div className="mt-1.5 flex flex-wrap gap-1">
                <SessionSummaryPill label="模式" value={currentModeLabel} />
                <SessionSummaryPill label="阶段" value={currentPhaseLabel} />
                <SessionSummaryPill label="技能" value={normalizedSkill || "自动选择"} />
                <SessionSummaryPill label="材料" value={`${selectedArtifactIDs.length} 份`} />
                <SessionSummaryPill label="状态" value={currentRunStatusLabel} tone="accent" />
              </div>
              <p className="mt-1 text-[10px] leading-4 text-[rgba(115,137,161,0.68)]">
                人格：{getPersonaMeta(config.persona).label} · 联网：{config.enableWebSearch ? "开启" : "关闭"}
              </p>
            </div>
            <Button
              type="button"
              variant="outline"
              onClick={onClose}
              className="field-surface h-8 rounded-full px-3.5 text-[13px] text-[rgb(55,65,81)]"
            >
              收起
            </Button>
          </div>
        </div>

        <div className="surface-secondary mt-4 inline-flex w-full max-w-fit rounded-[1.15rem] p-1 shadow-[inset_0_1px_0_rgba(255,255,255,0.9)]">
          {panels.map((panel) => (
            <button
              key={panel.key}
              type="button"
              onClick={() => onActivePanelChange(panel.key)}
              className={`rounded-[0.95rem] px-4 py-2 text-sm font-medium transition-all ${
                panel.key === activePanel
                  ? "bg-[rgba(255,255,255,0.96)] text-[rgb(37,99,235)] shadow-[0_10px_20px_rgba(37,99,235,0.08)]"
                  : "text-[rgba(71,85,105,0.78)] hover:text-[rgb(37,99,235)]"
              }`}
            >
              {panel.label}
            </button>
          ))}
        </div>

        <div className="mt-2 min-h-0 flex-1 overflow-y-auto pr-1">
          {activePanel === "persona" ? (
            <div className="surface-secondary h-full rounded-[1.2rem] px-4 py-4">
              <div className="relative">
                <select
                  value={config.persona ?? "rigorous"}
                  onChange={(event) =>
                    onConfigChange((current) => ({
                      ...current,
                      persona: event.target.value as InterviewPersona,
                    }))
                  }
                  className="field-surface h-11 w-full appearance-none rounded-[1rem] px-4 pr-10"
                >
                  {personaOptions.map((option) => (
                    <option key={option.value} value={option.value}>
                      {option.label}
                    </option>
                  ))}
                </select>
                <ChevronDown className="pointer-events-none absolute right-4 top-1/2 h-4 w-4 -translate-y-1/2 text-[rgba(97,123,150,0.72)]" />
              </div>
              <p className="mt-3 text-sm leading-7 text-[rgba(115,137,161,0.82)]">
                {getPersonaMeta(config.persona).hint}。切换后会在下一轮追问生效，不会影响当前已在运行的回复。
              </p>
            </div>
          ) : null}

          {activePanel === "skill" ? (
            <div className="surface-secondary h-full rounded-[1.2rem] px-4 py-4">
              <div className="flex flex-wrap items-center gap-2">
                <div className="relative min-w-[220px] flex-1">
                  <select
                    value={normalizedSkill || "__auto__"}
                    onChange={(event) =>
                      onConfigChange((current) => ({
                        ...current,
                        skill: event.target.value === "__auto__" ? "" : event.target.value,
                        skillFocuses: [],
                      }))
                    }
                    className="field-surface h-11 w-full appearance-none rounded-[1rem] px-4 pr-10"
                  >
                    <option value="__auto__">自动选择</option>
                    {skills
                      .filter((skill) => typeof skill.name === "string" && skill.name.trim())
                      .map((skill) => (
                        <option key={skill.name} value={skill.name.trim()}>
                          {skill.name.trim()}
                        </option>
                      ))}
                  </select>
                  <ChevronDown className="pointer-events-none absolute right-4 top-1/2 h-4 w-4 -translate-y-1/2 text-[rgba(97,123,150,0.72)]" />
                </div>
                <label
                  className={`inline-flex h-10 items-center rounded-full px-4 text-sm ${
                    skillBusy
                      ? "cursor-default bg-[rgba(146,190,200,0.12)] text-[rgba(96,134,145,0.48)]"
                      : "cursor-pointer bg-[rgba(118,177,189,0.1)] text-[rgb(36,95,110)]"
                  }`}
                >
                  <FileUp className="mr-2 h-4 w-4" />
                  上传技能
                  <input type="file" className="hidden" disabled={skillBusy} accept=".zip,.skill,.md" onChange={(event) => void onSkillUpload(event)} />
                </label>
                <Button
                  type="button"
                  disabled={skillBusy}
                  onClick={onOpenCreateSkill}
                  className="h-10 rounded-full bg-[rgb(74,156,175)] px-4 text-primary-foreground hover:bg-[rgb(61,132,150)]"
                >
                  <Plus className="mr-2 h-4 w-4" />
                  新建技能
                </Button>
              </div>

              <div className="mt-4 rounded-[1.1rem] border border-[rgba(176,194,221,0.34)] bg-[rgba(248,251,255,0.9)] px-4 py-4">
                <div className="flex flex-wrap items-center gap-2">
                  <span className="tech-label text-[0.64rem] text-[rgba(97,123,150,0.66)]">技能元数据</span>
                  {activeSkill?.version ? <span className="chip-accent px-3 py-1 text-[11px] normal-case tracking-[0.04em]">v{activeSkill.version}</span> : null}
                  {activeSkill?.installSource ? <span className="chip-muted px-3 py-1 text-[11px] normal-case tracking-[0.04em]">{activeSkill.installSource}</span> : null}
                  {(activeSkill?.composedOf?.length ?? 0) > 0 ? (
                    <span className="chip-warning px-3 py-1 text-[11px] normal-case tracking-[0.04em]">组合 {activeSkill?.composedOf?.length}</span>
                  ) : null}
                </div>
                <p className="mt-3 text-sm leading-7 text-[rgba(115,137,161,0.82)]">
                  {normalizedSkill ? activeSkill?.description ?? normalizedSkill : "未指定时会使用默认 interview skill。"}
                </p>
              </div>

              <div className="mt-4 rounded-[1.1rem] border border-[rgba(176,194,221,0.34)] bg-[rgba(248,251,255,0.9)] px-4 py-4">
                <div className="flex items-center justify-between gap-3">
                  <div>
                    <p className="tech-label text-[0.64rem] text-[rgba(97,123,150,0.66)]">组合 focus</p>
                    <p className="mt-2 text-sm leading-7 text-[rgba(115,137,161,0.82)]">
                      同一场面试可以叠加多个 focus，后端会把它们一起写进 skill 约束和 rubric。
                    </p>
                  </div>
                  {selectedSkillFocuses.length > 0 ? (
                    <button
                      type="button"
                      className="text-xs text-[rgb(37,99,235)]"
                      onClick={() =>
                        onConfigChange((current) => ({
                          ...current,
                          skillFocuses: [],
                        }))
                      }
                    >
                      清空
                    </button>
                  ) : null}
                </div>
                <div className="mt-3 flex flex-wrap gap-2">
                  {(activeSkill?.focusAreas ?? []).length > 0 ? (
                    activeSkill?.focusAreas?.map((focus) => {
                      const active = selectedSkillFocuses.includes(focus);
                      return (
                        <button
                          key={focus}
                          type="button"
                          onClick={() =>
                            onConfigChange((current) => {
                              const currentFocuses = current.skillFocuses ?? [];
                              const nextFocuses = currentFocuses.includes(focus)
                                ? currentFocuses.filter((item) => item !== focus)
                                : [...currentFocuses, focus];
                              return {
                                ...current,
                                skillFocuses: nextFocuses,
                              };
                            })
                          }
                          className={`rounded-full border px-3 py-1.5 text-xs transition ${
                            active
                              ? "border-[rgba(191,219,254,0.96)] bg-[rgba(239,246,255,0.96)] text-[rgb(37,99,235)]"
                              : "border-[rgba(214,222,234,0.96)] bg-[rgba(255,255,255,0.92)] text-[rgba(71,85,105,0.82)]"
                          }`}
                        >
                          {focus}
                        </button>
                      );
                    })
                  ) : (
                    <p className="text-sm leading-7 text-[rgba(115,137,161,0.74)]">当前技能还没有声明可选 focus，可在技能编辑器里补充。</p>
                  )}
                </div>
              </div>
            </div>
          ) : null}

          {activePanel === "materials" ? (
            <div className="surface-secondary h-full rounded-[1.2rem] px-4 py-4">
              <div className="flex flex-wrap items-center gap-2">
                <label
                  className={`inline-flex h-10 items-center rounded-full px-4 text-sm ${
                    selectedConversationId && !isSubmitting
                      ? "cursor-pointer bg-[rgb(74,156,175)] text-primary-foreground"
                      : "cursor-default bg-[rgba(146,190,200,0.12)] text-[rgba(96,134,145,0.48)]"
                  }`}
                >
                  <Upload className="mr-2 h-4 w-4" />
                  上传材料
                  <input type="file" className="hidden" disabled={!selectedConversationId || isSubmitting} onChange={(event) => void onArtifactUpload(event)} />
                </label>
                <Button
                  type="button"
                  disabled={!selectedConversationId || isSubmitting}
                  onClick={onOpenCreateArtifact}
                  className="h-10 rounded-full bg-[rgba(118,177,189,0.1)] px-4 text-[rgb(36,95,110)] hover:bg-[rgba(118,177,189,0.16)]"
                >
                  <FilePenLine className="mr-2 h-4 w-4" />
                  新建材料
                </Button>
              </div>

              <div className="mt-4 max-h-64 space-y-2 overflow-y-auto rounded-[1rem] border border-[rgba(176,194,221,0.34)] bg-[rgba(248,251,255,0.9)] px-4 py-3">
                {artifacts.length === 0 ? (
                  <p className="text-sm text-[rgba(115,137,161,0.78)]">当前工作区还没有可绑定材料。</p>
                ) : (
                  artifacts.map((artifact) => {
                    const checked = selectedArtifactIDs.includes(artifact.id);
                    return (
                      <label key={artifact.id} className="flex cursor-pointer items-center gap-3 text-sm text-[rgb(72,91,114)]">
                        <input
                          type="checkbox"
                          checked={checked}
                          onChange={(event) =>
                            onSelectedArtifactIDsChange((current) =>
                              event.target.checked ? [...current, artifact.id] : current.filter((id) => id !== artifact.id)
                            )
                          }
                        />
                        <span className="truncate">{artifact.name}</span>
                      </label>
                    );
                  })
                )}
              </div>
            </div>
          ) : null}
        </div>

        <button
          type="button"
          onClick={onClose}
          className="absolute right-6 top-6 inline-flex h-9 w-9 items-center justify-center rounded-full border border-[rgba(223,228,238,0.88)] bg-[rgba(255,255,255,0.96)] text-[rgba(71,85,105,0.78)] transition hover:text-[rgb(37,99,235)]"
          aria-label="关闭设置"
        >
          <X className="h-4 w-4" />
        </button>
      </div>
    </div>
  );
}

function SessionSummaryPill({
  label,
  value,
  tone = "default",
}: {
  label: string;
  value: string;
  tone?: "default" | "accent";
}) {
  return (
    <span className={`inline-flex items-center gap-1 rounded-full border px-2 py-[3px] text-[10px] ${tone === "accent" ? "chip-info" : "chip-neutral"}`}>
      <span className="tech-label text-[0.58rem] text-[rgba(97,123,150,0.7)]">{label}</span>
      <span className="max-w-[16ch] truncate font-medium">{value}</span>
    </span>
  );
}
