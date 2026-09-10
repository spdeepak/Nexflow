package device

import (
	"context"

	"github.com/google/uuid"
)

type (
	service struct {
		querier Querier
	}
	Service interface {
		GetDeviceDetail(ctx context.Context) uuid.UUID
	}
)

func NewService(querier Querier) Service {
	return &service{
		querier: querier,
	}
}

func (s *service) GetDeviceDetail(ctx context.Context) uuid.UUID {
	id, err := s.querier.GetDeviceDetail(ctx)
	if err == nil {
		return id
	}
	id, _ = s.querier.AddDevice(ctx, uuid.New())
	return id
}
