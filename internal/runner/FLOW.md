# Chat run flow

The Wails desktop app exposes Go methods to the frontend. Chat is driven through
the `application` package (bindings) on top of the `runner` package, which
orchestrates ADK over the agent tree.

- Wails entrypoint: `cmd/app/main.go`
- App bindings: `../application`, `../application`
- Chat service + persistence orchestration: `internal/runner/chat.go`
- ADK orchestration: `internal/runner/runner.go`
- ADK session store (event persistence): `internal/sessions/service.go`

## 1. Click "Chat" (sidebar)

**Frontend-only navigation.** Once the chat view mounts, the frontend calls:

- **`App.ListSessions()`** — `../application` → `chat.Service.ListSessions()` (`internal/runner/chat.go:155`) → SQL query for sessions
- **`App.GetAgents()`** — `../application` → `agents.Service.ListRootAgents()` → SQL query for root agents

## 2. Click "New Chat"

**Frontend-only state change** — shows the agent selection UI. No backend call.

## 3. Select an agent

**Frontend-only** — agent list was already fetched by `GetAgents()`.

## 4. Click "Start Chat"

**`App.CreateSession(rootAgentID)`** — `../application`

```
App.CreateSession()
  → currentUser()                          // application/chat.go:57
    → userService.GetUserByExternalID()    // resolves user from deviceID
  → chatService.CreateSession()            // internal/runner/chat.go:135
    → uuid.NewV7()
    → sessionsQuery.CreateSession()        // SQL: INSERT into sessions
  → toSessionDTO()
```

## 5. Type message → Click "Send"

**`App.SendMessageSession(sessionID, message)`** — `../application`

This is the most complex flow:

```
App.SendMessageSession()
  → chatService.GetSession()               // SQL: SELECT from sessions
  → chatService.Run(..., runner.Options{   // internal/runner/chat.go:69
        Emitter: a.emit,                   // Wails runtime.EventsEmit
     })
    → runsQuery.CreateRun()                // SQL: INSERT into runs
    → invocation.ContextWithInvocation()   // tags all events with the run ID
    → orch.Run(Request{...}, onEvent)      // internal/runner/runner.go:85
      → agentService.GetAgent()            // SQL: SELECT root agent
      → agentService.ListAgentChildren()   // SQL: SELECT sub-agents
      → buildLLMAgent() x sub-agents       // runner.go:200
        → resolveModel() / getModelCredential()  // SQL: SELECT model_credentials
        → llmagent.New() with UIHook before/after callbacks
      → buildLLMAgentWithSubAgents()       // runner.go:230 — wires SubAgents + root hooks
      → runStage()                         // runner.go:132
        → adkRunner.Run()                  // ADK delegates to sub-agents autonomously
        → per event:
            - sessions.SessionStore.AppendEvent()
                                           // SQL: INSERT events + session_state
            - onEvent(StageEvent{...})     // runner.go:169
              → emit("chat:event", RunEvent)  // Wails runtime.EventsEmit
            - synthetic root final event   // runner.go:183 (UI marks root done)
    → runsQuery.UpdateRunStatus()          // SQL: UPDATE runs
```

Lifecycle ("Agent started/finished execution") is signalled by ADK's
`BeforeAgentCallbacks`/`AfterAgentCallbacks` installed via `UIHook`
(`internal/runner/runner.go:264`); it no longer relies on the event stream.

### Key files to target for tests

| Layer | File | Methods to test |
|---|---|---|
| **App bindings** | `../application` | `CreateSession`, `SendMessageSession`, `ListSessions`, `ListEvents`, `GetRun` |
| **Chat service** | `internal/runner/chat.go` | `Run`, `CreateSession`, `GetSession`, `ListSessions`, `DeleteSession`, `ListEvents` |
| **Session store** | `internal/sessions/service.go` | `AppendEvent`, `Create`, `Get` |
| **Runner** | `internal/runner/runner.go` | `Run`, `runStage`, `buildLLMAgent`, `buildLLMAgentWithSubAgents`, `resolveModel` |
| **Agents service** | `internal/agents/service.go` | `GetAgent`, `ListAgentChildren`, `ListRootAgents` |

For unit tests, mock the sqlc queriers (they're interfaces) and test the
service/orchestration layers in isolation. For integration tests, use a real
SQLite DB.