package users

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/spdeepak/nexflow/internal/schema"
)

type (
	service struct {
		query Querier
	}

	Service interface {
		// CreateAppUser registers the local app user identified by externalID
		// (the device serial number). The row primary key is generated; the
		// device id is only the external identity.
		CreateAppUser(ctx context.Context, externalID, name string) (schema.User, error)
		// EnsureAppUser returns the local app user for externalID, creating it
		// with name on first run.
		EnsureAppUser(ctx context.Context, externalID, name string) (schema.User, error)
		GetUserByExternalID(ctx context.Context, externalID string) (GetUserByExternalIDRow, error)
	}
)

func NewService(query Querier) Service {
	return &service{
		query: query,
	}
}

func (s *service) CreateAppUser(ctx context.Context, externalID, name string) (schema.User, error) {
	row, err := s.query.CreateUser(ctx, CreateUserParams{
		ID:         uuid.New(),
		ExternalID: externalID,
		Name:       name,
	})
	if err != nil {
		return schema.User{}, err
	}
	return newUser(row.ID, row.ExternalID, row.Name, row.CreatedAt), nil
}

func (s *service) EnsureAppUser(ctx context.Context, externalID, name string) (schema.User, error) {
	row, err := s.query.GetUserByExternalID(ctx, externalID)
	if err == nil {
		return newUser(row.ID, row.ExternalID, row.Name, row.CreatedAt), nil
	}
	return s.CreateAppUser(ctx, externalID, name)
}

func (s *service) GetUserByExternalID(ctx context.Context, externalID string) (GetUserByExternalIDRow, error) {
	return s.query.GetUserByExternalID(ctx, externalID)
}

func newUser(id uuid.UUID, externalID, name string, createdAt time.Time) schema.User {
	return schema.User{
		CreatedAt:  createdAt,
		ExternalID: externalID,
		ID:         id,
		Name:       name,
	}
}
