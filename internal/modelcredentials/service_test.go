package modelcredentials

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	cfg "github.com/spdeepak/nexflow/internal/config"
	"github.com/spdeepak/nexflow/internal/db"
	"github.com/spdeepak/nexflow/internal/enums"
	"github.com/spdeepak/nexflow/internal/users"
	"github.com/spdeepak/nexflow/schema"
)

type (
	testFixture struct {
		test         *testing.T
		userService  users.Service
		modelService Service
	}
)

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
		userService:  users.NewService(users.New(conn)),
		modelService: NewService(New(conn)),
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

func (tf *testFixture) createAppModel() schema.ModelCredential {
	tf.test.Helper()
	arg := schema.ModelCredentialCreate{
		ApiKey:      "schema-key",
		BaseUrl:     "http://localhost:11434/v1",
		ExtraConfig: schema.ModelCredentialCreateExtraConfig{},
		Provider:    "ollama",
		ModelName:   "minimax-m3:cloud",
		Scope:       enums.CredentialScopeApp,
		Title:       "test ollama minimax",
	}
	model, err := tf.modelService.CreateAppModelCredential(context.Background(), arg)
	require.NoError(tf.test, err)
	require.NotNil(tf.test, model)
	return model
}

func (tf *testFixture) createUserModel(user schema.User) schema.ModelCredential {
	tf.test.Helper()
	arg := schema.ModelCredentialCreate{
		ApiKey:      "schema-key",
		BaseUrl:     "http://localhost:11434/v1",
		ExtraConfig: schema.ModelCredentialCreateExtraConfig{},
		Provider:    "ollama",
		ModelName:   "minimax-m3:cloud",
		Scope:       enums.CredentialScopeApp,
		Title:       "test ollama minimax",
	}
	model, err := tf.modelService.CreateUserModelCredential(context.Background(), user.ID, arg)
	require.NoError(tf.test, err)
	require.NotNil(tf.test, model)
	return model
}

func TestCreateAppModel_OK(t *testing.T) {
	tf := newTestFixture(t)
	tf.createUser()
	tf.createAppModel()
}

func TestCreateUserModel_OK(t *testing.T) {
	tf := newTestFixture(t)
	tf.createUser()
	tf.createUserModel(tf.createUser())
}

func TestCreateAppModel_ConstraintCheckErr(t *testing.T) {
	query := NewMockQuerier(t)
	query.EXPECT().CreateAppModelCredential(mock.Anything, mock.Anything).Return(CreateAppModelCredentialRow{}, fmt.Errorf("model_credentials_owner_check"))

	modelService := NewService(query)
	arg := schema.ModelCredentialCreate{
		ApiKey:      "schema-key",
		BaseUrl:     "http://localhost:11434/v1",
		ExtraConfig: schema.ModelCredentialCreateExtraConfig{},
		Provider:    "ollama",
		ModelName:   "minimax-m3:cloud",
		Scope:       enums.CredentialScopeApp,
		Title:       "test ollama minimax",
	}
	credential, err := modelService.CreateAppModelCredential(context.Background(), arg)
	assert.Equal(t, err.Error(), "app level model credential cannot not be linked to a user")
	assert.Empty(t, credential)
}

func TestCreateAppModel_Err(t *testing.T) {
	query := NewMockQuerier(t)
	query.EXPECT().CreateAppModelCredential(mock.Anything, mock.Anything).Return(CreateAppModelCredentialRow{}, fmt.Errorf("error"))

	modelService := NewService(query)
	arg := schema.ModelCredentialCreate{
		ApiKey:      "schema-key",
		BaseUrl:     "http://localhost:11434/v1",
		ExtraConfig: schema.ModelCredentialCreateExtraConfig{},
		Provider:    "ollama",
		ModelName:   "minimax-m3:cloud",
		Scope:       enums.CredentialScopeApp,
		Title:       "test ollama minimax",
	}
	credential, err := modelService.CreateAppModelCredential(context.Background(), arg)
	assert.Error(t, err)
	assert.Empty(t, credential)
}
