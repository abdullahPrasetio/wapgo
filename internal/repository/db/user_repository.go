package db

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/abdullahPrasetio/wapgo/internal/domain/entity"
	domainrepo "github.com/abdullahPrasetio/wapgo/internal/domain/repository"
)

// userRepository sengaja unexported agar caller tidak bisa bergantung ke
// struct konkret ini — mereka hanya bisa memakai tipe interface-nya.
type userRepository struct {
	db *gorm.DB
}

// NewUserRepository membuat implementasi Postgres dari domainrepo.UserRepository.
// Return type-nya adalah interface (bukan *userRepository) supaya caller
// tidak perlu tahu bahwa di baliknya ada Postgres — cukup pakai kontraknya.
func NewUserRepository(db *gorm.DB) domainrepo.UserRepository {
	return &userRepository{db: db}
}

func (r *userRepository) FindByID(ctx context.Context, id uuid.UUID) (*entity.User, error) {
	var user entity.User
	result := r.db.WithContext(ctx).First(&user, "id = ?", id)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, gorm.ErrRecordNotFound
	}
	if result.Error != nil {
		return nil, fmt.Errorf("find user by id: %w", result.Error)
	}
	return &user, nil
}

func (r *userRepository) FindAll(ctx context.Context) ([]*entity.User, error) {
	var users []*entity.User
	if err := r.db.WithContext(ctx).Find(&users).Error; err != nil {
		return nil, fmt.Errorf("find all users: %w", err)
	}
	return users, nil
}

func (r *userRepository) Create(ctx context.Context, user *entity.User) error {
	if err := r.db.WithContext(ctx).Create(user).Error; err != nil {
		return fmt.Errorf("create user: %w", err)
	}
	return nil
}

func (r *userRepository) Update(ctx context.Context, user *entity.User) error {
	if err := r.db.WithContext(ctx).Save(user).Error; err != nil {
		return fmt.Errorf("update user: %w", err)
	}
	return nil
}

func (r *userRepository) Delete(ctx context.Context, id uuid.UUID) error {
	if err := r.db.WithContext(ctx).Delete(&entity.User{}, "id = ?", id).Error; err != nil {
		return fmt.Errorf("delete user: %w", err)
	}
	return nil
}

func (r *userRepository) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&entity.User{}).
		Where("email = ?", email).
		Count(&count).Error
	if err != nil {
		return false, fmt.Errorf("check email existence: %w", err)
	}
	return count > 0, nil
}
