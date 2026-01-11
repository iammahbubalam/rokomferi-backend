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
	err := getDB(ctx, r.db).
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
	return getDB(ctx, r.db).Create(cart).Error
}

func (r *orderRepository) UpdateCart(ctx context.Context, cart *domain.Cart) error {
	// This simple implementation saves the cart and its items.
	// For full sync, we might need to delete old items or handle merge logic.
	// Here we assume "cart" passed in has the Latest State of items.
	return getDB(ctx, r.db).Session(&gorm.Session{FullSaveAssociations: true}).Save(cart).Error
}

func (r *orderRepository) ClearCart(ctx context.Context, cartID string) error {
	return getDB(ctx, r.db).Where("cart_id = ?", cartID).Delete(&domain.CartItem{}).Error
}

// --- Order Implementation ---

func (r *orderRepository) CreateOrder(ctx context.Context, order *domain.Order) error {
	return getDB(ctx, r.db).Create(order).Error
}

func (r *orderRepository) GetByID(ctx context.Context, id string) (*domain.Order, error) {
	var order domain.Order
	if err := r.db.WithContext(ctx).Preload("Items").Preload("Items.Product").First(&order, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &order, nil
}

func (r *orderRepository) GetByUserID(ctx context.Context, userID string) ([]domain.Order, error) {
	var orders []domain.Order
	if err := r.db.WithContext(ctx).Preload("Items").Where("user_id = ?", userID).Order("created_at desc").Find(&orders).Error; err != nil {
		return nil, err
	}
	return orders, nil
}

// --- Admin Implementation ---

func (r *orderRepository) GetAll(ctx context.Context, page, limit int, status string) ([]domain.Order, int64, error) {
	var orders []domain.Order
	var total int64

	query := r.db.WithContext(ctx).Model(&domain.Order{})
	if status != "" {
		query = query.Where("status = ?", status)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	offset := (page - 1) * limit
	if err := query.Preload("Items").Order("created_at desc").Limit(limit).Offset(offset).Find(&orders).Error; err != nil {
		return nil, 0, err
	}

	return orders, total, nil
}

func (r *orderRepository) UpdateStatus(ctx context.Context, id, status string) error {
	return r.db.WithContext(ctx).Model(&domain.Order{}).Where("id = ?", id).Update("status", status).Error
}
