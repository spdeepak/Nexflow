package skills

import (
	"context"
	"database/sql"
	"encoding/json"
	"log/slog"
	"os"

	"github.com/google/uuid"

	"github.com/spdeepak/nexflow/internal/errors"
	"github.com/spdeepak/nexflow/internal/schema"
)

type (
	service struct {
		querier Querier
	}

	Service interface {
		CreateSkill(ctx context.Context, arg schema.SkillCreate, userID uuid.UUID) (schema.Skill, error)
		DeleteSkill(ctx context.Context, userId, id uuid.UUID) error
		GetAllAvailableSkill(ctx context.Context) ([]schema.Skill, error)
		GetAvailableSkill(ctx context.Context, arg GetAvailableSkillParams) ([]GetAvailableSkillRow, error)
		GetSkill(ctx context.Context, id uuid.UUID) (Skill, error)
		UpdateSkill(ctx context.Context, id uuid.UUID, arg schema.SkillUpdate) (Skill, error)
	}
)

func NewService(querier Querier) Service {
	return &service{
		querier: querier,
	}
}

func (s *service) CreateSkill(ctx context.Context, arg schema.SkillCreate, userID uuid.UUID) (schema.Skill, error) {
	var metadata []byte
	if arg.Metadata != nil {
		metadata, _ = json.Marshal(arg.Metadata)
	} else {
		metadata = []byte{}
	}

	var content sql.NullString
	var storageUri sql.NullString
	if arg.Content != nil {
		content = sql.NullString{String: *arg.Content, Valid: true}
	}
	if arg.StorageUri != nil {
		storageUri = sql.NullString{String: *arg.StorageUri, Valid: true}
	}
	id, _ := uuid.NewV7()
	createSkillParams := CreateSkillParams{
		ID:          id,
		UserID:      userID,
		Scope:       arg.Scope,
		Title:       arg.Title,
		ContentType: arg.ContentType,
		Content:     content,
		StorageUri:  storageUri,
		Metadata:    metadata,
	}

	skill, err := s.querier.CreateSkill(ctx, createSkillParams)
	if err != nil {
		slog.ErrorContext(ctx, "error creating skill", "err", err)
		return schema.Skill{}, errors.SkillCreationFailed
	}
	return schema.Skill{
		AgentCount:  0,
		Content:     &skill.Content.String,
		ContentType: skill.ContentType,
		CreatedAt:   skill.CreatedAt,
		ID:          skill.ID,
		IsActive:    skill.IsActive,
		Metadata:    nil,
		Scope:       skill.Scope,
		StorageUri:  &skill.StorageUri.String,
		Title:       skill.Title,
		UpdatedAt:   skill.UpdatedAt,
		UserID:      skill.UserID,
	}, nil
}

func (s *service) DeleteSkill(ctx context.Context, userId, id uuid.UUID) error {
	skill, err := s.GetSkill(ctx, id)
	if err != nil {
		slog.ErrorContext(ctx, "couldn't find skill", "err", err)
		return errors.SkillDeletionFailed
	}
	if skill.StorageUri.Valid {
		err = os.Remove(skill.StorageUri.String)
		if err != nil {
			slog.ErrorContext(ctx, "error deleting skill file", "err", err, "filePath", skill.StorageUri.String)
			return errors.SkillDeletionFailed
		}
	}
	err = s.querier.DeleteSkill(ctx, DeleteSkillParams{ID: id, UserID: userId})
	if err != nil {
		slog.ErrorContext(ctx, "error deleting skill", "err", err)
		return errors.SkillDeletionFailed
	}
	return nil
}

func (s *service) GetAvailableSkill(ctx context.Context, arg GetAvailableSkillParams) ([]GetAvailableSkillRow, error) {
	//TODO implement me
	panic("implement me")
}

func (s *service) GetAllAvailableSkill(ctx context.Context) ([]schema.Skill, error) {
	skill, err := s.querier.GetAllAvailableSkill(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "error getting all available skill", "err", err)
		return nil, err
	}
	allSkill := make([]schema.Skill, len(skill))
	for i, k := range skill {
		allSkill[i] = schema.Skill{
			AgentCount:  int(k.AgentCount),
			Content:     &k.Content.String,
			ContentType: k.ContentType,
			CreatedAt:   k.CreatedAt,
			ID:          k.ID,
			IsActive:    k.IsActive,
			StorageUri:  &k.StorageUri.String,
			Title:       k.Title,
			UpdatedAt:   k.UpdatedAt,
		}
	}
	return allSkill, nil
}

func (s *service) GetSkill(ctx context.Context, id uuid.UUID) (Skill, error) {
	skill, err := s.querier.GetSkill(ctx, id)
	if err != nil {
		slog.ErrorContext(ctx, "error getting skill", "err", err)
		return Skill{}, err
	}
	return skill, nil
}

func (s *service) UpdateSkill(ctx context.Context, id uuid.UUID, arg schema.SkillUpdate) (Skill, error) {
	updateParams := UpdateSkillParams{
		ID: id,
	}
	if arg.Content != nil {
		updateParams.Content = sql.NullString{String: *arg.Content, Valid: true}
	}
	if arg.ContentType != nil {
		updateParams.ContentType = *arg.ContentType
	}
	if arg.IsActive != nil {
		updateParams.IsActive = sql.NullBool{Bool: *arg.IsActive, Valid: true}
	}
	if arg.Metadata != nil {
		metadata, err := json.Marshal(arg.Metadata)
		if err != nil {
			return Skill{}, err
		}
		updateParams.Metadata = metadata
	}
	if arg.StorageUri != nil {
		updateParams.StorageUri = sql.NullString{String: *arg.StorageUri, Valid: true}
	}
	if arg.Title != nil {
		updateParams.Title = sql.NullString{String: *arg.Title, Valid: true}
	}
	slog.InfoContext(ctx, "updating skill", "updateParams", updateParams, "arg", arg)
	skill, err := s.querier.UpdateSkill(ctx, updateParams)
	if err != nil {
		return Skill{}, err
	}
	return skill, nil
}
