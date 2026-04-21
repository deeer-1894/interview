package mongo

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	mongodriver "go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"mockinterview/internal/protocol"
)

const (
	conversationsCollection = "conversations"
	tasksCollection         = "tasks"
	runsCollection          = "runs"
	messagesCollection      = "messages"
	eventsCollection        = "events"
	profilesCollection      = "profiles"
	memoryCollection        = "memories"
	artifactsCollection     = "artifacts"
)

type Repositories struct {
	db *mongodriver.Database
}

func New(db *mongodriver.Database) *Repositories {
	return &Repositories{db: db}
}

func (r *Repositories) CreateConversation(ctx context.Context, conversation protocol.Conversation) (protocol.Conversation, error) {
	if _, err := r.db.Collection(conversationsCollection).InsertOne(ctx, conversation); err != nil {
		return protocol.Conversation{}, fmt.Errorf("insert conversation: %w", err)
	}
	return conversation, nil
}

func (r *Repositories) UpdateConversation(ctx context.Context, conversation protocol.Conversation) (protocol.Conversation, error) {
	if _, err := r.db.Collection(conversationsCollection).ReplaceOne(ctx, bson.M{"id": conversation.ID}, conversation); err != nil {
		return protocol.Conversation{}, fmt.Errorf("replace conversation: %w", err)
	}
	return conversation, nil
}

func (r *Repositories) GetConversation(ctx context.Context, id string) (protocol.Conversation, error) {
	var conversation protocol.Conversation
	if err := r.db.Collection(conversationsCollection).FindOne(ctx, bson.M{"id": id}).Decode(&conversation); err != nil {
		return protocol.Conversation{}, fmt.Errorf("find conversation %s: %w", id, err)
	}
	return conversation, nil
}

func (r *Repositories) ListConversations(ctx context.Context) ([]protocol.Conversation, error) {
	cursor, err := r.db.Collection(conversationsCollection).Find(ctx, bson.D{}, options.Find().SetSort(bson.D{{Key: "updatedat", Value: -1}}))
	if err != nil {
		return nil, fmt.Errorf("find conversations: %w", err)
	}
	defer cursor.Close(ctx)

	var conversations []protocol.Conversation
	if err := cursor.All(ctx, &conversations); err != nil {
		return nil, fmt.Errorf("decode conversations: %w", err)
	}
	return conversations, nil
}

func (r *Repositories) CreateTask(ctx context.Context, task protocol.Task) (protocol.Task, error) {
	if _, err := r.db.Collection(tasksCollection).InsertOne(ctx, task); err != nil {
		return protocol.Task{}, fmt.Errorf("insert task: %w", err)
	}
	return task, nil
}

func (r *Repositories) GetTask(ctx context.Context, id string) (protocol.Task, error) {
	var task protocol.Task
	if err := r.db.Collection(tasksCollection).FindOne(ctx, bson.M{"id": id}).Decode(&task); err != nil {
		return protocol.Task{}, fmt.Errorf("find task %s: %w", id, err)
	}
	return task, nil
}

func (r *Repositories) ListTasksByConversation(ctx context.Context, conversationID string) ([]protocol.Task, error) {
	cursor, err := r.db.Collection(tasksCollection).Find(ctx, bson.M{"conversationid": conversationID}, options.Find().SetSort(bson.D{{Key: "updatedat", Value: -1}}))
	if err != nil {
		return nil, fmt.Errorf("find tasks: %w", err)
	}
	defer cursor.Close(ctx)

	var tasks []protocol.Task
	if err := cursor.All(ctx, &tasks); err != nil {
		return nil, fmt.Errorf("decode tasks: %w", err)
	}
	return tasks, nil
}

func (r *Repositories) CreateRun(ctx context.Context, run protocol.Run) (protocol.Run, error) {
	if _, err := r.db.Collection(runsCollection).InsertOne(ctx, run); err != nil {
		return protocol.Run{}, fmt.Errorf("insert run: %w", err)
	}
	return run, nil
}

func (r *Repositories) UpdateRun(ctx context.Context, run protocol.Run) (protocol.Run, error) {
	if _, err := r.db.Collection(runsCollection).ReplaceOne(ctx, bson.M{"id": run.ID}, run); err != nil {
		return protocol.Run{}, fmt.Errorf("replace run: %w", err)
	}
	return run, nil
}

func (r *Repositories) GetRun(ctx context.Context, id string) (protocol.Run, error) {
	var run protocol.Run
	if err := r.db.Collection(runsCollection).FindOne(ctx, bson.M{"id": id}).Decode(&run); err != nil {
		return protocol.Run{}, fmt.Errorf("find run %s: %w", id, err)
	}
	return run, nil
}

func (r *Repositories) ListRunsByConversation(ctx context.Context, conversationID string) ([]protocol.Run, error) {
	cursor, err := r.db.Collection(runsCollection).Find(ctx, bson.M{"conversationid": conversationID}, options.Find().SetSort(bson.D{{Key: "updatedat", Value: -1}}))
	if err != nil {
		return nil, fmt.Errorf("find runs: %w", err)
	}
	defer cursor.Close(ctx)

	var runs []protocol.Run
	if err := cursor.All(ctx, &runs); err != nil {
		return nil, fmt.Errorf("decode runs: %w", err)
	}
	return runs, nil
}

func (r *Repositories) CreateMessage(ctx context.Context, message protocol.Message) (protocol.Message, error) {
	if _, err := r.db.Collection(messagesCollection).InsertOne(ctx, message); err != nil {
		return protocol.Message{}, fmt.Errorf("insert message: %w", err)
	}
	return message, nil
}

func (r *Repositories) ListMessagesByRun(ctx context.Context, runID string) ([]protocol.Message, error) {
	cursor, err := r.db.Collection(messagesCollection).Find(ctx, bson.M{"runid": runID}, options.Find().SetSort(bson.D{{Key: "createdat", Value: 1}}))
	if err != nil {
		return nil, fmt.Errorf("find messages: %w", err)
	}
	defer cursor.Close(ctx)

	var messages []protocol.Message
	if err := cursor.All(ctx, &messages); err != nil {
		return nil, fmt.Errorf("decode messages: %w", err)
	}
	return messages, nil
}

func (r *Repositories) CreateEvent(ctx context.Context, event protocol.Event) (protocol.Event, error) {
	if _, err := r.db.Collection(eventsCollection).InsertOne(ctx, event); err != nil {
		return protocol.Event{}, fmt.Errorf("insert event: %w", err)
	}
	return event, nil
}

func (r *Repositories) GetProfile(ctx context.Context, id string) (protocol.CandidateProfile, error) {
	var profile protocol.CandidateProfile
	if err := r.db.Collection(profilesCollection).FindOne(ctx, bson.M{"id": id}).Decode(&profile); err != nil {
		return protocol.CandidateProfile{}, fmt.Errorf("find profile %s: %w", id, err)
	}
	return profile, nil
}

func (r *Repositories) SaveProfile(ctx context.Context, profile protocol.CandidateProfile) (protocol.CandidateProfile, error) {
	if _, err := r.db.Collection(profilesCollection).ReplaceOne(
		ctx,
		bson.M{"id": profile.ID},
		profile,
		options.Replace().SetUpsert(true),
	); err != nil {
		return protocol.CandidateProfile{}, fmt.Errorf("save profile: %w", err)
	}
	return profile, nil
}

func (r *Repositories) ListEventsByRun(ctx context.Context, runID string) ([]protocol.Event, error) {
	cursor, err := r.db.Collection(eventsCollection).Find(ctx, bson.M{"runid": runID}, options.Find().SetSort(bson.D{{Key: "timestamp", Value: 1}}))
	if err != nil {
		return nil, fmt.Errorf("find events: %w", err)
	}
	defer cursor.Close(ctx)

	var events []protocol.Event
	if err := cursor.All(ctx, &events); err != nil {
		return nil, fmt.Errorf("decode events: %w", err)
	}
	for i := range events {
		events[i].Payload = normalizeBSONValue(events[i].Payload)
	}
	return events, nil
}

func (r *Repositories) AppendMemory(ctx context.Context, record protocol.MemoryRecord) error {
	if _, err := r.db.Collection(memoryCollection).InsertOne(ctx, record); err != nil {
		return fmt.Errorf("insert memory: %w", err)
	}
	return nil
}

func (r *Repositories) ListMemory(ctx context.Context, runID string) ([]protocol.MemoryRecord, error) {
	cursor, err := r.db.Collection(memoryCollection).Find(ctx, bson.M{"runid": runID}, options.Find().SetSort(bson.D{{Key: "recordedat", Value: 1}}))
	if err != nil {
		return nil, fmt.Errorf("find memories: %w", err)
	}
	defer cursor.Close(ctx)

	var memories []protocol.MemoryRecord
	if err := cursor.All(ctx, &memories); err != nil {
		return nil, fmt.Errorf("decode memories: %w", err)
	}
	return memories, nil
}

func (r *Repositories) CreateArtifact(ctx context.Context, artifact protocol.Artifact) (protocol.Artifact, error) {
	if _, err := r.db.Collection(artifactsCollection).InsertOne(ctx, artifact); err != nil {
		return protocol.Artifact{}, fmt.Errorf("insert artifact: %w", err)
	}
	return artifact, nil
}

func (r *Repositories) UpdateArtifact(ctx context.Context, artifact protocol.Artifact) (protocol.Artifact, error) {
	if _, err := r.db.Collection(artifactsCollection).ReplaceOne(ctx, bson.M{"id": artifact.ID}, artifact); err != nil {
		return protocol.Artifact{}, fmt.Errorf("replace artifact: %w", err)
	}
	return artifact, nil
}

func (r *Repositories) GetArtifact(ctx context.Context, id string) (protocol.Artifact, error) {
	var artifact protocol.Artifact
	if err := r.db.Collection(artifactsCollection).FindOne(ctx, bson.M{"id": id}).Decode(&artifact); err != nil {
		return protocol.Artifact{}, fmt.Errorf("find artifact %s: %w", id, err)
	}
	return artifact, nil
}

func (r *Repositories) ListArtifactsByConversation(ctx context.Context, conversationID string) ([]protocol.Artifact, error) {
	cursor, err := r.db.Collection(artifactsCollection).Find(ctx, bson.M{"conversationid": conversationID}, options.Find().SetSort(bson.D{{Key: "createdat", Value: -1}}))
	if err != nil {
		return nil, fmt.Errorf("find artifacts: %w", err)
	}
	defer cursor.Close(ctx)

	var artifacts []protocol.Artifact
	if err := cursor.All(ctx, &artifacts); err != nil {
		return nil, fmt.Errorf("decode artifacts: %w", err)
	}
	return artifacts, nil
}

func (r *Repositories) DeleteArtifact(ctx context.Context, id string) error {
	result, err := r.db.Collection(artifactsCollection).DeleteOne(ctx, bson.M{"id": id})
	if err != nil {
		return fmt.Errorf("delete artifact %s: %w", id, err)
	}
	if result.DeletedCount == 0 {
		return fmt.Errorf("delete artifact %s: %w", id, mongo.ErrNoDocuments)
	}
	return nil
}

func EnsureIndexes(ctx context.Context, db *mongodriver.Database) error {
	models := []struct {
		name  string
		index mongodriver.IndexModel
	}{
		{conversationsCollection, mongodriver.IndexModel{Keys: bson.D{{Key: "updatedat", Value: -1}}}},
		{tasksCollection, mongodriver.IndexModel{Keys: bson.D{{Key: "conversationid", Value: 1}, {Key: "updatedat", Value: -1}}}},
		{runsCollection, mongodriver.IndexModel{Keys: bson.D{{Key: "conversationid", Value: 1}, {Key: "updatedat", Value: -1}}}},
		{messagesCollection, mongodriver.IndexModel{Keys: bson.D{{Key: "runid", Value: 1}, {Key: "createdat", Value: 1}}}},
		{eventsCollection, mongodriver.IndexModel{Keys: bson.D{{Key: "runid", Value: 1}, {Key: "timestamp", Value: 1}}}},
		{memoryCollection, mongodriver.IndexModel{Keys: bson.D{{Key: "runid", Value: 1}, {Key: "recordedat", Value: 1}}}},
		{artifactsCollection, mongodriver.IndexModel{Keys: bson.D{{Key: "conversationid", Value: 1}, {Key: "createdat", Value: -1}}}},
	}

	indexCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	for _, model := range models {
		if _, err := db.Collection(model.name).Indexes().CreateOne(indexCtx, model.index); err != nil {
			return fmt.Errorf("create index for %s: %w", model.name, err)
		}
	}
	return nil
}

func normalizeBSONValue(value any) any {
	switch typed := value.(type) {
	case primitive.D:
		return normalizeDocument(typed)
	case primitive.A:
		normalized := make([]any, 0, len(typed))
		for _, item := range typed {
			normalized = append(normalized, normalizeBSONValue(item))
		}
		return normalized
	case []any:
		normalized := make([]any, 0, len(typed))
		for _, item := range typed {
			normalized = append(normalized, normalizeBSONValue(item))
		}
		return normalized
	case map[string]any:
		normalized := make(map[string]any, len(typed))
		for key, item := range typed {
			normalized[key] = normalizeBSONValue(item)
		}
		return normalized
	default:
		return value
	}
}

func normalizeDocument(document primitive.D) map[string]any {
	normalized := make(map[string]any, len(document))
	for _, element := range document {
		normalized[element.Key] = normalizeBSONValue(element.Value)
	}
	return normalized
}
