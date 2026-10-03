package schema

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/spdeepak/nexflow/internal/enums"
)

func TestSkillCreateValidateStorageUriRequired(t *testing.T) {
	storageUri := "/tmp/some-skill"

	storageUriSkill := SkillCreate{
		Title:       "some-skill",
		ContentType: enums.SkillContentTypeStorageUri,
		Scope:       enums.SkillScopeUser,
		StorageUri:  &storageUri,
	}
	require.NoError(t, storageUriSkill.Validate())

	missingStorageUri := SkillCreate{
		Title:       "some-skill",
		ContentType: enums.SkillContentTypeStorageUri,
		Scope:       enums.SkillScopeUser,
	}
	require.Error(t, missingStorageUri.Validate())

	textSkill := SkillCreate{
		Title:       "some-skill",
		ContentType: enums.SkillContentTypeText,
		Scope:       enums.SkillScopeUser,
	}
	require.NoError(t, textSkill.Validate())
}
