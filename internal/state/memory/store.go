package memory

import (
	"context"
	"slices"
	"sync"
	"time"

	"github.com/google/uuid"

	"mockinterview/internal/interview/report"
	"mockinterview/internal/interview/resume"
	"mockinterview/internal/interview/session"
	"mockinterview/internal/interview/store"
)

type Store struct {
	mu sync.RWMutex

	sessions     map[string]session.InterviewSession
	messages     map[string][]session.Message
	resumes      map[string]resume.Profile
	reports      map[string]report.Scorecard
	locks        map[string]lockEntry
	checkpointID map[string]expiringString
	checkpoints  map[string][]byte
}

type lockEntry struct {
	token     string
	expiresAt time.Time
}

type expiringString struct {
	value     string
	expiresAt time.Time
}

func New() *Store {
	return &Store{
		sessions:     make(map[string]session.InterviewSession),
		messages:     make(map[string][]session.Message),
		resumes:      make(map[string]resume.Profile),
		reports:      make(map[string]report.Scorecard),
		locks:        make(map[string]lockEntry),
		checkpointID: make(map[string]expiringString),
		checkpoints:  make(map[string][]byte),
	}
}

func (s *Store) Bundle() store.Bundle {
	return store.Bundle{
		Sessions:    s,
		Messages:    s,
		Resumes:     s,
		Reports:     s,
		Runs:        s,
		Checkpoints: s,
	}
}

func (s *Store) CreateSession(_ context.Context, item session.InterviewSession) (session.InterviewSession, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	if item.ID == "" {
		item.ID = uuid.NewString()
	}
	if item.CreatedAt.IsZero() {
		item.CreatedAt = now
	}
	item.UpdatedAt = now

	if _, exists := s.sessions[item.ID]; exists {
		return session.InterviewSession{}, store.ErrConflict
	}
	s.sessions[item.ID] = item
	return item, nil
}

func (s *Store) GetSession(_ context.Context, userID string, sessionID string) (session.InterviewSession, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	item, ok := s.sessions[sessionID]
	if !ok || item.UserID != userID {
		return session.InterviewSession{}, store.ErrNotFound
	}
	return item, nil
}

func (s *Store) ListSessions(_ context.Context, userID string, limit int) ([]session.InterviewSession, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	items := make([]session.InterviewSession, 0)
	for _, item := range s.sessions {
		if item.UserID == userID {
			items = append(items, item)
		}
	}
	slices.SortFunc(items, func(a, b session.InterviewSession) int {
		return b.UpdatedAt.Compare(a.UpdatedAt)
	})
	return capLimit(items, limit), nil
}

func (s *Store) UpdateSession(_ context.Context, item session.InterviewSession) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	existing, ok := s.sessions[item.ID]
	if !ok || existing.UserID != item.UserID {
		return store.ErrNotFound
	}
	if item.CreatedAt.IsZero() {
		item.CreatedAt = existing.CreatedAt
	}
	item.UpdatedAt = time.Now()
	s.sessions[item.ID] = item
	return nil
}

func (s *Store) DeleteSession(_ context.Context, userID string, sessionID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	item, ok := s.sessions[sessionID]
	if !ok || item.UserID != userID {
		return store.ErrNotFound
	}
	delete(s.sessions, sessionID)
	delete(s.messages, sessionID)
	delete(s.reports, sessionID)
	delete(s.checkpointID, sessionID)
	return nil
}

func (s *Store) AppendMessage(_ context.Context, msg session.Message) (session.Message, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	item, ok := s.sessions[msg.SessionID]
	if !ok || item.UserID != msg.UserID {
		return session.Message{}, store.ErrNotFound
	}
	if msg.ID == "" {
		msg.ID = uuid.NewString()
	}
	if msg.CreatedAt.IsZero() {
		msg.CreatedAt = time.Now()
	}
	s.messages[msg.SessionID] = append(s.messages[msg.SessionID], msg)
	return msg, nil
}

func (s *Store) ListMessages(_ context.Context, userID string, sessionID string, limit int) ([]session.Message, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	item, ok := s.sessions[sessionID]
	if !ok || item.UserID != userID {
		return nil, store.ErrNotFound
	}
	items := append([]session.Message(nil), s.messages[sessionID]...)
	slices.SortFunc(items, func(a, b session.Message) int {
		return a.CreatedAt.Compare(b.CreatedAt)
	})
	if limit > 0 && len(items) > limit {
		items = items[len(items)-limit:]
	}
	return items, nil
}

func (s *Store) DeleteSessionMessages(_ context.Context, userID string, sessionID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	item, ok := s.sessions[sessionID]
	if !ok || item.UserID != userID {
		return store.ErrNotFound
	}
	delete(s.messages, sessionID)
	return nil
}

func (s *Store) SaveProfile(_ context.Context, profile resume.Profile) (resume.Profile, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	if profile.ID == "" {
		profile.ID = uuid.NewString()
	}
	if profile.CreatedAt.IsZero() {
		profile.CreatedAt = now
	}
	profile.UpdatedAt = now
	s.resumes[profile.ID] = profile
	return profile, nil
}

func (s *Store) GetProfile(_ context.Context, userID string, profileID string) (resume.Profile, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	item, ok := s.resumes[profileID]
	if !ok || item.UserID != userID {
		return resume.Profile{}, store.ErrNotFound
	}
	return item, nil
}

func (s *Store) LatestProfile(_ context.Context, userID string) (resume.Profile, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var latest resume.Profile
	for _, item := range s.resumes {
		if item.UserID != userID {
			continue
		}
		if latest.ID == "" || item.UpdatedAt.After(latest.UpdatedAt) {
			latest = item
		}
	}
	if latest.ID == "" {
		return resume.Profile{}, store.ErrNotFound
	}
	return latest, nil
}

func (s *Store) SaveScorecard(_ context.Context, scorecard report.Scorecard) (report.Scorecard, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if scorecard.ID == "" {
		scorecard.ID = uuid.NewString()
	}
	if scorecard.CreatedAt.IsZero() {
		scorecard.CreatedAt = time.Now()
	}
	s.reports[scorecard.SessionID] = scorecard
	return scorecard, nil
}

func (s *Store) GetScorecard(_ context.Context, userID string, sessionID string) (report.Scorecard, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	item, ok := s.reports[sessionID]
	if !ok || item.UserID != userID {
		return report.Scorecard{}, store.ErrNotFound
	}
	return item, nil
}

func (s *Store) AcquireRunLock(_ context.Context, key string, ttl time.Duration) (string, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	if item, ok := s.locks[key]; ok && item.expiresAt.After(now) {
		return "", false, nil
	}
	token := uuid.NewString()
	s.locks[key] = lockEntry{token: token, expiresAt: now.Add(ttl)}
	return token, true, nil
}

func (s *Store) ReleaseRunLock(_ context.Context, key string, token string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	item, ok := s.locks[key]
	if !ok {
		return nil
	}
	if item.token == token {
		delete(s.locks, key)
	}
	return nil
}

func (s *Store) SetCheckpointID(_ context.Context, sessionID string, checkpointID string, ttl time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.checkpointID[sessionID] = expiringString{value: checkpointID, expiresAt: time.Now().Add(ttl)}
	return nil
}

func (s *Store) GetCheckpointID(_ context.Context, sessionID string) (string, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	item, ok := s.checkpointID[sessionID]
	if !ok || item.expiresAt.Before(time.Now()) {
		return "", false, nil
	}
	return item.value, true, nil
}

func (s *Store) Get(_ context.Context, checkpointID string) ([]byte, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	item, ok := s.checkpoints[checkpointID]
	if !ok {
		return nil, false, nil
	}
	return append([]byte(nil), item...), true, nil
}

func (s *Store) Set(_ context.Context, checkpointID string, checkpoint []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.checkpoints[checkpointID] = append([]byte(nil), checkpoint...)
	return nil
}

func capLimit[T any](items []T, limit int) []T {
	if limit <= 0 || len(items) <= limit {
		return items
	}
	return items[:limit]
}
