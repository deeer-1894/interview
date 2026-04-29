package resume

import "time"

type Profile struct {
	ID        string    `json:"id" bson:"_id,omitempty"`
	UserID    string    `json:"userId" bson:"userId"`
	RawText   string    `json:"rawText,omitempty" bson:"rawText,omitempty"`
	Summary   string    `json:"summary,omitempty" bson:"summary,omitempty"`
	Skills    []string  `json:"skills,omitempty" bson:"skills,omitempty"`
	Projects  []Project `json:"projects,omitempty" bson:"projects,omitempty"`
	CreatedAt time.Time `json:"createdAt" bson:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt" bson:"updatedAt"`
}

type Project struct {
	ID        string   `json:"id" bson:"id"`
	Name      string   `json:"name" bson:"name"`
	Domain    string   `json:"domain,omitempty" bson:"domain,omitempty"`
	TechStack []string `json:"techStack,omitempty" bson:"techStack,omitempty"`
	Claims    []string `json:"claims,omitempty" bson:"claims,omitempty"`
	Evidence  []string `json:"evidence,omitempty" bson:"evidence,omitempty"`
	Summary   string   `json:"summary,omitempty" bson:"summary,omitempty"`
}

type ProjectBrief struct {
	ID      string `json:"id" bson:"id"`
	Name    string `json:"name" bson:"name"`
	Domain  string `json:"domain,omitempty" bson:"domain,omitempty"`
	Summary string `json:"summary,omitempty" bson:"summary,omitempty"`
}

func (p Project) Brief() ProjectBrief {
	return ProjectBrief{
		ID:      p.ID,
		Name:    p.Name,
		Domain:  p.Domain,
		Summary: p.Summary,
	}
}
