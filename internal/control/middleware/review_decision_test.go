package middleware

import (
	"testing"

	"mockinterview/internal/protocol"
)

func TestBuildReviewDecisionAuditIncludesLatestHistorySignals(t *testing.T) {
	t.Parallel()

	state := &protocol.RunInterviewState{
		Phase:      protocol.PhaseProbe,
		Round:      3,
		Difficulty: 2,
		LastDecision: &protocol.NextStepDecision{
			Reason:           protocol.ReasonMissingTradeoff,
			Explanation:      "继续追问 tradeoff。",
			RecommendedFocus: []string{"tradeoff_expression"},
			KeepTopic:        true,
		},
		History: []protocol.InterviewRoundSnapshot{
			{
				Round:                  3,
				WeakSignals:            []string{"too_generic"},
				StrongSignals:          []string{"implementation_detail"},
				WeakSignalConfidence:   map[string]float64{"too_generic": 0.88},
				StrongSignalConfidence: map[string]float64{"implementation_detail": 0.91},
			},
		},
	}

	audit := BuildReviewDecisionAudit(state, protocol.ModeStandard, protocol.CandidateProfile{
		RecommendedFocus: []string{"observability"},
		RecurringGaps:    []string{"tradeoff_expression"},
	}, "interviewer.v2")
	if audit == nil {
		t.Fatalf("expected decision audit")
	}
	if audit.Decision.Reason != protocol.ReasonMissingTradeoff {
		t.Fatalf("unexpected decision reason: %q", audit.Decision.Reason)
	}
	if !audit.Analysis.TooGeneric {
		t.Fatalf("expected too_generic weak signal to set tooGeneric")
	}
	if !audit.Analysis.HasConcreteImplementation {
		t.Fatalf("expected implementation_detail strong signal to set hasConcreteImplementation")
	}
	if audit.Analysis.WeakSignalConfidence["too_generic"] < 0.8 {
		t.Fatalf("expected weak signal confidence to be preserved, got %#v", audit.Analysis.WeakSignalConfidence)
	}
	if len(audit.ProfileFocus) == 0 {
		t.Fatalf("expected profile focus to be present")
	}
	if audit.PromptVersion != "interviewer.v2" {
		t.Fatalf("expected prompt version to be preserved, got %q", audit.PromptVersion)
	}
}
