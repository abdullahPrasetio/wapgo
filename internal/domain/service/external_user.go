package service

import (
	"context"

	"github.com/abdullahPrasetio/wapgo/internal/domain/entity"
)

// ExternalUserService is the interface for fetching user data from another microservice.
// Implementation lives in pkg/httpclient/user_client.go (Fase v0.3).
type ExternalUserService interface {
	GetUser(ctx context.Context, id string) (*entity.User, error)
}
