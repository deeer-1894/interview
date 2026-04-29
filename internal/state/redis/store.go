package redis

import (
	"context"
	"time"

	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
)

const (
	lockPrefix       = "offerbot:run:lock:"
	checkpointPrefix = "offerbot:adk:checkpoint:"
	sessionCPPrefix  = "offerbot:session:checkpoint:"
)

type Store struct {
	client        goredis.UniversalClient
	checkpointTTL time.Duration
}

func New(client goredis.UniversalClient, checkpointTTL time.Duration) *Store {
	if checkpointTTL <= 0 {
		checkpointTTL = 24 * time.Hour
	}
	return &Store{client: client, checkpointTTL: checkpointTTL}
}

func (s *Store) AcquireRunLock(ctx context.Context, key string, ttl time.Duration) (string, bool, error) {
	if ttl <= 0 {
		ttl = 5 * time.Minute
	}
	token := uuid.NewString()
	ok, err := s.client.SetNX(ctx, lockPrefix+key, token, ttl).Result()
	if err != nil {
		return "", false, err
	}
	if !ok {
		return "", false, nil
	}
	return token, true, nil
}

func (s *Store) ReleaseRunLock(ctx context.Context, key string, token string) error {
	script := goredis.NewScript(`
if redis.call("get", KEYS[1]) == ARGV[1] then
	return redis.call("del", KEYS[1])
end
return 0
`)
	return script.Run(ctx, s.client, []string{lockPrefix + key}, token).Err()
}

func (s *Store) SetCheckpointID(ctx context.Context, sessionID string, checkpointID string, ttl time.Duration) error {
	if ttl <= 0 {
		ttl = s.checkpointTTL
	}
	return s.client.Set(ctx, sessionCPPrefix+sessionID, checkpointID, ttl).Err()
}

func (s *Store) GetCheckpointID(ctx context.Context, sessionID string) (string, bool, error) {
	value, err := s.client.Get(ctx, sessionCPPrefix+sessionID).Result()
	if err == goredis.Nil {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return value, true, nil
}

func (s *Store) Get(ctx context.Context, checkpointID string) ([]byte, bool, error) {
	value, err := s.client.Get(ctx, checkpointPrefix+checkpointID).Bytes()
	if err == goredis.Nil {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	return value, true, nil
}

func (s *Store) Set(ctx context.Context, checkpointID string, checkpoint []byte) error {
	return s.client.Set(ctx, checkpointPrefix+checkpointID, checkpoint, s.checkpointTTL).Err()
}
