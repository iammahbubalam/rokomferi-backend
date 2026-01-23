package usecase

import (
	"context"
	"fmt"
	"rokomferi-backend/internal/domain"
	"rokomferi-backend/pkg/utils"
)

type OrderUsecase struct {
	orderRepo   domain.OrderRepository
	productRepo domain.ProductRepository
	couponRepo  domain.CouponRepository // L9: Added CouponRepo
	txManager   domain.TransactionManager
}

func NewOrderUsecase(repo domain.OrderRepository, pRepo domain.ProductRepository, cRepo domain.CouponRepository, txManager domain.TransactionManager) *OrderUsecase {
	return &OrderUsecase{
		orderRepo:   repo,
		productRepo: pRepo,
		couponRepo:  cRepo,
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

func (u *OrderUsecase) AddToCart(ctx context.Context, userID string, productID string, variantID *string, quantity int) (*domain.Cart, error) {
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
	items, err := u.orderRepo.UpsertCartItemAtomic(ctx, userID, productID, variantID, newTotal)
	if err != nil || len(items) == 0 {
		if variantID != nil {
			return nil, fmt.Errorf("insufficient stock or product unavailable for variant %s", *variantID)
		} else {
			return nil, fmt.Errorf("insufficient stock or product unavailable")
		}
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

func (u *OrderUsecase) UpdateCartItemQuantity(ctx context.Context, userID string, productID string, variantID *string, quantity int) (*domain.Cart, error) {
	if quantity <= 0 {
		return u.RemoveFromCart(ctx, userID, productID)
	}

	// 🔥 ATOMIC: 1 DB CALL DOES EVERYTHING 🔥
	items, err := u.orderRepo.UpsertCartItemAtomic(ctx, userID, productID, variantID, quantity)
	if err != nil {
		return nil, fmt.Errorf("failed to update cart: %w", err)
	}

	// If no items returned, the atomic operation didn't insert/update anything
	// This means EITHER user has no cart OR product stock is insufficient
	if len(items) == 0 {
		// Simplified error since we rely on atomic query
		return nil, fmt.Errorf("unable to update cart: insufficient stock or invalid item")
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
	Address    domain.JSONB      `json:"address"`
	Payment    string            `json:"paymentMethod"`
	Items      []CheckoutItemReq `json:"items,omitempty"` // Optional: For Direct Checkout
	CouponCode string            `json:"couponCode,omitempty"`
}

type CheckoutItemReq struct {
	ProductID string `json:"productId"`
	Quantity  int    `json:"quantity"`
}

// ApplyCouponResp represents the result of applying a coupon
type ApplyCouponResp struct {
	Valid          bool    `json:"valid"`
	Code           string  `json:"code"`
	DiscountAmount float64 `json:"discountAmount"`
	NewTotal       float64 `json:"newTotal"`
	Message        string  `json:"message"`
}

func (u *OrderUsecase) ApplyCoupon(ctx context.Context, userID, code string) (*ApplyCouponResp, error) {
	// 1. Get Cart
	cart, err := u.GetMyCart(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get cart")
	}

	// 2. Calculate Subtotal
	var subtotal float64
	for _, item := range cart.Items {
		price := item.Product.BasePrice
		if item.Product.SalePrice != nil {
			price = *item.Product.SalePrice
		}
		subtotal += price * float64(item.Quantity)
	}

	// 3. Validate Coupon
	res, err := u.couponRepo.ValidateCoupon(ctx, code, subtotal)
	if err != nil {
		// If validation query returns no rows or error
		return &ApplyCouponResp{Valid: false, Message: "Invalid coupon code"}, nil
	}

	if res.ValidationStatus != "valid" {
		return &ApplyCouponResp{Valid: false, Message: fmt.Sprintf("Coupon is %s", res.ValidationStatus), Code: code}, nil
	}

	// 4. Calculate Discount
	discount := 0.0
	if res.Type == "percentage" {
		discount = subtotal * (res.Value / 100)
	} else {
		discount = res.Value
	}

	// Cap discount at subtotal (no negative total)
	if discount > subtotal {
		discount = subtotal
	}

	return &ApplyCouponResp{
		Valid:          true,
		Code:           code,
		DiscountAmount: discount,
		NewTotal:       subtotal - discount,
		Message:        "Coupon applied successfully",
	}, nil
}

func (u *OrderUsecase) Checkout(ctx context.Context, userID string, req CheckoutReq) (*domain.Order, error) {
	// 1. Determine Source (Direct vs Cart)
	var processItems []domain.CartItem
	isDirect := false
	var cartID string

	if len(req.Items) > 0 {
		isDirect = true
		for _, item := range req.Items {
			processItems = append(processItems, domain.CartItem{
				ProductID: item.ProductID,
				Quantity:  item.Quantity,
			})
		}
	} else {
		cart, err := u.GetMyCart(ctx, userID)
		if err != nil || len(cart.Items) == 0 {
			return nil, fmt.Errorf("cart is empty")
		}
		processItems = cart.Items
		cartID = cart.ID
	}

	// 2. Calculate Total & Prepare Order Items
	var total float64
	var orderItems []domain.OrderItem

	for _, item := range processItems {
		product, err := u.productRepo.GetProductByID(ctx, item.ProductID)
		if err != nil {
			return nil, fmt.Errorf("product %s not found", item.ProductID)
		}

		// Verify Variant & Pricing
		var price float64
		// Default to product price
		price = product.BasePrice
		if product.SalePrice != nil {
			price = *product.SalePrice
		}

		// Find relevant variant logic
		var targetVariantID string
		if item.VariantID != nil {
			targetVariantID = *item.VariantID
		}

		// L9 Fix: Iterate variants to find price override and validate ID
		foundVariant := false
		if len(product.Variants) > 0 {
			// If target is empty, use first/default
			if targetVariantID == "" {
				targetVariantID = product.Variants[0].ID
			}
			for _, v := range product.Variants {
				if v.ID == targetVariantID {
					foundVariant = true
					// Check for variant price override
					if v.Price != nil {
						price = *v.Price
					}
					// Check for variant sale price
					if v.SalePrice != nil {
						price = *v.SalePrice
					}
					break
				}
			}
		} else {
			// No variants? Should not happen with SSOT backfill, but handle gracefully
			if targetVariantID == "" {
				return nil, fmt.Errorf("product %s has no inventory variants", product.Name)
			}
		}

		if !foundVariant && len(product.Variants) > 0 {
			return nil, fmt.Errorf("variant %s not found for product %s", targetVariantID, product.Name)
		}

		total += price * float64(item.Quantity)

		// Use local variable for safe pointer
		variantIDPtr := &targetVariantID

		orderItems = append(orderItems, domain.OrderItem{
			ID:        utils.GenerateUUID(),
			ProductID: item.ProductID,
			VariantID: variantIDPtr,
			Quantity:  item.Quantity,
			Price:     price,
		})
	}

	// 3. Apply Coupon (if provided)
	if req.CouponCode != "" {
		// Re-validate strictly inside checkout
		res, err := u.couponRepo.ValidateCoupon(ctx, req.CouponCode, total)
		if err != nil || res.ValidationStatus != "valid" {
			return nil, fmt.Errorf("invalid or expired coupon: %s", req.CouponCode)
		}

		discount := 0.0
		if res.Type == "percentage" {
			discount = total * (res.Value / 100)
		} else {
			discount = res.Value
		}
		if discount > total {
			discount = total
		}
		total -= discount

		// Note: We might want to store the discount/coupon in the order payload for records.
		// Since we didn't add columns to Orders table yet, we assume TotalAmount reflects the discounted price.
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

	// 4. Transaction: Create Order, Update Stock, Increment Coupon, Clear Cart
	err := u.txManager.Do(ctx, func(txCtx context.Context) error {
		if err := u.orderRepo.CreateOrder(txCtx, order); err != nil {
			return err
		}

		// Update Stock
		for _, item := range order.Items {
			if item.VariantID == nil {
				return fmt.Errorf("item %s has no variant ID", item.ProductID)
			}
			if err := u.productRepo.UpdateStock(txCtx, *item.VariantID, -item.Quantity, "order_placed", order.ID); err != nil {
				return err
			}
		}

		// Increment Coupon Logic
		if req.CouponCode != "" {
			// We need the ID. ValidateCoupon gives us the result.
			// Ideally we fetch it or use the result from earlier if we refactored.
			// For safety/speed reuse the fetch:
			coupon, err := u.couponRepo.GetCouponByCode(txCtx, req.CouponCode)
			if err == nil {
				u.couponRepo.IncrementCouponUsage(txCtx, coupon.ID)
			}
		}

		// Clear Cart
		if !isDirect && cartID != "" {
			if err := u.orderRepo.ClearCart(txCtx, cartID); err != nil {
				return err
			}
		}
		return nil
	})

	if err != nil {
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
