package repository

import (
	"context"

	"github.com/google/uuid"

	"github.com/abdullahPrasetio/wapgo/internal/domain/entity"
)

// UserRepository defines persistence operations for the user domain.
// Implementation lives in internal/repository/postgres/.
type UserRepository interface {
	FindByID(ctx context.Context, id uuid.UUID) (*entity.User, error)
	FindAll(ctx context.Context) ([]*entity.User, error)
	Create(ctx context.Context, user *entity.User) error
	Update(ctx context.Context, user *entity.User) error
	Delete(ctx context.Context, id uuid.UUID) error
	ExistsByEmail(ctx context.Context, email string) (bool, error)
}
