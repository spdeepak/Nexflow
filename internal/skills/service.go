package skills

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"

	"github.com/spdeepak/nexflow/internal/enums"
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
		GetSkillByTitle(ctx context.Context, title string) (Skill, error)
		SyncGlobalSkills(ctx context.Context, userID uuid.UUID) error
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
	globalSkill := false
	if arg.Content != nil {
		content = sql.NullString{String: *arg.Content, Valid: true}
	}
	if arg.GlobalSkill != nil {
		globalSkill = *arg.GlobalSkill
	}
	if arg.ContentType == enums.SkillContentTypeStorageUri {
		skillMDPath, err := resolveSkillStoragePath(arg.StorageUri)
		if err != nil {
			slog.ErrorContext(ctx, "invalid skill location for storage uri skill", "err", err, "storageUri", arg.StorageUri)
			return schema.Skill{}, errors.SkillInvalidLocation
		}
		storageUri = sql.NullString{String: skillMDPath, Valid: true}
	} else if arg.StorageUri != nil {
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
		GlobalSkill: globalSkill,
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

// resolveSkillStoragePath validates the location of a storage uri skill and
// returns the path of its SKILL.md file. The location may either be the skill
// folder itself or a direct path to its SKILL.md file. It returns an error when
// the location does not exist or does not contain a SKILL.md file.
func resolveSkillStoragePath(storageUri *string) (string, error) {
	if storageUri == nil || strings.TrimSpace(*storageUri) == "" {
		return "", fmt.Errorf("storage uri is empty")
	}

	path := strings.TrimSpace(*storageUri)
	info, err := os.Stat(path)
	if err != nil {
		return "", fmt.Errorf("cannot access skill location %q: %w", path, err)
	}

	if info.IsDir() {
		skillMDPath := filepath.Join(path, "SKILL.md")
		if _, err = os.Stat(skillMDPath); err != nil {
			return "", fmt.Errorf("SKILL.md not found in folder %q: %w", path, err)
		}
		return skillMDPath, nil
	}

	if filepath.Base(path) != "SKILL.md" {
		return "", fmt.Errorf("skill location %q is not a SKILL.md file", path)
	}
	return path, nil
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

func (s *service) GetSkillByTitle(ctx context.Context, title string) (Skill, error) {
	skill, err := s.querier.GetSkillByTitle(ctx, title)
	if err != nil {
		slog.ErrorContext(ctx, "error getting skill by title", "err", err)
		return Skill{}, err
	}
	return skill, nil
}

func (s *service) SyncGlobalSkills(ctx context.Context, userID uuid.UUID) error {
	home, err := os.UserHomeDir()
	if err != nil {
		slog.ErrorContext(ctx, "failed to get user home dir", "err", err)
		return err
	}

	globalSkillsDir := filepath.Join(home, ".agents", "skills")
	entries, err := os.ReadDir(globalSkillsDir)
	if err != nil {
		if os.IsNotExist(err) {
			slog.DebugContext(ctx, "global skills directory does not exist", "path", globalSkillsDir)
			return nil
		}
		slog.ErrorContext(ctx, "failed to read global skills dir", "err", err)
		return err
	}

	// Collect skill titles from filesystem
	skillPaths := make(map[string]string) // title -> skillMDPath
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		skillDir := filepath.Join(globalSkillsDir, entry.Name())
		skillMDPath := filepath.Join(skillDir, "SKILL.md")

		// Check if SKILL.md exists
		if _, err = os.Stat(skillMDPath); os.IsNotExist(err) {
			continue
		}

		// Use directory name as title
		title := entry.Name()
		skillPaths[title] = skillDir
	}

	// Get all storageUri skills from DB with app scope
	dbSkills, err := s.querier.GetSkillsByContentTypeAndScope(ctx, GetSkillsByContentTypeAndScopeParams{
		ContentType: enums.SkillContentTypeStorageUri,
		Scope:       enums.SkillScopeUser,
	})
	if err != nil {
		slog.ErrorContext(ctx, "failed to get storageUri skills from DB", "err", err)
		return err
	}

	// Build map for O(1) lookups
	dbSkillsByTitle := make(map[string]Skill, len(dbSkills))
	for _, dbSkill := range dbSkills {
		dbSkillsByTitle[dbSkill.Title] = dbSkill
	}

	// Delete skills that are in DB but not in filesystem anymore
	for _, dbSkill := range dbSkills {
		if _, exists := skillPaths[dbSkill.Title]; !exists {
			slog.InfoContext(ctx, "Deleting orphaned skill", "title", dbSkill.Title, "id", dbSkill.ID)
			if err = s.querier.DeleteSkill(ctx, DeleteSkillParams{ID: dbSkill.ID, UserID: dbSkill.UserID}); err != nil {
				slog.ErrorContext(ctx, "failed to delete orphaned skill", "title", dbSkill.Title, "err", err)
			}
		}
	}

	// Add new skills from filesystem
	for title, skillMDPath := range skillPaths {
		if _, exists := dbSkillsByTitle[title]; exists {
			continue
		}
		globalSkill := true
		skillCreate := schema.SkillCreate{
			Title:       title,
			ContentType: enums.SkillContentTypeStorageUri,
			Content:     nil,
			Scope:       enums.SkillScopeUser,
			StorageUri:  &skillMDPath,
			GlobalSkill: &globalSkill,
			Metadata:    nil,
		}

		_, err = s.CreateSkill(ctx, skillCreate, userID)
		if err != nil {
			slog.ErrorContext(ctx, "failed to create skill from global", "title", title, "err", err)
		}
	}

	return nil
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
