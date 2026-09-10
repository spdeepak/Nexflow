# Database Schema

> Requires PostgreSQL 18+ for the built-in `uuidv7()` (time-ordered UUID). If using the web service. GUI apps use sqlite.
> For PG < 18, install the `pg_uuidv7` extension or replace with `gen_random_uuid()`.

## Custom ENUM Types

| Type                | Values                                          | Used By                                      |
|---------------------|-------------------------------------------------|----------------------------------------------|
| `credential_scope`  | `app`, `user`                                   | `model_credentials.scope`, `skills.scope` |
| `agent_mode`        | `chat`, `task`, `single_turn`                   | `agents.mode`                                |
| `credential_source` | `auto`, `user`, `app`                           | `agents.credential_source`                   |
| `run_status`        | `running`, `completed`, `failed`, `interrupted` | `runs.status`                                |
| `tool_call_status`  | `ok`, `error`, `long_running`                   | `tool_calls.status`                          |
| `event_role`        | `user`, `model`, `function`                     | `events.role`                                |

## Tables

### users

End users of the application. One row per authenticated user (the app is multi-tenant: every user owns their own agents,
sessions and model keys). `external_id` holds the identity from your auth provider (OAuth sub, email, ...).

### model_credentials

API keys for LLM providers (openai, anthropic, google, ollama, ...).

Two ownership tiers:

- `scope = 'app'` — owned by the application itself, shared by all users (`user_id` is NULL). Seeded at deployment/boot.
- `scope = 'user'` — owned by a single user, who can add N keys for N models and pick which one an agent uses.

The key is NEVER stored in plaintext: `api_key_cipher` is AES-256-GCM ciphertext, `base64(nonce || ciphertext)`,
encrypted with a master key kept outside the DB (env var, secret manager or KMS). `key_version` supports rotation
without rewriting every row.

PG treats NULLs as distinct in a plain UNIQUE, so partial indexes enforce uniqueness:

- `uq_cred_user` on `(user_id, provider, model_name) WHERE scope = 'user'`
- `uq_cred_app` on `(provider, model_name) WHERE scope = 'app'`

### agents

Agent definitions that form the agent tree (`parent_agent_id` references the parent, NULL = root). Each agent is built
from an ADK `llmagent.Config` at runtime. A user chooses, per agent, which model to run and which credential to use:

- `credential_source = 'auto'` — prefer the user's own key, fall back to the app's shared key
- `credential_source = 'user'` — only the user's key
- `credential_source = 'app'` — always the app's key

`config_json` keeps the full `llmagent.Config` so the agent can be recreated for replay/multi-instance deployments.
Static skill is attached many-to-many via `agent_skills`.

### sessions

One chat thread between a user and an agent tree, mapping 1:1 to the ADK Session. `app_name` distinguishes the same user
across multiple apps; a user may have many sessions (UNIQUE on `user_id` + `app_name`). `root_agent_id` is the
agent tree this session runs against. Conversation contents live in `events`, state in `session_state`, artifacts in
`artifacts`.

### events

Every individual interaction in a session, mapping 1:1 to the ADK Event: user input, model response, tool call/response,
etc. `seq` gives the ordered position within the session. `author` is the emitting agent name (or `"user"`), `branch` is
the agent path (`"agent_1.agent_2"`) used to scope history per sub-agent. `content_json` holds the ADK message parts
(text, thought, function_call, ...), `actions_json` the EventActions (stateDelta, transferToAgent, artifactDelta).
`is_final` marks `Event.IsFinalResponse()`. `invocation_id` groups all events belonging to one run.

### session_state

The key-value session state, mapping 1:1 to the ADK State. Keys carry scope prefixes (`app:`, `user:`, `temp:`) that ADK
uses to share values within/across sessions. One row per (`session_id`, `key`).

### artifacts

Files produced during agent execution (e.g. a generated report, image or CSV), versioned per filename as in the ADK
ArtifactDelta. `storage_uri` points to the object (s3/gcs/local disk); the file body is NOT stored here.

### runs

One invocation of an agent tree: a single user input processed to a final output, matching the ADK InvocationID. A run
spans multiple events across possibly several agents, all sharing the same `invocation_id`. Captures status, error and
aggregate cost for observability/billing.

### tool_calls

Denormalized record of every tool/function invocation an agent made, for analytics (which tool, how long,
success/failure, per-run). The full payload is already in `events.content_json`; this table is the queryable projection.
`status` `'long_running'` matches ADK `Event.LongRunningToolIDs`.

### skills

Static skill (documents, FAQs, product docs, ...) that agents can draw on at runtime (RAG). Separate from ADK's
memory service, which only handles long-term conversation memory. Same ownership tiers as `model_credentials`:

- `scope = 'app'` — skill shared across all users (`user_id` is NULL)
- `scope = 'user'` — skill owned by a single user

The body is either stored inline (`content`) or externally (`storage_uri`, e.g. s3/gcs); for RAG a pgvector `embedding`
column can be added and indexed. Attach skill to agents via `agent_skills`.

### agent_skills

Join table: which skill is attached to which agent (many-to-many). At runtime an agent retrieves only the documents
linked here, so the same doc can be reused across agents without duplication.

### mcp_server

MCP (Model Context Protocol) server configurations. Each row describes a remote or local MCP server that agents can
connect to for external tool capabilities (API calls, file system access, database queries, etc.).

Two transport modes:

- `streamable_http` — connects to a remote MCP server over HTTP (JSON-RPC). Requires `endpoint` (URL).
- `stdio` — launches a local MCP server as a subprocess. Requires `command` (binary path) and optionally `args`.

Authentication is optional and stored as a JSON blob (`auth_config`) resolved at runtime by `auth_type` (e.g. `api_key`,
`oauth`). The `allowed_tools` JSON array restricts which tools from the server are exposed to the agent; NULL means all
discovered tools are available.

HITL (Human-in-the-Loop) confirmation is controlled by two columns:

- `require_confirmation` — static boolean flag; when true, every tool call from this MCP server requires user approval.
- `confirmation_rules` — per-tool rules as a JSON object with a `rules` array and a `default` boolean. Each rule matches
  on `toolName` and optionally `matchArgs` (wildcard patterns). Evaluated at runtime to build an ADK
  `RequireConfirmationProvider` function. Takes precedence over the static flag when present.

### agent_mcp_server

Join table: which MCP server is attached to which agent (many-to-many). At runtime the runner builds an `mcptoolset.Config`
for each linked MCP server and passes them as `Toolsets` on the agent's `llmagent.Config`. MCPs are scoped per-agent to
limit tool surface area, reduce token usage, and enforce least-privilege access.