package application

import (
	"encoding/base64"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/google/uuid"

	"github.com/spdeepak/nexflow/internal/enums"
	"github.com/spdeepak/nexflow/internal/errors"
	"github.com/spdeepak/nexflow/internal/schema"
	"github.com/spdeepak/nexflow/internal/util"
)

func skillsDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "."
	}
	skillDirPath := filepath.Join(home, ".local/share/nexflow/skills")
	_ = os.MkdirAll(skillDirPath, 0o700)
	return skillDirPath
}

func (a *App) CreateSkill(params schema.SkillCreate) error {
	if err := params.Validate(); err != nil {
		return err
	}
	_, err := a.skillService.CreateSkill(a.ctx, params, a.deviceID)
	return err
}

func (a *App) CreateSkillFromFile(params schema.SkillCreate, fileDataBase64 string) error {
	if err := params.Validate(); err != nil {
		return err
	}
	fileData, err := base64.StdEncoding.DecodeString(fileDataBase64)
	if err != nil {
		return fmt.Errorf("failed to decode file data: %w", err)
	}

	extractedText, err := util.GetFileContents(fileData)
	if err != nil {
		slog.ErrorContext(a.ctx, "Failed to extract PDF content", "error", err)
		return fmt.Errorf("failed to read PDF content: %w", err)
	}
	params.ContentType = "pdf"
	params.Content = &extractedText
	skill, err := a.skillService.CreateSkill(a.ctx, params, a.deviceID)
	if err != nil {
		return err
	}

	fileName := skill.ID.String() + ".pdf"
	filePath := filepath.Join(skillsDir(), fileName)

	if err = os.WriteFile(filePath, fileData, 0o600); err != nil {
		slog.ErrorContext(a.ctx, "Failed to write PDF file", "error", err)
		return fmt.Errorf("failed to save PDF file: %w", err)
	}

	_, err = a.skillService.UpdateSkill(a.ctx, skill.ID, schema.SkillUpdate{
		StorageUri: &filePath,
	})
	if err != nil {
		_ = os.Remove(filePath)
		_ = a.DeleteSkill(skill.ID)
		return errors.SkillCreationFailed
	}
	return nil
}

func (a *App) GetSkill() ([]schema.Skill, error) {
	items, err := a.skillService.GetAllAvailableSkill(a.ctx)
	if err != nil {
		slog.ErrorContext(a.ctx, "Error getting skill", "error", err)
		return nil, err
	}
	result := make([]schema.Skill, len(items))
	for i, k := range items {
		storageUri := ""
		if k.StorageUri != nil {
			storageUri = *k.StorageUri
		}
		result[i] = schema.Skill{
			ID:          k.ID,
			Title:       k.Title,
			ContentType: k.ContentType,
			Content:     k.Content,
			StorageUri:  &storageUri,
			IsActive:    k.IsActive,
		}
	}
	return result, nil
}

func (a *App) UpdateSkill(skillId uuid.UUID, params schema.SkillUpdate) error {
	if err := params.Validate(); err != nil {
		return err
	}
	_, err := a.skillService.UpdateSkill(a.ctx, skillId, params)
	return err
}

func (a *App) UpdateSkillFromFile(skillId uuid.UUID, params schema.SkillUpdate, fileDataBase64 string) error {
	if err := params.Validate(); err != nil {
		return err
	}
	fileData, err := base64.StdEncoding.DecodeString(fileDataBase64)
	if err != nil {
		return fmt.Errorf("failed to decode file data: %w", err)
	}

	fileName := skillId.String() + ".pdf"
	filePath := filepath.Join(skillsDir(), fileName)

	if err = os.WriteFile(filePath, fileData, 0o600); err != nil {
		slog.ErrorContext(a.ctx, "Failed to write PDF file", "error", err)
		return fmt.Errorf("failed to save PDF file: %w", err)
	}

	extractedText, err := util.GetFileContents(fileData)
	if err != nil {
		_ = os.Remove(filePath)
		slog.ErrorContext(a.ctx, "Failed to extract PDF content, please try again", "error", err)
		return fmt.Errorf("failed to read PDF content: %w", err)
	}

	contentType := enums.SkillContentTypePDF
	params.ContentType = &contentType
	params.Content = &extractedText
	params.StorageUri = &filePath

	_, err = a.skillService.UpdateSkill(a.ctx, skillId, params)
	return err
}

func (a *App) GetSkillPDF(skillId uuid.UUID) (string, error) {
	skill, err := a.skillService.GetSkill(a.ctx, skillId)
	if err != nil {
		return "", err
	}
	if !skill.StorageUri.Valid || skill.StorageUri.String == "" {
		return "", nil
	}
	data, err := os.ReadFile(skill.StorageUri.String)
	if err != nil {
		return "", fmt.Errorf("failed to read PDF file: %w", err)
	}
	return base64.StdEncoding.EncodeToString(data), nil
}

func (a *App) DeleteSkill(id uuid.UUID) error {
	slog.Info("Deleting skill", "id", id)
	return a.skillService.DeleteSkill(a.ctx, a.deviceID, id)
}
