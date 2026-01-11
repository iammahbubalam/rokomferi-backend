package postgres

import (
	"context"
	"rokomferi-backend/internal/domain"

	"gorm.io/gorm"
)

type orderRepository struct {
	db *gorm.DB
}

func NewOrderRepository(db *gorm.DB) domain.OrderRepository {
	db.AutoMigrate(&domain.Cart{}, &domain.CartItem{}, &domain.Order{}, &domain.OrderItem{})
	return &orderRepository{db: db}
}

// --- Cart Implementation ---

func (r *orderRepository) GetCartByUserID(ctx context.Context, userID string) (*domain.Cart, error) {
	var cart domain.Cart
	err := r.db.WithContext(ctx).
		Preload("Items").
		Preload("Items.Product").
		Where("user_id = ?", userID).
		First(&cart).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &cart, nil
}

func (r *orderRepository) CreateCart(ctx context.Context, cart *domain.Cart) error {
	return r.db.WithContext(ctx).Create(cart).Error
}

func (r *orderRepository) UpdateCart(ctx context.Context, cart *domain.Cart) error {
	// This simple implementation saves the cart and its items.
	// For full sync, we might need to delete old items or handle merge logic.
	// Here we assume "cart" passed in has the Latest State of items.
	return r.db.WithContext(ctx).Session(&gorm.Session{FullSaveAssociations: true}).Save(cart).Error
}

func (r *orderRepository) ClearCart(ctx context.Context, cartID string) error {
	return r.db.WithContext(ctx).Where("cart_id = ?", cartID).Delete(&domain.CartItem{}).Error
}

// --- Order Implementation ---

func (r *orderRepository) CreateOrder(ctx context.Context, order *domain.Order) error {
	return r.db.WithContext(ctx).Create(order).Error
}

func (r *orderRepository) GetOrderByID(ctx context.Context, id string) (*domain.Order, error) {
	var order domain.Order
	err := r.db.WithContext(ctx).Preload("Items.Product").First(&order, "id = ?", id).Error
	return &order, err
}

func (r *orderRepository) GetOrdersByUserID(ctx context.Context, userID string) ([]domain.Order, error) {
	var orders []domain.Order
	err := r.db.WithContext(ctx).Preload("Items").Where("user_id = ?", userID).Order("created_at DESC").Find(&orders).Error
	return orders, err
}
