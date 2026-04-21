# Python Agent Interview Topics

## Core Areas

- Async execution: `asyncio`, task groups, cancellation, timeouts, semaphores
- Tool and model integration: structured outputs, retries, partial failures, streaming
- Orchestration: custom graphs, queues, state machines, framework tradeoffs
- State and memory: message history, retrieval boundaries, cache design, persistence
- Reliability: dead-letter handling, idempotency, fallback flows, guardrails
- Observability: tracing, logs, prompt versioning, dataset-based evals
- Deployment: workers, web services, schedulers, dependency isolation, secrets handling

## Sample Questions

1. Design a Python agent service that uses multiple tools concurrently but stays within rate limits.
2. Explain how you would debug an `asyncio`-based agent that sometimes hangs under load.
3. Compare LangGraph or similar orchestration frameworks with a custom-built workflow engine.
4. Describe how you would evaluate whether a prompt change improved the agent in production.
5. Walk through how you would structure state persistence for long-running agent tasks.

## Follow-Up Prompts

- What failure mode did you optimize for?
- How would you cap concurrency safely?
- Where would you add tracing spans?
- How would you detect silent quality regressions?
- Which component would you rewrite first if latency doubled?

## Scoring Anchors

- Strong junior: understands Python basics, can reason about API calls and simple async workflows
- Strong mid: explains concurrency and reliability tradeoffs with practical implementation detail
- Strong senior: designs robust orchestration, debugging, evaluation, and deployment strategy
- Strong staff: frames platform abstractions, operating constraints, governance, and team standards
