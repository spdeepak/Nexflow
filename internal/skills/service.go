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
			GlobalSkill: k.GlobalSkill,
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

	allSkillDirectories, err := findSkillsDirs(home)
	if err != nil {
		slog.ErrorContext(ctx, "failed to find skills directories", "err", err)
		return err
	}

	// Collect skill folders across every skills directory before reconciling:
	// a skill found in one directory must not be treated as orphaned while
	// syncing another one. The first directory listing a title wins.
	skillPaths := make(map[string]string) // title -> skill folder
	for _, skillsDir := range allSkillDirectories {
		entries, err := os.ReadDir(skillsDir)
		if err != nil {
			if os.IsNotExist(err) {
				slog.DebugContext(ctx, "skills directory does not exist", "path", skillsDir)
				continue
			}
			slog.ErrorContext(ctx, "failed to read global skills dir", "err", err)
			return err
		}

		for _, entry := range entries {
			if _, exists := skillPaths[entry.Name()]; exists {
				continue
			}

			// Installed skills are often symlinked folders, and DirEntry.IsDir
			// does not follow symlinks, so stat the folder instead.
			skillDir := filepath.Join(skillsDir, entry.Name())
			if info, err := os.Stat(skillDir); err != nil || !info.IsDir() {
				continue
			}
			if info, err := os.Stat(filepath.Join(skillDir, "SKILL.md")); err != nil || info.IsDir() {
				continue
			}

			// Use directory name as title
			skillPaths[entry.Name()] = skillDir
		}
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

	// Delete synced skills that are no longer in the filesystem. Skills the
	// user added manually point outside the scanned directories and are kept.
	for _, dbSkill := range dbSkills {
		if !dbSkill.GlobalSkill {
			continue
		}
		if _, exists := skillPaths[dbSkill.Title]; !exists {
			slog.InfoContext(ctx, "Deleting orphaned skill", "title", dbSkill.Title, "id", dbSkill.ID)
			if err = s.querier.DeleteSkill(ctx, DeleteSkillParams{ID: dbSkill.ID, UserID: dbSkill.UserID}); err != nil {
				slog.ErrorContext(ctx, "failed to delete orphaned skill", "title", dbSkill.Title, "err", err)
			}
		}
	}

	// Add new skills from filesystem
	for title, skillDir := range skillPaths {
		if _, exists := dbSkillsByTitle[title]; exists {
			continue
		}
		globalSkill := true
		skillCreate := schema.SkillCreate{
			Title:       title,
			ContentType: enums.SkillContentTypeStorageUri,
			Content:     nil,
			Scope:       enums.SkillScopeUser,
			StorageUri:  &skillDir,
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

// findSkillsDirs returns every ~/.<name>/skills directory that exists.
func findSkillsDirs(home string) ([]string, error) {
	entries, err := os.ReadDir(home)
	if err != nil {
		return nil, err
	}

	var result []string
	for _, e := range entries {
		if !strings.HasPrefix(e.Name(), ".") {
			continue
		}

		dirPath := filepath.Join(home, e.Name())

		// e.IsDir() is false for symlinks to directories, so stat the path instead
		info, err := os.Stat(dirPath)
		if err != nil || !info.IsDir() {
			continue
		}

		skillsPath := filepath.Join(dirPath, "skills")
		if info, err = os.Stat(skillsPath); err == nil && info.IsDir() {
			result = append(result, skillsPath)
		}
	}
	return result, nil
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
	slog.DebugContext(ctx, "updating skill", "id", id)
	skill, err := s.querier.UpdateSkill(ctx, updateParams)
	if err != nil {
		return Skill{}, err
	}
	return skill, nil
}
