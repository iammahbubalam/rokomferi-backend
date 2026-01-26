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
	Address         domain.JSONB      `json:"address"`
	Payment         string            `json:"paymentMethod"`
	Items           []CheckoutItemReq `json:"items,omitempty"` // Optional: For Direct Checkout
	CouponCode      string            `json:"couponCode,omitempty"`
	PaymentTrxID    string            `json:"paymentTrxId,omitempty"`
	PaymentProvider string            `json:"paymentProvider,omitempty"`
	PaymentPhone    string            `json:"paymentPhone,omitempty"`
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

	// 2. Calculate Total & Prepare Order Items & Check Pre-order
	var total float64
	var orderItems []domain.OrderItem
	var preOrderDepositRequired float64
	isPreOrderOrder := false

	// TODO: Get from Config/DB (Admin Setting)
	const PreOrderPercentage = 0.50

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
					// Update VariantID for checking stock status if variant had one (currently Product level)
					// Product.StockStatus is global for now.
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

		itemTotal := price * float64(item.Quantity)
		total += itemTotal

		// Pre-order Calculation
		if product.StockStatus == "pre_order" {
			isPreOrderOrder = true
			preOrderDepositRequired += itemTotal * PreOrderPercentage
		}

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

	// 4. Pre-Order Validation
	paymentDetails := domain.JSONB{}
	paymentStatus := "pending"
	paidAmount := 0.0

	if isPreOrderOrder && preOrderDepositRequired > 0 {
		if req.PaymentTrxID == "" || req.PaymentProvider == "" || req.PaymentPhone == "" {
			return nil, fmt.Errorf("pre-order items require partial payment info (TrxID, Provider, Phone)")
		}
		paymentStatus = "pending_verification"
		paidAmount = preOrderDepositRequired

		// Map details
		detailsMap := map[string]interface{}{
			"provider":         req.PaymentProvider,
			"transaction_id":   req.PaymentTrxID,
			"sender_number":    req.PaymentPhone,
			"deposit_required": preOrderDepositRequired,
		}
		paymentDetails = domain.JSONB(detailsMap)
	}

	order := &domain.Order{
		ID:              utils.GenerateUUID(),
		UserID:          userID,
		Status:          "pending",
		TotalAmount:     total,
		ShippingAddress: req.Address,
		PaymentMethod:   req.Payment,
		PaymentStatus:   paymentStatus,
		PaidAmount:      paidAmount,
		IsPreOrder:      isPreOrderOrder,
		PaymentDetails:  paymentDetails,
		Items:           orderItems,
	}

	// 5. Transaction: Create Order, Update Stock, Increment Coupon, Clear Cart
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

func (u *OrderUsecase) GetAllOrders(ctx context.Context, filter domain.OrderFilter) ([]domain.Order, int64, error) {
	return u.orderRepo.GetAll(ctx, filter)
}

func (u *OrderUsecase) GetOrder(ctx context.Context, id string) (*domain.Order, error) {
	return u.orderRepo.GetByID(ctx, id)
}

func (u *OrderUsecase) UpdateOrderStatus(ctx context.Context, orderID, newStatus, note, actorID string) error {
	// 1. Get existing order
	order, err := u.orderRepo.GetByID(ctx, orderID)
	if err != nil {
		return err
	}

	oldStatus := order.Status

	// Terminal Status Check - REMOVED for Admin flexibility
	// Admins need to be able to correct mistakes (e.g. accidentally marked as Fake/Cancelled).
	// if order.Status == domain.OrderStatusCancelled ||
	//    order.Status == domain.OrderStatusRefunded ||
	//    order.Status == domain.OrderStatusReturned ||
	//    order.Status == domain.OrderStatusFake {
	// 	return fmt.Errorf("cannot update order with terminal status: %s", order.Status)
	// }

	// 2. Validate Transition (L9 State Machine)
	if err := u.validateOrderTransition(order, newStatus); err != nil {
		return err
	}

	return u.txManager.Do(ctx, func(txCtx context.Context) error {
		// 3. Handle Side Effects (Stock, Payment, Analytics triggers)
		if err := u.handleOrderStateSideEffects(txCtx, order, newStatus, actorID); err != nil {
			return err
		}

		// 4. Update Status
		if err := u.orderRepo.UpdateStatus(txCtx, orderID, newStatus); err != nil {
			return err
		}

		// 5. Create History Entry
		// Determine Reason
		finalReason := note
		if finalReason == "" {
			finalReason = fmt.Sprintf("System: Status changed from %s to %s", oldStatus, newStatus)
		}

		var reasonPtr *string
		if finalReason != "" {
			reasonPtr = &finalReason
		}

		history := domain.OrderHistory{
			OrderID:        orderID,
			PreviousStatus: &oldStatus,
			NewStatus:      newStatus,
			Reason:         reasonPtr,
			CreatedBy:      &actorID,
		}
		if err := u.orderRepo.CreateOrderHistory(txCtx, &history); err != nil {
			return fmt.Errorf("failed to record history: %w", err)
		}

		return nil
	})
}

// L9: Strict State Transition Rules
func (u *OrderUsecase) validateOrderTransition(order *domain.Order, newStatus string) error {
	current := order.Status

	// Identity transition (no-op)
	if current == newStatus {
		return nil
	}

	// Define Allowed Transitions Graph
	// Key: Current Status -> Value: Allowed Next Statuses
	validTransitions := map[string][]string{
		domain.OrderStatusPending:             {domain.OrderStatusProcessing, domain.OrderStatusCancelled, domain.OrderStatusFake},
		domain.OrderStatusPendingVerification: {domain.OrderStatusProcessing, domain.OrderStatusCancelled, domain.OrderStatusFake},
		domain.OrderStatusProcessing:          {domain.OrderStatusShipped, domain.OrderStatusCancelled, domain.OrderStatusFake},
		domain.OrderStatusShipped:             {domain.OrderStatusDelivered, domain.OrderStatusReturned, domain.OrderStatusFake, domain.OrderStatusCancelled}, // Cancelled allowed if shipment recalled
		domain.OrderStatusDelivered:           {domain.OrderStatusPaid, domain.OrderStatusReturned},                                                           // Paid is the success path after Delivery for COD
		domain.OrderStatusPaid:                {domain.OrderStatusReturned, domain.OrderStatusRefunded},                                                       // Can only return/refund after valid payment

		// Terminal-ish states (Admin can mostly reset if needed, but with caution)
		domain.OrderStatusCancelled: {domain.OrderStatusPending},                               // Allow reactivation
		domain.OrderStatusRefunded:  {domain.OrderStatusPending},                               // Allow correction/reactivation
		domain.OrderStatusReturned:  {domain.OrderStatusPending, domain.OrderStatusProcessing}, // Reprocess
		domain.OrderStatusFake:      {domain.OrderStatusPending},                               // Correction
	}

	allowed, exists := validTransitions[current]
	// If current status not in map (e.g. legacy status), allow cautious admin override
	if !exists {
		// Log warning? For now, if we are robust, we block unknown states.
		// But let's allow "Manual Admin Override" if strictly needed?
		// User said "Strictly implement logic". So we BLOCK.
		return fmt.Errorf("invalid current logic state: %s", current)
	}

	isAllowed := false
	for _, s := range allowed {
		if s == newStatus {
			isAllowed = true
			break
		}
	}

	if !isAllowed {
		return fmt.Errorf("invalid transition: cannot go from %s to %s", current, newStatus)
	}

	return nil
}

// L9: Side Effects Handler
func (u *OrderUsecase) handleOrderStateSideEffects(ctx context.Context, order *domain.Order, newStatus, actorID string) error {
	// A. Stock Management
	// Restock conditions: Cancelled, Returned, Fake, Refunded (if considered a return)
	// Note: Move TO Refunded is usually handled by ProcessRefund, but if done manually here, we assume it implies restock?
	// Let's keep manual move to Refunded as NO-OP for stock unless explicitly known.
	// Actually, if we allow manual move TO Refunded, we should probably Restock.
	// But `ProcessRefund` does that.
	// Limit: UpdateStatus to 'refunded' manually here implies we just change label.
	// But we are focusing on REACTIVATION here.

	isRestockTarget := newStatus == domain.OrderStatusCancelled ||
		newStatus == domain.OrderStatusReturned ||
		newStatus == domain.OrderStatusFake

	// If we are MOVING TO a restock state FROM a non-restock state
	// (Check if we weren't already cancelled/returned to prevent double restock if re-applying)
	wasRestocked := order.Status == domain.OrderStatusCancelled ||
		order.Status == domain.OrderStatusReturned ||
		order.Status == domain.OrderStatusFake ||
		order.Status == domain.OrderStatusRefunded // L9 Fix: Refunded implies stock returns usually

	if isRestockTarget && !wasRestocked {
		reason := fmt.Sprintf("auto-restock: %s", newStatus)
		for _, item := range order.Items {
			targetID := item.ProductID
			if item.VariantID != nil {
				targetID = *item.VariantID
			}
			if err := u.productRepo.UpdateStock(ctx, targetID, item.Quantity, reason, order.ID); err != nil {
				return fmt.Errorf("failed to restock %s: %v", targetID, err)
			}
		}
	}

	// Deduct Stock if reactivating (e.g. Cancelled -> Pending)
	if wasRestocked && !isRestockTarget {
		reason := fmt.Sprintf("stock-deduct: reactivation to %s", newStatus)
		for _, item := range order.Items {
			targetID := item.ProductID
			if item.VariantID != nil {
				targetID = *item.VariantID
			}
			// Use negative quantity to deduct
			if err := u.productRepo.UpdateStock(ctx, targetID, -item.Quantity, reason, order.ID); err != nil {
				return fmt.Errorf("failed to deduct stock %s: %v", targetID, err)
			}
		}
	}

	// B. Payment Status Synchronization
	// Forward: Delivered -> Paid
	if newStatus == domain.OrderStatusPaid {
		// Only auto-mark as paid if not already paid
		if order.PaymentStatus != domain.PaymentStatusPaid {
			if err := u.orderRepo.UpdatePaymentStatus(ctx, order.ID, domain.PaymentStatusPaid); err != nil {
				return err
			}
			// Should we set PaidAmount to TotalAmount? Secure approach: Yes.
			// Assuming we have a SetPaidAmount repo method... if not, skip for now or rely on separate payment flow.
			// Ideally update PaidAmount column too.
		}
	}

	return nil
}

// VerifyOrderPayment verifies a pre-order payment
func (u *OrderUsecase) VerifyOrderPayment(ctx context.Context, orderID, adminID string) error {
	order, err := u.orderRepo.GetByID(ctx, orderID)
	if err != nil {
		return err
	}

	if !order.IsPreOrder {
		return fmt.Errorf("order is not a pre-order")
	}
	if order.Status != "pending_verification" {
		return fmt.Errorf("order status is %s, cannot verify payment", order.Status)
	}

	oldStatus := order.Status

	// Atomic Update: Status -> processing, PaymentStatus -> partial_paid
	return u.txManager.Do(ctx, func(txCtx context.Context) error {
		if err := u.orderRepo.UpdateStatus(txCtx, orderID, "processing"); err != nil {
			return err
		}
		if err := u.orderRepo.UpdatePaymentStatus(txCtx, orderID, "partial_paid"); err != nil {
			return err
		}

		// Record History
		reason := "Payment Verified"
		history := &domain.OrderHistory{
			OrderID:        orderID,
			PreviousStatus: &oldStatus,
			NewStatus:      "processing",
			Reason:         &reason,
			CreatedBy:      &adminID,
		}
		if err := u.orderRepo.CreateOrderHistory(txCtx, history); err != nil {
			return err
		}

		return nil
	})
}

// UpdatePaymentStatus updates the payment status of an order manually
func (u *OrderUsecase) UpdatePaymentStatus(ctx context.Context, orderID, newStatus, actorID string) error {
	order, err := u.orderRepo.GetByID(ctx, orderID)
	if err != nil {
		return err
	}

	oldPaymentStatus := order.PaymentStatus
	if oldPaymentStatus == newStatus {
		return nil
	}

	return u.txManager.Do(ctx, func(txCtx context.Context) error {
		if err := u.orderRepo.UpdatePaymentStatus(txCtx, orderID, newStatus); err != nil {
			return err
		}

		// Record History
		reason := fmt.Sprintf("Payment status changed: %s -> %s", oldPaymentStatus, newStatus)
		history := &domain.OrderHistory{
			OrderID:        orderID,
			PreviousStatus: &order.Status, // Status didn't change
			NewStatus:      order.Status,
			Reason:         &reason,
			CreatedBy:      &actorID,
		}
		if err := u.orderRepo.CreateOrderHistory(txCtx, history); err != nil {
			return err
		}
		return nil
	})
}

// ProcessRefund handles the refund logic
func (u *OrderUsecase) ProcessRefund(ctx context.Context, orderID string, amount float64, reason string, restock bool, adminID string) error {
	// 1. Get Order
	order, err := u.orderRepo.GetByID(ctx, orderID)
	if err != nil {
		return err
	}

	// 2. Validate Refund
	if amount <= 0 {
		return fmt.Errorf("refund amount must be positive")
	}
	remainingRefundable := order.PaidAmount - order.RefundedAmount
	if amount > remainingRefundable {
		return fmt.Errorf("cannot refund %.2f (max refundable: %.2f)", amount, remainingRefundable)
	}

	// Determine if status should change (e.g. if full refund -> refunded)
	// For now, partial refund doesn't change order status usually, but full refund might.
	// L9: Let's assume explicit status change is handled separately via UpdateStatus,
	// OR if restock is true, maybe we should mark as refunded?
	// Current logic just updates refunded amount.
	// Ensure we log this action.

	// 3. Execute Transaction
	return u.txManager.Do(ctx, func(txCtx context.Context) error {
		// Create Refund & Update Order
		if err := u.orderRepo.CreateRefund(txCtx, orderID, amount, reason, restock, &adminID); err != nil {
			return err
		}

		// Handle Stock Restoration
		if restock {
			for _, item := range order.Items {
				targetID := item.ProductID
				if item.VariantID != nil {
					targetID = *item.VariantID
				}
				// Restock
				if err := u.productRepo.UpdateStock(txCtx, targetID, item.Quantity, "refund_restock", orderID); err != nil {
					return fmt.Errorf("failed to restock item %s: %v", targetID, err)
				}
			}
		}

		// Record History (Log the refund action)
		// Previous status is same as current if we don't change it.
		// We log "refunded" action but status might remain "delivered" or "cancelled".
		// Let's log it as a status update if we change status, but here we are just refunding money.
		// But the audit log is `order_history` which tracks status.
		// Maybe we just log a "note" entry? The schema requires new_status.
		// Let's use current status as new_status but add reason "Refunded X Amount".

		histReason := fmt.Sprintf("Refunded %.2f: %s", amount, reason)
		history := &domain.OrderHistory{
			OrderID:        orderID,
			PreviousStatus: &order.Status,
			NewStatus:      order.Status, // Status didn't change unless we force it
			Reason:         &histReason,
			CreatedBy:      &adminID,
		}
		// If it was a full refund and restock, maybe auto-set to refunded?
		// User asked for robust, so automation is good.
		if restock && amount >= remainingRefundable {
			history.NewStatus = domain.OrderStatusRefunded
			if err := u.orderRepo.UpdateStatus(txCtx, orderID, domain.OrderStatusRefunded); err != nil {
				return err
			}
		}

		if err := u.orderRepo.CreateOrderHistory(txCtx, history); err != nil {
			return err
		}

		return nil
	})
}

// GetOrderHistory retrieves the history logs for an order
func (u *OrderUsecase) GetOrderHistory(ctx context.Context, orderID string) ([]domain.OrderHistory, error) {
	return u.orderRepo.GetOrderHistory(ctx, orderID)
}
