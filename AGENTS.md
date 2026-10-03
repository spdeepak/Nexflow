# Nexflow — Agent Engineering Instructions

## 1. Project Overview

Nexflow is a local-first, Wails v2 desktop application for designing, managing, and running hierarchical AI-agent
systems.

The application is primarily a Go application with a vanilla HTML/CSS/JavaScript frontend.

Core runtime responsibilities include:

- Hierarchical AI agents
- Google Agent Development Kit (ADK) orchestration
- Agent-to-agent delegation through ADK `SubAgents`
- Model credential management
- Static agent skills
- MCP server configuration and toolsets
- MCP OAuth authentication
- Human-in-the-loop tool confirmation
- Persistent chat sessions
- ADK event persistence
- Run history and status
- SQLite-backed desktop persistence
- Generated type-safe SQL access through sqlc
- Generated schema/types and mocks
- Wails bindings between Go and the frontend

The application is intentionally local-first. Avoid introducing external services or infrastructure unless the task
explicitly requires them.

---

# 2. Technology Stack

Use the versions already declared by the repository.

| Area             | Technology                                     |
|------------------|------------------------------------------------|
| Language         | Go 1.26.6                                      |
| Desktop          | Wails v2                                       |
| Frontend         | Vanilla HTML/CSS/JavaScript                    |
| AI runtime       | Google ADK v2                                  |
| LLM abstraction  | ADK model interfaces / OpenAI-compatible model |
| MCP              | `github.com/modelcontextprotocol/go-sdk`       |
| Database         | SQLite                                         |
| SQLite driver    | `modernc.org/sqlite`                           |
| Migrations       | `golang-migrate`                               |
| SQL generation   | sqlc                                           |
| Mock generation  | mockery                                        |
| Validation       | `go-playground/validator`                      |
| Configuration    | Viper/config files                             |
| IDs              | UUID                                           |
| Testing          | Go `testing` + testify where already used      |
| Desktop bindings | Wails generated bindings                       |

Do not upgrade dependencies as part of an unrelated feature.

Do not introduce a new dependency merely because it makes a small piece of code more convenient.

Prefer the Go standard library when it provides a reasonable solution.

---

# 3. Source-of-Truth Rule

Before modifying code:

1. Read the relevant existing implementation.
2. Read the relevant SQL queries and schema.
3. Read the relevant generated types/interfaces if they affect the change.
4. Read the relevant architecture documentation.
5. Search all call sites before changing an exported function, interface, database column, or persisted representation.
6. Follow existing patterns before introducing a new abstraction.

Do not assume the README describes every current implementation detail. The code, SQL schema, Makefile, and architecture
documents are authoritative.

Important architecture documentation:

- `internal/runner/FLOW.md`
- `migrations/README.md`
- `docs/RECOMMENDATIONS.md`

---

# 4. Repository Structure

Important directories:

```text
cmd/
  app/
    main.go
    frontend/
  server/
  mock_oauth/

internal/
  application/       # Wails application bindings
  agents/            # Agent definitions and agent tree operations
  agentskills/       # Skill management
  agentmcp/          # Agent ↔ MCP relationships
  mcpstore/          # MCP OAuth/token infrastructure
  modelcredentials/  # LLM credential management
  runner/             # ADK orchestration
  sessions/           # ADK session/event persistence
  runs/               # Run persistence
  sessionstate/       # Session state persistence
  artifacts/          # Artifact persistence
  enums/              # Domain enums
  schema/             # Generated/domain schema types
  errors/             # Application errors
  util/               # Existing utility code

migrations/            # Database migrations and schema documentation
sqls/                  # SQL consumed by sqlc
configs/               # Runtime configuration
docs/                  # Architecture/design documentation
Makefile               # Canonical developer commands
```

Keep new code in the domain package responsible for the behavior.

Do not create generic packages such as:

```text
internal/common/
internal/helpers/
internal/managers/
internal/utils/
```

unless there is a demonstrated existing convention that requires it.

---

# 5. Go Engineering Rules

Write idiomatic Go.

## Prefer

- Small functions
- Explicit control flow
- Early returns
- Explicit error handling
- Small interfaces
- Interfaces defined by consumers
- Concrete implementations by default
- `context.Context` for I/O and long-running operations
- Table-driven tests
- Clear domain-oriented packages
- Standard-library solutions where practical

## Avoid

- Java/Spring-style abstractions
- Generic repository/service layers
- Interfaces with many methods
- Global mutable state
- Unnecessary generics
- `any` when a concrete type is possible
- Reflection unless necessary
- Hidden goroutines
- `time.Sleep` as synchronization
- Swallowing errors
- Large god functions
- Premature abstraction

Every goroutine must have a clear ownership and shutdown mechanism.

Every external operation should have appropriate context propagation.

Wrap errors with useful context:

```go
return fmt.Errorf("failed to load agent %s: %w", id, err)
```

Do not wrap errors merely to add noise.

---

# 6. Dependency Policy

Do not add a new dependency without first checking whether:

1. The standard library already solves the problem.
2. An existing dependency already provides the functionality.
3. The dependency is necessary for the architecture rather than merely convenient.

If a dependency is genuinely necessary, explain why it is needed and why an existing dependency or standard-library
implementation is insufficient.

Never modify `go.mod` manually when the Go toolchain can make the change safely.

---

# 7. Database Rules

Nexflow uses SQLite for the desktop application.

The database is persisted locally and contains important user-owned state.

Database changes must be treated as API changes.

Before changing database behavior:

1. Inspect the existing migration.
2. Inspect affected SQL queries.
3. Inspect sqlc configuration.
4. Inspect generated types/interfaces.
5. Determine whether existing development databases need migration/reset handling.
6. Update migration/schema documentation when appropriate.
7. Regenerate generated code.

Never change only the generated sqlc code.

Generated files are outputs, not sources of truth.

---

# 8. SQL / sqlc Rules

SQL lives under:

```text
sqls/
```

Generated sqlc code must not be edited manually.

When changing SQL:

```bash
make generate
```

This regenerates the generated code.

Pay particular attention to:

- UUID handling
- nullable columns
- JSON columns
- enum mappings
- `database/sql` nullable types
- ownership/scope filtering
- ordering semantics
- foreign-key relationships
- user isolation

Queries involving user-owned resources must preserve the application's ownership model.

Do not accidentally expose app-scoped resources as user-owned or vice versa.

---

# 9. Generated Code

The repository uses generated code.

The canonical generation command is:

```bash
make generate
```

This regenerates:

- sqlc code
- mockery mocks
- other generated Go artifacts used by the project

Do not manually edit generated files.

After changing source schemas, SQL, interfaces, or generation inputs, regenerate and inspect the diff.

Do not commit unrelated generated changes.

---

# 10. Agent Architecture

The agent runtime is built around Google ADK.

The important flow is:

```text
Wails frontend
    ↓
application bindings
    ↓
runner
    ↓
agent service
    ↓
load root agent
    ↓
load sub-agents
    ↓
resolve model credentials
    ↓
resolve skills
    ↓
resolve MCP toolsets
    ↓
construct ADK agents
    ↓
root agent + SubAgents
    ↓
ADK Runner
    ↓
ADK events
    ↓
session/event persistence
    ↓
Wails event stream
    ↓
frontend
```

See:

```text
internal/runner/FLOW.md
```

before changing chat/run behavior.

---

# 11. Google ADK Rules

ADK is the runtime authority for agent execution.

Do not reimplement ADK functionality unless there is a demonstrated limitation requiring it.

In particular:

- Use ADK `SubAgents` for agent delegation.
- Use ADK toolsets for tools.
- Use ADK session semantics for session state.
- Preserve ADK event semantics.
- Preserve ADK invocation IDs.
- Preserve ADK final-response semantics.
- Use ADK callbacks for lifecycle behavior where already established.
- Preserve ADK HITL confirmation behavior.

A sub-agent's final response is not necessarily the final user response.

The root agent owns the final response in the current orchestration model.

Do not change this behavior accidentally.

---

# 12. Agent Tree Rules

Agents are persisted as a tree:

```text
root agent
├── sub-agent
├── sub-agent
└── sub-agent
```

`parent_agent_id` defines the relationship.

Child ordering is persisted and is significant.

The order returned from:

```text
GetSubAgents
ListAgentChildren
```

must remain deterministic.

Do not replace ordered queries with unordered queries.

When constructing ADK `SubAgents`, preserve the persisted order.

Do not silently sort agents alphabetically unless that is explicitly required.

---

# 13. Agent Configuration

An agent can have:

- name
- description
- instruction
- global instruction
- mode
- model
- model credential
- credential source
- model generation configuration
- skills
- MCP servers
- sub-agents

Do not duplicate agent configuration in multiple persistence layers.

Persisted agent configuration is the source from which the runtime agent should be reconstructed.

---

# 14. Model Credential Rules

Credentials can be:

```text
app-scoped
user-scoped
```

The credential source can determine whether the runner may use:

```text
auto
user
app
```

Follow the existing credential-resolution rules.

Explicitly selected credentials must not be silently replaced unless the current fallback behavior explicitly permits
it.

Never log:

- API keys
- bearer tokens
- refresh tokens
- client secrets
- decrypted credential payloads

Be particularly careful with:

```text
slog
fmt.Printf
JSON serialization
errors
debug logging
```

Do not log credential-containing structs.

---

# 15. Skills

Skills are ADK toolsets and are attached to agents.

Skills may be:

- stored inline
- stored through a storage URI
- represented as static instructions/documents

The runner resolves skills into an ADK skill toolset.

When modifying skill behavior:

1. Inspect `internal/agentskills`.
2. Inspect the database schema.
3. Inspect the agent-to-skill relationship.
4. Inspect `runner.resolveSkills`.
5. Verify how the skill is exposed to ADK.
6. Test both inline and filesystem/storage-backed skills where applicable.

Do not confuse:

```text
Nexflow persisted skills
```

with:

```text
coding-agent SKILL.md files
```

They are related concepts but serve different runtime purposes.

---

# 16. MCP Architecture

MCP servers are attached to agents through a many-to-many relationship.

Supported transports include:

```text
streamable_http
stdio
```

The runtime builds ADK MCP toolsets from the persisted configuration.

MCP configuration can include:

- endpoint
- command
- arguments
- authentication
- OAuth
- allowed tools
- confirmation requirements
- confirmation rules

When modifying MCP behavior, inspect:

```text
internal/agentmcp/
internal/mcpstore/
internal/runner/
migrations/
sqls/
```

Do not implement a second MCP abstraction if ADK's existing MCP toolset can provide the required behavior.

---

# 17. MCP Security

Treat MCP servers as potentially privileged integrations.

An MCP server may provide access to:

- files
- databases
- APIs
- network resources
- external accounts
- destructive operations

Therefore:

- Preserve `allowed_tools`.
- Preserve HITL confirmation.
- Do not automatically expose newly discovered tools.
- Do not broaden tool access implicitly.
- Do not bypass confirmation rules.
- Validate persisted MCP configuration.
- Treat command paths and arguments as security-sensitive.
- Treat endpoints as security-sensitive.
- Never log authentication secrets.

For stdio MCP servers, remember that:

```go
exec.Command(...)
```

launches an external process.

Do not construct shell commands unnecessarily.

Do not change stdio execution to:

```text
sh -c
bash -c
zsh -c
```

unless there is a compelling, reviewed requirement.

---

# 18. MCP Tool Allow-Lists

`allowed_tools` is a security and token-budget boundary.

If it is configured, only the explicitly permitted tools should be exposed to the agent.

Do not silently interpret an empty/non-empty allow-list differently from the established schema semantics.

When changing tool filtering, add tests for:

- allowed tool
- disallowed tool
- multiple allowed tools
- unknown tool
- empty allow-list
- nil/null configuration

---

# 19. HITL / Tool Confirmation

Human-in-the-loop behavior is a first-class feature.

A tool call may cause a run to enter:

```text
interrupted
```

state and later resume after the user confirms or rejects the operation.

Do not convert an interrupted run into a failed run.

Do not lose the pending tool-call information.

Do not execute a confirmation-required tool twice.

When modifying HITL behavior, test:

```text
normal tool call
confirmation requested
user approves
user rejects
run resumes
run remains interrupted
```

---

# 20. Runner Rules

`internal/runner/runner.go` is the central orchestration component.

Changes here require extra care because it connects:

```text
agents
models
skills
MCP
ADK
sessions
events
HITL
Wails
```

Avoid adding unrelated responsibilities to the runner.

If a new behavior can be isolated into an existing domain package, do so.

When changing runner behavior, inspect:

```text
internal/runner/FLOW.md
```

and update it if the execution flow materially changes.

---

# 21. Event Streaming

ADK events are persisted and streamed to the frontend.

Do not assume:

```text
one ADK event = one user-visible response
```

Events may represent:

- user input
- model output
- partial output
- tool calls
- tool responses
- sub-agent activity
- state changes
- final responses
- HITL interruptions

Preserve event ordering.

Preserve invocation IDs.

Preserve session IDs.

Do not introduce asynchronous processing that can reorder persisted events unless ordering is explicitly designed and
tested.

---

# 22. Session Persistence

Sessions map application chat sessions to ADK sessions.

A session can contain:

- events
- session state
- artifacts
- runs

Do not delete or overwrite historical events merely to simplify a new feature.

Treat historical session data as immutable wherever practical.

---

# 23. Wails Rules

This project uses Wails **v2**.

Do not use Wails v3 APIs.

Do not mix Wails v2 and v3 examples.

The Wails entry point is:

```text
cmd/app/main.go
```

The frontend is:

```text
cmd/app/frontend/
```

Wails bindings are generated.

Do not manually edit generated Wails binding files.

If a Go method exposed to the frontend changes:

1. Update the Go implementation.
2. Regenerate/rebuild Wails bindings as appropriate.
3. Update frontend callers.
4. Verify the application builds.

---

# 24. Frontend Rules

The frontend intentionally uses vanilla HTML/CSS/JavaScript.

Do not introduce React, Vue, Svelte, Angular, or another frontend framework unless explicitly requested.

Prefer the existing frontend patterns.

Avoid introducing a frontend build system for a small UI feature when the current architecture can support the change
directly.

Keep business logic in Go when it belongs to the backend/domain.

Keep presentation state in JavaScript when it belongs to the UI.

---

# 25. Wails Development

Canonical development command:

```bash
make wails-dev
```

This command resets the local app database and generated Wails frontend artifacts before launching Wails development
mode.

Therefore:

**Do not use `make wails-dev` if you need to preserve the existing local development database.**

For a normal development reset, it is the canonical command.

For GoLand debugging, use the existing `wails-app` configuration with:

```text
-tags production
CGO_LDFLAGS=-framework UniformTypeIdentifiers
```

Do not invent a second debugging configuration unless necessary.

---

# 26. Wails Build

Canonical packaged build:

```bash
make wails-build
```

This produces:

```text
cmd/app/build/bin/Nexflow.app
```

The Makefile performs the required macOS Wails build setup and ad-hoc codesigning.

To launch the packaged binary directly on macOS:

```bash
make wails-run
```

Do not claim that a build is successful without actually running the build command or equivalent verification.

---

# 27. Backend Commands

Use the Makefile rather than inventing custom commands.

Build:

```bash
make build
```

Run server:

```bash
make run
```

Lint:

```bash
make lint
```

Test:

```bash
make test
```

Backend lint:

```bash
make lint-backend
```

Backend tests:

```bash
make test-backend
```

Coverage:

```bash
make test-backend-coverage
```

Generation:

```bash
make generate
```

Complete dependency/generation refresh:

```bash
make complete
```

---

# 28. Testing Requirements

Every behavioral change should have tests.

At minimum:

- New business logic → unit tests
- New SQL behavior → SQL/service tests
- New runner behavior → runner tests
- New MCP behavior → MCP/runner tests
- New skill behavior → skill/runner tests
- New Wails binding behavior → application-level tests where practical
- New persistence behavior → SQLite integration test where appropriate

Use the testing pyramid:

```text
many small unit tests
        ↓
service/orchestration tests
        ↓
fewer SQLite integration tests
        ↓
small number of desktop/manual tests
```

Do not rely exclusively on manual UI testing.

---

# 29. Test Doubles

The repository already uses interfaces and generated mocks.

Use the existing testing strategy.

For unit tests:

- mock sqlc queriers where appropriate
- use small fakes when they are clearer
- isolate external systems
- do not require a running LLM provider

For integration tests:

- use a real SQLite database
- apply the relevant migrations
- test actual SQL behavior

Do not make tests depend on:

- real OpenAI credentials
- real Google credentials
- real Anthropic credentials
- real Ollama
- external MCP servers
- network availability

unless the test is explicitly an integration/e2e test and is clearly separated from normal CI tests.

---

# 30. TDD / Implementation Discipline

For meaningful feature work or bug fixes:

1. Understand the existing behavior.
2. Add or update a test that captures the desired behavior.
3. Run the test and confirm the relevant failure.
4. Implement the smallest change.
5. Run the focused tests.
6. Run broader tests.
7. Run lint/build checks.
8. Inspect the final diff.

Do not write large amounts of implementation code and only then discover how it should be tested.

For trivial mechanical changes, use judgment.

---

# 31. Validation

Agent input and persisted configuration must be validated.

The project uses:

```text
github.com/go-playground/validator/v10
```

When adding dependent fields, follow the existing validation style.

For example, if a field is required only for a particular enum value, validation should express that relationship rather
than relying solely on runtime assumptions.

Do not duplicate validation rules across frontend and backend when the backend can enforce the invariant
authoritatively.

Frontend validation may still be used for user experience.

---

# 32. Configuration

Configuration lives under:

```text
configs/
```

Do not put secrets into source-controlled configuration.

Never add real:

- API keys
- OAuth tokens
- refresh tokens
- client secrets
- passwords

to tracked files.

Use the existing configuration/secrets mechanisms.

Be careful when logging configuration objects because configuration may contain credentials.

---

# 33. File and Process Safety

Nexflow can interact with the local filesystem and launch MCP stdio processes.

Treat user-controlled paths and commands as security-sensitive.

When handling:

- file paths
- storage URIs
- MCP commands
- MCP arguments
- downloaded files
- skill directories

validate and normalize inputs appropriately.

Do not introduce shell injection vulnerabilities.

Prefer direct process execution with argument arrays over shell command construction.

---

# 34. Logging

Use the existing `log/slog` approach.

Logs should answer:

```text
what happened
where it happened
which resource was involved
what failed
```

Use structured fields.

Do not log secrets.

Do not log complete credential structs.

Avoid excessive debug logging in hot paths.

Do not leave temporary debugging statements in production code.

---

# 35. Error Handling

Errors should contain useful operation context.

Prefer:

```go
return fmt.Errorf("failed to create MCP toolset for %s: %w", server.Name, err)
```

over:

```go
return err
```

when additional context materially helps diagnosis.

Do not expose sensitive implementation details to end users merely because they are useful in logs.

Keep internal error context separate from user-facing messages where appropriate.

---

# 36. Concurrency

Agent execution and event streaming can be concurrent.

Before adding goroutines:

- identify ownership
- identify cancellation
- identify shutdown
- identify synchronization
- determine whether event ordering matters

Never use `time.Sleep` as synchronization.

Use:

- contexts
- channels
- explicit synchronization
- deterministic test mechanisms

where appropriate.

---

# 37. Changes to Public/Shared Interfaces

Before changing an interface:

```text
grep/search all implementations
grep/search all consumers
grep/search all generated mocks
```

Then update:

1. interface
2. implementation
3. mocks
4. tests
5. callers

Run:

```bash
make generate
```

if mocks are generated.

Do not leave partially updated interfaces.

---

# 38. Generated Wails / SQL / Mock Files

Never hand-edit:

- generated sqlc files
- generated mockery files
- generated Wails bindings
- other generated artifacts

Modify the source and regenerate.

If generated output unexpectedly changes a large number of unrelated files, stop and inspect the generation inputs
rather than blindly committing the diff.

---

# 39. Documentation

Update documentation when behavior or architecture changes.

Especially update:

```text
internal/runner/FLOW.md
migrations/README.md
docs/RECOMMENDATIONS.md
README.md
```

when their documented behavior becomes incorrect.

Do not add documentation that merely repeats obvious code.

Architecture documentation should explain decisions and system behavior, not implementation trivia.

---

# 40. Git Hygiene

Keep changes focused.

Do not:

- reformat unrelated files
- update dependencies unnecessarily
- rename unrelated symbols
- rewrite working architecture
- commit generated artifacts unrelated to the task
- mix refactoring with feature work unless necessary

Before finishing:

```bash
git diff --stat
git diff
git status
```

Review the complete diff.

Remove accidental files and unrelated changes.

---

# 41. Definition of Done

A change is not complete merely because the code compiles.

Before declaring completion, verify as applicable:

```bash
gofmt
make generate
make lint-backend
make test-backend
go build ./...
```

For desktop changes:

```bash
make wails-build
```

For UI development:

```bash
make wails-dev
```

Use the smallest appropriate verification set during iteration, then run the broader relevant checks before completion.

If a check cannot be run, explicitly state which check was not run and why.

Never claim that tests, builds, or lint passed without actually running them.

---

# 42. Important Existing Runtime Semantics

Preserve these behaviors unless the task explicitly changes them:

### Root agent

The root agent is the orchestrator and ultimately produces the final user response.

### Sub-agents

ADK handles delegation through the root agent's `SubAgents`.

### Skills

Skills are converted into ADK skill toolsets and attached to the corresponding agent.

### MCP

MCP servers are converted into ADK MCP toolsets and attached to the corresponding agent.

### Tool restrictions

MCP `allowed_tools` restricts the tool surface exposed to the agent.

### HITL

Confirmation-required MCP tools can interrupt a run and resume later.

### Events

ADK events are persisted and streamed to the frontend.

### Runs

A run corresponds to an invocation and has an explicit lifecycle/status.

### Credentials

Model credentials are resolved according to explicit credential selection and credential-source rules.

Do not change these semantics accidentally while modifying adjacent code.

---

# 43. When Working on a Feature

For a feature spanning multiple layers, follow this order:

```text
1. Understand existing architecture
2. Define persisted/data model changes
3. Update migrations if necessary
4. Update SQL
5. Regenerate sqlc/mocks
6. Update domain/service logic
7. Update runner/orchestration
8. Update Wails bindings
9. Update frontend
10. Add/update tests
11. Run generation
12. Run lint
13. Run tests
14. Build
15. Review diff
```

Not every feature needs every step.

Do not modify the frontend first when the authoritative behavior belongs in the backend.

---

# 44. Preferred Engineering Style

Optimize for:

```text
correctness
clarity
testability
security
small changes
maintainability
```

over:

```text
cleverness
abstraction
maximum code reuse
minimum line count
premature optimization
```

The best implementation is usually the smallest implementation that fits the existing architecture cleanly.

When two approaches are functionally equivalent, prefer the one that:

1. uses fewer abstractions,
2. uses existing project patterns,
3. has fewer dependencies,
4. is easier to test,
5. makes failure behavior explicit.

---

# 45. Final Rule

Before modifying Nexflow, understand how the change travels through:

```text
UI
→ Wails binding
→ application service
→ domain service
→ persistence
→ runner
→ ADK
→ model / skill / MCP
→ events
→ persistence
→ UI
```

A change that looks local may affect several of these boundaries.

Inspect the complete path before making architectural changes.