package postgres

import (
	"context"
	"errors"
	"rokomferi-backend/internal/domain"

	"gorm.io/gorm"
)

type userRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) domain.UserRepository {
	// Auto Migrate the schema including new tables
	if err := db.AutoMigrate(&domain.User{}, &domain.Address{}, &domain.RefreshToken{}); err != nil {
		panic(err)
	}
	return &userRepository{db: db}
}

func (r *userRepository) Create(ctx context.Context, user *domain.User) error {
	return r.db.WithContext(ctx).Create(user).Error
}

func (r *userRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	var user domain.User
	if err := r.db.WithContext(ctx).Where("email = ?", email).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

func (r *userRepository) GetByID(ctx context.Context, id string) (*domain.User, error) {
	var user domain.User
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// --- Addresses ---

func (r *userRepository) AddAddress(ctx context.Context, addr *domain.Address) error {
	return r.db.WithContext(ctx).Create(addr).Error
}

func (r *userRepository) GetAddresses(ctx context.Context, userID string) ([]domain.Address, error) {
	var addrs []domain.Address
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).Find(&addrs).Error
	return addrs, err
}

// --- Refresh Tokens ---

func (r *userRepository) SaveRefreshToken(ctx context.Context, token *domain.RefreshToken) error {
	return r.db.WithContext(ctx).Create(token).Error
}

func (r *userRepository) GetRefreshToken(ctx context.Context, token string) (*domain.RefreshToken, error) {
	var rt domain.RefreshToken
	err := r.db.WithContext(ctx).Where("token = ?", token).First(&rt).Error
	return &rt, err
}

func (r *userRepository) RevokeRefreshToken(ctx context.Context, token string) error {
	// We can either delete it or set revoked=true.
	// User preferences: secure but simple. Deleting is simplest revocation.
	// But maintaining history (Revoked=true) is 7/10 better for auditing.
	// Let's delete for "Simple", but Wait, "revoked" field exists in struct. So let's use it.
	return r.db.WithContext(ctx).Model(&domain.RefreshToken{}).
		Where("token = ?", token).
		Update("revoked", true).Error
}
