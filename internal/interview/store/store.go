package store

import (
	"context"
	"errors"
	"time"

	"mockinterview/internal/interview/report"
	"mockinterview/internal/interview/resume"
	"mockinterview/internal/interview/session"
)

var (
	ErrNotFound = errors.New("not found")
	ErrConflict = errors.New("conflict")
)

type SessionStore interface {
	CreateSession(ctx context.Context, s session.InterviewSession) (session.InterviewSession, error)
	GetSession(ctx context.Context, userID string, sessionID string) (session.InterviewSession, error)
	ListSessions(ctx context.Context, userID string, limit int) ([]session.InterviewSession, error)
	UpdateSession(ctx context.Context, s session.InterviewSession) error
	DeleteSession(ctx context.Context, userID string, sessionID string) error
}

type MessageStore interface {
	AppendMessage(ctx context.Context, msg session.Message) (session.Message, error)
	ListMessages(ctx context.Context, userID string, sessionID string, limit int) ([]session.Message, error)
	DeleteSessionMessages(ctx context.Context, userID string, sessionID string) error
}

type ResumeStore interface {
	SaveProfile(ctx context.Context, profile resume.Profile) (resume.Profile, error)
	GetProfile(ctx context.Context, userID string, profileID string) (resume.Profile, error)
	LatestProfile(ctx context.Context, userID string) (resume.Profile, error)
}

type ReportStore interface {
	SaveScorecard(ctx context.Context, scorecard report.Scorecard) (report.Scorecard, error)
	GetScorecard(ctx context.Context, userID string, sessionID string) (report.Scorecard, error)
}

type RunStore interface {
	AcquireRunLock(ctx context.Context, key string, ttl time.Duration) (token string, ok bool, err error)
	ReleaseRunLock(ctx context.Context, key string, token string) error
	SetCheckpointID(ctx context.Context, sessionID string, checkpointID string, ttl time.Duration) error
	GetCheckpointID(ctx context.Context, sessionID string) (string, bool, error)
}

type CheckpointStore interface {
	Get(ctx context.Context, checkpointID string) ([]byte, bool, error)
	Set(ctx context.Context, checkpointID string, checkpoint []byte) error
}

type Bundle struct {
	Sessions    SessionStore
	Messages    MessageStore
	Resumes     ResumeStore
	Reports     ReportStore
	Runs        RunStore
	Checkpoints CheckpointStore
}
