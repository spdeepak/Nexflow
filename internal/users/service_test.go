package users

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	cfg "github.com/spdeepak/nexflow/internal/config"
	"github.com/spdeepak/nexflow/internal/db"
)

func newTestService(t *testing.T) Service {
	t.Helper()

	dbConfig := cfg.DBConfig{
		DBName:          "app.db",
		MaxOpenConns:    1,
		MaxIdleConns:    1,
		ConnMaxLifetime: time.Hour,
		ConnMaxIdleTime: 30 * time.Minute,
	}

	require.NoError(t, db.RunMigrations(dbConfig))
	t.Cleanup(func() {
		_ = os.Remove("app.db")
		_ = os.Remove("app.db-shm")
		_ = os.Remove("app.db-wal")
	})

	return NewService(New(db.Connect(dbConfig)))
}

func TestEnsureAppUser_CreatesOnFirstRun(t *testing.T) {
	svc := newTestService(t)

	user, err := svc.EnsureAppUser(context.Background(), "Q9KDVVQ5R9", "Deepak")
	require.NoError(t, err)
	require.Equal(t, "Q9KDVVQ5R9", user.ExternalID)
	require.Equal(t, "Deepak", user.Name)
	require.NotEqual(t, uuid.Nil, user.ID, "the user row must have its own uuid key, not the device serial")
}

func TestEnsureAppUser_IsIdempotentForTheSameDevice(t *testing.T) {
	svc := newTestService(t)

	first, err := svc.EnsureAppUser(context.Background(), "Q9KDVVQ5R9", "Deepak")
	require.NoError(t, err)

	second, err := svc.EnsureAppUser(context.Background(), "Q9KDVVQ5R9", "Renamed")
	require.NoError(t, err)

	require.Equal(t, first.ID, second.ID, "an existing device must not get a second user row")
	require.Equal(t, first.Name, second.Name, "the stored name wins over the supplied one")
	require.Equal(t, first.CreatedAt, second.CreatedAt)
}

func TestEnsureAppUser_KeepsDevicesDistinct(t *testing.T) {
	svc := newTestService(t)

	first, err := svc.EnsureAppUser(context.Background(), "SERIAL-ONE", "Deepak")
	require.NoError(t, err)
	second, err := svc.EnsureAppUser(context.Background(), "SERIAL-TWO", "Someone Else")
	require.NoError(t, err)

	require.NotEqual(t, first.ID, second.ID)
	require.NotEqual(t, first.ExternalID, second.ExternalID)
}

func TestCreateAppUser_GeneratesIDIndependentOfExternalID(t *testing.T) {
	svc := newTestService(t)

	user, err := svc.CreateAppUser(context.Background(), "Q9KDVVQ5R9", "Deepak")
	require.NoError(t, err)

	require.Equal(t, "Q9KDVVQ5R9", user.ExternalID)
	require.NotEqual(t, "Q9KDVVQ5R9", user.ID.String(), "the primary key must not be the device serial")
}
