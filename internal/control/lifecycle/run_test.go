package lifecycle

import (
	"testing"
	"time"

	"mockinterview/internal/protocol"
)

func TestBuildRunOutputOutcomeMarksWrapupWhenEvaluationExplicitlyRequested(t *testing.T) {
	t.Parallel()

	now := time.Now()
	outcome := BuildRunOutputOutcome(
		RunOutputInput{
			Task: protocol.Task{
				Config: protocol.InterviewConfig{
					OutputStyle: protocol.OutputInterviewPlusScore,
					TimeBudget:  "15 分钟",
				}.WithDefaults(),
			},
			Run: protocol.Run{
				ID:     "run_1",
				Input:  "请结束这场面试并给我评分总结",
				Status: protocol.RunRunning,
				InterviewState: &protocol.RunInterviewState{
					Phase: protocol.PhaseStress,
					Round: 3,
					LastDecision: &protocol.NextStepDecision{
						Reason:           protocol.ReasonPressureTest,
						Explanation:      "继续加压验证稳定性。",
						RecommendedFocus: []string{"observability", "tradeoff"},
					},
				},
			},
			Messages: []protocol.Message{
				{Role: "assistant", Content: "请详细解释你的超时控制策略。"},
				{Role: "user", Content: "我会用 context 控制超时和取消。"},
			},
			Output: "这是最后一轮的收尾总结。",
			Rubric: protocol.Rubric{
				Title:   "Go Agent Rubric",
				Anchors: []string{"并发控制", "可观测性"},
			},
		},
		now,
	)

	if !outcome.EvaluationPending {
		t.Fatalf("expected evaluation to be pending")
	}
	if outcome.Run.Phase != protocol.RunPhaseEvaluating {
		t.Fatalf("expected run phase evaluating, got %s", outcome.Run.Phase)
	}
	if outcome.Run.InterviewState == nil || outcome.Run.InterviewState.Phase != protocol.PhaseWrapup {
		t.Fatalf("expected interview state phase wrapup, got %#v", outcome.Run.InterviewState)
	}
	if outcome.Run.InterviewState.LastDecision == nil {
		t.Fatalf("expected wrapup decision to be synthesized")
	}
	if outcome.Run.InterviewState.LastDecision.Reason != protocol.ReasonWrapupRequested {
		t.Fatalf("expected wrapup reason, got %#v", outcome.Run.InterviewState.LastDecision)
	}
	if len(outcome.Run.InterviewState.LastDecision.RecommendedFocus) != 2 {
		t.Fatalf("expected recommended focus to be preserved, got %#v", outcome.Run.InterviewState.LastDecision)
	}
}

func TestBuildRunOutputOutcomeKeepsInterviewOpenForRegularTurn(t *testing.T) {
	t.Parallel()

	now := time.Now()
	outcome := BuildRunOutputOutcome(
		RunOutputInput{
			Task: protocol.Task{
				Config: protocol.InterviewConfig{
					OutputStyle: protocol.OutputInterviewPlusStudy,
					TimeBudget:  "45 分钟",
				}.WithDefaults(),
			},
			Run: protocol.Run{
				ID:     "run_regular",
				Input:  "我会继续解释 cancel 和 checkpoint 的原子性。",
				Status: protocol.RunResuming,
				Phase:  protocol.RunPhaseInterviewing,
				InterviewState: &protocol.RunInterviewState{
					Phase: protocol.PhaseProbe,
					Round: 2,
				},
			},
			Messages: []protocol.Message{
				{Role: "assistant", Content: "请解释 cancel 和 checkpoint 的原子性设计。"},
				{Role: "user", Content: "我会用 CAS 和 append-only event 保证一致性。"},
			},
			Output: "那你再具体说说，数据库写失败时你如何 repair 和补偿？",
			Rubric: protocol.Rubric{
				Title:   "Go Agent Rubric",
				Anchors: []string{"并发控制", "可观测性"},
			},
		},
		now,
	)

	if outcome.EvaluationPending {
		t.Fatalf("did not expect evaluation to be pending")
	}
	if outcome.Run.Phase != protocol.RunPhaseInterviewing {
		t.Fatalf("expected run phase interviewing, got %s", outcome.Run.Phase)
	}
	if outcome.Run.Status != protocol.RunWaitingClarify {
		t.Fatalf("expected turn to settle into waiting clarify status, got %s", outcome.Run.Status)
	}
	if outcome.Run.CompletedAt != nil {
		t.Fatalf("expected non-terminal interview turn to keep completedAt empty")
	}
	if outcome.ReviewSnapshot != nil {
		t.Fatalf("did not expect final review snapshot for a normal interview turn")
	}
}
