package domain

import (
	"context"
	"time"
)

type User struct {
	ID        string    `json:"id" gorm:"primaryKey;type:varchar(50)"` // u_12345
	Email     string    `json:"email" gorm:"uniqueIndex;not null"`
	Role      string    `json:"role" gorm:"default:'customer'"`
	FirstName string    `json:"firstName"`
	LastName  string    `json:"lastName"`
	Avatar    string    `json:"avatar"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type UserRepository interface {
	Create(ctx context.Context, user *User) error
	GetByEmail(ctx context.Context, email string) (*User, error)
	GetByID(ctx context.Context, id string) (*User, error)
}
