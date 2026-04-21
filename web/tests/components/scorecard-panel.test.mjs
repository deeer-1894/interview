import test from "node:test";
import assert from "node:assert/strict";
import React from "react";
import { renderToStaticMarkup } from "react-dom/server";

import { ScorecardPanel } from "../../.test-dist/src/components/review/scorecard-panel.js";

test("ScorecardPanel renders score summary, focus, dimensions and study plan", () => {
  const html = renderToStaticMarkup(
    React.createElement(ScorecardPanel, {
      scorecard: {
        title: "Go Agent 面试评分",
        summary: "系统设计清晰，但 tradeoff 解释还不够具体。",
        anchors: ["tradeoff", "reliability"],
        dimensionScores: [
          { name: "System Design", score: 4, rationale: "拆解结构完整" },
          { name: "Tradeoffs", score: 2, rationale: "缺少替代方案比较" },
        ],
        strengths: ["架构拆解清晰"],
        gaps: ["tradeoff 对比不够具体"],
        improvements: ["先补充两个候选方案的取舍"],
        studyPlan: ["练习用两句话讲清方案差异"],
      },
      outputStyle: "interview_plus_score_and_study_plan",
      retryDisabled: false,
      latestFailure: null,
    }),
  );

  assert.match(html, /Go Agent 面试评分/);
  assert.match(html, /系统设计清晰，但 tradeoff 解释还不够具体/);
  assert.match(html, /当前最该先修/);
  assert.match(html, /先补充两个候选方案的取舍/);
  assert.match(html, /System Design/);
  assert.match(html, /4\/5/);
  assert.match(html, /学习计划/);
  assert.match(html, /练习用两句话讲清方案差异/);
});

test("ScorecardPanel shows fallback copy and retry affordance when scorecard is absent", () => {
  const html = renderToStaticMarkup(
    React.createElement(ScorecardPanel, {
      scorecard: null,
      outputStyle: "interview_plus_score",
      retryDisabled: true,
      latestFailure: {
        id: "event_1",
        conversationId: "conv_1",
        taskId: "task_1",
        runId: "run_1",
        type: "run.failed",
        timestamp: new Date().toISOString(),
        payload: { error: "network" },
      },
      onRetry: () => {},
    }),
  );

  assert.match(html, /等待评分结果/);
  assert.match(html, /先看这里判断本场整体表现/);
  assert.match(html, /运行到达评分阶段后，结构化评分会显示在这里/);
  assert.match(html, /恢复运行/);
  assert.match(html, /启动恢复运行/);
  assert.match(html, /disabled/);
});
