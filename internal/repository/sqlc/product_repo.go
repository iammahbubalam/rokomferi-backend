package sqlcrepo

import (
	"context"
	"encoding/json"
	"fmt"
	"rokomferi-backend/db/sqlc"
	"rokomferi-backend/internal/domain"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type productRepository struct {
	db      *pgxpool.Pool
	queries *sqlc.Queries
}

func NewProductRepository(db *pgxpool.Pool) domain.ProductRepository {
	return &productRepository{
		db:      db,
		queries: sqlc.New(db),
	}
}

// --- Helpers ---

func numericToFloat64(n pgtype.Numeric) float64 {
	if !n.Valid {
		return 0
	}
	f, _ := n.Float64Value()
	return f.Float64
}

func float64ToNumeric(f float64) pgtype.Numeric {
	var n pgtype.Numeric
	n.Scan(strconv.FormatFloat(f, 'f', -1, 64))
	return n
}

func float64PtrToNumeric(f *float64) pgtype.Numeric {
	var n pgtype.Numeric
	if f != nil {
		n.Scan(strconv.FormatFloat(*f, 'f', -1, 64))
	}
	return n
}

func numericToFloat64Ptr(n pgtype.Numeric) *float64 {
	if !n.Valid {
		return nil
	}
	f, _ := n.Float64Value()
	val := f.Float64
	return &val
}

// --- Mappers ---

func sqlcProductToDomain(p sqlc.Product) domain.Product {
	prod := domain.Product{
		ID:                uuidToString(p.ID),
		Name:              p.Name,
		Slug:              p.Slug,
		SKU:               p.Sku,
		Description:       ptrString(p.Description),
		BasePrice:         numericToFloat64(p.BasePrice),
		SalePrice:         numericToFloat64Ptr(p.SalePrice),
		Stock:             int(p.Stock),
		StockStatus:       ptrString(p.StockStatus),
		LowStockThreshold: int(p.LowStockThreshold),
		IsFeatured:        p.IsFeatured,
		IsActive:          p.IsActive,
		CreatedAt:         pgtimeToTime(p.CreatedAt),
		UpdatedAt:         pgtimeToTime(p.UpdatedAt),
		MetaTitle:         ptrString(p.MetaTitle),
		MetaDescription:   ptrString(p.MetaDescription),
		Keywords:          ptrString(p.MetaKeywords),
		OGImage:           ptrString(p.OgImage),
		Brand:             ptrString(p.Brand),
		Tags:              p.Tags,
	}

	// Handle Media (JSONB)
	if len(p.Media) > 0 {
		prod.Media = domain.RawJSON(p.Media)
		mapMediaToImages(&prod)
	}

	// Handle Attributes
	if len(p.Attributes) > 0 {
		var attrs domain.JSONB
		json.Unmarshal(p.Attributes, &attrs)
		prod.Attributes = attrs
	}

	// Handle Specs
	if len(p.Specifications) > 0 {
		var specs domain.JSONB
		json.Unmarshal(p.Specifications, &specs)
		prod.Specs = specs
	}

	// Handle Warranty Info
	if len(p.WarrantyInfo) > 0 {
		var warranty domain.JSONB
		json.Unmarshal(p.WarrantyInfo, &warranty)
		prod.WarrantyInfo = warranty
	}

	return prod
}

func mapMediaToImages(p *domain.Product) {
	if len(p.Media) == 0 {
		return
	}
	var arr []string
	if err := json.Unmarshal([]byte(p.Media), &arr); err == nil {
		p.Images = arr
		return
	}
	var obj map[string]interface{}
	if err := json.Unmarshal([]byte(p.Media), &obj); err == nil {
		if imgs, ok := obj["images"].([]interface{}); ok {
			p.Images = make([]string, len(imgs))
			for i, v := range imgs {
				if str, ok := v.(string); ok {
					p.Images[i] = str
				}
			}
		}
	}
}

func mapImagesToMedia(p *domain.Product) []byte {
	if p.Images != nil {
		bytes, _ := json.Marshal(p.Images)
		return bytes
	}
	return nil
}

func sqlcCategoryToDomain(c sqlc.Category) domain.Category {
	var parentID *string
	if c.ParentID.Valid {
		pid := uuidToString(c.ParentID)
		parentID = &pid
	}
	return domain.Category{
		ID:              uuidToString(c.ID),
		Name:            c.Name,
		Slug:            c.Slug,
		ParentID:        parentID,
		OrderIndex:      int(c.OrderIndex),
		Icon:            ptrString(c.Icon),
		Image:           ptrString(c.Image),
		IsActive:        c.IsActive,
		ShowInNav:       c.ShowInNav,
		IsFeatured:      c.IsFeatured,
		MetaTitle:       ptrString(c.MetaTitle),
		MetaDescription: ptrString(c.MetaDescription),
		Keywords:        ptrString(c.Keywords),
	}
}

func sqlcCollectionToDomain(c sqlc.Collection) domain.Collection {
	return domain.Collection{
		ID:              uuidToString(c.ID),
		Title:           c.Title,
		Slug:            c.Slug,
		Description:     ptrString(c.Description),
		Image:           ptrString(c.Image),
		Story:           ptrString(c.Story),
		IsActive:        c.IsActive,
		CreatedAt:       pgtimeToTime(c.CreatedAt),
		UpdatedAt:       pgtimeToTime(c.UpdatedAt),
		MetaTitle:       ptrString(c.MetaTitle),
		MetaDescription: ptrString(c.MetaDescription),
		Keywords:        ptrString(c.MetaKeywords),
		OGImage:         ptrString(c.OgImage),
	}
}

func sqlcVariantToDomain(v sqlc.Variant) domain.Variant {
	variant := domain.Variant{
		ID:        uuidToString(v.ID),
		ProductID: uuidToString(v.ProductID),
		Name:      v.Name,
		Stock:     int(v.Stock),
		SKU:       ptrString(v.Sku),
		Price:     numericToFloat64Ptr(v.Price),
		SalePrice: numericToFloat64Ptr(v.SalePrice),
		Images:    v.Images,
		Weight:    numericToFloat64Ptr(v.Weight),
		Barcode:   ptrString(v.Barcode),
	}

	if len(v.Attributes) > 0 {
		var attrs domain.JSONB
		json.Unmarshal(v.Attributes, &attrs)
		variant.Attributes = attrs
	}

	if len(v.Dimensions) > 0 {
		var dims domain.JSONB
		json.Unmarshal(v.Dimensions, &dims)
		variant.Dimensions = dims
	}

	return variant
}

func sqlcInventoryLogToDomain(l sqlc.InventoryLog) domain.InventoryLog {
	var variantID *string
	if l.VariantID.Valid {
		vid := uuidToString(l.VariantID)
		variantID = &vid
	}
	return domain.InventoryLog{
		ID:           uint(l.ID),
		ProductID:    uuidToString(l.ProductID),
		VariantID:    variantID,
		ChangeAmount: int(l.ChangeAmount),
		Reason:       l.Reason,
		ReferenceID:  l.ReferenceID,
		CreatedAt:    pgtimeToTime(l.CreatedAt),
	}
}

func sqlcReviewToDomain(r sqlc.Review) domain.Review {
	return domain.Review{
		ID:        uuidToString(r.ID),
		ProductID: uuidToString(r.ProductID),
		UserID:    uuidToString(r.UserID),
		Rating:    int(r.Rating),
		Comment:   ptrString(r.Comment),
		CreatedAt: pgtimeToTime(r.CreatedAt),
	}
}

// --- Category Methods ---

func (r *productRepository) GetCategoryTree(ctx context.Context) ([]domain.Category, error) {
	return r.buildCategoryTree(ctx, false)
}

func (r *productRepository) GetNavCategoryTree(ctx context.Context) ([]domain.Category, error) {
	return r.buildCategoryTree(ctx, true)
}

func (r *productRepository) buildCategoryTree(ctx context.Context, navOnly bool) ([]domain.Category, error) {
	var roots []sqlc.Category
	var err error

	if navOnly {
		roots, err = r.queries.GetActiveNavCategories(ctx)
	} else {
		roots, err = r.queries.GetRootCategories(ctx)
	}
	if err != nil {
		return nil, err
	}

	result := make([]domain.Category, len(roots))
	for i, root := range roots {
		cat := sqlcCategoryToDomain(root)
		cat.Children, _ = r.getChildrenRecursive(ctx, root.ID, navOnly, 3)
		result[i] = cat
	}
	return result, nil
}

func (r *productRepository) getChildrenRecursive(ctx context.Context, parentID pgtype.UUID, navOnly bool, depth int) ([]domain.Category, error) {
	if depth <= 0 {
		return nil, nil
	}

	var children []sqlc.Category
	var err error

	if navOnly {
		children, err = r.queries.GetActiveChildCategories(ctx, parentID)
	} else {
		children, err = r.queries.GetChildCategories(ctx, parentID)
	}
	if err != nil {
		return nil, err
	}

	result := make([]domain.Category, len(children))
	for i, child := range children {
		cat := sqlcCategoryToDomain(child)
		cat.Children, _ = r.getChildrenRecursive(ctx, child.ID, navOnly, depth-1)
		result[i] = cat
	}
	return result, nil
}

func (r *productRepository) GetCategoryBySlug(ctx context.Context, slug string) (*domain.Category, error) {
	c, err := r.queries.GetCategoryBySlug(ctx, slug)
	if err != nil {
		if err.Error() == "no rows in result set" {
			return nil, nil
		}
		return nil, err
	}
	cat := sqlcCategoryToDomain(c)
	return &cat, nil
}

func (r *productRepository) CreateCategory(ctx context.Context, category *domain.Category) error {
	var parentID pgtype.UUID
	if category.ParentID != nil {
		parentID = stringToUUID(*category.ParentID)
	}

	created, err := r.queries.CreateCategory(ctx, sqlc.CreateCategoryParams{
		Name:            category.Name,
		Slug:            category.Slug,
		ParentID:        parentID,
		OrderIndex:      int32(category.OrderIndex),
		Icon:            strPtr(category.Icon),
		Image:           strPtr(category.Image),
		IsActive:        category.IsActive,
		ShowInNav:       category.ShowInNav,
		MetaTitle:       strPtr(category.MetaTitle),
		MetaDescription: strPtr(category.MetaDescription),
		Keywords:        strPtr(category.Keywords),
		IsFeatured:      category.IsFeatured,
	})
	if err != nil {
		return err
	}
	category.ID = uuidToString(created.ID)
	return nil
}

func (r *productRepository) UpdateCategory(ctx context.Context, category *domain.Category) error {
	var parentID pgtype.UUID
	if category.ParentID != nil {
		parentID = stringToUUID(*category.ParentID)
	}

	_, err := r.queries.UpdateCategory(ctx, sqlc.UpdateCategoryParams{
		ID:              stringToUUID(category.ID),
		Name:            category.Name,
		Slug:            category.Slug,
		ParentID:        parentID,
		OrderIndex:      int32(category.OrderIndex),
		Icon:            strPtr(category.Icon),
		Image:           strPtr(category.Image),
		IsActive:        category.IsActive,
		ShowInNav:       category.ShowInNav,
		MetaTitle:       strPtr(category.MetaTitle),
		MetaDescription: strPtr(category.MetaDescription),
		Keywords:        strPtr(category.Keywords),
		IsFeatured:      category.IsFeatured,
	})
	return err
}

func (r *productRepository) DeleteCategory(ctx context.Context, id string) error {
	return r.queries.DeleteCategory(ctx, stringToUUID(id))
}

func (r *productRepository) ReorderCategories(ctx context.Context, updates []domain.CategoryReorderItem) error {
	for _, item := range updates {
		var parentID pgtype.UUID
		if item.ParentID != nil {
			parentID = stringToUUID(*item.ParentID)
		}
		err := r.queries.UpdateCategoryOrder(ctx, sqlc.UpdateCategoryOrderParams{
			ID:         stringToUUID(item.ID),
			OrderIndex: int32(item.OrderIndex),
			ParentID:   parentID,
		})
		if err != nil {
			return err
		}
	}
	return nil
}

// --- Collection Methods ---

func (r *productRepository) GetCollections(ctx context.Context) ([]domain.Collection, error) {
	cols, err := r.queries.GetActiveCollections(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]domain.Collection, len(cols))
	for i, c := range cols {
		result[i] = sqlcCollectionToDomain(c)
	}
	return result, nil
}

func (r *productRepository) GetAllCollections(ctx context.Context) ([]domain.Collection, error) {
	cols, err := r.queries.GetAllCollections(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]domain.Collection, len(cols))
	for i, c := range cols {
		result[i] = sqlcCollectionToDomain(c)
	}
	return result, nil
}

func (r *productRepository) GetCollectionBySlug(ctx context.Context, slug string) (*domain.Collection, error) {
	col, err := r.queries.GetCollectionBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}
	collection := sqlcCollectionToDomain(col)

	// Get products for collection
	products, err := r.queries.GetProductsForCollection(ctx, col.ID)
	if err == nil {
		collection.Products = make([]domain.Product, len(products))
		for i, p := range products {
			prod := sqlcProductToDomain(p)
			collection.Products[i] = prod
		}
	}

	return &collection, nil
}

func (r *productRepository) CreateCollection(ctx context.Context, collection *domain.Collection) error {
	created, err := r.queries.CreateCollection(ctx, sqlc.CreateCollectionParams{
		Title:           collection.Title,
		Slug:            collection.Slug,
		Description:     strPtr(collection.Description),
		Image:           strPtr(collection.Image),
		Story:           strPtr(collection.Story),
		IsActive:        collection.IsActive,
		MetaTitle:       strPtr(collection.MetaTitle),
		MetaDescription: strPtr(collection.MetaDescription),
		MetaKeywords:    strPtr(collection.Keywords),
		OgImage:         strPtr(collection.OGImage),
	})
	if err != nil {
		return err
	}
	collection.ID = uuidToString(created.ID)
	collection.CreatedAt = pgtimeToTime(created.CreatedAt)
	collection.UpdatedAt = pgtimeToTime(created.UpdatedAt)
	return nil
}

func (r *productRepository) UpdateCollection(ctx context.Context, collection *domain.Collection) error {
	_, err := r.queries.UpdateCollection(ctx, sqlc.UpdateCollectionParams{
		ID:              stringToUUID(collection.ID),
		Title:           collection.Title,
		Slug:            collection.Slug,
		Description:     strPtr(collection.Description),
		Image:           strPtr(collection.Image),
		Story:           strPtr(collection.Story),
		IsActive:        collection.IsActive,
		MetaTitle:       strPtr(collection.MetaTitle),
		MetaDescription: strPtr(collection.MetaDescription),
		MetaKeywords:    strPtr(collection.Keywords),
		OgImage:         strPtr(collection.OGImage),
	})
	return err
}

func (r *productRepository) DeleteCollection(ctx context.Context, id string) error {
	return r.queries.DeleteCollection(ctx, stringToUUID(id))
}

func (r *productRepository) AddProductToCollection(ctx context.Context, collectionID, productID string) error {
	return r.queries.AddProductToCollection(ctx, sqlc.AddProductToCollectionParams{
		ProductID:    stringToUUID(productID),
		CollectionID: stringToUUID(collectionID),
	})
}

func (r *productRepository) RemoveProductFromCollection(ctx context.Context, collectionID, productID string) error {
	return r.queries.RemoveProductFromCollection(ctx, sqlc.RemoveProductFromCollectionParams{
		ProductID:    stringToUUID(productID),
		CollectionID: stringToUUID(collectionID),
	})
}

// --- Product Methods ---

func (r *productRepository) GetProducts(ctx context.Context, filter domain.ProductFilter) ([]domain.Product, int64, error) {
	limit := int32(filter.Limit)
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	// Default to true if nil (show active products by default)
	var isActive *bool
	trueVal := true
	if filter.IsActive != nil {
		isActive = filter.IsActive
	} else {
		isActive = &trueVal
	}

	products, err := r.queries.GetProducts(ctx, sqlc.GetProductsParams{
		IsActive:   isActive,
		IsFeatured: filter.IsFeatured,
		Limit:      limit,
		Offset:     int32(filter.Offset),
	})
	if err != nil {
		return nil, 0, err
	}

	count, err := r.queries.CountProducts(ctx, sqlc.CountProductsParams{
		IsActive:   isActive,
		IsFeatured: filter.IsFeatured,
	})
	if err != nil {
		return nil, 0, err
	}

	result := make([]domain.Product, len(products))
	for i, p := range products {
		result[i] = sqlcProductToDomain(p)
	}

	return result, count, nil
}

func (r *productRepository) GetProductBySlug(ctx context.Context, slug string) (*domain.Product, error) {
	p, err := r.queries.GetProductBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}
	prod := sqlcProductToDomain(p)

	// Load variants
	variants, _ := r.queries.GetVariantsByProductID(ctx, p.ID)
	prod.Variants = make([]domain.Variant, len(variants))
	for i, v := range variants {
		prod.Variants[i] = sqlcVariantToDomain(v)
	}

	// Load categories (Optimized Batch Fetch)
	catIDs, _ := r.queries.GetCategoryIDsForProduct(ctx, p.ID)
	if len(catIDs) > 0 {
		// Fetch all categories in 1 query
		cats, err := r.queries.GetCategoriesByIDs(ctx, catIDs)
		if err != nil {
			// Log error but don't fail, return empty categories
			// or we could return error. For now, best effort.
			return nil, err
		}

		prod.Categories = make([]domain.Category, len(cats))
		for i, c := range cats {
			prod.Categories[i] = sqlcCategoryToDomain(c)
		}
	} else {
		prod.Categories = []domain.Category{}
	}

	return &prod, nil
}

func (r *productRepository) GetProductByID(ctx context.Context, id string) (*domain.Product, error) {
	p, err := r.queries.GetProductByID(ctx, stringToUUID(id))
	if err != nil {
		return nil, err
	}
	prod := sqlcProductToDomain(p)

	// Load variants
	variants, _ := r.queries.GetVariantsByProductID(ctx, p.ID)
	prod.Variants = make([]domain.Variant, len(variants))
	for i, v := range variants {
		prod.Variants[i] = sqlcVariantToDomain(v)
	}

	// Load categories (Optimized Batch Fetch)
	catIDs, _ := r.queries.GetCategoryIDsForProduct(ctx, p.ID)
	if len(catIDs) > 0 {
		cats, err := r.queries.GetCategoriesByIDs(ctx, catIDs)
		if err != nil {
			return nil, err
		}
		prod.Categories = make([]domain.Category, len(cats))
		for i, c := range cats {
			prod.Categories[i] = sqlcCategoryToDomain(c)
		}
	} else {
		prod.Categories = []domain.Category{}
	}

	return &prod, nil
}

func (r *productRepository) UpdateStock(ctx context.Context, productID string, quantity int, reason, referenceID string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	qtx := r.queries.WithTx(tx)

	// Update stock
	rows, err := qtx.UpdateProductStock(ctx, sqlc.UpdateProductStockParams{
		ID:    stringToUUID(productID),
		Stock: int32(quantity),
	})
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("insufficient stock or product not found: %s", productID)
	}

	// Create log
	var variantID pgtype.UUID
	_, err = qtx.CreateInventoryLog(ctx, sqlc.CreateInventoryLogParams{
		ProductID:    stringToUUID(productID),
		VariantID:    variantID,
		ChangeAmount: int32(quantity),
		Reason:       reason,
		ReferenceID:  referenceID,
	})
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (r *productRepository) GetInventoryLogs(ctx context.Context, productID string, limit, offset int) ([]domain.InventoryLog, int64, error) {
	var prodUUID pgtype.UUID
	if productID != "" {
		prodUUID = stringToUUID(productID)
	}

	logs, err := r.queries.GetInventoryLogs(ctx, sqlc.GetInventoryLogsParams{
		Column1: prodUUID,
		Limit:   int32(limit),
		Offset:  int32(offset),
	})
	if err != nil {
		return nil, 0, err
	}

	count, err := r.queries.CountInventoryLogs(ctx, prodUUID)
	if err != nil {
		return nil, 0, err
	}

	result := make([]domain.InventoryLog, len(logs))
	for i, l := range logs {
		result[i] = sqlcInventoryLogToDomain(l)
	}

	return result, count, nil
}

// --- Admin Product Methods ---

func (r *productRepository) CreateProduct(ctx context.Context, product *domain.Product) error {
	if product.CreatedAt.IsZero() {
		product.CreatedAt = time.Now()
	}
	if product.UpdatedAt.IsZero() {
		product.UpdatedAt = time.Now()
	}

	mediaBytes := mapImagesToMedia(product)
	attrsBytes, _ := json.Marshal(product.Attributes)
	specsBytes, _ := json.Marshal(product.Specs)

	// Marshal additional JSONB fields
	warrantyBytes, _ := json.Marshal(product.WarrantyInfo)

	created, err := r.queries.CreateProduct(ctx, sqlc.CreateProductParams{
		Name:              product.Name,
		Slug:              product.Slug,
		Sku:               product.SKU,
		Description:       strPtr(product.Description),
		BasePrice:         float64ToNumeric(product.BasePrice),
		SalePrice:         float64PtrToNumeric(product.SalePrice),
		Stock:             int32(product.Stock),
		StockStatus:       strPtr(product.StockStatus),
		LowStockThreshold: int32(product.LowStockThreshold),
		IsFeatured:        product.IsFeatured,
		IsActive:          product.IsActive,
		Media:             mediaBytes,
		Attributes:        attrsBytes,
		Specifications:    specsBytes,
		MetaTitle:         strPtr(product.MetaTitle),
		MetaDescription:   strPtr(product.MetaDescription),
		MetaKeywords:      strPtr(product.Keywords),
		OgImage:           strPtr(product.OGImage),
		Brand:             strPtr(product.Brand),
		Tags:              product.Tags,
		WarrantyInfo:      warrantyBytes,
	})
	if err != nil {
		return err
	}

	product.ID = uuidToString(created.ID)
	product.CreatedAt = pgtimeToTime(created.CreatedAt)
	product.UpdatedAt = pgtimeToTime(created.UpdatedAt)

	// Add categories
	for _, cat := range product.Categories {
		r.queries.AddProductCategory(ctx, sqlc.AddProductCategoryParams{
			ProductID:  created.ID,
			CategoryID: stringToUUID(cat.ID),
		})
	}

	// Add variants
	for _, v := range product.Variants {
		vAttributes, _ := json.Marshal(v.Attributes)
		vDimensions, _ := json.Marshal(v.Dimensions)

		_, err := r.queries.CreateVariant(ctx, sqlc.CreateVariantParams{
			ProductID:  created.ID,
			Name:       v.Name,
			Stock:      int32(v.Stock),
			Sku:        strPtr(v.SKU),
			Attributes: vAttributes,
			Price:      float64PtrToNumeric(v.Price),
			SalePrice:  float64PtrToNumeric(v.SalePrice),
			Images:     v.Images,
			Weight:     float64PtrToNumeric(v.Weight),
			Dimensions: vDimensions,
			Barcode:    strPtr(v.Barcode),
		})
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *productRepository) UpdateProduct(ctx context.Context, product *domain.Product) error {
	mediaBytes := mapImagesToMedia(product)
	attrsBytes, _ := json.Marshal(product.Attributes)
	specsBytes, _ := json.Marshal(product.Specs)

	// Marshal additional JSONB fields
	warrantyBytes, _ := json.Marshal(product.WarrantyInfo)

	_, err := r.queries.UpdateProduct(ctx, sqlc.UpdateProductParams{
		ID:                stringToUUID(product.ID),
		Name:              product.Name,
		Slug:              product.Slug,
		Description:       strPtr(product.Description),
		BasePrice:         float64ToNumeric(product.BasePrice),
		SalePrice:         float64PtrToNumeric(product.SalePrice),
		Stock:             int32(product.Stock),
		StockStatus:       strPtr(product.StockStatus),
		LowStockThreshold: int32(product.LowStockThreshold),
		IsFeatured:        product.IsFeatured,
		IsActive:          product.IsActive,
		Media:             mediaBytes,
		Attributes:        attrsBytes,
		Specifications:    specsBytes,
		MetaTitle:         strPtr(product.MetaTitle),
		MetaDescription:   strPtr(product.MetaDescription),
		MetaKeywords:      strPtr(product.Keywords),
		OgImage:           strPtr(product.OGImage),
		Brand:             strPtr(product.Brand),
		Tags:              product.Tags,
		WarrantyInfo:      warrantyBytes,
	})
	if err != nil {
		return err
	}

	// Update categories
	productUUID := stringToUUID(product.ID)
	r.queries.ClearProductCategories(ctx, productUUID)
	for _, cat := range product.Categories {
		r.queries.AddProductCategory(ctx, sqlc.AddProductCategoryParams{
			ProductID:  productUUID,
			CategoryID: stringToUUID(cat.ID),
		})
	}

	// Update variants (Replace Strategy: Delete all, Re-create)
	if err := r.queries.DeleteVariantsByProductID(ctx, productUUID); err != nil {
		return err
	}
	for _, v := range product.Variants {
		vAttributes, _ := json.Marshal(v.Attributes)
		vDimensions, _ := json.Marshal(v.Dimensions)

		_, err := r.queries.CreateVariant(ctx, sqlc.CreateVariantParams{
			ProductID:  productUUID,
			Name:       v.Name,
			Stock:      int32(v.Stock),
			Sku:        strPtr(v.SKU),
			Attributes: vAttributes,
			Price:      float64PtrToNumeric(v.Price),
			SalePrice:  float64PtrToNumeric(v.SalePrice),
			Images:     v.Images,
			Weight:     float64PtrToNumeric(v.Weight),
			Dimensions: vDimensions,
			Barcode:    strPtr(v.Barcode),
		})
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *productRepository) UpdateProductStatus(ctx context.Context, id string, isActive bool) error {
	return r.queries.UpdateProductStatus(ctx, sqlc.UpdateProductStatusParams{
		ID:       stringToUUID(id),
		IsActive: isActive,
	})
}

func (r *productRepository) DeleteProduct(ctx context.Context, id string) error {
	return r.queries.DeleteProduct(ctx, stringToUUID(id))
}

// --- Reviews ---

func (r *productRepository) CreateReview(ctx context.Context, review *domain.Review) error {
	created, err := r.queries.CreateReview(ctx, sqlc.CreateReviewParams{
		ProductID: stringToUUID(review.ProductID),
		UserID:    stringToUUID(review.UserID),
		Rating:    int32(review.Rating),
		Comment:   strPtr(review.Comment),
	})
	if err != nil {
		return err
	}
	review.ID = uuidToString(created.ID)
	review.CreatedAt = pgtimeToTime(created.CreatedAt)
	return nil
}

func (r *productRepository) GetReviews(ctx context.Context, productID string) ([]domain.Review, error) {
	rows, err := r.queries.GetReviewsByProductID(ctx, stringToUUID(productID))
	if err != nil {
		return nil, err
	}

	result := make([]domain.Review, len(rows))
	for i, row := range rows {
		result[i] = domain.Review{
			ID:        uuidToString(row.ID),
			ProductID: uuidToString(row.ProductID),
			UserID:    uuidToString(row.UserID),
			Rating:    int(row.Rating),
			Comment:   ptrString(row.Comment),
			CreatedAt: pgtimeToTime(row.CreatedAt),
			User: domain.User{
				FirstName: ptrString(row.FirstName),
				LastName:  ptrString(row.LastName),
				Avatar:    ptrString(row.Avatar),
			},
		}
	}
	return result, nil
}
