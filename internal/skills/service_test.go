package skills

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	cfg "github.com/spdeepak/nexflow/internal/config"
	"github.com/spdeepak/nexflow/internal/db"
	"github.com/spdeepak/nexflow/internal/enums"
	"github.com/spdeepak/nexflow/internal/errors"
	"github.com/spdeepak/nexflow/internal/schema"
	"github.com/spdeepak/nexflow/internal/users"
)

type testFixture struct {
	test         *testing.T
	skillService Service
	userService  users.Service
}

func newTestFixture(test *testing.T) *testFixture {
	test.Helper()

	dbConfig := cfg.DBConfig{
		DBName:          "app.db",
		MaxOpenConns:    1,
		MaxIdleConns:    1,
		ConnMaxLifetime: time.Hour,
		ConnMaxIdleTime: 30 * time.Minute,
	}

	require.NoError(test, db.RunMigrations(dbConfig))
	test.Cleanup(func() {
		require.NoError(test, os.Remove("app.db"))
		require.NoError(test, os.Remove("app.db-shm"))
		require.NoError(test, os.Remove("app.db-wal"))
	})

	conn := db.Connect(dbConfig)
	return &testFixture{
		test:         test,
		skillService: NewService(New(conn)),
		userService:  users.NewService(users.New(conn)),
	}
}

func (tf *testFixture) createUser() schema.User {
	tf.test.Helper()
	user, err := tf.userService.CreateAppUser(context.Background(), uuid.NewString(), "Deepak")
	require.NoError(tf.test, err)
	require.NotNil(tf.test, user)
	return user
}

func TestSyncGlobalSkills(t *testing.T) {
	// Point os.UserHomeDir() at a fixture home so the test does not depend on
	// whatever skills happen to be installed on the machine running it.
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	globalSkillsDir := filepath.Join(home, ".agents", "skills")
	skillDir := filepath.Join(globalSkillsDir, "test-skill")
	require.NoError(t, os.MkdirAll(skillDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("# Test Skill"), 0o600))

	// A folder without SKILL.md must be ignored.
	require.NoError(t, os.MkdirAll(filepath.Join(globalSkillsDir, "not-a-skill"), 0o755))

	fixture := newTestFixture(t)
	user := fixture.createUser()
	ctx := context.Background()

	// Sync creates the skill found on the filesystem.
	require.NoError(t, fixture.skillService.SyncGlobalSkills(ctx, user.ID))
	skill, err := fixture.skillService.GetAllAvailableSkill(ctx)
	require.NoError(t, err)
	require.Len(t, skill, 1)
	require.Equal(t, "test-skill", skill[0].Title)
	require.NotNil(t, skill[0].StorageUri)
	require.Equal(t, filepath.Join(skillDir, "SKILL.md"), *skill[0].StorageUri)

	// A second sync must not create duplicates.
	require.NoError(t, fixture.skillService.SyncGlobalSkills(ctx, user.ID))
	skill, err = fixture.skillService.GetAllAvailableSkill(ctx)
	require.NoError(t, err)
	require.Len(t, skill, 1)

	// Removing the folder removes the orphaned skill from the DB.
	require.NoError(t, os.RemoveAll(skillDir))
	require.NoError(t, fixture.skillService.SyncGlobalSkills(ctx, user.ID))
	skill, err = fixture.skillService.GetAllAvailableSkill(ctx)
	require.NoError(t, err)
	require.Empty(t, skill)
}

func TestSyncGlobalSkillsAcrossDirectories(t *testing.T) {
	root := t.TempDir()
	home := filepath.Join(root, "home")
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	// Installed skills are commonly linked into a skills directory, so the
	// folder itself is a symlink.
	linkedTarget := filepath.Join(root, "store", "linked-skill")
	require.NoError(t, os.MkdirAll(linkedTarget, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(linkedTarget, "SKILL.md"), []byte("# Linked"), 0o600))
	agentsSkillsDir := filepath.Join(home, ".agents", "skills")
	require.NoError(t, os.MkdirAll(agentsSkillsDir, 0o755))
	require.NoError(t, os.Symlink(linkedTarget, filepath.Join(agentsSkillsDir, "linked-skill")))

	// A second skills directory with its own skill.
	otherSkillsDir := filepath.Join(home, ".claude", "skills")
	otherSkillDir := filepath.Join(otherSkillsDir, "other-skill")
	require.NoError(t, os.MkdirAll(otherSkillDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(otherSkillDir, "SKILL.md"), []byte("# Other"), 0o600))

	fixture := newTestFixture(t)
	user := fixture.createUser()
	ctx := context.Background()

	for range 2 {
		require.NoError(t, fixture.skillService.SyncGlobalSkills(ctx, user.ID))
		skills, err := fixture.skillService.GetAllAvailableSkill(ctx)
		require.NoError(t, err)
		titles := make([]string, len(skills))
		for i, skill := range skills {
			titles[i] = skill.Title
		}
		require.ElementsMatch(t, []string{"linked-skill", "other-skill"}, titles)
	}
}

func TestSyncGlobalSkillsKeepsUserSkills(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	require.NoError(t, os.MkdirAll(filepath.Join(home, ".agents", "skills"), 0o755))

	fixture := newTestFixture(t)
	user := fixture.createUser()
	ctx := context.Background()

	ownSkillDir := filepath.Join(t.TempDir(), "own-skill")
	require.NoError(t, os.MkdirAll(ownSkillDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(ownSkillDir, "SKILL.md"), []byte("# Own"), 0o600))
	_, err := fixture.skillService.CreateSkill(ctx, schema.SkillCreate{
		Title:       "own-skill",
		ContentType: enums.SkillContentTypeStorageUri,
		Scope:       enums.SkillScopeUser,
		StorageUri:  &ownSkillDir,
	}, user.ID)
	require.NoError(t, err)

	require.NoError(t, fixture.skillService.SyncGlobalSkills(ctx, user.ID))
	skills, err := fixture.skillService.GetAllAvailableSkill(ctx)
	require.NoError(t, err)
	require.Len(t, skills, 1)
	require.Equal(t, "own-skill", skills[0].Title)
}

func TestSyncGlobalSkillsMissingGlobalDir(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	fixture := newTestFixture(t)
	user := fixture.createUser()

	require.NoError(t, fixture.skillService.SyncGlobalSkills(context.Background(), user.ID))
	skill, err := fixture.skillService.GetAllAvailableSkill(context.Background())
	require.NoError(t, err)
	require.Empty(t, skill)
}

func TestCreateSkillStorageUriLocation(t *testing.T) {
	fixture := newTestFixture(t)
	user := fixture.createUser()
	ctx := context.Background()

	t.Run("missing storage uri", func(t *testing.T) {
		_, err := fixture.skillService.CreateSkill(ctx, schema.SkillCreate{
			Title:       "missing-uri-skill",
			ContentType: enums.SkillContentTypeStorageUri,
			Scope:       enums.SkillScopeUser,
		}, user.ID)
		require.ErrorIs(t, err, errors.SkillInvalidLocation)
	})

	t.Run("non existent location", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "does-not-exist")
		_, err := fixture.skillService.CreateSkill(ctx, schema.SkillCreate{
			Title:       "non-existent-location-skill",
			ContentType: enums.SkillContentTypeStorageUri,
			Scope:       enums.SkillScopeUser,
			StorageUri:  &path,
		}, user.ID)
		require.ErrorIs(t, err, errors.SkillInvalidLocation)
	})

	t.Run("folder without SKILL.md", func(t *testing.T) {
		dir := t.TempDir()
		_, err := fixture.skillService.CreateSkill(ctx, schema.SkillCreate{
			Title:       "invalid-location-skill",
			ContentType: enums.SkillContentTypeStorageUri,
			Scope:       enums.SkillScopeUser,
			StorageUri:  &dir,
		}, user.ID)
		require.ErrorIs(t, err, errors.SkillInvalidLocation)
	})

	t.Run("folder with SKILL.md", func(t *testing.T) {
		dir := t.TempDir()
		skillMDPath := filepath.Join(dir, "SKILL.md")
		require.NoError(t, os.WriteFile(skillMDPath, []byte("# Skill"), 0o600))

		skill, err := fixture.skillService.CreateSkill(ctx, schema.SkillCreate{
			Title:       "valid-location-skill",
			ContentType: enums.SkillContentTypeStorageUri,
			Scope:       enums.SkillScopeUser,
			StorageUri:  &dir,
		}, user.ID)
		require.NoError(t, err)
		require.NotNil(t, skill.StorageUri)
		require.Equal(t, skillMDPath, *skill.StorageUri)
	})
}
