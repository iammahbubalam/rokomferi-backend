package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"mime/multipart"
	"net/textproto"
	"os"
	"path/filepath"
	"rokomferi-backend/config"
	"rokomferi-backend/pkg/storage"

	"github.com/jackc/pgx/v5/pgxpool"
)

var r2Storage *storage.R2Storage

func main() {
	cfg := config.LoadConfig()

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, cfg.DBUrl)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer pool.Close()

	// Initialize R2 Storage
	r2Storage, err = storage.NewR2Storage(
		cfg.R2AccountID,
		cfg.R2AccessKeyID,
		cfg.R2AccessKeySecret,
		cfg.R2BucketName,
		cfg.R2PublicURL,
	)
	if err != nil {
		log.Printf("⚠️ Warning: R2 Storage not configured: %v", err)
		log.Println("   Images will use local paths instead of R2 URLs")
	}

	log.Println("🌱 Starting Database Seeder...")

	catMap := seedCategories(ctx, pool)
	users := seedUsers(ctx, pool)
	productIDs := seedProducts(ctx, pool, catMap)
	seedCollections(ctx, pool, productIDs)
	seedReviews(ctx, pool, users, productIDs)
	seedAddresses(ctx, pool, users)
	seedCartsAndOrders(ctx, pool, users, productIDs)
	seedInventoryLogs(ctx, pool, productIDs)

	log.Println("✅ Database Seeding Completed!")
}

// uploadImage uploads a local image to R2 and returns the URL
func uploadImage(relativePath string) string {
	if r2Storage == nil {
		return relativePath // R2 not configured, return local path
	}

	// Path relative to backend folder
	localPath := filepath.Join("../rokomferi-frontend/public", relativePath)

	file, err := os.Open(localPath)
	if err != nil {
		log.Printf("    ⚠️ Failed to open file %s: %v", localPath, err)
		return relativePath
	}
	defer file.Close()

	// Create mock file header
	fileInfo, _ := file.Stat()
	header := &multipart.FileHeader{
		Filename: filepath.Base(localPath),
		Size:     fileInfo.Size(),
		Header:   make(textproto.MIMEHeader),
	}

	// Set content type based on extension
	ext := filepath.Ext(localPath)
	switch ext {
	case ".png":
		header.Header.Set("Content-Type", "image/png")
	case ".jpg", ".jpeg":
		header.Header.Set("Content-Type", "image/jpeg")
	case ".svg":
		header.Header.Set("Content-Type", "image/svg+xml")
	case ".webp":
		header.Header.Set("Content-Type", "image/webp")
	}

	url, err := r2Storage.UploadFile(file, header)
	if err != nil {
		log.Printf("    ⚠️ Failed to upload %s to R2: %v", relativePath, err)
		return relativePath
	}

	log.Printf("    📤 Uploaded: %s", filepath.Base(relativePath))
	return url
}

func seedCategories(ctx context.Context, pool *pgxpool.Pool) map[string]string {
	log.Println("  📁 Seeding Categories...")
	catMap := make(map[string]string)

	// Top Level Categories
	cats := []struct{ Name, Slug string }{
		{"Women", "women"},
		{"Men", "men"},
		{"Accessories", "accessories"},
	}

	for i, c := range cats {
		var id string
		err := pool.QueryRow(ctx, "SELECT id FROM categories WHERE slug = $1", c.Slug).Scan(&id)
		if err != nil {
			err = pool.QueryRow(ctx,
				"INSERT INTO categories (name, slug, order_index, is_active, show_in_nav) VALUES ($1, $2, $3, true, true) RETURNING id",
				c.Name, c.Slug, i+1).Scan(&id)
			if err != nil {
				log.Printf("    ⚠️ Failed to create category %s: %v", c.Name, err)
				continue
			}
		}
		catMap[c.Slug] = id
	}

	// Sub Categories
	subCats := []struct{ Name, Slug, Parent string }{
		{"Sarees", "sarees", "women"},
		{"Kurtis", "kurtis", "women"},
		{"Three Piece", "three-piece", "women"},
		{"Panjabi", "panjabi", "men"},
		{"Jewelry", "jewelry", "accessories"},
		{"Bags & Potlis", "bags", "accessories"},
	}

	for i, sc := range subCats {
		var id string
		err := pool.QueryRow(ctx, "SELECT id FROM categories WHERE slug = $1", sc.Slug).Scan(&id)
		if err != nil {
			parentID := catMap[sc.Parent]
			err = pool.QueryRow(ctx,
				"INSERT INTO categories (name, slug, parent_id, order_index, is_active, show_in_nav) VALUES ($1, $2, $3, $4, true, true) RETURNING id",
				sc.Name, sc.Slug, parentID, i+1).Scan(&id)
			if err != nil {
				log.Printf("    ⚠️ Failed to create subcategory %s: %v", sc.Name, err)
				continue
			}
		}
		catMap[sc.Slug] = id
	}

	return catMap
}

func seedUsers(ctx context.Context, pool *pgxpool.Pool) []string {
	log.Println("  👤 Seeding Users...")
	users := []struct{ Email, FirstName, LastName string }{
		{"alice@example.com", "Alice", "Wonder"},
		{"bob@example.com", "Bob", "Builder"},
		{"charlie@example.com", "Charlie", "Chaplin"},
		{"diana@example.com", "Diana", "Ross"},
	}

	var userIDs []string
	for _, u := range users {
		var id string
		err := pool.QueryRow(ctx, "SELECT id FROM users WHERE email = $1", u.Email).Scan(&id)
		if err != nil {
			err = pool.QueryRow(ctx,
				"INSERT INTO users (email, first_name, last_name, role) VALUES ($1, $2, $3, 'customer') RETURNING id",
				u.Email, u.FirstName, u.LastName).Scan(&id)
			if err != nil {
				log.Printf("    ⚠️ Failed to create user %s: %v", u.Email, err)
				continue
			}
		}
		userIDs = append(userIDs, id)
	}
	return userIDs
}

func seedProducts(ctx context.Context, pool *pgxpool.Pool, catMap map[string]string) []string {
	log.Println("  📦 Seeding Products...")

	products := []struct {
		Name, Slug, SKU, Description, Category string
		BasePrice                              float64
		SalePrice                              *float64
		Stock                                  int
		IsFeatured                             bool
		ImagePath                              string // Local path in /assets/
	}{
		{"Indigo Silk Panjabi", "indigo-silk-panjabi", "ISP-001", "Premium indigo silk panjabi with intricate embroidery.", "panjabi", 4500, nil, 50, true, "/assets/panjabi-indigo-silk.png"},
		{"Platinum White Panjabi", "platinum-white-panjabi", "PWP-001", "Classic platinum white cotton panjabi suitable for summer.", "panjabi", 3500, nil, 100, false, "/assets/panjabi-platinum-white.png"},
		{"Sage Green Panjabi", "sage-green-panjabi", "SGP-001", "Elegant sage green panjabi for festive occasions.", "panjabi", 5200, floatPtr(4500), 30, true, "/assets/panjabi-sage-green.png"},
		{"Crimson Bridal Banarasi", "crimson-bridal-banarasi", "CBB-001", "Authentic crimson Banarasi saree with gold zari work.", "sarees", 25000, floatPtr(22000), 5, true, "/assets/saree-crimson-bridal.png"},
		{"Midnight Blue Katan", "midnight-blue-katan", "MBK-001", "Stunning midnight blue katan saree with silver motifs.", "sarees", 12500, nil, 15, true, "/assets/saree-blue-katan.png"},
		{"Emerald Green Kurti", "emerald-green-kurti", "EGK-001", "Comfortable emerald green cotton kurti.", "kurtis", 2500, floatPtr(1999), 60, false, "/assets/kurti-emerald.png"},
		{"Ivory Khadi Kurti", "ivory-khadi-kurti", "IKK-001", "Minimalist ivory khadi kurti with wooden buttons.", "kurtis", 2200, nil, 45, false, "/assets/kurti-ivory-khadi.png"},
		{"Ruby Cotton Kurti", "ruby-cotton-kurti", "RCK-001", "Vibrant ruby red kurti for daily wear.", "kurtis", 1800, nil, 80, false, "/assets/kurti-ruby-cotton.png"},
		{"Black Georgette Set", "black-georgette-set", "BGS-001", "Stylish black georgette three-piece suit.", "three-piece", 4800, floatPtr(4200), 25, true, "/assets/threepiece-black-georgette.png"},
		{"Lilac Chiffon Suit", "lilac-chiffon-suit", "LCS-001", "Soft lilac chiffon suit with digital print.", "three-piece", 5500, nil, 20, true, "/assets/threepiece-lilac-chiffon.png"},
		{"Peach Cotton Set", "peach-cotton-set", "PCS-001", "Breathable peach cotton three-piece.", "three-piece", 3200, nil, 40, false, "/assets/threepiece-peach.png"},
		{"Pearl Choker Set", "pearl-choker-set", "PCHK-001", "Elegant imitation pearl choker with earrings.", "jewelry", 1500, nil, 100, false, "/assets/accessory-pearl-choker.png"},
		{"Silver Oxidized Jhumka", "silver-oxidized-jhumka", "SOJ-001", "Traditional silver oxidized jhumka earrings.", "jewelry", 850, floatPtr(699), 150, false, "/assets/accessory-silver-jhumka.png"},
		{"Velvet Potli Bag", "velvet-potli-bag", "VPB-001", "Embroidered velvet potli bag for weddings.", "bags", 1200, nil, 50, false, "/assets/accessory-velvet-potli.png"},
	}

	var productIDs []string
	for _, p := range products {
		var id string
		err := pool.QueryRow(ctx, "SELECT id FROM products WHERE slug = $1", p.Slug).Scan(&id)
		if err != nil {
			// Upload image to R2
			imageURL := uploadImage(p.ImagePath)
			mediaBytes, _ := json.Marshal([]string{imageURL})

			var salePrice interface{} = nil
			if p.SalePrice != nil {
				salePrice = *p.SalePrice
			}

			err = pool.QueryRow(ctx,
				`INSERT INTO products (name, slug, sku, description, base_price, sale_price, stock, is_featured, is_active, media)
				 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, true, $9) RETURNING id`,
				p.Name, p.Slug, p.SKU, p.Description, p.BasePrice, salePrice, p.Stock, p.IsFeatured, mediaBytes).Scan(&id)
			if err != nil {
				log.Printf("    ⚠️ Failed to create product %s: %v", p.Name, err)
				continue
			}
			fmt.Printf("    ✅ Created: %s\n", p.Name)

			// Link to category
			catID := catMap[p.Category]
			if catID != "" {
				_, _ = pool.Exec(ctx, "INSERT INTO product_categories (product_id, category_id) VALUES ($1, $2) ON CONFLICT DO NOTHING", id, catID)
			}
		} else {
			// Update existing product's image if needed
			imageURL := uploadImage(p.ImagePath)
			mediaBytes, _ := json.Marshal([]string{imageURL})
			_, _ = pool.Exec(ctx, "UPDATE products SET media = $1 WHERE id = $2", mediaBytes, id)
		}
		productIDs = append(productIDs, id)
	}
	return productIDs
}

func seedCollections(ctx context.Context, pool *pgxpool.Pool, productIDs []string) {
	log.Println("  🏷️ Seeding Collections...")

	collections := []struct {
		Title, Slug, Description, Story, ImagePath string
	}{
		{"Moonlit Silence", "eid-2025", "The Eid 2025 Edit", "In the stillness of the crescent moon, find the luxury of connection.", "/assets/eid-hero.png"},
		{"Legacy of Loom", "heritage", "The Heritage Edit", "Celebrate the hands that weave history. Authentic Katan, Muslin, and Silk.", "/assets/eid-editorial.png"},
		{"Wedding Guest", "wedding-guest", "For the golden hours of reunion.", "Designed for elegance and grace at every celebration.", "/assets/eid-hero-group.png"},
	}

	for _, c := range collections {
		var id string
		err := pool.QueryRow(ctx, "SELECT id FROM collections WHERE slug = $1", c.Slug).Scan(&id)
		if err != nil {
			// Upload image to R2
			imageURL := uploadImage(c.ImagePath)

			err = pool.QueryRow(ctx,
				"INSERT INTO collections (title, slug, description, story, image, is_active) VALUES ($1, $2, $3, $4, $5, true) RETURNING id",
				c.Title, c.Slug, c.Description, c.Story, imageURL).Scan(&id)
			if err != nil {
				log.Printf("    ⚠️ Failed to create collection %s: %v", c.Title, err)
				continue
			}
			fmt.Printf("    ✅ Created: %s\n", c.Title)

			// Add random products to collection
			numProds := rand.Intn(4) + 2
			for i := 0; i < numProds && i < len(productIDs); i++ {
				pid := productIDs[rand.Intn(len(productIDs))]
				_, _ = pool.Exec(ctx, "INSERT INTO product_collections (product_id, collection_id) VALUES ($1, $2) ON CONFLICT DO NOTHING", pid, id)
			}
		} else {
			// Update existing collection's image
			imageURL := uploadImage(c.ImagePath)
			_, _ = pool.Exec(ctx, "UPDATE collections SET image = $1 WHERE id = $2", imageURL, id)
		}
	}
}

func seedReviews(ctx context.Context, pool *pgxpool.Pool, userIDs []string, productIDs []string) {
	log.Println("  ⭐ Seeding Reviews...")

	comments := []string{
		"Absolutely loved the fabric quality!",
		"Fits perfectly, true to size.",
		"Color is slightly different than picture, but still good.",
		"Fast delivery and great packaging.",
		"Will definitely order again.",
		"Value for money.",
		"Elegant and classy.",
	}

	for _, pid := range productIDs {
		numReviews := rand.Intn(3) + 1
		for i := 0; i < numReviews; i++ {
			uid := userIDs[rand.Intn(len(userIDs))]
			rating := rand.Intn(3) + 3
			comment := comments[rand.Intn(len(comments))]

			_, _ = pool.Exec(ctx,
				"INSERT INTO reviews (product_id, user_id, rating, comment) VALUES ($1, $2, $3, $4) ON CONFLICT DO NOTHING",
				pid, uid, rating, comment)
		}
	}
}

func seedAddresses(ctx context.Context, pool *pgxpool.Pool, userIDs []string) {
	log.Println("  📍 Seeding Addresses...")

	if len(userIDs) == 0 {
		return
	}

	_, _ = pool.Exec(ctx,
		`INSERT INTO addresses (user_id, label, first_name, last_name, phone, district, thana, address_line, postal_code, is_default)
		 VALUES ($1, 'Home', 'Alice', 'Wonder', '+8801712345678', 'Dhaka', 'Gulshan', '123 Main Street, House 5', '1212', true)
		 ON CONFLICT DO NOTHING`,
		userIDs[0])
}

func seedCartsAndOrders(ctx context.Context, pool *pgxpool.Pool, userIDs []string, productIDs []string) {
	log.Println("  🛒 Seeding Carts and Orders...")

	if len(userIDs) == 0 || len(productIDs) == 0 {
		return
	}

	// Cart for first user
	var cartID string
	err := pool.QueryRow(ctx, "SELECT id FROM carts WHERE user_id = $1", userIDs[0]).Scan(&cartID)
	if err != nil {
		err = pool.QueryRow(ctx, "INSERT INTO carts (user_id) VALUES ($1) RETURNING id", userIDs[0]).Scan(&cartID)
		if err == nil {
			_, _ = pool.Exec(ctx, "INSERT INTO cart_items (cart_id, product_id, quantity) VALUES ($1, $2, 2)", cartID, productIDs[0])
		}
	}

	// Order for first user
	shippingAddr := map[string]string{
		"firstName":   "Alice",
		"lastName":    "Wonder",
		"phone":       "+8801712345678",
		"district":    "Dhaka",
		"thana":       "Gulshan",
		"addressLine": "123 Main Street",
	}
	addrBytes, _ := json.Marshal(shippingAddr)

	var orderID string
	err = pool.QueryRow(ctx,
		`INSERT INTO orders (user_id, status, total_amount, shipping_address, payment_method, payment_status)
		 VALUES ($1, 'delivered', 25000, $2, 'cod', 'paid') RETURNING id`,
		userIDs[0], addrBytes).Scan(&orderID)
	if err == nil {
		_, _ = pool.Exec(ctx, "INSERT INTO order_items (order_id, product_id, quantity, price) VALUES ($1, $2, 1, 25000)", orderID, productIDs[0])
	}
}

func seedInventoryLogs(ctx context.Context, pool *pgxpool.Pool, productIDs []string) {
	log.Println("  📊 Seeding Inventory Logs...")

	for _, pid := range productIDs {
		_, _ = pool.Exec(ctx,
			"INSERT INTO inventory_logs (product_id, change_amount, reason, reference_id) VALUES ($1, 50, 'initial_stock', 'SEED-001')",
			pid)
	}
}

func floatPtr(f float64) *float64 {
	return &f
}
