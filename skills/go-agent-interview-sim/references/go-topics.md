# Go Agent Interview Topics

## Core Areas

- Concurrency primitives: goroutines, channels, `context.Context`, cancellation propagation
- Networked agent execution: HTTP clients, connection reuse, request deadlines, retries
- Tool execution: schema validation, typed request/response models, sandboxing boundaries
- Workflow control: queues, worker pools, fan-out/fan-in, backpressure
- State and memory: transient run state, checkpoints, persistent stores
- Reliability: idempotency, deduplication, poison jobs, circuit breakers
- Observability: structured logs, metrics, traces, replay support
- Evaluation: offline eval sets, regression suites, failure tagging, latency and cost budgets

## Sample Questions

1. Design a Go service that executes multi-step agents with tool calling and strict per-step timeouts.
2. Explain how you would prevent goroutine leaks in an agent runtime that calls slow external tools.
3. Compare channels and explicit workflow state machines for agent orchestration.
4. Describe how you would persist agent run state so a crashed worker can resume safely.
5. Walk through how you would instrument an agent platform for debugging bad tool selections.

## Follow-Up Prompts

- What fails first at 10x traffic?
- Where would you enforce cancellation?
- How would you make the run replayable?
- What metrics would tell you the planner is degrading?
- Which parts must be idempotent?

## Scoring Anchors

- Strong junior: knows Go basics, can reason about context cancellation and HTTP calls, needs help on production design
- Strong mid: connects concurrency choices to reliability, debugging, and throughput tradeoffs
- Strong senior: designs robust orchestration, failure recovery, evaluation, and observability strategy
- Strong staff: explains platform boundaries, operating model, cost and safety controls, and team-level standards
