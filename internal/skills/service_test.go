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

	dbConfig := cfg.AppConfig{
		DBConfig: cfg.DBConfig{
			Host:              "localhost",
			Port:              "5432",
			DBName:            "app.db",
			UserName:          "admin",
			Password:          "admin",
			SSLMode:           "disable",
			Timeout:           10 * time.Second,
			MaxRetry:          3,
			ConnectTimeout:    5 * time.Second,
			StatementTimeout:  30 * time.Second,
			MaxOpenConns:      1,
			MaxIdleConns:      1,
			ConnMaxLifetime:   1 * time.Hour,
			ConnMaxIdleTime:   30 * time.Minute,
			HealthCheckPeriod: 1 * time.Minute,
		},
	}.DBConfig

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
	id := uuid.New()
	user, err := tf.userService.CreateAppUser(context.Background(), id, "Deepak")
	require.NoError(tf.test, err)
	require.NotNil(tf.test, user)
	return user
}

func TestSyncGlobalSkills(t *testing.T) {
	fixture := newTestFixture(t)
	user := fixture.createUser()
	err := fixture.skillService.SyncGlobalSkills(context.Background(), user.ID)
	require.NoError(t, err)
	skill, err := fixture.skillService.GetAllAvailableSkill(context.Background())
	require.NoError(t, err)
	require.NotEmpty(t, skill)
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
