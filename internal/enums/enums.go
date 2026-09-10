package enums

import "database/sql/driver"

type CredentialScope string

const (
	CredentialScopeApp  CredentialScope = "app"
	CredentialScopeUser CredentialScope = "user"
)

func (e CredentialScope) String() string { return string(e) }

func (e CredentialScope) Value() (driver.Value, error) { return string(e), nil }

type SkillScope string

const (
	SkillScopeApp  SkillScope = "app"
	SkillScopeUser SkillScope = "user"
)

func (e SkillScope) String() string { return string(e) }

func (e SkillScope) Value() (driver.Value, error) { return string(e), nil }

type CredentialSource string

const (
	CredentialSourceAuto CredentialSource = "auto"
	CredentialSourceUser CredentialSource = "user"
	CredentialSourceApp  CredentialSource = "app"
)

func (e CredentialSource) String() string { return string(e) }

func (e CredentialSource) Value() (driver.Value, error) { return string(e), nil }

type RunStatus string

const (
	RunStatusRunning     RunStatus = "running"
	RunStatusCompleted   RunStatus = "completed"
	RunStatusFailed      RunStatus = "failed"
	RunStatusInterrupted RunStatus = "interrupted"
)

func (e RunStatus) String() string { return string(e) }

func (e RunStatus) Value() (driver.Value, error) { return string(e), nil }

type ToolCallStatus string

const (
	ToolCallStatusOk          ToolCallStatus = "ok"
	ToolCallStatusError       ToolCallStatus = "error"
	ToolCallStatusLongRunning ToolCallStatus = "long_running"
)

func (e ToolCallStatus) String() string { return string(e) }

func (e ToolCallStatus) Value() (driver.Value, error) { return string(e), nil }

type EventRole string

const (
	EventRoleUser     EventRole = "user"
	EventRoleModel    EventRole = "model"
	EventRoleFunction EventRole = "function"
)

func (e EventRole) String() string { return string(e) }

func (e EventRole) Value() (driver.Value, error) { return string(e), nil }

type SkillContentType string

const (
	SkillContentTypeText     SkillContentType = "text"
	SkillContentTypeMarkDown SkillContentType = "markdown"
	SkillContentTypePDF      SkillContentType = "pdf"
	SkillContentTypeCSV      SkillContentType = "csv"
	SkillContentTypeJson     SkillContentType = "json"
)

func (s SkillContentType) String() string { return string(s) }

func (s SkillContentType) Value() (driver.Value, error) { return string(s), nil }

type McpTransport string

const (
	McpTransportStreamableHttp = "streamable_http"
	McpTransportStdIO          = "stdio"
)

func (m McpTransport) String() string { return string(m) }

func (m McpTransport) Value() (driver.Value, error) { return string(m), nil }

type McpAuthType string

const (
	McpAuthTypeApiKey = "api_key"
	McpAuthTypeOAUTH  = "oauth"
)

func (m McpAuthType) String() string { return string(m) }

func (m McpAuthType) Value() (driver.Value, error) { return string(m), nil }
