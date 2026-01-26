package sqlcrepo

import (
	"context"
	"encoding/json"
	"rokomferi-backend/db/sqlc"
	"rokomferi-backend/internal/domain"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type orderRepository struct {
	db      *pgxpool.Pool
	queries *sqlc.Queries
}

func NewOrderRepository(db *pgxpool.Pool) domain.OrderRepository {
	return &orderRepository{
		db:      db,
		queries: sqlc.New(db),
	}
}

// --- Mappers ---

func sqlcCartToDomain(c sqlc.Cart, items []sqlc.GetCartItemsRow) *domain.Cart {
	cart := &domain.Cart{
		ID:        uuidToString(c.ID),
		CreatedAt: pgtimeToTime(c.CreatedAt),
		UpdatedAt: pgtimeToTime(c.UpdatedAt),
	}
	if c.UserID.Valid {
		uid := uuidToString(c.UserID)
		cart.UserID = &uid
	}

	cart.Items = make([]domain.CartItem, len(items))
	for i, item := range items {
		cart.Items[i] = domain.CartItem{
			ID:        uuidToString(item.ID),
			CartID:    uuidToString(item.CartID),
			ProductID: uuidToString(item.ProductID),
			Quantity:  int(item.Quantity),
			Product: domain.Product{
				ID:        uuidToString(item.ProductID),
				Name:      item.Name,
				Slug:      item.Slug,
				BasePrice: numericToFloat64(item.BasePrice),
				SalePrice: numericToFloat64Ptr(item.SalePrice),
			},
		}
		if item.VariantID.Valid {
			vid := uuidToString(item.VariantID)
			cart.Items[i].VariantID = &vid
		}
		// Parse media for images
		if len(item.Media) > 0 {
			cart.Items[i].Product.Media = domain.RawJSON(item.Media)
			mapMediaToImages(&cart.Items[i].Product)
		}
	}
	return cart
}

func sqlcOrderToDomain(o sqlc.Order, items []sqlc.GetOrderItemsRow) *domain.Order {
	order := &domain.Order{
		ID:            uuidToString(o.ID),
		UserID:        uuidToString(o.UserID),
		Status:        o.Status,
		TotalAmount:   numericToFloat64(o.TotalAmount),
		PaymentMethod: ptrString(o.PaymentMethod),
		PaymentStatus: ptrString(o.PaymentStatus),
		PaidAmount:    numericToFloat64(o.PaidAmount),
		IsPreOrder:    o.IsPreorder,
		CreatedAt:     pgtimeToTime(o.CreatedAt),
		UpdatedAt:     pgtimeToTime(o.UpdatedAt),
	}

	if len(o.PaymentDetails) > 0 {
		var details domain.JSONB
		json.Unmarshal(o.PaymentDetails, &details)
		order.PaymentDetails = details
	}

	// Parse shipping address
	if len(o.ShippingAddress) > 0 {
		var addr domain.JSONB
		json.Unmarshal(o.ShippingAddress, &addr)
		order.ShippingAddress = addr
	}

	// ... items mapping (unchanged, will be included by context) ...
	order.Items = make([]domain.OrderItem, len(items))
	for i, item := range items {
		order.Items[i] = domain.OrderItem{
			ID:        uuidToString(item.ID),
			OrderID:   uuidToString(item.OrderID),
			ProductID: uuidToString(item.ProductID),
			Quantity:  int(item.Quantity),
			Price:     numericToFloat64(item.Price),
			Product: domain.Product{
				Name: item.Name,
				Slug: item.Slug,
			},
		}
		if item.VariantID.Valid {
			vid := uuidToString(item.VariantID)
			order.Items[i].VariantID = &vid
		}
		if len(item.Media) > 0 {
			order.Items[i].Product.Media = domain.RawJSON(item.Media)
			mapMediaToImages(&order.Items[i].Product)
		}
	}
	return order
}

// ...

// --- Cart Methods ---

func (r *orderRepository) GetCartByUserID(ctx context.Context, userID string) (*domain.Cart, error) {
	cart, err := r.queries.GetCartByUserID(ctx, stringToUUID(userID))
	if err != nil {
		if err.Error() == "no rows in result set" {
			return nil, nil
		}
		return nil, err
	}

	items, err := r.queries.GetCartItems(ctx, cart.ID)
	if err != nil {
		return nil, err
	}

	return sqlcCartToDomain(cart, items), nil
}

func (r *orderRepository) CreateCart(ctx context.Context, cart *domain.Cart) error {
	var userID pgtype.UUID
	if cart.UserID != nil {
		userID = stringToUUID(*cart.UserID)
	}

	created, err := r.queries.CreateCart(ctx, userID)
	if err != nil {
		return err
	}
	cart.ID = uuidToString(created.ID)
	cart.CreatedAt = pgtimeToTime(created.CreatedAt)
	cart.UpdatedAt = pgtimeToTime(created.UpdatedAt)
	return nil
}

func (r *orderRepository) GetCartWithItems(ctx context.Context, userID string) ([]domain.CartItem, error) {
	rows, err := r.queries.GetCartWithItems(ctx, stringToUUID(userID))
	if err != nil {
		return nil, err
	}

	items := make([]domain.CartItem, 0, len(rows))
	for _, row := range rows {
		// Skip rows where item_id is null (empty cart)
		if !row.ItemID.Valid {
			continue
		}

		item := domain.CartItem{
			ID:        uuidToString(row.ItemID),
			CartID:    uuidToString(row.CartID),
			ProductID: uuidToString(row.ProductID),
			Quantity:  int(*row.Quantity),
			Product: domain.Product{
				ID:        uuidToString(row.ProductID),
				Name:      *row.Name,
				Slug:      *row.Slug,
				BasePrice: numericToFloat64(row.BasePrice),
				SalePrice: numericToFloat64Ptr(row.SalePrice),
			},
		}

		if row.VariantID.Valid {
			vid := uuidToString(row.VariantID)
			item.VariantID = &vid
		}

		if len(row.Media) > 0 {
			item.Product.Media = domain.RawJSON(row.Media)
			mapMediaToImages(&item.Product)
		}

		items = append(items, item)
	}

	return items, nil
}

func (r *orderRepository) UpsertCartItemAtomic(ctx context.Context, userID, productID string, variantID *string, quantity int) ([]domain.CartItem, error) {
	var variantUUID pgtype.UUID
	if variantID != nil {
		variantUUID = stringToUUID(*variantID)
	}

	rows, err := r.queries.UpsertCartItemAtomic(ctx, sqlc.UpsertCartItemAtomicParams{
		UserID:    stringToUUID(userID),
		ProductID: stringToUUID(productID),
		VariantID: variantUUID,
		Quantity:  int32(quantity),
	})
	if err != nil {
		return nil, err
	}

	// Map rows to CartItems
	items := make([]domain.CartItem, len(rows))
	for i, row := range rows {
		items[i] = domain.CartItem{
			ID:        row.ID.String(),
			CartID:    row.CartID.String(),
			ProductID: row.ProductID.String(),
			Quantity:  int(row.Quantity),
			Product: domain.Product{
				ID:        row.ProductID.String(),
				Name:      row.Name,
				Slug:      row.Slug,
				BasePrice: numericToFloat64(row.BasePrice),
				SalePrice: numericToFloat64Ptr(row.SalePrice),
			},
		}
		if row.VariantID.Valid {
			vid := uuidToString(row.VariantID)
			items[i].VariantID = &vid
		}
		if len(row.Media) > 0 {
			items[i].Product.Media = domain.RawJSON(row.Media)
			mapMediaToImages(&items[i].Product)
		}
	}

	return items, nil
}

func (r *orderRepository) AtomicRemoveCartItem(ctx context.Context, userID, productID string) error {
	return r.queries.AtomicRemoveCartItem(ctx, sqlc.AtomicRemoveCartItemParams{
		UserID:    stringToUUID(userID),
		ProductID: stringToUUID(productID),
	})
}

func (r *orderRepository) ClearCart(ctx context.Context, cartID string) error {
	return r.queries.ClearCart(ctx, stringToUUID(cartID))
}

// --- Order Methods ---

func (r *orderRepository) CreateOrder(ctx context.Context, order *domain.Order) error {
	shippingAddrBytes, _ := json.Marshal(order.ShippingAddress)
	paymentDetailsBytes, _ := json.Marshal(order.PaymentDetails)

	created, err := r.queries.CreateOrder(ctx, sqlc.CreateOrderParams{
		UserID:          stringToUUID(order.UserID),
		Status:          order.Status,
		TotalAmount:     float64ToNumeric(order.TotalAmount),
		ShippingAddress: shippingAddrBytes,
		PaymentMethod:   strPtr(order.PaymentMethod),
		PaymentStatus:   strPtr(order.PaymentStatus),
		PaidAmount:      float64ToNumeric(order.PaidAmount),
		PaymentDetails:  paymentDetailsBytes,
		IsPreorder:      order.IsPreOrder,
	})
	if err != nil {
		return err
	}

	order.ID = uuidToString(created.ID)
	order.CreatedAt = pgtimeToTime(created.CreatedAt)
	order.UpdatedAt = pgtimeToTime(created.UpdatedAt)

	// Create order items
	for i := range order.Items {
		item := &order.Items[i]
		var variantID pgtype.UUID
		if item.VariantID != nil {
			variantID = stringToUUID(*item.VariantID)
		}

		createdItem, err := r.queries.CreateOrderItem(ctx, sqlc.CreateOrderItemParams{
			OrderID:   created.ID,
			ProductID: stringToUUID(item.ProductID),
			VariantID: variantID,
			Quantity:  int32(item.Quantity),
			Price:     float64ToNumeric(item.Price),
		})
		if err != nil {
			return err
		}
		item.ID = uuidToString(createdItem.ID)
		item.OrderID = order.ID
	}

	return nil
}

// ...

func (r *orderRepository) GetByID(ctx context.Context, id string) (*domain.Order, error) {
	order, err := r.queries.GetOrderByID(ctx, stringToUUID(id))
	if err != nil {
		return nil, err
	}

	items, err := r.queries.GetOrderItems(ctx, order.ID)
	if err != nil {
		return nil, err
	}

	return sqlcOrderToDomain(order, items), nil
}

func (r *orderRepository) GetByUserID(ctx context.Context, userID string) ([]domain.Order, error) {
	orders, err := r.queries.GetOrdersByUserID(ctx, stringToUUID(userID))
	if err != nil {
		return nil, err
	}

	result := make([]domain.Order, len(orders))
	for i, o := range orders {
		items, _ := r.queries.GetOrderItems(ctx, o.ID)
		order := sqlcOrderToDomain(o, items)
		result[i] = *order
	}
	return result, nil
}

// --- Admin Methods ---

func (r *orderRepository) GetAll(ctx context.Context, page, limit int, status string) ([]domain.Order, int64, error) {
	offset := (page - 1) * limit

	orders, err := r.queries.GetAllOrders(ctx, sqlc.GetAllOrdersParams{
		Column1: status,
		Limit:   int32(limit),
		Offset:  int32(offset),
	})
	if err != nil {
		return nil, 0, err
	}

	count, err := r.queries.CountOrders(ctx, status)
	if err != nil {
		return nil, 0, err
	}

	result := make([]domain.Order, len(orders))
	for i, o := range orders {
		result[i] = domain.Order{
			ID:            uuidToString(o.ID),
			UserID:        uuidToString(o.UserID),
			Status:        o.Status,
			TotalAmount:   numericToFloat64(o.TotalAmount),
			PaymentMethod: ptrString(o.PaymentMethod),
			PaymentStatus: ptrString(o.PaymentStatus),
			PaidAmount:    numericToFloat64(o.PaidAmount),
			IsPreOrder:    o.IsPreorder,
			CreatedAt:     pgtimeToTime(o.CreatedAt),
			UpdatedAt:     pgtimeToTime(o.UpdatedAt),
			User: domain.User{
				Email:     o.Email,
				FirstName: ptrString(o.FirstName),
				LastName:  ptrString(o.LastName),
			},
		}
		if len(o.PaymentDetails) > 0 {
			var details domain.JSONB
			json.Unmarshal(o.PaymentDetails, &details)
			result[i].PaymentDetails = details
		}
		if len(o.ShippingAddress) > 0 {
			var addr domain.JSONB
			json.Unmarshal(o.ShippingAddress, &addr)
			result[i].ShippingAddress = addr
		}
	}

	return result, count, nil
}

func (r *orderRepository) UpdateStatus(ctx context.Context, id, status string) error {
	return r.queries.UpdateOrderStatus(ctx, sqlc.UpdateOrderStatusParams{
		ID:     stringToUUID(id),
		Status: status,
	})
}

func (r *orderRepository) HasPurchasedProduct(ctx context.Context, userID, productID string) (bool, error) {
	return r.queries.HasPurchasedProduct(ctx, sqlc.HasPurchasedProductParams{
		UserID:    stringToUUID(userID),
		ProductID: stringToUUID(productID),
	})
}
