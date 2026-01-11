package domain

import (
	"context"
	"time"
)

// --- Cart Entities ---

type Cart struct {
	ID        string     `json:"id" gorm:"primaryKey"`
	UserID    *string    `json:"userId"` // Optional: guest carts could be supported, but for now we link to User if logged in
	Items     []CartItem `json:"items" gorm:"foreignKey:CartID"`
	CreatedAt time.Time  `json:"createdAt"`
	UpdatedAt time.Time  `json:"updatedAt"`
}

type CartItem struct {
	ID        string  `json:"id" gorm:"primaryKey"`
	CartID    string  `json:"cartId"`
	ProductID string  `json:"productId"`
	Product   Product `json:"product" gorm:"foreignKey:ProductID"`
	VariantID *string `json:"variantId"`
	Quantity  int     `json:"quantity"`
}

// --- Order Entities ---

type Order struct {
	ID              string      `json:"id" gorm:"primaryKey"`
	UserID          string      `json:"userId"`
	User            User        `json:"user" gorm:"foreignKey:UserID"`
	Status          string      `json:"status"` // pending, processing, shipped, delivered, cancelled
	TotalAmount     float64     `json:"totalAmount"`
	ShippingAddress JSONB       `json:"shippingAddress" gorm:"type:jsonb"`
	PaymentMethod   string      `json:"paymentMethod"`
	PaymentStatus   string      `json:"paymentStatus"`
	Items           []OrderItem `json:"items" gorm:"foreignKey:OrderID"`
	CreatedAt       time.Time   `json:"createdAt"`
	UpdatedAt       time.Time   `json:"updatedAt"`
}

type OrderItem struct {
	ID        string  `json:"id" gorm:"primaryKey"`
	OrderID   string  `json:"orderId"`
	ProductID string  `json:"productId"`
	Product   Product `json:"product" gorm:"foreignKey:ProductID"`
	VariantID *string `json:"variantId"`
	Quantity  int     `json:"quantity"`
	Price     float64 `json:"price"` // Price at time of purchase
}

// --- Interfaces ---

type OrderRepository interface {
	// Cart
	GetCartByUserID(ctx context.Context, userID string) (*Cart, error)
	CreateCart(ctx context.Context, cart *Cart) error
	UpdateCart(ctx context.Context, cart *Cart) error // Updates/Adds items
	ClearCart(ctx context.Context, cartID string) error

	// Order
	CreateOrder(ctx context.Context, order *Order) error
	GetOrderByID(ctx context.Context, id string) (*Order, error)
	GetOrdersByUserID(ctx context.Context, userID string) ([]Order, error)
}
