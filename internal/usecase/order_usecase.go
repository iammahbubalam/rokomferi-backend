package usecase

import (
	"context"
	"fmt"
	"log"
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
	// 🔥 OPTIMIZED: 1 DB call gets cart + all items with product details
	items, err := u.orderRepo.GetCartWithItems(ctx, userID)
	if err != nil {
		// If query failed due to no cart existing, create one
		if err.Error() == "no rows in result set" || err.Error() == "sql: no rows in result set" {
			cart := &domain.Cart{
				ID:     utils.GenerateUUID(),
				UserID: &userID,
			}
			if createErr := u.orderRepo.CreateCart(ctx, cart); createErr != nil {
				return nil, createErr
			}
			return cart, nil
		}
		return nil, err
	}

	// Build cart from items (use GetCartByUserID to get cart ID if needed)
	cart, cartErr := u.orderRepo.GetCartByUserID(ctx, userID)
	if cartErr != nil {
		// Shouldn't happen since GetCartWithItems succeeded, but handle it
		cart = &domain.Cart{
			ID:     utils.GenerateUUID(),
			UserID: &userID,
		}
		if createErr := u.orderRepo.CreateCart(ctx, cart); createErr != nil {
			return nil, createErr
		}
	}

	cart.Items = items
	return cart, nil
}

func (u *OrderUsecase) AddToCart(ctx context.Context, userID string, productID string, quantity int) (*domain.Cart, error) {
	// Get cart and existing quantity
	cart, err := u.GetMyCart(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Check existing quantity
	existingQty := 0
	for _, item := range cart.Items {
		if item.ProductID == productID {
			existingQty = item.Quantity
			break
		}
	}

	// Calculate new total
	newTotal := existingQty + quantity

	// Use atomic upsert with new total
	items, err := u.orderRepo.UpsertCartItemAtomic(ctx, userID, productID, nil, newTotal)
	if err != nil || len(items) == 0 {
		// Get product for better error message
		product, pErr := u.productRepo.GetProductByID(ctx, productID)
		if pErr != nil {
			return nil, fmt.Errorf("product not found")
		}
		if product.Stock < newTotal {
			return nil, fmt.Errorf("insufficient stock for %s (available: %d)", product.Name, product.Stock)
		}
		if product.StockStatus == "out_of_stock" {
			return nil, fmt.Errorf("product %s is out of stock", product.Name)
		}

		// Debug why atomic query failed
		log.Printf("DEBUG AddToCart: Cart exists (id=%s), Product exists (id=%s, stock=%d, stock_status=%v), Existing qty=%d, Adding=%d, New total=%d, but atomic query returned %d items",
			cart.ID, productID, product.Stock, product.StockStatus, existingQty, quantity, newTotal, len(items))

		return nil, fmt.Errorf("failed to add to cart")
	}

	// Return cart with all items
	return &domain.Cart{
		ID:     items[0].CartID,
		UserID: &userID,
		Items:  items,
	}, nil
}

// RemoveFromCart removes a product from the user's cart
func (u *OrderUsecase) RemoveFromCart(ctx context.Context, userID string, productID string) (*domain.Cart, error) {
	// Atomic O(1) Remove
	if err := u.orderRepo.AtomicRemoveCartItem(ctx, userID, productID); err != nil {
		return nil, err
	}

	// Fetch fresh cart to return
	return u.GetMyCart(ctx, userID)
}

func (u *OrderUsecase) UpdateCartItemQuantity(ctx context.Context, userID string, productID string, quantity int) (*domain.Cart, error) {
	if quantity <= 0 {
		return u.RemoveFromCart(ctx, userID, productID)
	}

	// 🔥 ATOMIC: 1 DB CALL DOES EVERYTHING 🔥
	items, err := u.orderRepo.UpsertCartItemAtomic(ctx, userID, productID, nil, quantity)
	if err != nil {
		return nil, fmt.Errorf("failed to update cart: %w", err)
	}

	// If no items returned, the atomic operation didn't insert/update anything
	// This means EITHER user has no cart OR product stock is insufficient
	if len(items) == 0 {
		// Check what actually failed
		// 1. Does user have a cart?
		cart, cartErr := u.GetMyCart(ctx, userID)
		if cartErr != nil {
			return nil, fmt.Errorf("cart not found")
		}

		// 2. Does product exist and have enough stock?
		product, pErr := u.productRepo.GetProductByID(ctx, productID)
		if pErr != nil {
			return nil, fmt.Errorf("product not found")
		}

		// 3. Stock validation
		if product.Stock < quantity {
			return nil, fmt.Errorf("insufficient stock (available: %d)", product.Stock)
		}
		if product.StockStatus == "out_of_stock" {
			return nil, fmt.Errorf("product is out of stock")
		}

		// If we got here, cart exists, product exists, stock is sufficient
		// But atomic query still failed - this is likely a bug in the SQL
		log.Printf("DEBUG: Cart exists (id=%s), Product exists (id=%s, stock=%d), Requested qty=%d, but atomic query returned 0 items",
			cart.ID, productID, product.Stock, quantity)
		return nil, fmt.Errorf("unable to update cart: cart_id=%s, product_id=%s", cart.ID, productID)
	}

	// Success! Build cart from returned items
	cart := &domain.Cart{
		UserID: &userID,
		Items:  items,
	}

	if len(items) > 0 {
		cart.ID = items[0].CartID
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
