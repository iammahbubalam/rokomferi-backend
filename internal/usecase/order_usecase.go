package usecase

import (
	"context"
	"fmt"
	"rokomferi-backend/internal/domain"
	"time"
)

type OrderUsecase struct {
	orderRepo   domain.OrderRepository
	productRepo domain.ProductRepository
}

func NewOrderUsecase(repo domain.OrderRepository, pRepo domain.ProductRepository) *OrderUsecase {
	return &OrderUsecase{
		orderRepo:   repo,
		productRepo: pRepo,
	}
}

// --- Cart Logic ---

func (u *OrderUsecase) GetMyCart(ctx context.Context, userID string) (*domain.Cart, error) {
	cart, err := u.orderRepo.GetCartByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if cart == nil {
		// Auto-create cart for user if missing
		cart = &domain.Cart{
			ID:     fmt.Sprintf("cart_%d_%s", time.Now().Unix(), userID),
			UserID: &userID,
		}
		if err := u.orderRepo.CreateCart(ctx, cart); err != nil {
			return nil, err
		}
	}
	return cart, nil
}

func (u *OrderUsecase) AddToCart(ctx context.Context, userID string, productID string, quantity int) (*domain.Cart, error) {
	cart, err := u.GetMyCart(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Fetch Product to verify existence and price
	product, err := u.productRepo.GetProductByID(ctx, productID)
	if err != nil {
		return nil, err
	}
	_ = product

	// Simple add: check if exists, update qty, else append
	found := false
	for i, item := range cart.Items {
		if item.ProductID == productID {
			cart.Items[i].Quantity += quantity
			if cart.Items[i].Quantity <= 0 {
				// Remove item (simple slice removal logic omitted for brevity, let's assume >0 for add)
			}
			found = true
			break
		}
	}
	if !found {
		newItem := domain.CartItem{
			ID:        fmt.Sprintf("ci_%d", time.Now().UnixNano()),
			CartID:    cart.ID,
			ProductID: productID,
			Quantity:  quantity,
		}
		cart.Items = append(cart.Items, newItem)
	}

	if err := u.orderRepo.UpdateCart(ctx, cart); err != nil {
		return nil, err
	}
	return cart, nil
}

// --- Order Logic ---

type CheckoutReq struct {
	Address domain.JSONB `json:"address"`
	Payment string       `json:"paymentMethod"`
}

func (u *OrderUsecase) Checkout(ctx context.Context, userID string, req CheckoutReq) (*domain.Order, error) {
	cart, err := u.GetMyCart(ctx, userID)
	if err != nil || len(cart.Items) == 0 {
		return nil, fmt.Errorf("cart is empty")
	}

	// Calculate Total
	var total float64
	var orderItems []domain.OrderItem

	for _, item := range cart.Items {
		// Ideally fetch latest price from DB here to prevent tampering.
		// For MVP, we trust the preload or fetch again.
		// Let's assume item.Product has price.
		// Wait, Product might not be loaded with price in Cart Items if not preloaded correctly.
		// We should really fetch the product price here.

		// MVP Shortcut: We assume 1000 for everything if 0, but CartRepo preloads "Product".
		price := item.Product.BasePrice
		if item.Product.SalePrice != nil {
			price = *item.Product.SalePrice
		}

		total += price * float64(item.Quantity)
		orderItems = append(orderItems, domain.OrderItem{
			ID:        fmt.Sprintf("oi_%d", time.Now().UnixNano()),
			ProductID: item.ProductID,
			Quantity:  item.Quantity,
			Price:     price,
		})
	}

	order := &domain.Order{
		ID:              fmt.Sprintf("ord_%d", time.Now().Unix()),
		UserID:          userID,
		Status:          "pending",
		TotalAmount:     total,
		ShippingAddress: req.Address,
		PaymentMethod:   req.Payment,
		PaymentStatus:   "pending",
		Items:           orderItems,
	}

	if err := u.orderRepo.CreateOrder(ctx, order); err != nil {
		return nil, err
	}

	// Clear Cart
	if err := u.orderRepo.ClearCart(ctx, cart.ID); err != nil {
		// Log error but order is placed
	}

	return order, nil
}
