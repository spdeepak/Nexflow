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
	skill, err := fixture.skillService.GetAvailableSkill(ctx, user.ID)
	require.NoError(t, err)
	require.Len(t, skill, 1)
	require.Equal(t, "test-skill", skill[0].Title)
	require.NotNil(t, skill[0].StorageUri)
	require.Equal(t, filepath.Join(skillDir, "SKILL.md"), *skill[0].StorageUri)

	// A second sync must not create duplicates.
	require.NoError(t, fixture.skillService.SyncGlobalSkills(ctx, user.ID))
	skill, err = fixture.skillService.GetAvailableSkill(ctx, user.ID)
	require.NoError(t, err)
	require.Len(t, skill, 1)

	// Removing the folder removes the orphaned skill from the DB.
	require.NoError(t, os.RemoveAll(skillDir))
	require.NoError(t, fixture.skillService.SyncGlobalSkills(ctx, user.ID))
	skill, err = fixture.skillService.GetAvailableSkill(ctx, user.ID)
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
		skills, err := fixture.skillService.GetAvailableSkill(ctx, user.ID)
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
	skills, err := fixture.skillService.GetAvailableSkill(ctx, user.ID)
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
	skill, err := fixture.skillService.GetAvailableSkill(context.Background(), user.ID)
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

func createTextSkill(t *testing.T, fixture *testFixture, owner uuid.UUID, title string, scope enums.SkillScope) schema.Skill {
	t.Helper()
	content := "content of " + title
	skill, err := fixture.skillService.CreateSkill(context.Background(), schema.SkillCreate{
		Title:       title,
		ContentType: enums.SkillContentTypeText,
		Content:     &content,
		Scope:       scope,
	}, owner)
	require.NoError(t, err)
	return skill
}

func TestGetAvailableSkillScopes(t *testing.T) {
	fixture := newTestFixture(t)
	owner := fixture.createUser()
	other := fixture.createUser()
	ctx := context.Background()

	appSkill := createTextSkill(t, fixture, owner.ID, "scoped-app-skill", enums.SkillScopeApp)
	ownerSkill := createTextSkill(t, fixture, owner.ID, "scoped-owner-skill", enums.SkillScopeUser)
	createTextSkill(t, fixture, other.ID, "scoped-other-skill", enums.SkillScopeUser)

	listFor := func(user uuid.UUID) []string {
		t.Helper()
		items, err := fixture.skillService.GetAvailableSkill(ctx, user)
		require.NoError(t, err)
		titles := make([]string, len(items))
		for i, item := range items {
			titles[i] = item.Title
		}
		return titles
	}

	require.ElementsMatch(t, []string{"scoped-app-skill", "scoped-owner-skill"}, listFor(owner.ID))
	require.ElementsMatch(t, []string{"scoped-app-skill", "scoped-other-skill"}, listFor(other.ID))

	// The app-scope row must be stored with user_id NULL (skills_owner_check).
	row, err := fixture.skillService.GetSkill(ctx, appSkill.ID)
	require.NoError(t, err)
	require.Equal(t, uuid.Nil, row.UserID)
	require.Equal(t, enums.SkillScopeApp, row.Scope)

	// Sanity: user-scope rows keep their owner.
	row, err = fixture.skillService.GetSkill(ctx, ownerSkill.ID)
	require.NoError(t, err)
	require.Equal(t, owner.ID, row.UserID)
}

func TestCreateSkillOwnershipAndDuplicateTitle(t *testing.T) {
	fixture := newTestFixture(t)
	owner := fixture.createUser()
	ctx := context.Background()

	cases := []struct {
		name    string
		pre     *schema.SkillCreate
		params  schema.SkillCreate
		wantErr error
		assert  func(t *testing.T, got schema.Skill, err error)
	}{
		{
			name:   "app scope stores null owner",
			params: textCreate("create-app-skill", enums.SkillScopeApp),
			assert: func(t *testing.T, got schema.Skill, err error) {
				require.NoError(t, err)
				require.Equal(t, uuid.Nil, got.UserID)
				require.Equal(t, enums.SkillScopeApp, got.Scope)
			},
		},
		{
			name:   "user scope stores owner",
			params: textCreate("create-user-skill", enums.SkillScopeUser),
			assert: func(t *testing.T, got schema.Skill, err error) {
				require.NoError(t, err)
				require.Equal(t, owner.ID, got.UserID)
				require.Equal(t, enums.SkillScopeUser, got.Scope)
			},
		},
		{
			name:    "duplicate title rejected",
			pre:     ptr(textCreate("create-dup-skill", enums.SkillScopeUser)),
			params:  textCreate("create-dup-skill", enums.SkillScopeApp),
			wantErr: errors.SkillTitleTaken,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.pre != nil {
				_, err := fixture.skillService.CreateSkill(ctx, *tc.pre, owner.ID)
				require.NoError(t, err)
			}
			got, err := fixture.skillService.CreateSkill(ctx, tc.params, owner.ID)
			if tc.wantErr != nil {
				require.ErrorIs(t, err, tc.wantErr)
				return
			}
			tc.assert(t, got, err)
		})
	}
}

func textCreate(title string, scope enums.SkillScope) schema.SkillCreate {
	content := "content of " + title
	return schema.SkillCreate{
		Title:       title,
		ContentType: enums.SkillContentTypeText,
		Content:     &content,
		Scope:       scope,
	}
}

func ptr[T any](v T) *T { return &v }

func TestDeleteSkill(t *testing.T) {
	// Keep AppSkillsDir inside the test sandbox so file-ownership assertions
	// never touch the real ~/.local/share/nexflow/skills.
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	fixture := newTestFixture(t)
	owner := fixture.createUser()
	other := fixture.createUser()
	ctx := context.Background()

	writeSkillMD := func(t *testing.T, dir string) string {
		t.Helper()
		require.NoError(t, os.MkdirAll(dir, 0o755))
		path := filepath.Join(dir, "SKILL.md")
		require.NoError(t, os.WriteFile(path, []byte("# Skill"), 0o600))
		return path
	}
	createStorageSkill := func(t *testing.T, title, dir string) uuid.UUID {
		t.Helper()
		skill, err := fixture.skillService.CreateSkill(ctx, schema.SkillCreate{
			Title:       title,
			ContentType: enums.SkillContentTypeStorageUri,
			Scope:       enums.SkillScopeUser,
			StorageUri:  &dir,
		}, owner.ID)
		require.NoError(t, err)
		return skill.ID
	}

	cases := []struct {
		name string
		// setup returns the skill id and, for file skills, the SKILL.md path.
		setup    func(t *testing.T) (uuid.UUID, string)
		deleteBy uuid.UUID
		wantErr  error
		rowGone  bool
		fileGone bool // only checked when setup returned a file path
	}{
		{
			name: "user scope skill is deleted",
			setup: func(t *testing.T) (uuid.UUID, string) {
				return createTextSkill(t, fixture, owner.ID, "del-user-skill", enums.SkillScopeUser).ID, ""
			},
			deleteBy: owner.ID,
			rowGone:  true,
		},
		{
			name: "app scope skill is deleted",
			setup: func(t *testing.T) (uuid.UUID, string) {
				return createTextSkill(t, fixture, owner.ID, "del-app-skill", enums.SkillScopeApp).ID, ""
			},
			deleteBy: owner.ID,
			rowGone:  true,
		},
		{
			name: "skill owned by another user survives",
			setup: func(t *testing.T) (uuid.UUID, string) {
				return createTextSkill(t, fixture, other.ID, "del-foreign-skill", enums.SkillScopeUser).ID, ""
			},
			deleteBy: owner.ID,
			wantErr:  errors.SkillNotFound,
			rowGone:  false,
		},
		{
			name: "file outside the skills dir survives",
			setup: func(t *testing.T) (uuid.UUID, string) {
				dir := filepath.Join(t.TempDir(), "outside-skill")
				file := writeSkillMD(t, dir)
				return createStorageSkill(t, "del-outside-skill", dir), file
			},
			deleteBy: owner.ID,
			rowGone:  true,
			fileGone: false,
		},
		{
			name: "file inside the skills dir is removed",
			setup: func(t *testing.T) (uuid.UUID, string) {
				dir := filepath.Join(AppSkillsDir(), "inside-skill")
				file := writeSkillMD(t, dir)
				return createStorageSkill(t, "del-inside-skill", dir), file
			},
			deleteBy: owner.ID,
			rowGone:  true,
			fileGone: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			id, file := tc.setup(t)

			err := fixture.skillService.DeleteSkill(ctx, tc.deleteBy, id)
			if tc.wantErr != nil {
				require.ErrorIs(t, err, tc.wantErr)
			} else {
				require.NoError(t, err)
			}

			// A failed delete must never report success while the row survives,
			// and a successful delete must actually remove the row.
			_, getErr := fixture.skillService.GetSkill(ctx, id)
			if tc.rowGone {
				require.Error(t, getErr)
			} else {
				require.NoError(t, getErr)
			}

			if file != "" {
				_, statErr := os.Stat(file)
				if tc.fileGone {
					require.ErrorIs(t, statErr, os.ErrNotExist)
				} else {
					require.NoError(t, statErr, "user file must survive deletion")
				}
			}
		})
	}
}
