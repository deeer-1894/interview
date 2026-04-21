package service

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"

	"mockinterview/internal/protocol"
	artifactstorage "mockinterview/internal/storage/artifacts"
)

func (a *App) UploadArtifactFile(
	ctx context.Context,
	conversationID string,
	taskID string,
	runID string,
	filename string,
	declaredContentType string,
	content io.Reader,
) (protocol.Artifact, error) {
	conversationID = strings.TrimSpace(conversationID)
	if conversationID == "" {
		return protocol.Artifact{}, fmt.Errorf("conversationId is required")
	}

	filename = strings.TrimSpace(filename)
	if filename == "" {
		return protocol.Artifact{}, fmt.Errorf("file name is required")
	}

	now := time.Now()
	baseName := filepath.Base(filename)
	contentType := artifactstorage.DetectContentType(baseName, declaredContentType)
	storageKey := filepath.Join(conversationID, fmt.Sprintf("%s_%s", uuid.NewString(), baseName))

	return a.UploadArtifact(ctx, protocol.Artifact{
		ID:             uuid.NewString(),
		ConversationID: conversationID,
		TaskID:         strings.TrimSpace(taskID),
		RunID:          strings.TrimSpace(runID),
		Name:           baseName,
		ContentType:    contentType,
		StorageKey:     storageKey,
		CreatedAt:      now,
	}, content)
}

func (a *App) CreateTextArtifactContent(
	ctx context.Context,
	conversationID string,
	taskID string,
	runID string,
	name string,
	contentType string,
	content string,
) (protocol.Artifact, error) {
	conversationID = strings.TrimSpace(conversationID)
	if conversationID == "" {
		return protocol.Artifact{}, fmt.Errorf("conversationId is required")
	}

	name = strings.TrimSpace(name)
	if name == "" {
		return protocol.Artifact{}, fmt.Errorf("name is required")
	}

	now := time.Now()
	baseName := filepath.Base(name)
	storageKey := filepath.Join(conversationID, fmt.Sprintf("%s_%s", uuid.NewString(), baseName))

	return a.CreateTextArtifact(ctx, protocol.Artifact{
		ID:             uuid.NewString(),
		ConversationID: conversationID,
		TaskID:         strings.TrimSpace(taskID),
		RunID:          strings.TrimSpace(runID),
		Name:           baseName,
		ContentType:    artifactstorage.DetectContentType(baseName, contentType),
		StorageKey:     storageKey,
		CreatedAt:      now,
	}, content)
}

func (a *App) UpdateTextArtifactContent(
	ctx context.Context,
	id string,
	name string,
	contentType string,
	taskID string,
	runID string,
	content string,
) (protocol.Artifact, error) {
	artifact, err := a.GetArtifact(ctx, id)
	if err != nil {
		return protocol.Artifact{}, err
	}
	if !editableTextArtifact(artifact) {
		return protocol.Artifact{}, fmt.Errorf("only text artifacts can be edited inline")
	}

	name = strings.TrimSpace(name)
	if name != "" {
		artifact.Name = filepath.Base(name)
	}
	if normalizedType := strings.TrimSpace(contentType); normalizedType != "" || name != "" {
		artifact.ContentType = artifactstorage.DetectContentType(artifact.Name, normalizedType)
	}
	if normalizedTaskID := strings.TrimSpace(taskID); normalizedTaskID != "" {
		artifact.TaskID = normalizedTaskID
	}
	if normalizedRunID := strings.TrimSpace(runID); normalizedRunID != "" {
		artifact.RunID = normalizedRunID
	}

	return a.UpdateTextArtifact(ctx, artifact, content)
}

func editableTextArtifact(artifact protocol.Artifact) bool {
	contentType := strings.ToLower(strings.TrimSpace(artifact.ContentType))
	if strings.HasPrefix(contentType, "text/") {
		return true
	}

	switch contentType {
	case "application/json", "application/ld+json", "application/xml", "application/yaml", "application/x-yaml":
		return true
	}

	switch strings.ToLower(filepath.Ext(artifact.Name)) {
	case ".md", ".txt", ".json", ".yaml", ".yml", ".xml", ".csv", ".tsv":
		return true
	default:
		return false
	}
}
