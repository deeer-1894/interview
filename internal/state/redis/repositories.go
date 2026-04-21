package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	goredis "github.com/redis/go-redis/v9"

	"mockinterview/internal/protocol"
)

type Repositories struct {
	client *goredis.Client
	prefix string
	ttl    time.Duration
}

func New(client *goredis.Client, prefix string, ttl time.Duration) *Repositories {
	if prefix == "" {
		prefix = "mockinterview"
	}
	if ttl <= 0 {
		ttl = 24 * time.Hour
	}
	return &Repositories{
		client: client,
		prefix: prefix,
		ttl:    ttl,
	}
}

func (r *Repositories) Save(ctx context.Context, snapshot protocol.CheckpointSnapshot) error {
	return r.setJSON(ctx, r.checkpointKey(snapshot.RunID), snapshot, r.ttl)
}

func (r *Repositories) Load(ctx context.Context, runID string) (protocol.CheckpointSnapshot, error) {
	var snapshot protocol.CheckpointSnapshot
	if err := r.getJSON(ctx, r.checkpointKey(runID), &snapshot); err != nil {
		return protocol.CheckpointSnapshot{}, err
	}
	return snapshot, nil
}

func (r *Repositories) SaveClarify(ctx context.Context, request protocol.ClarifyRequest) error {
	return r.setJSON(ctx, r.clarifyKey(request.RunID), request, r.ttl)
}

func (r *Repositories) GetPendingClarify(ctx context.Context, runID string) (protocol.ClarifyRequest, error) {
	var request protocol.ClarifyRequest
	if err := r.getJSON(ctx, r.clarifyKey(runID), &request); err != nil {
		return protocol.ClarifyRequest{}, err
	}
	if request.Status != "pending" {
		return protocol.ClarifyRequest{}, fmt.Errorf("pending clarify for run %s not found", runID)
	}
	return request, nil
}

func (r *Repositories) ResolveClarify(ctx context.Context, runID string) error {
	request, err := r.GetPendingClarify(ctx, runID)
	if err != nil {
		return err
	}
	request.Status = "resolved"
	return r.setJSON(ctx, r.clarifyKey(runID), request, r.ttl)
}

func (r *Repositories) checkpointKey(runID string) string {
	return fmt.Sprintf("%s:checkpoint:%s", r.prefix, runID)
}

func (r *Repositories) clarifyKey(runID string) string {
	return fmt.Sprintf("%s:clarify:%s", r.prefix, runID)
}

func (r *Repositories) setJSON(ctx context.Context, key string, value any, ttl time.Duration) error {
	payload, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("marshal %s: %w", key, err)
	}
	if err := r.client.Set(ctx, key, payload, ttl).Err(); err != nil {
		return fmt.Errorf("set %s: %w", key, err)
	}
	return nil
}

func (r *Repositories) getJSON(ctx context.Context, key string, target any) error {
	payload, err := r.client.Get(ctx, key).Bytes()
	if err != nil {
		if err == goredis.Nil {
			return fmt.Errorf("%s not found", key)
		}
		return fmt.Errorf("get %s: %w", key, err)
	}
	if err := json.Unmarshal(payload, target); err != nil {
		return fmt.Errorf("unmarshal %s: %w", key, err)
	}
	return nil
}
