package service

import (
	"context"
	"time"

	"mockinterview/internal/protocol"
)

type adkCheckpointStore struct {
	repo CheckpointRepository
	run  protocol.Run
	task protocol.Task
}

func newADKCheckpointStore(repo CheckpointRepository, run protocol.Run, task protocol.Task) *adkCheckpointStore {
	return &adkCheckpointStore{
		repo: repo,
		run:  run,
		task: task,
	}
}

func (s *adkCheckpointStore) Get(ctx context.Context, checkPointID string) ([]byte, bool, error) {
	snapshot, err := s.repo.Load(ctx, checkPointID)
	if err != nil {
		return nil, false, err
	}
	if len(snapshot.RawState) == 0 {
		return nil, false, nil
	}
	return append([]byte(nil), snapshot.RawState...), true, nil
}

func (s *adkCheckpointStore) Set(ctx context.Context, checkPointID string, checkPoint []byte) error {
	snapshot, err := s.repo.Load(ctx, checkPointID)
	if err != nil {
		snapshot = protocol.CheckpointSnapshot{
			RunID:          s.run.ID,
			ConversationID: s.run.ConversationID,
			TaskID:         s.run.TaskID,
			Prompt:         s.task.Prompt,
			Input:          s.run.Input,
			RunStatus:      s.run.Status,
			RunPhase:       s.run.Phase,
			InterviewState: s.run.InterviewState,
			Config:         s.task.Config.WithDefaults(),
			ModelConfig:    s.task.ModelConfig,
		}
	}
	snapshot.RawState = append([]byte(nil), checkPoint...)
	snapshot.UpdatedAt = time.Now()
	return s.repo.Save(ctx, snapshot)
}
