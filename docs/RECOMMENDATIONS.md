# Recommendations

This document captures design recommendations that were considered during
development but are not part of the current implementation scope. They are
recorded so the intent and rationale are preserved for future work.

## Sub-agent ordering — runtime "order drives transfer options"

### Goal

Match Google ADK semantics where the order in which sub- agents appear in a
parent agent's `sub_agents` list drives the LLM's `transfer_to_agent_<name>`
tool ordering and `SequentialAgent` execution order.

### What was implemented

The data model now persists a per-parent `position` for each agent:

- `agents.position` column (per-parent sequence order).
- New sub-agents append at the end of the parent's list.
- `ListAgentChildren` returns sub-agents in `position ASC` order.
- A `ReorderAgentChildren` query + service/app methods allow the UI to reorder
  sub-agents (up/down arrows and drag-and-drop).

### What was NOT implemented (recommended follow-up)

The DB → ADK runtime wiring that would actually consume this order at execution
time:

1. Build the ADK agent tree from persisted DB agents instead of the hardcoded
   demo in `agent/agent.go`.
2. For each parent agent, load its sub-agents via `ListAgentChildren` (already
   ordered by `position`) and pass them into `llmagent.Config.SubAgents` in that
   order, so the `transfer_to_agent_<name>` tools are exposed in position order
   and `SequentialAgent` executes them in that same order.

This is intentionally deferred to keep the data/UI ordering change self-contained
and independently testable.

## Existing-dev-database migration strategy

The ordering column was added directly to `migrations/0001_initial.up.sql`
rather than as a new migration file (per project instruction). Because
`golang-migrate` tracks already-applied versions, an existing development
database will NOT pick up the new column automatically — the dev DB must be
reset (dropped and recreated). Fresh databases will include the column.

## Draft: DB-driven agent loader (sketch for follow-up)

Below is an illustrative outline of the runtime wiring that should consume the
ordering once built. It is a sketch, not working code.

```go
// Build the parent LLM agent's SubAgents from the DB, preserving position order.
children, err := svc.ListAgentChildren(ctx, parent.ID) // already ORDER BY position
var subAgents []agent.Agent
for _, child := range children {
    sub := buildAgent(ctx, child) // recursive; child of a child gets its own ordered list
    subAgents = append(subAgents, sub)
}
parentLLM, err := llmagent.New(llmagent.Config{
    Name:       parent.Name,
    Model:      modelFor(parent),
    Instruction: parent.Instruction,
    SubAgents:  subAgents, // order drives transfer tool order & SequentialAgent order
})
```

Deferred decisions:

- How the root agent for a session is selected (`sessions.root_agent_id`).
- How model/credentials are mapped per agent (`model_credential_id`,
  `credential_source`) into an ADK `agent.Model`.
- Whether to build a `SequentialAgent` when a parent's mode indicates a strict
  workflow, versus an `LlmAgent` where order only affects transfer-tool order.