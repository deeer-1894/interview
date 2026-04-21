import type { CandidateProfile, ProfileDimension, ReviewSummary } from "@/lib/types";

type CandidateProfilePanelProps = {
  profile: CandidateProfile | null;
  reviewSummary?: ReviewSummary | null;
  getPersonaLabel: (persona?: string | null) => string;
};

export function CandidateProfilePanel({
  profile,
  reviewSummary,
  getPersonaLabel,
}: CandidateProfilePanelProps) {
  const dimensions = profile?.dimensions ?? [];
  const radar = profile?.radar ?? [];
  const growthCurves = profile?.growthCurves ?? [];
  const personaUsage = profile?.personaUsage ?? [];
  const stableStrengths = profile?.stableStrengths ?? [];
  const recurringGaps = profile?.recurringGaps ?? [];
  const recommendedFocus = profile?.recommendedFocus ?? [];
  const recentChanges = profile?.recentChanges ?? [];

  return (
    <section className="panel-card">
      <div className="flex items-start justify-between gap-4">
        <div>
          <p className="tech-label text-[0.68rem] text-[rgba(97,123,150,0.62)]">跨会话画像</p>
          <h2 className="mt-2 font-display text-2xl text-[rgb(72,91,114)]">
            {profile?.interviewCount ? `累计 ${profile.interviewCount} 场面试` : "等待画像沉淀"}
          </h2>
        </div>
        {profile?.lastPersona ? (
          <div className="chip-info px-3 py-1.5 text-[0.68rem] tracking-[0.2em]">
            最近人格 · {getPersonaLabel(profile.lastPersona)}
          </div>
        ) : null}
      </div>
      <p className="mt-3 max-w-[64ch] text-sm leading-7 text-[rgba(115,137,161,0.78)]">
        这些跨会话结论会逐渐影响下一场面试的追问重点，让训练越来越针对你的真实薄弱项。
      </p>

      {!profile || profile.interviewCount === 0 ? (
        <p className="mt-5 rounded-[1.3rem] border border-dashed border-[rgba(153,191,201,0.18)] px-4 py-5 text-sm leading-7 text-[rgba(115,137,161,0.78)]">
          评分完成后，这里会跨会话累计优势、短板、人格使用偏好和建议训练重点。
        </p>
      ) : (
        <div className="mt-5 space-y-5">
          <div className="grid gap-3 md:grid-cols-2">
            <ProfileFactCard label="最近技能" value={profile.lastSkill || "默认 interview"} />
            <ProfileFactCard label="最近更新" value={profile.updatedAt ? new Date(profile.updatedAt).toLocaleString() : "刚刚"} />
          </div>

          <div>
            <p className="tech-label text-[0.68rem] text-[rgba(97,123,150,0.62)]">能力维度</p>
            {dimensions.length === 0 ? (
              <p className="mt-3 text-sm leading-7 text-[rgba(115,137,161,0.78)]">当前还没有足够证据形成维度画像。</p>
            ) : (
              <div className="mt-3 space-y-3">
                {dimensions.map((dimension) => (
                  <ProfileDimensionRow key={dimension.key} dimension={dimension} />
                ))}
              </div>
            )}
          </div>

          {radar.length > 0 ? (
            <div>
              <p className="tech-label text-[0.68rem] text-[rgba(97,123,150,0.62)]">雷达图底稿</p>
              <div className="mt-3 grid gap-3 sm:grid-cols-2 xl:grid-cols-3">
                {radar.map((point) => (
                  <div key={point.key} className="panel-card-soft">
                    <div className="flex items-center justify-between gap-3">
                      <p className="text-sm font-medium text-[rgb(31,41,55)]">{point.label}</p>
                      <span className="chip-info px-2.5 py-1 text-[0.68rem] tracking-[0.18em]">
                        {point.normalizedScore}/100
                      </span>
                    </div>
                    <div className="mt-3 h-2 overflow-hidden rounded-full bg-[rgba(226,232,240,0.96)]">
                      <div
                        className="h-full rounded-full bg-[linear-gradient(90deg,rgba(14,165,233,0.86),rgba(125,211,252,0.9))]"
                        style={{ width: `${point.normalizedScore}%` }}
                      />
                    </div>
                  </div>
                ))}
              </div>
            </div>
          ) : null}

          {growthCurves.length > 0 ? (
            <div>
              <p className="tech-label text-[0.68rem] text-[rgba(97,123,150,0.62)]">跨场次趋势</p>
              <div className="mt-3 space-y-3">
                {growthCurves.map((curve) => (
                  <GrowthCurveRow key={curve.key} label={curve.label} points={curve.points ?? []} />
                ))}
              </div>
            </div>
          ) : null}

          {personaUsage.length > 0 ? (
            <div>
              <p className="tech-label text-[0.68rem] text-[rgba(97,123,150,0.62)]">人格使用偏好</p>
              <div className="mt-3 flex flex-wrap gap-2">
                {personaUsage.map((entry) => (
                  <span
                    key={entry.persona}
                    className="chip-info px-3 py-1.5 text-sm normal-case tracking-[0.04em]"
                  >
                    {getPersonaLabel(entry.persona)} · {entry.count}
                  </span>
                ))}
              </div>
            </div>
          ) : null}

          <ProfileBulletSection title="稳定优势" items={stableStrengths} tone="neutral" />
          <ProfileBulletSection title="重复短板" items={recurringGaps} tone="warning" />
          <ProfileBulletSection title="本场再次命中的历史弱项" items={reviewSummary?.historicalWeaknessesHit ?? []} tone="warning" />
          <ProfileBulletSection title="本场新增弱项" items={reviewSummary?.newWeaknesses ?? []} tone="warning" />
          <ProfileBulletSection title="本场修正项" items={reviewSummary?.resolvedWeaknesses ?? []} tone="success" />
          <ProfileBulletSection title="推荐训练重点" items={recommendedFocus} tone="accent" />
          <ProfileBulletSection title="最近变化" items={recentChanges} tone="neutral" />
        </div>
      )}
    </section>
  );
}

function ProfileFactCard({ label, value }: { label: string; value: string }) {
  return (
    <div className="panel-card-soft">
      <p className="tech-label text-[0.64rem] text-[rgba(97,123,150,0.62)]">{label}</p>
      <p className="mt-2 text-sm leading-6 text-[rgb(31,41,55)]">{value}</p>
    </div>
  );
}

function ProfileDimensionRow({ dimension }: { dimension: ProfileDimension }) {
  const score = Math.max(-6, Math.min(6, dimension.score));
  const normalized = typeof dimension.normalizedScore === "number" ? dimension.normalizedScore : Math.round(((score + 6) / 12) * 100);
  const toneClass =
    normalized >= 67
      ? "bg-[linear-gradient(90deg,rgba(59,130,246,0.9),rgba(147,197,253,0.92))]"
      : normalized <= 40
        ? "bg-[linear-gradient(90deg,rgba(96,165,250,0.56),rgba(191,219,254,0.84))]"
        : "bg-[linear-gradient(90deg,rgba(191,219,254,0.7),rgba(219,234,254,0.92))]";
  const trendPoints = dimension.trend ?? [];
  const deltaLabel =
    typeof dimension.recentDelta === "number" && dimension.recentDelta !== 0
      ? dimension.recentDelta > 0
        ? `本场 +${dimension.recentDelta}`
        : `本场 ${dimension.recentDelta}`
      : "本场持平";

  return (
    <div className="panel-card-soft">
      <div className="flex items-start justify-between gap-3">
        <div>
          <p className="text-sm font-medium text-[rgb(31,41,55)]">{dimension.label}</p>
          {dimension.summary ? <p className="mt-1 text-xs leading-6 text-[rgba(100,116,139,0.82)]">{dimension.summary}</p> : null}
        </div>
        <div className="flex flex-wrap items-center justify-end gap-2">
          <div className="chip-info px-2.5 py-1 text-[0.68rem] tracking-[0.18em]">{normalized}/100</div>
          <div className="chip-neutral px-2.5 py-1 text-[0.68rem] tracking-[0.2em]">证据 {dimension.evidenceCount}</div>
        </div>
      </div>
      <div className="mt-3 h-2.5 overflow-hidden rounded-full bg-[rgba(226,232,240,0.96)]">
        <div className={`h-full rounded-full ${toneClass}`} style={{ width: `${normalized}%` }} />
      </div>
      <div className="mt-3 flex flex-wrap items-center justify-between gap-3 text-xs leading-6 text-[rgba(100,116,139,0.82)]">
        <span>{deltaLabel}</span>
        <span>{dimension.lastUpdatedAt ? `最近命中 ${new Date(dimension.lastUpdatedAt).toLocaleString()}` : "等待更多样本"}</span>
      </div>
      {trendPoints.length > 1 ? <TrendStrip points={trendPoints} /> : null}
    </div>
  );
}

function GrowthCurveRow({
  label,
  points,
}: {
  label: string;
  points: NonNullable<CandidateProfile["growthCurves"]>[number]["points"];
}) {
  if (!points || points.length === 0) {
    return null;
  }

  const latest = points[points.length - 1];

  return (
    <div className="panel-card-soft">
      <div className="flex items-center justify-between gap-3">
        <p className="text-sm font-medium text-[rgb(31,41,55)]">{label}</p>
        <span className="chip-info px-2.5 py-1 text-[0.68rem] tracking-[0.18em]">{latest.normalizedScore}/100</span>
      </div>
      <TrendStrip points={points} compact />
    </div>
  );
}

function TrendStrip({
  points,
  compact = false,
}: {
  points: NonNullable<ProfileDimension["trend"]>;
  compact?: boolean;
}) {
  if (points.length === 0) {
    return null;
  }

  return (
    <div className={compact ? "mt-3" : "mt-3 rounded-[1rem] bg-[rgba(248,250,252,0.82)] px-3 py-3"}>
      <div className="flex items-end gap-1.5">
        {points.map((point, index) => (
          <div
            key={`${point.timestamp}-${index}`}
            className="flex-1 rounded-full bg-[linear-gradient(180deg,rgba(14,165,233,0.84),rgba(191,219,254,0.92))]"
            style={{ height: `${Math.max(10, Math.round(point.normalizedScore * 0.44))}px` }}
            title={`${new Date(point.timestamp).toLocaleString()} · ${point.normalizedScore}/100`}
          />
        ))}
      </div>
    </div>
  );
}

function ProfileBulletSection({
  title,
  items,
  tone,
}: {
  title: string;
  items: string[];
  tone: "neutral" | "warning" | "accent" | "success";
}) {
  if (items.length === 0) {
    return null;
  }

  const toneClass =
    tone === "warning"
      ? "border-[rgba(253,230,138,0.92)] bg-[rgba(254,252,232,0.96)] text-[rgb(161,98,7)]"
      : tone === "success"
        ? "border-[rgba(187,247,208,0.96)] bg-[rgba(240,253,244,0.96)] text-[rgb(21,128,61)]"
        : tone === "accent"
          ? "border-[rgba(219,234,254,0.96)] bg-[rgba(248,250,255,0.96)] text-[rgb(29,78,216)]"
          : "border-[rgba(153,191,201,0.16)] bg-[rgba(252,255,255,0.9)] text-[rgb(31,41,55)]";

  return (
    <div>
      <p className="tech-label text-[0.68rem] text-[rgba(97,123,150,0.62)]">{title}</p>
      <div className="mt-3 space-y-2.5">
        {items.map((item, index) => (
          <div key={`${title}-${index}-${item}`} className={`rounded-[1.15rem] border px-4 py-3 text-sm leading-7 ${toneClass}`}>
            {item}
          </div>
        ))}
      </div>
    </div>
  );
}
