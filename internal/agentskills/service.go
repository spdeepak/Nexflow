package agentskills

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/google/uuid"

	"github.com/spdeepak/nexflow/internal/util"
	"github.com/spdeepak/nexflow/schema"
)

type (
	service struct {
		querier Querier
	}

	Service interface {
		AttachSkillToAgent(ctx context.Context, arg AttachSkillToAgentParams) error
		DetachSkillFromAgent(ctx context.Context, arg DetachSkillFromAgentParams) error
		ListAgentSkill(ctx context.Context, agentID uuid.UUID) ([]schema.Skill, error)
	}
)

func NewService(querier Querier) Service {
	return &service{
		querier: querier,
	}
}

func (s *service) AttachSkillToAgent(ctx context.Context, arg AttachSkillToAgentParams) error {
	err := s.querier.AttachSkillToAgent(ctx, arg)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to attach Skill to Agent", "error", err)
		return err
	}
	return nil
}

func (s *service) DetachSkillFromAgent(ctx context.Context, arg DetachSkillFromAgentParams) error {
	err := s.querier.DetachSkillFromAgent(ctx, arg)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to detach Skill from Agent", "error", err)
		return err
	}
	return nil
}

func (s *service) ListAgentSkill(ctx context.Context, agentID uuid.UUID) ([]schema.Skill, error) {
	agentSkills, err := s.querier.ListAgentSkill(ctx, agentID)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to list AgentSkill", "error", err)
		return nil, err
	}
	skills := make([]schema.Skill, len(agentSkills))
	for index, agentSkill := range agentSkills {
		skills[index] = schema.Skill{
			ID:          agentSkill.ID,
			UserID:      agentSkill.UserID,
			Scope:       agentSkill.Scope,
			Title:       agentSkill.Title,
			ContentType: agentSkill.ContentType,
			Content:     util.GetOptionalString(agentSkill.Content),
			StorageUri:  util.GetOptionalString(agentSkill.StorageUri),
			Metadata:    unmarshalMetadata(agentSkill.Metadata),
			IsActive:    agentSkill.IsActive,
			CreatedAt:   agentSkill.CreatedAt,
			UpdatedAt:   agentSkill.UpdatedAt,
		}
	}
	return skills, nil
}

func unmarshalMetadata(raw json.RawMessage) schema.SkillMetadata {
	if raw == nil {
		return schema.SkillMetadata{}
	}
	var m schema.SkillMetadata
	if err := json.Unmarshal(raw, &m); err != nil {
		return schema.SkillMetadata{}
	}
	return m
}
