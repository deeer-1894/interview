package runtime

import (
	"context"

	"github.com/cloudwego/eino/adk"

	"mockinterview/internal/protocol"
)

type RunContext struct {
	Context         context.Context
	Request         protocol.RunRequest
	Prompt          string
	PromptVersion   string
	Task            protocol.Task
	Run             protocol.Run
	Messages        []protocol.Message
	Result          RunResultContext
	Resolved        RunResolvedContext
	Executor        Executor
	Recorder        EventRecorder
	Callbacks       []Callback
	Tools           ToolGateway
	ADKCheckpoints  adk.CheckPointStore
	ClarifyRequests ClarifyRepository
	MiddlewareChain []string
	MiddlewareTrace []string
	middlewareSpecs map[string]MiddlewareSpec
}

type RunResultContext struct {
	Output  string
	Summary string
}

type RunResolvedContext struct {
	Interview RunInterviewContext
	Execution RunExecutionContext
}

type RunInterviewContext struct {
	Skill      protocol.SkillSpec
	Rubric     protocol.Rubric
	Profile    protocol.CandidateProfile
	Checkpoint protocol.CheckpointSnapshot
}

type RunExecutionContext struct {
	Plan       protocol.ExecutionPlan
	Memory     []protocol.MemoryRecord
	Artifacts  []protocol.Artifact
	WebResults []protocol.WebSearchResult
}

func (c *RunContext) ConfigureMiddlewareChain(specs []MiddlewareSpec) {
	c.MiddlewareChain = make([]string, 0, len(specs))
	c.MiddlewareTrace = nil
	c.middlewareSpecs = make(map[string]MiddlewareSpec, len(specs))
	for _, spec := range specs {
		spec = normalizeMiddlewareSpec(spec)
		c.MiddlewareChain = append(c.MiddlewareChain, spec.Name)
		c.middlewareSpecs[spec.Name] = spec
	}
}

func (c *RunContext) MarkMiddlewareCompleted(name string) {
	name = normalizeMiddlewareName(name)
	if name == "" {
		return
	}
	c.MiddlewareTrace = append(c.MiddlewareTrace, name)
}

func (c *RunContext) MiddlewareSpec(name string) MiddlewareSpec {
	if c == nil || len(c.middlewareSpecs) == 0 {
		return MiddlewareSpec{}
	}
	return c.middlewareSpecs[normalizeMiddlewareName(name)]
}
