CREATE TABLE device
(
    id UUID NOT NULL PRIMARY KEY
);

CREATE TRIGGER device_one_row_check
    BEFORE INSERT ON device
    WHEN (SELECT COUNT(*) FROM device) >= 1
BEGIN
    SELECT RAISE(ABORT, 'Only one row is allowed in the device table');
END;

CREATE TABLE users
(
    id          UUID        NOT NULL PRIMARY KEY,
    external_id TEXT UNIQUE NOT NULL,
    name        TEXT        NOT NULL,
    created_at  DATETIME    NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE model_credentials
(
    id             UUID             NOT NULL PRIMARY KEY,
    title          VARCHAR(20)      NOT NULL DEFAULT '',
    user_id        UUID REFERENCES users (id) ON DELETE CASCADE,
    provider       TEXT             NOT NULL,
    model_name     TEXT             NOT NULL,
    base_url       TEXT,
    scope          credential_scope NOT NULL DEFAULT 'app',
    api_key_cipher TEXT,
    key_version    INTEGER          NOT NULL DEFAULT 1,
    is_active      BOOLEAN          NOT NULL DEFAULT 1,
    created_at     DATETIME         NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_by     UUID REFERENCES users (id),
    updated_at     DATETIME         NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT model_credentials_owner_check CHECK (
        (scope = 'app' AND user_id IS NULL) OR
        (scope = 'user' AND user_id IS NOT NULL)
        ),
    CONSTRAINT model_credentials_title_length CHECK (length(title) <= 20)
);
CREATE UNIQUE INDEX uq_cred_user ON model_credentials (user_id, provider, model_name) WHERE scope = 'user';
CREATE UNIQUE INDEX uq_cred_app ON model_credentials (provider, model_name) WHERE scope = 'app';

CREATE TABLE agents
(
    id                  UUID              NOT NULL PRIMARY KEY,
    name                TEXT              NOT NULL,
    parent_agent_id     UUID REFERENCES agents (id) ON DELETE CASCADE,
    description         TEXT,
    instruction         TEXT,
    global_instruction  TEXT,
    mode                agent_mode        NOT NULL DEFAULT 'chat',
    model_name          TEXT,
    model_credential_id UUID REFERENCES model_credentials (id) ON DELETE RESTRICT,
    credential_source   credential_source NOT NULL DEFAULT 'auto',
    model_config        JSONB             NOT NULL DEFAULT '{}',
    config_json         JSONB             NOT NULL DEFAULT '{}',
    is_active           BOOLEAN           NOT NULL DEFAULT 1,
    position            INTEGER           NOT NULL DEFAULT 0,
    created_at          DATETIME          NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at          DATETIME          NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (parent_agent_id, name)
);
CREATE INDEX idx_agents_parent_position ON agents (parent_agent_id, position);

CREATE TABLE sessions
(
    id            UUID     NOT NULL PRIMARY KEY,
    user_id       UUID     NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    app_name      TEXT     NOT NULL,
    root_agent_id UUID REFERENCES agents (id),
    last_update   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_sessions_user_app ON sessions (user_id, app_name, last_update);

CREATE TABLE events
(
    id              UUID       NOT NULL PRIMARY KEY,
    session_id      UUID       NOT NULL REFERENCES sessions (id) ON DELETE CASCADE,
    invocation_id   UUID       NOT NULL,
    seq             BIGINT     NOT NULL,
    branch          TEXT       NOT NULL DEFAULT '',
    isolation_scope TEXT       NOT NULL DEFAULT '',
    author          TEXT       NOT NULL,
    role            event_role NOT NULL,
    content_json    JSONB      NOT NULL DEFAULT '{}',
    actions_json    JSONB      NOT NULL DEFAULT '{}',
    is_partial      BOOLEAN    NOT NULL DEFAULT 0,
    is_final        BOOLEAN    NOT NULL DEFAULT 0,
    token_usage     JSONB      NOT NULL DEFAULT '{}',
    output_json     JSONB      NOT NULL DEFAULT '{}',
    created_at      DATETIME   NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (session_id, seq)
);
CREATE INDEX idx_events_session ON events (session_id, seq);
CREATE INDEX idx_events_invocation ON events (invocation_id);

CREATE TABLE session_state
(
    session_id UUID     NOT NULL REFERENCES sessions (id) ON DELETE CASCADE,
    key        TEXT     NOT NULL,
    value      JSONB    NOT NULL,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (session_id, key)
);

CREATE TABLE artifacts
(
    id           UUID     NOT NULL PRIMARY KEY,
    session_id   UUID     NOT NULL REFERENCES sessions (id) ON DELETE CASCADE,
    filename     TEXT     NOT NULL,
    version      BIGINT   NOT NULL DEFAULT 1,
    agent_name   TEXT,
    storage_uri  TEXT     NOT NULL,
    size_bytes   BIGINT,
    content_type TEXT,
    created_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (session_id, filename, version)
);

CREATE TABLE runs
(
    id            UUID       NOT NULL PRIMARY KEY,
    session_id    UUID       NOT NULL REFERENCES sessions (id) ON DELETE CASCADE,
    invocation_id UUID       NOT NULL,
    root_agent_id UUID REFERENCES agents (id),
    status        run_status NOT NULL DEFAULT 'running',
    started_at    DATETIME   NOT NULL DEFAULT CURRENT_TIMESTAMP,
    finished_at   DATETIME,
    error         TEXT,
    UNIQUE (invocation_id)
);
CREATE INDEX idx_runs_session ON runs (session_id);

CREATE TABLE tool_calls
(
    id          UUID             NOT NULL PRIMARY KEY,
    event_id    UUID             NOT NULL REFERENCES events (id) ON DELETE CASCADE,
    run_id      UUID REFERENCES runs (id),
    agent_name  TEXT             NOT NULL,
    tool_name   TEXT             NOT NULL,
    arguments   JSONB            NOT NULL DEFAULT '{}',
    result_json JSONB            NOT NULL DEFAULT '{}',
    status      tool_call_status NOT NULL DEFAULT 'ok',
    duration_ms BIGINT,
    created_at  DATETIME         NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_tool_calls_run ON tool_calls (run_id, tool_name);

CREATE TABLE skills
(
    id           UUID        NOT NULL PRIMARY KEY,
    user_id      UUID REFERENCES users (id) ON DELETE CASCADE,
    scope        skill_scope NOT NULL DEFAULT 'app',
    title        TEXT        NOT NULL,
    content_type TEXT        NOT NULL DEFAULT 'text',
    content      TEXT,
    storage_uri  TEXT,
    metadata     JSONB       NOT NULL DEFAULT '{}',
    is_active    BOOLEAN     NOT NULL DEFAULT 1,
    created_at   DATETIME    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at   DATETIME    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT skills_owner_check CHECK (
        (scope = 'app' AND user_id IS NULL) OR
        (scope = 'user' AND user_id IS NOT NULL)
        ),
    CONSTRAINT skills_content_check CHECK (
        (content IS NOT NULL) OR (storage_uri IS NOT NULL)
        )
);

CREATE TABLE agent_skills
(
    agent_id   UUID     NOT NULL REFERENCES agents (id) ON DELETE CASCADE,
    skill_id   UUID     NOT NULL REFERENCES skills (id) ON DELETE CASCADE,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (agent_id, skill_id)
);
CREATE INDEX idx_agent_skills_skill ON agent_skills (skill_id);

CREATE TABLE mcp_server
(
    id                   UUID     NOT NULL PRIMARY KEY,
    user_id              UUID     NOT NULL REFERENCES users (id),
    name                 TEXT     NOT NULL,
    endpoint             TEXT     NOT NULL,
    transport            TEXT     NOT NULL DEFAULT 'streamable_http',
    command              JSONB,
    args                 JSONB,
    auth_type            TEXT,
    auth_config          JSONB,
    allowed_tools        JSONB,
    require_confirmation BOOLEAN  NOT NULL DEFAULT false,
    confirmation_rules   JSONB,
    is_active            BOOLEAN  NOT NULL DEFAULT true,
    created_at           DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at           DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT empty_text_check CHECK ( length(name) > 0 AND length(endpoint) > 0 )
);

CREATE TABLE agent_mcp_server
(
    agent_id      UUID NOT NULL REFERENCES agents (id) ON DELETE RESTRICT,
    mcp_server_id UUID NOT NULL REFERENCES mcp_server (id) ON DELETE RESTRICT,
    PRIMARY KEY (agent_id, mcp_server_id)
);