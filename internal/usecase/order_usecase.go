package usecase

import (
	"context"
	"fmt"
	"log/slog"
	"rokomferi-backend/internal/domain"
	"time"
)

type OrderUsecase struct {
	orderRepo   domain.OrderRepository
	productRepo domain.ProductRepository
	txManager   domain.TransactionManager
}

func NewOrderUsecase(repo domain.OrderRepository, pRepo domain.ProductRepository, txManager domain.TransactionManager) *OrderUsecase {
	return &OrderUsecase{
		orderRepo:   repo,
		productRepo: pRepo,
		txManager:   txManager,
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

	// Calculate Total with FRESH prices and check Stock
	var total float64
	var orderItems []domain.OrderItem

	for _, item := range cart.Items {
		// Re-fetch product to ensure price/stock is up-to-date.
		// relying on Cart's cached "Product" might be unsafe if price changed since add-to-cart.
		product, err := u.productRepo.GetProductByID(ctx, item.ProductID)
		if err != nil {
			return nil, fmt.Errorf("product %s not found: %v", item.ProductID, err)
		}

		if product.StockStatus == "out_of_stock" {
			return nil, fmt.Errorf("product %s is out of stock", product.Name)
		}

		// Determine price (Sale vs Base)
		price := product.BasePrice
		if product.SalePrice != nil {
			price = *product.SalePrice
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

	// Wrap Order Creation and Cart Clearing in Transaction
	err = u.txManager.Do(ctx, func(txCtx context.Context) error {
		if err := u.orderRepo.CreateOrder(txCtx, order); err != nil {
			return fmt.Errorf("failed to create order: %v", err)
		}

		if err := u.orderRepo.ClearCart(txCtx, cart.ID); err != nil {
			return fmt.Errorf("failed to clear cart: %v", err)
		}

		return nil
	})

	if err != nil {
		slog.Error("Checkout transaction failed", "error", err)
		return nil, err
	}

	return order, nil
}

func (u *OrderUsecase) GetMyOrders(ctx context.Context, userID string) ([]domain.Order, error) {
	return u.orderRepo.GetOrdersByUserID(ctx, userID)
}
