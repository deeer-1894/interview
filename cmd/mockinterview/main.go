package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
	mongodriver "go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"mockinterview/internal/interview/resume"
	interviewrt "mockinterview/internal/interview/runtime"
	"mockinterview/internal/interview/session"
	"mockinterview/internal/interview/store"
	statemongo "mockinterview/internal/state/mongo"
	stateredis "mockinterview/internal/state/redis"
)

const (
	devEmail    = "shiyi123@123.com"
	devPassword = "shiyi123456"
	devToken    = "dev-token-shiyi123"
)

type app struct {
	stores store.Bundle
	rt     *interviewrt.Runtime
}

type messageOutput struct {
	Text      string
	Candidate bool
}

func newStores(ctx context.Context) (store.Bundle, func(), error) {
	mongoURI := strings.TrimSpace(os.Getenv("MONGO_URI"))
	mongoDatabase := strings.TrimSpace(os.Getenv("MONGO_DATABASE"))
	redisAddr := strings.TrimSpace(os.Getenv("REDIS_ADDR"))
	if mongoURI == "" {
		mongoURI = "mongodb://localhost:27017"
	}
	if mongoDatabase == "" {
		mongoDatabase = "mockinterview"
	}
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}

	connectCtx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()

	mongoClient, err := mongodriver.Connect(connectCtx, options.Client().ApplyURI(mongoURI))
	if err != nil {
		return store.Bundle{}, nil, fmt.Errorf("connect mongo: %w", err)
	}
	if err := mongoClient.Ping(connectCtx, nil); err != nil {
		_ = mongoClient.Disconnect(context.Background())
		return store.Bundle{}, nil, fmt.Errorf("ping mongo: %w", err)
	}

	redisClient := goredis.NewClient(&goredis.Options{
		Addr:     redisAddr,
		Password: os.Getenv("REDIS_PASSWORD"),
	})
	if err := redisClient.Ping(connectCtx).Err(); err != nil {
		_ = redisClient.Close()
		_ = mongoClient.Disconnect(context.Background())
		return store.Bundle{}, nil, fmt.Errorf("ping redis: %w", err)
	}

	mongoStore := statemongo.New(mongoClient.Database(mongoDatabase))
	redisStore := stateredis.New(redisClient, 24*time.Hour)
	closeFn := func() {
		_ = redisClient.Close()
		_ = mongoClient.Disconnect(context.Background())
	}
	return mongoStore.Bundle(redisStore, redisStore), closeFn, nil
}

func main() {
	addr := flag.String("addr", ":8080", "listen address")
	serve := flag.Bool("serve", false, "start HTTP server")
	flag.Parse()
	if !*serve {
		fmt.Println("use -serve to start the mock interview server")
		return
	}

	ctx := context.Background()
	stores, closeStores, err := newStores(ctx)
	if err != nil {
		panic(err)
	}
	defer closeStores()

	tools := interviewrt.NewGraphTools()
	tools = append(tools, interviewrt.NewClarifyTool())

	patchToolCalls, err := interviewrt.NewPatchToolCallsMiddleware(ctx)
	if err != nil {
		panic(err)
	}
	allowedTools := append(interviewrt.GraphToolNames(), interviewrt.ControlToolNames()...)
	chatModel, modelConfig, err := interviewrt.NewModelFromEnv(ctx)
	if err != nil {
		panic(err)
	}
	fmt.Printf("using model provider=%s model=%s baseURL=%s\n", modelConfig.Provider, modelConfig.Name, modelConfig.BaseURL)
	rt, err := interviewrt.New(ctx, interviewrt.Config{
		Model:           chatModel,
		Tools:           tools,
		Stores:          stores,
		EnableStreaming: true,
		Handlers: []adk.ChatModelAgentMiddleware{
			interviewrt.NewTraceMiddleware(),
			interviewrt.NewSafeToolMiddleware(allowedTools, 8*time.Second),
			interviewrt.NewSessionContextMiddleware(stores),
			interviewrt.NewContextIsolationMiddleware(),
			patchToolCalls,
			interviewrt.NewQuestionQualityMiddleware(),
		},
	})
	if err != nil {
		panic(err)
	}

	a := &app{stores: stores, rt: rt}
	router := a.router()
	if err := router.Run(*addr); err != nil {
		panic(err)
	}
}

func (a *app) router() *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery(), cors())

	r.GET("/api/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})
	r.POST("/api/login", a.login)

	api := r.Group("/api", auth())
	api.GET("/profile", a.profile)
	api.POST("/profile", a.saveProfile)
	api.GET("/skills", a.skills)
	api.GET("/sessions", a.sessions)
	api.POST("/sessions", a.createSession)
	api.POST("/sessions/stream", a.createSessionStreaming)
	api.DELETE("/sessions/:id", a.deleteSession)
	api.GET("/sessions/:id/messages", a.messages)
	api.POST("/sessions/:id/messages", a.postMessage)
	api.POST("/sessions/:id/messages/stream", a.streamPostMessage)
	api.POST("/sessions/:id/resume", a.resumeSession)
	api.GET("/sessions/:id/stream", a.streamMessage)
	api.GET("/sessions/:id/report", a.scorecard)
	return r
}

func cors() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Headers", "Authorization, Content-Type")
		c.Header("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
		if c.Request.Method == http.MethodOptions {
			c.Status(http.StatusNoContent)
			c.Abort()
			return
		}
		c.Next()
	}
}

func auth() gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		queryToken := c.Query("token")
		if header != "Bearer "+devToken && queryToken != devToken {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			c.Abort()
			return
		}
		c.Set("userID", devEmail)
		c.Next()
	}
}

func (a *app) login(c *gin.Context) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Email != devEmail || req.Password != devPassword {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"token": devToken, "user": gin.H{"id": devEmail, "email": devEmail}})
}

func (a *app) profile(c *gin.Context) {
	profile, err := a.stores.Resumes.LatestProfile(c.Request.Context(), userID(c))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, profile)
}

func (a *app) saveProfile(c *gin.Context) {
	var req struct {
		ID       string           `json:"id"`
		RawText  string           `json:"rawText"`
		Summary  string           `json:"summary"`
		Skills   []string         `json:"skills"`
		Projects []resume.Project `json:"projects"`
	}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if strings.TrimSpace(req.RawText) == "" && len(req.Projects) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "rawText or projects is required"})
		return
	}
	now := time.Now()
	profile := resume.Profile{
		ID:        req.ID,
		UserID:    userID(c),
		RawText:   req.RawText,
		Summary:   req.Summary,
		Skills:    req.Skills,
		Projects:  req.Projects,
		CreatedAt: now,
		UpdatedAt: now,
	}
	saved, err := a.stores.Resumes.SaveProfile(c.Request.Context(), profile)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, saved)
}

func (a *app) skills(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"skills": []string{
		"backend-system-design",
		"agent-runtime-and-tools",
		"go-concurrency-runtime",
		"python-automation-agent",
		"data-pipeline-and-storage",
		"content-safety-and-risk",
	}})
}

func (a *app) sessions(c *gin.Context) {
	items, err := a.stores.Sessions.ListSessions(c.Request.Context(), userID(c), 50)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if items == nil {
		items = []session.InterviewSession{}
	}
	c.JSON(http.StatusOK, gin.H{"sessions": items})
}

func (a *app) createSession(c *gin.Context) {
	var req struct {
		Role            string `json:"role"`
		Level           string `json:"level"`
		Mode            string `json:"mode"`
		ActiveProjectID string `json:"activeProjectId"`
		ResumeProfileID string `json:"resumeProfileId"`
	}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.ResumeProfileID == "" {
		profile, err := a.stores.Resumes.LatestProfile(c.Request.Context(), userID(c))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "resume profile is required"})
			return
		}
		req.ResumeProfileID = profile.ID
	}
	now := time.Now()
	item := session.NewInterviewSession(now)
	item.ID = uuid.NewString()
	item.UserID = userID(c)
	item.Role = nonEmptyString(req.Role, "后端/AI Agent 工程师")
	item.Level = nonEmptyString(req.Level, "中高级")
	item.Mode = nonEmptyString(req.Mode, "long")
	item.ResumeProfileID = req.ResumeProfileID
	item.ActiveProjectID = req.ActiveProjectID
	item.Round = 1
	created, err := a.stores.Sessions.CreateSession(c.Request.Context(), item)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if err := a.generateOpening(c.Request.Context(), created); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, created)
}

func (a *app) createSessionStreaming(c *gin.Context) {
	var req struct {
		Role            string `json:"role"`
		Level           string `json:"level"`
		Mode            string `json:"mode"`
		ActiveProjectID string `json:"activeProjectId"`
		ResumeProfileID string `json:"resumeProfileId"`
	}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.ResumeProfileID == "" {
		profile, err := a.stores.Resumes.LatestProfile(c.Request.Context(), userID(c))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "resume profile is required"})
			return
		}
		req.ResumeProfileID = profile.ID
	}

	now := time.Now()
	item := session.NewInterviewSession(now)
	item.ID = uuid.NewString()
	item.UserID = userID(c)
	item.Role = nonEmptyString(req.Role, "后端/AI Agent 工程师")
	item.Level = nonEmptyString(req.Level, "中高级")
	item.Mode = nonEmptyString(req.Mode, "long")
	item.ResumeProfileID = req.ResumeProfileID
	item.ActiveProjectID = req.ActiveProjectID
	item.Round = 1
	created, err := a.stores.Sessions.CreateSession(c.Request.Context(), item)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")
	writeSSE(c, "session", created)

	assistant, err := a.generateOpeningStreaming(c.Request.Context(), c, created)
	if err != nil {
		writeSSE(c, "error", gin.H{"error": err.Error()})
		return
	}
	writeSSE(c, "done", gin.H{"session": created, "assistant": assistant})
}

func (a *app) deleteSession(c *gin.Context) {
	if err := a.stores.Sessions.DeleteSession(c.Request.Context(), userID(c), c.Param("id")); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

func (a *app) messages(c *gin.Context) {
	items, err := a.stores.Messages.ListMessages(c.Request.Context(), userID(c), c.Param("id"), 100)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"messages": items})
}

func (a *app) postMessage(c *gin.Context) {
	var req struct {
		Content string `json:"content"`
	}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	result, err := a.runTurn(c.Request.Context(), userID(c), c.Param("id"), req.Content)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}

func (a *app) streamPostMessage(c *gin.Context) {
	var req struct {
		Content string `json:"content"`
	}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if strings.TrimSpace(req.Content) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "content is required"})
		return
	}

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	if err := a.runTurnStreaming(c.Request.Context(), c, userID(c), c.Param("id"), req.Content); err != nil {
		writeSSE(c, "error", gin.H{"error": err.Error()})
	}
}

func (a *app) resumeSession(c *gin.Context) {
	var req struct {
		CheckpointID string `json:"checkpointId"`
		InterruptID  string `json:"interruptId"`
		Content      string `json:"content"`
	}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	result, err := a.resumeTurn(c.Request.Context(), userID(c), c.Param("id"), req.CheckpointID, req.InterruptID, req.Content)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}

func (a *app) streamMessage(c *gin.Context) {
	content := c.Query("message")
	if strings.TrimSpace(content) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "message is required"})
		return
	}
	result, err := a.runTurn(c.Request.Context(), userID(c), c.Param("id"), content)
	if err != nil {
		c.SSEvent("error", gin.H{"error": err.Error()})
		return
	}
	for _, event := range result.Events {
		c.SSEvent(event.Type, event.Data)
		c.Writer.Flush()
	}
	c.SSEvent("done", result)
}

func (a *app) runTurnStreaming(ctx context.Context, c *gin.Context, userID string, sessionID string, content string) error {
	current, err := a.stores.Sessions.GetSession(ctx, userID, sessionID)
	if err != nil {
		return err
	}
	userMsg := session.Message{SessionID: sessionID, UserID: userID, Role: session.RoleUser, Content: content}
	if _, err := a.stores.Messages.AppendMessage(ctx, userMsg); err != nil {
		return err
	}

	iter, checkpointID, err := a.rt.RunTurn(ctx, interviewrt.TurnRequest{UserID: userID, SessionID: sessionID, Input: content})
	if err != nil {
		return err
	}

	var result turnResult
	result.CheckpointID = checkpointID
	var assistant strings.Builder
	var candidate strings.Builder
	seenToolOutput := false
	for {
		event, ok := iter.Next()
		if !ok {
			break
		}
		if event == nil {
			continue
		}
		if event.Err != nil {
			return event.Err
		}
		if event.Action != nil {
			if event.Action.CustomizedAction != nil {
				apiEvent := apiEvent{Type: "custom", Data: event.Action.CustomizedAction}
				result.Events = append(result.Events, apiEvent)
				writeSSE(c, apiEvent.Type, apiEvent.Data)
			}
			if event.Action.Interrupted != nil {
				result.Interrupted = event.Action.Interrupted
				apiEvent := apiEvent{Type: "interrupted", Data: event.Action.Interrupted}
				result.Events = append(result.Events, apiEvent)
				writeSSE(c, apiEvent.Type, apiEvent.Data)
			}
		}
		if event.Output == nil || event.Output.MessageOutput == nil {
			continue
		}
		if event.Output.MessageOutput.Role == schema.Tool {
			seenToolOutput = true
		}
		output, err := streamMessageOutput(c, event.Output.MessageOutput, seenToolOutput)
		if err != nil {
			return err
		}
		if output.Text != "" {
			if output.Candidate {
				candidate.WriteString(output.Text)
			} else {
				assistant.WriteString(output.Text)
			}
		}
	}

	result.Assistant = chooseAssistantText(assistant.String(), candidate.String())
	if strings.TrimSpace(assistant.String()) == "" && strings.TrimSpace(result.Assistant) != "" {
		writeSSE(c, "delta", result.Assistant)
	}
	if strings.TrimSpace(result.Assistant) == "" && result.Interrupted == nil {
		return fmt.Errorf("empty assistant response")
	}
	if result.Assistant != "" {
		assistantMsg := session.Message{SessionID: sessionID, UserID: userID, Role: session.RoleAssistant, Content: result.Assistant}
		if _, err := a.stores.Messages.AppendMessage(ctx, assistantMsg); err != nil {
			return err
		}
		current.Round++
		current.Status = session.StatusRunning
		current.CheckpointID = checkpointID
		if err := a.stores.Sessions.UpdateSession(ctx, current); err != nil {
			return err
		}
	}
	if result.Interrupted != nil {
		current.Status = session.StatusInterrupted
		current.CheckpointID = checkpointID
		_ = a.stores.Sessions.UpdateSession(ctx, current)
	}
	writeSSE(c, "done", result)
	return nil
}

func writeSSE(c *gin.Context, event string, data interface{}) {
	payload, err := json.Marshal(data)
	if err != nil {
		payload = []byte(`{"error":"failed to encode event"}`)
		event = "error"
	}
	_, _ = fmt.Fprintf(c.Writer, "event: %s\n", event)
	_, _ = fmt.Fprintf(c.Writer, "data: %s\n\n", payload)
	c.Writer.Flush()
}

func streamMessageOutput(c *gin.Context, output *adk.MessageVariant, live bool) (messageOutput, error) {
	if output == nil {
		return messageOutput{}, nil
	}
	if output.Role != schema.Assistant {
		_, err := drainMessageOutput(output)
		return messageOutput{}, err
	}
	if output.IsStreaming && output.MessageStream != nil {
		var b strings.Builder
		hasToolCall := false
		for {
			frame, err := output.MessageStream.Recv()
			if err == io.EOF {
				break
			}
			if err != nil {
				return messageOutput{}, err
			}
			if frame == nil {
				continue
			}
			if len(frame.ToolCalls) > 0 {
				hasToolCall = true
			}
			if frame.Content == "" {
				continue
			}
			b.WriteString(frame.Content)
			if live {
				writeSSE(c, "delta", frame.Content)
			}
		}
		if hasToolCall {
			return messageOutput{Text: b.String(), Candidate: true}, nil
		}
		if !live && b.String() != "" {
			writeSSE(c, "delta", b.String())
		}
		return messageOutput{Text: b.String()}, nil
	}
	if output.Message != nil && output.Message.Content != "" {
		if len(output.Message.ToolCalls) > 0 {
			return messageOutput{Text: output.Message.Content, Candidate: true}, nil
		}
		writeSSE(c, "delta", output.Message.Content)
		return messageOutput{Text: output.Message.Content}, nil
	}
	return messageOutput{}, nil
}

func collectMessageOutput(output *adk.MessageVariant) (messageOutput, error) {
	if output == nil {
		return messageOutput{}, nil
	}
	if output.Role != schema.Assistant {
		_, err := drainMessageOutput(output)
		return messageOutput{}, err
	}
	if output.IsStreaming && output.MessageStream != nil {
		var b strings.Builder
		hasToolCall := false
		for {
			frame, err := output.MessageStream.Recv()
			if err == io.EOF {
				break
			}
			if err != nil {
				return messageOutput{}, err
			}
			if frame != nil {
				if len(frame.ToolCalls) > 0 {
					hasToolCall = true
				}
				b.WriteString(frame.Content)
			}
		}
		if hasToolCall {
			return messageOutput{Text: b.String(), Candidate: true}, nil
		}
		return messageOutput{Text: b.String()}, nil
	}
	if output.Message != nil {
		if len(output.Message.ToolCalls) > 0 {
			return messageOutput{Text: output.Message.Content, Candidate: true}, nil
		}
		return messageOutput{Text: output.Message.Content}, nil
	}
	return messageOutput{}, nil
}

func drainMessageOutput(output *adk.MessageVariant) (string, error) {
	if output == nil || !output.IsStreaming || output.MessageStream == nil {
		return "", nil
	}
	for {
		_, err := output.MessageStream.Recv()
		if err == io.EOF {
			return "", nil
		}
		if err != nil {
			return "", err
		}
	}
}

func (a *app) scorecard(c *gin.Context) {
	card, err := a.stores.Reports.GetScorecard(c.Request.Context(), userID(c), c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, card)
}

type turnResult struct {
	CheckpointID string      `json:"checkpointId"`
	Assistant    string      `json:"assistant"`
	Events       []apiEvent  `json:"events"`
	Interrupted  interface{} `json:"interrupted,omitempty"`
}

type apiEvent struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

func (a *app) runTurn(ctx context.Context, userID string, sessionID string, content string) (turnResult, error) {
	current, err := a.stores.Sessions.GetSession(ctx, userID, sessionID)
	if err != nil {
		return turnResult{}, err
	}
	userMsg := session.Message{SessionID: sessionID, UserID: userID, Role: session.RoleUser, Content: content}
	if _, err := a.stores.Messages.AppendMessage(ctx, userMsg); err != nil {
		return turnResult{}, err
	}

	iter, checkpointID, err := a.rt.RunTurn(ctx, interviewrt.TurnRequest{UserID: userID, SessionID: sessionID, Input: content})
	if err != nil {
		return turnResult{}, err
	}

	var result turnResult
	result.CheckpointID = checkpointID
	var assistant strings.Builder
	var candidate strings.Builder
	for {
		event, ok := iter.Next()
		if !ok {
			break
		}
		if event == nil {
			continue
		}
		if event.Err != nil {
			return turnResult{}, event.Err
		}
		if event.Action != nil {
			if event.Action.CustomizedAction != nil {
				result.Events = append(result.Events, apiEvent{Type: "custom", Data: event.Action.CustomizedAction})
			}
			if event.Action.Interrupted != nil {
				result.Interrupted = event.Action.Interrupted
				result.Events = append(result.Events, apiEvent{Type: "interrupted", Data: event.Action.Interrupted})
			}
		}
		if event.Output != nil && event.Output.MessageOutput != nil {
			output, err := collectMessageOutput(event.Output.MessageOutput)
			if err != nil {
				return turnResult{}, err
			}
			if output.Text != "" {
				if output.Candidate {
					candidate.WriteString(output.Text)
				} else {
					assistant.WriteString(output.Text)
				}
			}
		}
	}
	result.Assistant = chooseAssistantText(assistant.String(), candidate.String())
	if result.Assistant != "" {
		result.Events = append(result.Events, apiEvent{Type: "assistant", Data: result.Assistant})
	}
	if strings.TrimSpace(result.Assistant) == "" && result.Interrupted == nil {
		return turnResult{}, fmt.Errorf("empty assistant response")
	}
	if result.Assistant != "" {
		assistantMsg := session.Message{SessionID: sessionID, UserID: userID, Role: session.RoleAssistant, Content: result.Assistant}
		if _, err := a.stores.Messages.AppendMessage(ctx, assistantMsg); err != nil {
			return turnResult{}, err
		}
		current.Round++
		current.Status = session.StatusRunning
		current.CheckpointID = checkpointID
		if err := a.stores.Sessions.UpdateSession(ctx, current); err != nil {
			return turnResult{}, err
		}
	}
	if result.Interrupted != nil {
		current.Status = session.StatusInterrupted
		current.CheckpointID = checkpointID
		_ = a.stores.Sessions.UpdateSession(ctx, current)
	}
	return result, nil
}

func (a *app) generateOpening(ctx context.Context, item session.InterviewSession) error {
	iter, checkpointID, err := a.rt.RunTurn(ctx, interviewrt.TurnRequest{
		UserID:    item.UserID,
		SessionID: item.ID,
		Input:     "开始面试",
	})
	if err != nil {
		return err
	}
	result, err := consumeIterator(iter)
	if err != nil {
		return err
	}
	if strings.TrimSpace(result.Assistant) == "" {
		return fmt.Errorf("empty assistant response")
	}
	assistantMsg := session.Message{SessionID: item.ID, UserID: item.UserID, Role: session.RoleAssistant, Content: result.Assistant}
	if _, err := a.stores.Messages.AppendMessage(ctx, assistantMsg); err != nil {
		return err
	}
	item.Status = session.StatusRunning
	item.CheckpointID = checkpointID
	return a.stores.Sessions.UpdateSession(ctx, item)
}

func (a *app) generateOpeningStreaming(ctx context.Context, c *gin.Context, item session.InterviewSession) (string, error) {
	iter, checkpointID, err := a.rt.RunTurn(ctx, interviewrt.TurnRequest{
		UserID:    item.UserID,
		SessionID: item.ID,
		Input:     "开始面试",
	})
	if err != nil {
		return "", err
	}

	var assistant strings.Builder
	var candidate strings.Builder
	seenToolOutput := false
	for {
		event, ok := iter.Next()
		if !ok {
			break
		}
		if event == nil {
			continue
		}
		if event.Err != nil {
			return "", event.Err
		}
		if event.Action != nil {
			if event.Action.CustomizedAction != nil {
				writeSSE(c, "custom", event.Action.CustomizedAction)
			}
			if event.Action.Interrupted != nil {
				writeSSE(c, "interrupted", event.Action.Interrupted)
			}
		}
		if event.Output == nil || event.Output.MessageOutput == nil {
			continue
		}
		if event.Output.MessageOutput.Role == schema.Tool {
			seenToolOutput = true
		}
		output, err := streamMessageOutput(c, event.Output.MessageOutput, seenToolOutput)
		if err != nil {
			return "", err
		}
		if output.Text != "" {
			if output.Candidate {
				candidate.WriteString(output.Text)
			} else {
				assistant.WriteString(output.Text)
			}
		}
	}

	assistantText := chooseAssistantText(assistant.String(), candidate.String())
	if strings.TrimSpace(assistant.String()) == "" && strings.TrimSpace(assistantText) != "" {
		writeSSE(c, "delta", assistantText)
	}
	if strings.TrimSpace(assistantText) == "" {
		return "", fmt.Errorf("empty assistant response")
	}
	if strings.TrimSpace(assistantText) != "" {
		assistantMsg := session.Message{SessionID: item.ID, UserID: item.UserID, Role: session.RoleAssistant, Content: assistantText}
		if _, err := a.stores.Messages.AppendMessage(ctx, assistantMsg); err != nil {
			return "", err
		}
	}
	item.Status = session.StatusRunning
	item.CheckpointID = checkpointID
	if err := a.stores.Sessions.UpdateSession(ctx, item); err != nil {
		return "", err
	}
	return assistantText, nil
}

func (a *app) resumeTurn(ctx context.Context, userID string, sessionID string, checkpointID string, interruptID string, content string) (turnResult, error) {
	current, err := a.stores.Sessions.GetSession(ctx, userID, sessionID)
	if err != nil {
		return turnResult{}, err
	}
	if strings.TrimSpace(content) != "" {
		userMsg := session.Message{SessionID: sessionID, UserID: userID, Role: session.RoleUser, Content: content}
		if _, err := a.stores.Messages.AppendMessage(ctx, userMsg); err != nil {
			return turnResult{}, err
		}
	}
	resumeData := map[string]any(nil)
	if interruptID != "" {
		resumeData = map[string]any{interruptID: content}
	}
	iter, usedCheckpointID, err := a.rt.ResumeTurn(ctx, interviewrt.ResumeRequest{
		UserID:       userID,
		SessionID:    sessionID,
		CheckpointID: checkpointID,
		ResumeData:   resumeData,
		SessionValues: map[string]any{
			"last_answer": content,
		},
	})
	if err != nil {
		return turnResult{}, err
	}
	result, err := consumeIterator(iter)
	if err != nil {
		return turnResult{}, err
	}
	result.CheckpointID = usedCheckpointID
	if result.Assistant != "" {
		assistantMsg := session.Message{SessionID: sessionID, UserID: userID, Role: session.RoleAssistant, Content: result.Assistant}
		if _, err := a.stores.Messages.AppendMessage(ctx, assistantMsg); err != nil {
			return turnResult{}, err
		}
		current.Round++
		current.Status = session.StatusRunning
		current.CheckpointID = usedCheckpointID
		if err := a.stores.Sessions.UpdateSession(ctx, current); err != nil {
			return turnResult{}, err
		}
	}
	return result, nil
}

func consumeIterator(iter *adk.AsyncIterator[*adk.AgentEvent]) (turnResult, error) {
	var result turnResult
	var assistant strings.Builder
	var candidate strings.Builder
	for {
		event, ok := iter.Next()
		if !ok {
			break
		}
		if event == nil {
			continue
		}
		if event.Err != nil {
			return turnResult{}, event.Err
		}
		if event.Action != nil {
			if event.Action.CustomizedAction != nil {
				result.Events = append(result.Events, apiEvent{Type: "custom", Data: event.Action.CustomizedAction})
			}
			if event.Action.Interrupted != nil {
				result.Interrupted = event.Action.Interrupted
				result.Events = append(result.Events, apiEvent{Type: "interrupted", Data: event.Action.Interrupted})
			}
		}
		if event.Output != nil && event.Output.MessageOutput != nil {
			output, err := collectMessageOutput(event.Output.MessageOutput)
			if err != nil {
				return turnResult{}, err
			}
			if output.Text != "" {
				if output.Candidate {
					candidate.WriteString(output.Text)
				} else {
					assistant.WriteString(output.Text)
				}
			}
		}
	}
	result.Assistant = chooseAssistantText(assistant.String(), candidate.String())
	if result.Assistant != "" {
		result.Events = append(result.Events, apiEvent{Type: "assistant", Data: result.Assistant})
	}
	if strings.TrimSpace(result.Assistant) == "" && result.Interrupted == nil {
		return turnResult{}, fmt.Errorf("empty assistant response")
	}
	return result, nil
}

func chooseAssistantText(primary string, candidate string) string {
	if strings.TrimSpace(primary) != "" {
		return primary
	}
	return candidate
}

func userID(c *gin.Context) string {
	value, _ := c.Get("userID")
	userID, _ := value.(string)
	return userID
}

func nonEmptyString(value string, fallback string) string {
	if strings.TrimSpace(value) != "" {
		return value
	}
	return fallback
}
