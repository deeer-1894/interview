package runtime

import (
	"context"
	"errors"
	"fmt"

	"github.com/cloudwego/eino/adk"

	"mockinterview/internal/interview/resume"
	"mockinterview/internal/interview/session"
	"mockinterview/internal/interview/store"
)

const defaultHistoryLimit = 12

type SessionContextMiddleware struct {
	adk.BaseChatModelAgentMiddleware

	Stores       store.Bundle
	HistoryLimit int
}

func NewSessionContextMiddleware(stores store.Bundle) *SessionContextMiddleware {
	return &SessionContextMiddleware{Stores: stores, HistoryLimit: defaultHistoryLimit}
}

func (m *SessionContextMiddleware) BeforeAgent(ctx context.Context, runCtx *adk.ChatModelAgentContext) (context.Context, *adk.ChatModelAgentContext, error) {
	values := adk.GetSessionValues(ctx)
	userID, _ := values["user_id"].(string)
	sessionID, _ := values["session_id"].(string)
	if userID == "" || sessionID == "" {
		return ctx, nil, errors.New("session context requires user_id and session_id")
	}
	if m.Stores.Sessions == nil || m.Stores.Resumes == nil || m.Stores.Messages == nil {
		return ctx, nil, errors.New("session context requires session, resume, and message stores")
	}

	interviewSession, err := m.Stores.Sessions.GetSession(ctx, userID, sessionID)
	if err != nil {
		return ctx, nil, fmt.Errorf("load interview session: %w", err)
	}

	profile, err := m.loadProfile(ctx, userID, interviewSession.ResumeProfileID)
	if err != nil {
		return ctx, nil, err
	}
	activeProject, otherProjects := selectProject(profile.Projects, interviewSession.ActiveProjectID)

	limit := m.HistoryLimit
	if limit <= 0 {
		limit = defaultHistoryLimit
	}
	recentMessages, err := m.Stores.Messages.ListMessages(ctx, userID, sessionID, limit)
	if err != nil {
		return ctx, nil, fmt.Errorf("load recent messages: %w", err)
	}

	lastAnswer, _ := values["last_answer"].(string)
	turn := session.TurnContext{
		SessionID:       interviewSession.ID,
		UserID:          interviewSession.UserID,
		Role:            interviewSession.Role,
		Level:           interviewSession.Level,
		Mode:            interviewSession.Mode,
		Round:           interviewSession.Round,
		ResumeSummary:   profile.Summary,
		ResumeSkills:    profile.Skills,
		ResumeRawText:   profile.RawText,
		ActiveProject:   activeProject,
		OtherProjects:   otherProjects,
		CoveredTopics:   interviewSession.CoveredTopics,
		RecentQuestions: interviewSession.RecentQuestions,
		RecentMessages:  recentMessages,
		LastAnswer:      lastAnswer,
	}

	return WithTurnContext(ctx, turn), runCtx, nil
}

func (m *SessionContextMiddleware) loadProfile(ctx context.Context, userID string, profileID string) (resume.Profile, error) {
	if profileID != "" {
		profile, err := m.Stores.Resumes.GetProfile(ctx, userID, profileID)
		if err == nil {
			return profile, nil
		}
		return resume.Profile{}, fmt.Errorf("load resume profile: %w", err)
	}
	profile, err := m.Stores.Resumes.LatestProfile(ctx, userID)
	if err != nil {
		return resume.Profile{}, fmt.Errorf("load latest resume profile: %w", err)
	}
	return profile, nil
}

func selectProject(projects []resume.Project, activeProjectID string) (resume.Project, []resume.ProjectBrief) {
	if len(projects) == 0 {
		return resume.Project{}, nil
	}

	activeIndex := 0
	if activeProjectID != "" {
		for i, project := range projects {
			if project.ID == activeProjectID {
				activeIndex = i
				break
			}
		}
	}

	active := projects[activeIndex]
	others := make([]resume.ProjectBrief, 0, len(projects)-1)
	for i, project := range projects {
		if i != activeIndex {
			others = append(others, project.Brief())
		}
	}
	return active, others
}
