package bootstrap

import (
	"context"
	"fmt"
	"os"
	"time"

	goredis "github.com/redis/go-redis/v9"
	mongodriver "go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	controlservice "mockinterview/internal/control/service"
	mongostate "mockinterview/internal/state/mongo"
	redisstate "mockinterview/internal/state/redis"
)

type Bundle struct {
	Conversations controlservice.ConversationRepository
	Tasks         controlservice.TaskRepository
	Runs          controlservice.RunRepository
	Messages      controlservice.MessageRepository
	Events        controlservice.EventRepository
	Profiles      controlservice.ProfileRepository
	Checkpoints   controlservice.CheckpointRepository
	Clarifies     controlservice.ClarifyRequestRepository
	Memories      controlservice.MemoryRepository
	Artifacts     controlservice.ArtifactRepository
	Close         func(context.Context) error
}

func NewFromEnv(ctx context.Context) (Bundle, error) {
	return NewPersistentBundle(ctx)
}

func NewPersistentBundle(ctx context.Context) (Bundle, error) {
	redisAddr := firstNonEmpty(os.Getenv("REDIS_ADDR"), "localhost:6379")
	redisPassword := os.Getenv("REDIS_PASSWORD")
	redisDB := 0
	redisClient := goredis.NewClient(&goredis.Options{
		Addr:     redisAddr,
		Password: redisPassword,
		DB:       redisDB,
	})
	if err := redisClient.Ping(ctx).Err(); err != nil {
		return Bundle{}, fmt.Errorf("connect redis: %w", err)
	}

	mongoURI := firstNonEmpty(os.Getenv("MONGO_URI"), "mongodb://localhost:27017")
	mongoDatabase := firstNonEmpty(os.Getenv("MONGO_DATABASE"), "mockinterview")
	mongoCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	mongoClient, err := mongodriver.Connect(mongoCtx, options.Client().ApplyURI(mongoURI))
	if err != nil {
		_ = redisClient.Close()
		return Bundle{}, fmt.Errorf("connect mongo: %w", err)
	}
	if err := mongoClient.Ping(mongoCtx, nil); err != nil {
		_ = redisClient.Close()
		_ = mongoClient.Disconnect(context.Background())
		return Bundle{}, fmt.Errorf("ping mongo: %w", err)
	}

	db := mongoClient.Database(mongoDatabase)
	if err := mongostate.EnsureIndexes(ctx, db); err != nil {
		_ = redisClient.Close()
		_ = mongoClient.Disconnect(context.Background())
		return Bundle{}, fmt.Errorf("ensure mongo indexes: %w", err)
	}

	mongoRepositories := mongostate.New(db)
	conversations, tasks, runs, messages, events, profiles, _, _, memories, artifacts := mongostate.NewAdapters(mongoRepositories)
	redisRepositories := redisstate.New(redisClient, firstNonEmpty(os.Getenv("REDIS_PREFIX"), "mockinterview"), 24*time.Hour)
	checkpoints, clarifies := redisstate.NewAdapters(redisRepositories)

	return Bundle{
		Conversations: conversations,
		Tasks:         tasks,
		Runs:          runs,
		Messages:      messages,
		Events:        events,
		Profiles:      profiles,
		Checkpoints:   checkpoints,
		Clarifies:     clarifies,
		Memories:      memories,
		Artifacts:     artifacts,
		Close: func(closeCtx context.Context) error {
			var firstErr error
			if err := redisClient.Close(); err != nil && firstErr == nil {
				firstErr = err
			}
			if err := mongoClient.Disconnect(closeCtx); err != nil && firstErr == nil {
				firstErr = err
			}
			return firstErr
		},
	}, nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
