package users

import (
	"context"

	"github.com/google/uuid"

	"github.com/spdeepak/nexflow/internal/schema"
)

type (
	service struct {
		query Querier
	}

	Service interface {
		CreateUser(ctx context.Context, arg CreateUserParams) (CreateUserRow, error)
		CreateAppUser(ctx context.Context, deviceId uuid.UUID, name string) (schema.User, error)
		DeleteUser(ctx context.Context, id uuid.UUID) error
		GetUser(ctx context.Context, id uuid.UUID) (GetUserRow, error)
		GetUserByExternalID(ctx context.Context, externalID string) (GetUserByExternalIDRow, error)
	}
)

func NewService(query Querier) Service {
	return &service{
		query: query,
	}
}

func (s *service) CreateUser(ctx context.Context, arg CreateUserParams) (CreateUserRow, error) {
	//TODO implement me
	panic("implement me")
}

func (s *service) CreateAppUser(ctx context.Context, deviceId uuid.UUID, name string) (schema.User, error) {
	user := CreateUserParams{
		ID:         deviceId,
		ExternalID: deviceId.String(),
		Name:       name,
	}
	row, err := s.query.CreateUser(ctx, user)
	if err != nil {
		return schema.User{}, err
	}
	return schema.User{
		CreatedAt:  row.CreatedAt,
		ExternalID: row.ExternalID,
		ID:         row.ID,
		Name:       row.Name,
	}, nil
}

func (s *service) DeleteUser(ctx context.Context, id uuid.UUID) error {
	//TODO implement me
	panic("implement me")
}

func (s *service) GetUser(ctx context.Context, id uuid.UUID) (GetUserRow, error) {
	//TODO implement me
	panic("implement me")
}

func (s *service) GetUserByExternalID(ctx context.Context, externalID string) (GetUserByExternalIDRow, error) {
	return s.query.GetUserByExternalID(ctx, externalID)
}
