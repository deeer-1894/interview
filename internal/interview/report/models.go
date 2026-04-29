package report

import "time"

type Scorecard struct {
	ID        string          `json:"id" bson:"_id,omitempty"`
	SessionID string          `json:"sessionId" bson:"sessionId"`
	UserID    string          `json:"userId" bson:"userId"`
	Summary   string          `json:"summary" bson:"summary"`
	Overall   float64         `json:"overall" bson:"overall"`
	Items     []ScorecardItem `json:"items,omitempty" bson:"items,omitempty"`
	Advice    []string        `json:"advice,omitempty" bson:"advice,omitempty"`
	CreatedAt time.Time       `json:"createdAt" bson:"createdAt"`
}

type ScorecardItem struct {
	Dimension string   `json:"dimension" bson:"dimension"`
	Score     float64  `json:"score" bson:"score"`
	Evidence  []string `json:"evidence,omitempty" bson:"evidence,omitempty"`
	Advice    string   `json:"advice,omitempty" bson:"advice,omitempty"`
}
