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

	// LOCK 2: API Gatekeeper - Strict Stock Validation

	// Check existing quantity in cart
	existingQty := 0
	for _, item := range cart.Items {
		if item.ProductID == productID {
			existingQty = item.Quantity
			break
		}
	}

	totalRequested := existingQty + quantity

	if product.Stock < totalRequested {
		return nil, fmt.Errorf("insufficient stock for %s (requested total: %d, available: %d)", product.Name, totalRequested, product.Stock)
	}
	if product.StockStatus == "out_of_stock" {
		return nil, fmt.Errorf("product %s is currently out of stock", product.Name)
	}

	// Optimized O(1) Upsert
	cartItem := domain.CartItem{
		Product:   *product,
		ProductID: productID,
		Quantity:  quantity,
	}

	// Helper to check current qty if adding relatively (logic simplification: assuming passed quantity is what to ADD)
	// But UpsertCartItem in SQL does: quantity = cart_items.quantity + EXCLUDED.quantity
	// So we just pass the delta 'quantity'.

	if err := u.orderRepo.UpsertCartItem(ctx, cart.ID, cartItem); err != nil {
		return nil, err
	}

	// Fetch fresh cart to return
	return u.GetMyCart(ctx, userID)
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
	Address domain.JSONB      `json:"address"`
	Payment string            `json:"paymentMethod"`
	Items   []CheckoutItemReq `json:"items,omitempty"` // Optional: For Direct Checkout
}

type CheckoutItemReq struct {
	ProductID string `json:"productId"`
	Quantity  int    `json:"quantity"`
}

func (u *OrderUsecase) Checkout(ctx context.Context, userID string, req CheckoutReq) (*domain.Order, error) {
	// 1. Determine Source (Direct vs Cart)
	var processItems []domain.CartItem
	isDirect := false
	var cartID string

	if len(req.Items) > 0 {
		// Direct Checkout Mode
		isDirect = true
		for _, item := range req.Items {
			processItems = append(processItems, domain.CartItem{
				ProductID: item.ProductID,
				Quantity:  item.Quantity,
			})
		}
	} else {
		// Normal Cart Checkout Mode
		cart, err := u.GetMyCart(ctx, userID)
		if err != nil || len(cart.Items) == 0 {
			return nil, fmt.Errorf("cart is empty")
		}
		processItems = cart.Items
		cartID = cart.ID
	}

	// Calculate Total with FRESH prices and check Stock
	var total float64
	var orderItems []domain.OrderItem

	for _, item := range processItems {
		// Re-fetch product to ensure price/stock is up-to-date.
		product, err := u.productRepo.GetProductByID(ctx, item.ProductID)
		if err != nil {
			return nil, fmt.Errorf("product %s not found: %v", item.ProductID, err)
		}

		if product.Stock < item.Quantity { // Explicit Quantity Check
			return nil, fmt.Errorf("product %s has insufficient stock (requested: %d, available: %d)", product.Name, item.Quantity, product.Stock)
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
	err := u.txManager.Do(ctx, func(txCtx context.Context) error {
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

		// 3. Clear Cart (ONLY if not direct checkout)
		if !isDirect && cartID != "" {
			if err := u.orderRepo.ClearCart(txCtx, cartID); err != nil {
				return fmt.Errorf("failed to clear cart: %v", err)
			}
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
