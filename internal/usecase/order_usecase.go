package usecase

import (
	"context"
	"fmt"
	"log/slog"
	"rokomferi-backend/internal/domain"
	"rokomferi-backend/pkg/utils"
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
			ID:     utils.GenerateUUID(),
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
				// Remove item if quantity goes to 0 or below
				cart.Items = append(cart.Items[:i], cart.Items[i+1:]...)
			}
			found = true
			break
		}
	}
	if !found && quantity > 0 {
		newItem := domain.CartItem{
			ID:        utils.GenerateUUID(),
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

// RemoveFromCart removes a product from the user's cart
func (u *OrderUsecase) RemoveFromCart(ctx context.Context, userID string, productID string) (*domain.Cart, error) {
	cart, err := u.GetMyCart(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Find and remove the item
	for i, item := range cart.Items {
		if item.ProductID == productID {
			cart.Items = append(cart.Items[:i], cart.Items[i+1:]...)
			break
		}
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
			ID:        utils.GenerateUUID(),
			ProductID: item.ProductID,
			Quantity:  item.Quantity,
			Price:     price,
		})
	}

	order := &domain.Order{
		ID:              utils.GenerateUUID(),
		UserID:          userID,
		Status:          "pending",
		TotalAmount:     total,
		ShippingAddress: req.Address,
		PaymentMethod:   req.Payment,
		PaymentStatus:   "pending",
		Items:           orderItems,
	}

	// Wrap Order Creation, Cart Clearing, and Stock Update in Transaction
	err = u.txManager.Do(ctx, func(txCtx context.Context) error {
		// 1. Create Order
		if err := u.orderRepo.CreateOrder(txCtx, order); err != nil {
			return fmt.Errorf("failed to create order: %v", err)
		}

		// 2. Decrement Stock for each item
		for _, item := range order.Items {
			// Decrease stock: pass negative quantity
			if err := u.productRepo.UpdateStock(txCtx, item.ProductID, -item.Quantity, "order_placed", order.ID); err != nil {
				return fmt.Errorf("failed to update stock for %s: %v", item.ProductID, err)
			}
		}

		// 3. Clear Cart
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
	return u.orderRepo.GetByUserID(ctx, userID)
}

// --- Admin Usecase ---

func (u *OrderUsecase) GetAllOrders(ctx context.Context, page, limit int, status string) ([]domain.Order, int64, error) {
	return u.orderRepo.GetAll(ctx, page, limit, status)
}

func (u *OrderUsecase) UpdateOrderStatus(ctx context.Context, orderID, newStatus string) error {
	// 1. Get existing order to check current status and items
	order, err := u.orderRepo.GetByID(ctx, orderID)
	if err != nil {
		return err
	}

	// 2. Prevent invalid transitions (optional, simple check for now)
	if order.Status == "cancelled" {
		return fmt.Errorf("cannot update a cancelled order")
	}

	// 3. Handle Stock Reconciliation
	// If we are cancelling, we must RESTOCK.
	if newStatus == "cancelled" && order.Status != "cancelled" {
		// Use TransactionManager? ideally yes.
		return u.txManager.Do(ctx, func(txCtx context.Context) error {
			// Update Status
			if err := u.orderRepo.UpdateStatus(txCtx, orderID, newStatus); err != nil {
				return err
			}
			// Restock Items
			for _, item := range order.Items {
				// Positive quantity to add back
				if err := u.productRepo.UpdateStock(txCtx, item.ProductID, item.Quantity, "order_cancelled", orderID); err != nil {
					return fmt.Errorf("failed to restock %s: %v", item.ProductID, err)
				}
			}
			return nil
		})
	}

	// Simple status update for other cases (e.g. processing -> shipped)
	return u.orderRepo.UpdateStatus(ctx, orderID, newStatus)
}
