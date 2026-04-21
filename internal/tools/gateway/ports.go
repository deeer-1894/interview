package gateway

import (
	"context"

	"mockinterview/internal/protocol"
)

type CheckpointStore interface {
	Save(ctx context.Context, snapshot protocol.CheckpointSnapshot) error
	Load(ctx context.Context, runID string) (protocol.CheckpointSnapshot, error)
}

type MemoryStore interface {
	Append(ctx context.Context, record protocol.MemoryRecord) error
	List(ctx context.Context, runID string) ([]protocol.MemoryRecord, error)
}
