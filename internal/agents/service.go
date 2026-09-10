package agents

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"google.golang.org/genai"

	"github.com/spdeepak/nexflow/internal/agentmcp"
	"github.com/spdeepak/nexflow/internal/agentskills"
	"github.com/spdeepak/nexflow/internal/errors"
	"github.com/spdeepak/nexflow/internal/util"
	"github.com/spdeepak/nexflow/schema"
)

type (
	service struct {
		querier           Querier
		agentSkillService agentskills.Service
		agentMCPService   agentmcp.Service
	}

	Service interface {
		CreateRootAgent(ctx context.Context, arg schema.AgentCreate) (Agent, error)
		CreateSubAgent(ctx context.Context, arg schema.AgentCreate) (Agent, error)
		DeleteAgent(ctx context.Context, id uuid.UUID) error
		GetAgent(ctx context.Context, id uuid.UUID) (schema.Agent, error)
		GetAgentDetail(ctx context.Context, id uuid.UUID) (schema.AgentDetail, error)
		GetRootAgent(ctx context.Context, id uuid.UUID) (schema.Agent, error)
		GetSubAgents(ctx context.Context, parentAgentID uuid.UUID) ([]schema.Agent, error)
		ListRootAgents(ctx context.Context) ([]schema.Agent, error)
		ReorderAgentChildren(ctx context.Context, parentAgentID uuid.UUID, orderedIDs []uuid.UUID) error
		UpdateAgent(ctx context.Context, id uuid.UUID, arg schema.AgentUpdate) (Agent, error)
	}
)

func NewService(querier Querier, agentSkillService agentskills.Service, agentMCPService agentmcp.Service) Service {
	return &service{
		querier:           querier,
		agentSkillService: agentSkillService,
		agentMCPService:   agentMCPService,
	}
}

func (s *service) CreateRootAgent(ctx context.Context, arg schema.AgentCreate) (Agent, error) {
	id, _ := uuid.NewV7()
	params := CreateRootAgentParams{
		ID:                id,
		Name:              arg.Name,
		Description:       sql.NullString{String: arg.Description, Valid: true},
		Instruction:       util.GetSQLNullString(arg.Instruction),
		GlobalInstruction: util.GetSQLNullString(arg.GlobalInstruction),
		Mode:              arg.Mode,
		ModelName:         sql.NullString{String: arg.ModelName, Valid: true},
		ModelCredentialID: arg.ModelCredentialID,
		CredentialSource:  arg.CredentialSource,
		ModelConfig:       toRawMessage(arg.ModelConfig),
		ConfigJson:        json.RawMessage{},
	}
	agent, err := s.querier.CreateRootAgent(ctx, params)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to create root agent", "error", err)
		return Agent{}, errors.RootAgentCreateFailed
	}
	slog.DebugContext(ctx, "Created root agent", "agent", agent)
	return agent, nil
}

func (s *service) CreateSubAgent(ctx context.Context, arg schema.AgentCreate) (Agent, error) {
	if arg.ParentAgentID == nil {
		return Agent{}, fmt.Errorf("parent agent id is required")
	}
	id, _ := uuid.NewV7()
	params := CreateSubAgentParams{
		ID:                id,
		ParentAgentID:     *arg.ParentAgentID,
		Name:              arg.Name,
		Description:       sql.NullString{String: arg.Description, Valid: true},
		Instruction:       util.GetSQLNullString(arg.Instruction),
		GlobalInstruction: util.GetSQLNullString(arg.GlobalInstruction),
		Mode:              arg.Mode,
		ModelName:         sql.NullString{String: arg.ModelName, Valid: true},
		ModelCredentialID: arg.ModelCredentialID,
		CredentialSource:  arg.CredentialSource,
		ModelConfig:       toRawMessage(arg.ModelConfig),
		ConfigJson:        json.RawMessage{},
	}
	agent, err := s.querier.CreateSubAgent(ctx, params)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to create sub agent", "error", err)
		return Agent{}, errors.RootAgentCreateFailed
	}
	slog.DebugContext(ctx, "Created sub agent", "agent", agent)
	return agent, nil
}

func (s *service) DeleteAgent(ctx context.Context, id uuid.UUID) error {
	return s.querier.DeleteAgent(ctx, id)
}

func (s *service) GetAgent(ctx context.Context, id uuid.UUID) (schema.Agent, error) {
	agent, err := s.querier.GetAgent(ctx, id)
	if err != nil {
		return schema.Agent{}, err
	}
	apiAgent := schema.Agent{
		CreatedAt:         agent.CreatedAt,
		CredentialSource:  agent.CredentialSource,
		Description:       agent.Description.String,
		ID:                agent.ID,
		Instruction:       agent.Instruction.String,
		IsActive:          agent.IsActive,
		Mode:              agent.Mode,
		ModelCredentialID: agent.ModelCredentialID,
		ModelName:         agent.ModelName.String,
		Name:              agent.Name,
		UpdatedAt:         agent.UpdatedAt,
	}
	if agent.GlobalInstruction.Valid {
		apiAgent.GlobalInstruction = &agent.GlobalInstruction.String
	}
	if agent.ParentAgentID.Time() != 0 {
		apiAgent.ParentAgentID = &agent.ParentAgentID
	}
	configJson, err := agent.ConfigJson.MarshalJSON()
	if err != nil {
		slog.ErrorContext(ctx, "Failed to marshal agent to get its config", "error", err)
	} else if err := json.Unmarshal(configJson, &apiAgent); err != nil {
		slog.ErrorContext(ctx, "Failed to unmarshal config of agent to get its config", "error", err)
	}
	return apiAgent, nil
}

func (s *service) GetAgentDetail(ctx context.Context, id uuid.UUID) (schema.AgentDetail, error) {
	agent, err := s.GetAgent(ctx, id)
	if err != nil {
		return schema.AgentDetail{}, err
	}

	children, err := s.GetSubAgents(ctx, id)
	if err != nil {
		return schema.AgentDetail{}, err
	}

	agentSkills, err := s.agentSkillService.ListAgentSkill(ctx, id)
	if err != nil {
		return schema.AgentDetail{}, err
	}

	agentMCPs, err := s.agentMCPService.ListAgentMCP(ctx, id)
	if err != nil {
		return schema.AgentDetail{}, err
	}

	var configJson *schema.AgentConfigJson
	if agent.ConfigJson != nil {
		configJson = new(schema.AgentConfigJson)
		*configJson = *agent.ConfigJson
	}

	detail := schema.AgentDetail{
		ConfigJson:        configJson,
		CreatedAt:         agent.CreatedAt,
		CredentialSource:  agent.CredentialSource,
		Description:       agent.Description,
		GlobalInstruction: agent.GlobalInstruction,
		ID:                agent.ID,
		Instruction:       agent.Instruction,
		IsActive:          agent.IsActive,
		Mcps:              agentMCPs,
		Mode:              agent.Mode,
		ModelConfig:       agent.ModelConfig,
		ModelCredentialID: agent.ModelCredentialID,
		ModelName:         agent.ModelName,
		Name:              agent.Name,
		ParentAgentID:     agent.ParentAgentID,
		Position:          agent.Position,
		Skills:            agentSkills,
		SubAgents:         children,
		UpdatedAt:         agent.UpdatedAt,
	}

	return detail, nil
}

func (s *service) GetRootAgent(ctx context.Context, id uuid.UUID) (schema.Agent, error) {
	agent, err := s.querier.GetRootAgent(ctx, id)
	if err != nil {
		return schema.Agent{}, err
	}
	apiAgent := schema.Agent{
		CreatedAt:         agent.CreatedAt,
		CredentialSource:  agent.CredentialSource,
		Description:       agent.Description.String,
		ID:                agent.ID,
		Instruction:       agent.Instruction.String,
		IsActive:          agent.IsActive,
		Mode:              agent.Mode,
		ModelCredentialID: agent.ModelCredentialID,
		ModelName:         agent.ModelName.String,
		Name:              agent.Name,
		UpdatedAt:         agent.UpdatedAt,
	}
	if agent.GlobalInstruction.Valid {
		apiAgent.GlobalInstruction = &agent.GlobalInstruction.String
	}
	configJson, err := agent.ConfigJson.MarshalJSON()
	if err != nil {
		slog.ErrorContext(ctx, "Failed to marshal agent to get its config", "error", err)
	} else if err := json.Unmarshal(configJson, &apiAgent.ConfigJson); err != nil {
		slog.ErrorContext(ctx, "Failed to unmarshal config of agent to get its config", "error", err)
	}
	modelConfigJson, err := agent.ModelConfig.MarshalJSON()
	if err != nil {
		slog.ErrorContext(ctx, "Failed to marshal agent to get its model config", "error", err)
	} else if err := json.Unmarshal(modelConfigJson, &apiAgent.ModelConfig); err != nil {
		slog.ErrorContext(ctx, "Failed to unmarshal config of agent to get its model config", "error", err)
	}
	return apiAgent, nil
}

func (s *service) GetSubAgents(ctx context.Context, parentAgentID uuid.UUID) ([]schema.Agent, error) {
	childrenAgents, err := s.querier.ListAgentChildren(ctx, parentAgentID)
	if err != nil {
		return nil, err
	}
	apiAgents := make([]schema.Agent, 0, len(childrenAgents))
	for _, agent := range childrenAgents {
		position := int(agent.Position)
		apiAgents = append(apiAgents, schema.Agent{
			CreatedAt:         agent.CreatedAt,
			CredentialSource:  agent.CredentialSource,
			Description:       agent.Description.String,
			ID:                agent.ID,
			Instruction:       agent.Instruction.String,
			IsActive:          agent.IsActive,
			Mode:              agent.Mode,
			ModelCredentialID: agent.ModelCredentialID,
			ModelName:         agent.ModelName.String,
			Name:              agent.Name,
			Position:          &position,
		})
	}
	return apiAgents, nil
}

func (s *service) ReorderAgentChildren(ctx context.Context, parentAgentID uuid.UUID, orderedIDs []uuid.UUID) error {
	type AgentPosition struct {
		ID       uuid.UUID `json:"id"`
		Position int       `json:"position"`
	}

	agentPositions := make([]AgentPosition, len(orderedIDs))
	for index, id := range orderedIDs {
		agentPositions[index] = AgentPosition{
			ID:       id,
			Position: index + 1,
		}
	}
	agentPositionsJson, err := json.Marshal(agentPositions)
	if err != nil {
		return err
	}
	err = s.querier.ReorderSubAgents(ctx, ReorderSubAgentsParams{Assignments: string(agentPositionsJson), ParentAgentID: parentAgentID})
	if err != nil {
		slog.ErrorContext(ctx, "Failed to reorder subagents", "error", err, "agentPositions", agentPositions)
		return err
	}
	return nil
}

func (s *service) ListRootAgents(ctx context.Context) ([]schema.Agent, error) {
	agents, err := s.querier.ListRootAgents(ctx)
	if err != nil {
		return nil, err
	}
	apiAgents := make([]schema.Agent, len(agents))
	for i, agent := range agents {
		apiAgents[i] = schema.Agent{
			CreatedAt:         agent.CreatedAt,
			CredentialSource:  agent.CredentialSource,
			Description:       agent.Description.String,
			GlobalInstruction: new(util.GetStringFromSQLNullString(agent.GlobalInstruction)),
			ID:                agent.ID,
			Instruction:       agent.Instruction.String,
			IsActive:          agent.IsActive,
			Mode:              agent.Mode,
			ModelCredentialID: agent.ModelCredentialID,
			ModelName:         agent.ModelName.String,
			Name:              agent.Name,
			UpdatedAt:         agent.UpdatedAt,
		}
	}
	return apiAgents, nil
}

func (s *service) UpdateAgent(ctx context.Context, id uuid.UUID, arg schema.AgentUpdate) (Agent, error) {
	updateParams := UpdateAgentParams{
		ID: id,
	}
	updateParams.Description = util.GetSQLNullString(arg.Description)
	updateParams.GlobalInstruction = util.GetSQLNullString(arg.GlobalInstruction)
	updateParams.Instruction = util.GetSQLNullString(arg.Instruction)
	updateParams.ModelName = util.GetSQLNullString(arg.ModelName)

	if arg.CredentialSource != nil {
		updateParams.CredentialSource = sql.NullString{String: string(*arg.CredentialSource), Valid: true}
	}
	if arg.Mode != nil {
		updateParams.Mode = sql.NullString{String: string(*arg.Mode), Valid: true}
	}
	if arg.ModelConfig != nil {
		modelConfig, _ := json.Marshal(arg.ModelConfig)
		updateParams.ModelConfig = modelConfig
	}
	if arg.ModelCredentialID != nil {
		updateParams.ModelCredentialID = sql.NullString{String: arg.ModelCredentialID.String(), Valid: true}
	}
	if arg.IsActive != nil {
		updateParams.IsActive = sql.NullBool{Bool: *arg.IsActive, Valid: true}
	}

	agent, err := s.querier.UpdateAgent(ctx, updateParams)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to update agent", "error", err, "id", id, "args", arg)
		return Agent{}, err
	}
	return agent, nil
}

func toRawMessage(cfg *genai.GenerateContentConfig) json.RawMessage {
	if cfg == nil {
		return json.RawMessage{}
	}
	raw, err := json.Marshal(cfg)
	if err != nil {
		return json.RawMessage{}
	}
	return raw
}
