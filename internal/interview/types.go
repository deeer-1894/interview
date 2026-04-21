package interview

type Mode string

const (
	ModeStandard        Mode = "standard"
	ModeStress          Mode = "stress"
	ModeWeaknessFocused Mode = "weakness_focused"
	ModeSystemDesign    Mode = "system_design"
	ModeResumeDeepDive  Mode = "resume_deep_dive"
)

type Persona string

const (
	PersonaRigorous   Persona = "rigorous"
	PersonaCalm       Persona = "calm"
	PersonaSupportive Persona = "supportive"
	PersonaManager    Persona = "manager"
)

type OutputStyle string

const (
	OutputInterviewOnly      OutputStyle = "interview_only"
	OutputInterviewPlusScore OutputStyle = "interview_plus_score"
	OutputInterviewPlusStudy OutputStyle = "interview_plus_score_and_study_plan"
)

type InterviewConfig struct {
	Skill        string
	SkillFocuses []string
	Persona      Persona
	Level        string
	Focus        string
	Mode         Mode
	TimeBudget   string
	OutputStyle  OutputStyle
}

func (c InterviewConfig) WithDefaults() InterviewConfig {
	c.SkillFocuses = normalizeBulletList(c.SkillFocuses)
	if c.Level == "" {
		c.Level = "mid"
	}
	if c.Persona == "" {
		c.Persona = PersonaRigorous
	}
	if c.Focus == "" {
		c.Focus = "generalist"
	}
	if c.Mode == "" {
		c.Mode = ModeStandard
	}
	if c.TimeBudget == "" {
		c.TimeBudget = "25 minutes"
	}
	if c.OutputStyle == "" {
		c.OutputStyle = OutputInterviewPlusScore
	}

	return c
}
