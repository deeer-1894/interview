package interview

import (
	"strings"

	"mockinterview/internal/protocol"
)

type InterviewMode = protocol.InterviewMode

const (
	InterviewModeStandard        = protocol.ModeStandard
	InterviewModeStress          = protocol.ModeStress
	InterviewModeWeaknessFocused = protocol.ModeWeaknessFocused
	InterviewModeSystemDesign    = protocol.ModeSystemDesign
	InterviewModeResumeDeepDive  = protocol.ModeResumeDeepDive
)

func NormalizeInterviewMode(mode string) protocol.InterviewMode {
	switch strings.TrimSpace(strings.ToLower(mode)) {
	case string(protocol.ModeStress):
		return protocol.ModeStress
	case string(protocol.ModeWeaknessFocused):
		return protocol.ModeWeaknessFocused
	case string(protocol.ModeSystemDesign):
		return protocol.ModeSystemDesign
	case string(protocol.ModeResumeDeepDive):
		return protocol.ModeResumeDeepDive
	default:
		return protocol.ModeStandard
	}
}

func ModeInstruction(mode protocol.InterviewMode) string {
	switch NormalizeInterviewMode(string(mode)) {
	case protocol.ModeStress:
		return "Run this as a pressure interview. Push the candidate toward shorter answers, faster tradeoff calls, and more aggressive follow-up pressure."
	case protocol.ModeWeaknessFocused:
		return "Run this as a weakness-focused interview. Stay on historical weak areas longer and do not switch topics too early."
	case protocol.ModeSystemDesign:
		return "Run this as a system-design-focused interview. Prefer architecture, scaling, reliability, observability, and operational tradeoffs over syntax trivia."
	case protocol.ModeResumeDeepDive:
		return "Run this as a resume deep-dive. Use the candidate's stated experience and provided materials as the primary source of follow-up."
	default:
		return "Run this as a balanced standard interview with realistic follow-ups and calibrated pressure."
	}
}

