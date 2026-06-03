package entity

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// User is the GORM model and domain entity for the user aggregate.
type User struct {
	ID        uuid.UUID      `gorm:"type:varchar(36);primaryKey;not null" json:"id"`
	Name      string         `gorm:"size:100;not null"              json:"name"`
	Email     string         `gorm:"size:255;uniqueIndex;not null"  json:"email"`
	Password  string         `gorm:"size:255;not null"              json:"-"` // never expose hash
	CreatedAt time.Time      `                                      json:"created_at"`
	UpdatedAt time.Time      `                                      json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index"                          json:"-"`
}

// BeforeCreate sets a random UUID if the ID is not already set.
func (u *User) BeforeCreate(_ *gorm.DB) error {
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}
	return nil
}

// TableName overrides GORM's default table naming.
func (User) TableName() string { return "users" }
