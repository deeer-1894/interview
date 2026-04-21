package interview

import (
	"strings"

	"mockinterview/internal/protocol"
)

type StrategyName string

const (
	StrategyDefault         StrategyName = "default"
	StrategyWeaknessFocused StrategyName = "weakness-focused"
)

type SignalThresholds struct {
	MissingTradeoffReason            float64
	MissingTradeoffEscalate          float64
	MissingTradeoffAdversarial       float64
	MissingImplementationReason      float64
	TooGenericReason                 float64
	TooGenericEscalate               float64
	TooGenericAdversarial            float64
	TimeoutReason                    float64
	ObservabilityReason              float64
	IncreaseDifficultyTradeoff       float64
	IncreaseDifficultyImplementation float64
}

type DecisionPolicy struct {
	EscalatePressureFromRound int
	AdversarialFromPhase      protocol.InterviewPhase
	PreferWeaknessFocus       bool
	PreferTopicSwitch         bool
	TopicSwitchFromRound      int
	RecommendedFocus          []string
	Thresholds                SignalThresholds
}

type StrategyConfig struct {
	Name        StrategyName
	Description string
	Notes       []string
	Policy      DecisionPolicy
}

func ResolveStrategyConfig(cfg protocol.InterviewConfig, skill protocol.SkillSpec) StrategyConfig {
	cfg = cfg.WithDefaults()
	mode := NormalizeInterviewMode(string(cfg.Mode))
	strategy := baseStrategyConfig(mode)
	strategy.Policy = applyModePolicy(strategy.Policy, mode)
	strategy = applyPersonaStrategy(strategy, cfg.Persona)
	strategy = applySkillStrategy(strategy, skill)
	return strategy.withDefaults()
}

func baseStrategyConfig(mode protocol.InterviewMode) StrategyConfig {
	if mode == protocol.ModeWeaknessFocused {
		return StrategyConfig{
			Name:        StrategyWeaknessFocused,
			Description: "优先围绕历史弱项和当前弱信号继续深挖，减少无意义换题。",
			Notes:       []string{"优先围绕历史弱项继续追问"},
			Policy: DecisionPolicy{
				PreferWeaknessFocus: true,
				Thresholds: SignalThresholds{
					MissingImplementationReason: 0.56,
					TooGenericReason:            0.6,
				},
			},
		}
	}

	return StrategyConfig{
		Name:        StrategyDefault,
		Description: "在工程细节、tradeoff 和运行稳定性之间保持均衡追问。",
		Policy:      DecisionPolicy{},
	}
}

func applyModePolicy(policy DecisionPolicy, mode protocol.InterviewMode) DecisionPolicy {
	switch mode {
	case protocol.ModeStress:
		policy.EscalatePressureFromRound = 1
		policy.AdversarialFromPhase = protocol.PhaseWarmup
		policy.Thresholds.TooGenericEscalate = 0.66
		policy.Thresholds.TooGenericAdversarial = 0.74
		policy.Thresholds.MissingTradeoffEscalate = 0.66
		policy.Thresholds.MissingTradeoffAdversarial = 0.78
	case protocol.ModeWeaknessFocused:
		policy.PreferWeaknessFocus = true
		policy.Thresholds.MissingImplementationReason = 0.56
	case protocol.ModeSystemDesign:
		policy.PreferTopicSwitch = true
		policy.TopicSwitchFromRound = 1
		policy.RecommendedFocus = mergeDecisionFocus(policy.RecommendedFocus, []string{"system design", "reliability"})
	case protocol.ModeResumeDeepDive:
		policy.PreferWeaknessFocus = true
		policy.RecommendedFocus = mergeDecisionFocus(policy.RecommendedFocus, []string{"ownership", "resume evidence"})
	}
	return policy
}

func applyPersonaStrategy(strategy StrategyConfig, persona protocol.InterviewPersona) StrategyConfig {
	switch persona {
	case protocol.PersonaSupportive:
		strategy.Notes = append(strategy.Notes, "supportive persona 会更晚进入对抗式追问")
		strategy.Policy.AdversarialFromPhase = protocol.PhaseAdversarial
		strategy.Policy.EscalatePressureFromRound = maxInt(strategy.Policy.EscalatePressureFromRound, 3)
	case protocol.PersonaCalm:
		strategy.Notes = append(strategy.Notes, "calm persona 会优先保留澄清空间，再逐步加压")
		strategy.Policy.AdversarialFromPhase = protocol.PhaseAdversarial
	case protocol.PersonaManager:
		strategy.Notes = append(strategy.Notes, "manager persona 会额外关注 ownership 和业务判断")
		strategy.Policy.RecommendedFocus = mergeDecisionFocus(strategy.Policy.RecommendedFocus, []string{"ownership", "business impact"})
		strategy.Policy.PreferTopicSwitch = true
	default:
		strategy.Notes = append(strategy.Notes, "rigorous persona 保持默认追问强度")
	}
	return strategy
}

func applySkillStrategy(strategy StrategyConfig, skill protocol.SkillSpec) StrategyConfig {
	focusAreas := normalizeBulletList(skill.FocusAreas)

	if hasSkillFocus(focusAreas, "observability", "monitor", "metric", "trace", "log", "logging") {
		strategy.Notes = append(strategy.Notes, "observability 相关技能会降低可观测性弱项阈值")
		strategy.Policy.Thresholds.ObservabilityReason = 0.5
		strategy.Policy.RecommendedFocus = mergeDecisionFocus(strategy.Policy.RecommendedFocus, []string{"observability"})
	}
	if hasSkillFocus(focusAreas, "timeout", "latency", "reliability", "failure", "retry", "resilience") {
		strategy.Notes = append(strategy.Notes, "reliability 相关技能会更早追问 timeout 和失败恢复")
		strategy.Policy.Thresholds.TimeoutReason = 0.5
		strategy.Policy.RecommendedFocus = mergeDecisionFocus(strategy.Policy.RecommendedFocus, []string{"reliability", "timeout control"})
	}
	if hasSkillFocus(focusAreas, "system design", "architecture", "scaling") {
		strategy.Notes = append(strategy.Notes, "system design 相关技能会更早切题确认广度")
		strategy.Policy.PreferTopicSwitch = true
		strategy.Policy.TopicSwitchFromRound = 1
		strategy.Policy.RecommendedFocus = mergeDecisionFocus(strategy.Policy.RecommendedFocus, []string{"system design", "architecture"})
	}

	return strategy
}

func (c StrategyConfig) withDefaults() StrategyConfig {
	if c.Name == "" {
		c.Name = StrategyDefault
	}
	if strings.TrimSpace(c.Description) == "" {
		c.Description = "使用默认面试追问策略。"
	}
	c.Policy = c.Policy.withDefaults()
	c.Notes = normalizeBulletList(c.Notes)
	return c
}

func (p DecisionPolicy) withDefaults() DecisionPolicy {
	if p.EscalatePressureFromRound <= 0 {
		p.EscalatePressureFromRound = 2
	}
	if strings.TrimSpace(string(p.AdversarialFromPhase)) == "" {
		p.AdversarialFromPhase = protocol.PhaseProbe
	}
	if p.TopicSwitchFromRound <= 0 {
		p.TopicSwitchFromRound = 2
	}
	p.Thresholds = p.Thresholds.withDefaults()
	p.RecommendedFocus = normalizeBulletList(p.RecommendedFocus)
	return p
}

func (s SignalThresholds) withDefaults() SignalThresholds {
	if s.MissingTradeoffReason <= 0 {
		s.MissingTradeoffReason = 0.62
	}
	if s.MissingTradeoffEscalate <= 0 {
		s.MissingTradeoffEscalate = 0.72
	}
	if s.MissingTradeoffAdversarial <= 0 {
		s.MissingTradeoffAdversarial = 0.86
	}
	if s.MissingImplementationReason <= 0 {
		s.MissingImplementationReason = 0.62
	}
	if s.TooGenericReason <= 0 {
		s.TooGenericReason = 0.64
	}
	if s.TooGenericEscalate <= 0 {
		s.TooGenericEscalate = 0.72
	}
	if s.TooGenericAdversarial <= 0 {
		s.TooGenericAdversarial = 0.84
	}
	if s.TimeoutReason <= 0 {
		s.TimeoutReason = 0.58
	}
	if s.ObservabilityReason <= 0 {
		s.ObservabilityReason = 0.58
	}
	if s.IncreaseDifficultyTradeoff <= 0 {
		s.IncreaseDifficultyTradeoff = 0.68
	}
	if s.IncreaseDifficultyImplementation <= 0 {
		s.IncreaseDifficultyImplementation = 0.7
	}
	return s
}
