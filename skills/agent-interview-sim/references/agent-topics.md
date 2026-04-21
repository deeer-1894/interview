# Agent Interview Topics

## Core Areas

- Planning loops, step decomposition, and stopping criteria
- Tool selection, schema design, and error handling
- Memory layers: short-term context, long-term memory, retrieval, and summaries
- Workflow orchestration: loops, graphs, queues, supervisors, and checkpoints
- Evaluation: golden sets, simulation, human review, regression tracking
- Observability: traces, event logs, replay, incident timelines
- Safety: permissioning, policy checks, prompt injection resistance, budget limits
- Production operations: deployment, scaling, tenancy, rollouts, and fallback behavior

## Sample Questions

1. Design an agent architecture for a task that requires planning, tool use, and user clarification.
2. Explain how you would evaluate whether an agent became better after changing both the model and prompts.
3. Compare loop-based agents with graph-based orchestration in terms of control and reliability.
4. Describe how you would investigate a case where the agent appears successful but users are dissatisfied.
5. Walk through how you would enforce safety and cost limits in a multi-tenant agent platform.

## Follow-Up Prompts

- What would you log at each step?
- How would you replay the bad run?
- What would your rollback strategy be?
- How would you separate model failure from tool failure?
- Which metric can be gamed, and how would you defend against that?

## Scoring Anchors

- Strong junior: understands basic agent loop concepts and can describe tools and memory at a high level
- Strong mid: reasons about workflow, tradeoffs, and evaluation with practical detail
- Strong senior: designs robust agent systems with clear observability and recovery mechanisms
- Strong staff: explains platform-level standards, governance, rollout strategy, and operating model
