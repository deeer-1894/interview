package mongo

import (
	"context"
	"errors"
	"slices"
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	mongodriver "go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"mockinterview/internal/interview/report"
	"mockinterview/internal/interview/resume"
	"mockinterview/internal/interview/session"
	"mockinterview/internal/interview/store"
)

const (
	sessionsCollection = "interview_sessions"
	messagesCollection = "interview_messages"
	resumesCollection  = "resume_profiles"
	reportsCollection  = "interview_scorecards"
)

type Store struct {
	db *mongodriver.Database
}

func New(db *mongodriver.Database) *Store {
	return &Store{db: db}
}

func (s *Store) Bundle(runs store.RunStore, checkpoints store.CheckpointStore) store.Bundle {
	return store.Bundle{
		Sessions:    s,
		Messages:    s,
		Resumes:     s,
		Reports:     s,
		Runs:        runs,
		Checkpoints: checkpoints,
	}
}

func (s *Store) CreateSession(ctx context.Context, item session.InterviewSession) (session.InterviewSession, error) {
	now := time.Now()
	if item.ID == "" {
		item.ID = uuid.NewString()
	}
	if item.CreatedAt.IsZero() {
		item.CreatedAt = now
	}
	item.UpdatedAt = now

	_, err := s.db.Collection(sessionsCollection).InsertOne(ctx, item)
	if mongodriver.IsDuplicateKeyError(err) {
		return session.InterviewSession{}, store.ErrConflict
	}
	return item, err
}

func (s *Store) GetSession(ctx context.Context, userID string, sessionID string) (session.InterviewSession, error) {
	var item session.InterviewSession
	err := s.db.Collection(sessionsCollection).FindOne(ctx, bson.M{"_id": sessionID, "userId": userID}).Decode(&item)
	if isNotFound(err) {
		return session.InterviewSession{}, store.ErrNotFound
	}
	return item, err
}

func (s *Store) ListSessions(ctx context.Context, userID string, limit int) ([]session.InterviewSession, error) {
	opts := options.Find().SetSort(bson.D{{Key: "updatedAt", Value: -1}})
	if limit > 0 {
		opts.SetLimit(int64(limit))
	}
	cur, err := s.db.Collection(sessionsCollection).Find(ctx, bson.M{"userId": userID}, opts)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	var items []session.InterviewSession
	if err := cur.All(ctx, &items); err != nil {
		return nil, err
	}
	return items, nil
}

func (s *Store) UpdateSession(ctx context.Context, item session.InterviewSession) error {
	if item.ID == "" || item.UserID == "" {
		return store.ErrNotFound
	}
	item.UpdatedAt = time.Now()
	result, err := s.db.Collection(sessionsCollection).ReplaceOne(
		ctx,
		bson.M{"_id": item.ID, "userId": item.UserID},
		item,
	)
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return store.ErrNotFound
	}
	return nil
}

func (s *Store) DeleteSession(ctx context.Context, userID string, sessionID string) error {
	result, err := s.db.Collection(sessionsCollection).DeleteOne(ctx, bson.M{"_id": sessionID, "userId": userID})
	if err != nil {
		return err
	}
	if result.DeletedCount == 0 {
		return store.ErrNotFound
	}
	_, _ = s.db.Collection(messagesCollection).DeleteMany(ctx, bson.M{"sessionId": sessionID, "userId": userID})
	_, _ = s.db.Collection(reportsCollection).DeleteMany(ctx, bson.M{"sessionId": sessionID, "userId": userID})
	return nil
}

func (s *Store) AppendMessage(ctx context.Context, msg session.Message) (session.Message, error) {
	_, err := s.GetSession(ctx, msg.UserID, msg.SessionID)
	if err != nil {
		return session.Message{}, err
	}
	if msg.ID == "" {
		msg.ID = uuid.NewString()
	}
	if msg.CreatedAt.IsZero() {
		msg.CreatedAt = time.Now()
	}
	_, err = s.db.Collection(messagesCollection).InsertOne(ctx, msg)
	return msg, err
}

func (s *Store) ListMessages(ctx context.Context, userID string, sessionID string, limit int) ([]session.Message, error) {
	_, err := s.GetSession(ctx, userID, sessionID)
	if err != nil {
		return nil, err
	}

	opts := options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}})
	if limit > 0 {
		opts.SetLimit(int64(limit))
	}
	cur, err := s.db.Collection(messagesCollection).Find(ctx, bson.M{"sessionId": sessionID, "userId": userID}, opts)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	var items []session.Message
	if err := cur.All(ctx, &items); err != nil {
		return nil, err
	}
	slices.Reverse(items)
	return items, nil
}

func (s *Store) DeleteSessionMessages(ctx context.Context, userID string, sessionID string) error {
	_, err := s.GetSession(ctx, userID, sessionID)
	if err != nil {
		return err
	}
	_, err = s.db.Collection(messagesCollection).DeleteMany(ctx, bson.M{"sessionId": sessionID, "userId": userID})
	return err
}

func (s *Store) SaveProfile(ctx context.Context, profile resume.Profile) (resume.Profile, error) {
	now := time.Now()
	if profile.ID == "" {
		profile.ID = uuid.NewString()
	}
	if profile.CreatedAt.IsZero() {
		profile.CreatedAt = now
	}
	profile.UpdatedAt = now

	_, err := s.db.Collection(resumesCollection).ReplaceOne(
		ctx,
		bson.M{"_id": profile.ID, "userId": profile.UserID},
		profile,
		options.Replace().SetUpsert(true),
	)
	return profile, err
}

func (s *Store) GetProfile(ctx context.Context, userID string, profileID string) (resume.Profile, error) {
	var item resume.Profile
	err := s.db.Collection(resumesCollection).FindOne(ctx, bson.M{"_id": profileID, "userId": userID}).Decode(&item)
	if isNotFound(err) {
		return resume.Profile{}, store.ErrNotFound
	}
	return item, err
}

func (s *Store) LatestProfile(ctx context.Context, userID string) (resume.Profile, error) {
	var item resume.Profile
	err := s.db.Collection(resumesCollection).FindOne(
		ctx,
		bson.M{"userId": userID},
		options.FindOne().SetSort(bson.D{{Key: "updatedAt", Value: -1}}),
	).Decode(&item)
	if isNotFound(err) {
		return resume.Profile{}, store.ErrNotFound
	}
	return item, err
}

func (s *Store) SaveScorecard(ctx context.Context, scorecard report.Scorecard) (report.Scorecard, error) {
	if scorecard.ID == "" {
		scorecard.ID = uuid.NewString()
	}
	if scorecard.CreatedAt.IsZero() {
		scorecard.CreatedAt = time.Now()
	}
	_, err := s.db.Collection(reportsCollection).ReplaceOne(
		ctx,
		bson.M{"sessionId": scorecard.SessionID, "userId": scorecard.UserID},
		scorecard,
		options.Replace().SetUpsert(true),
	)
	return scorecard, err
}

func (s *Store) GetScorecard(ctx context.Context, userID string, sessionID string) (report.Scorecard, error) {
	var item report.Scorecard
	err := s.db.Collection(reportsCollection).FindOne(ctx, bson.M{"sessionId": sessionID, "userId": userID}).Decode(&item)
	if isNotFound(err) {
		return report.Scorecard{}, store.ErrNotFound
	}
	return item, err
}

func isNotFound(err error) bool {
	return errors.Is(err, mongodriver.ErrNoDocuments)
}
