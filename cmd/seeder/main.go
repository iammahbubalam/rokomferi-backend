package main

import (
	"log"
	"math/rand"
	"mime/multipart"
	"net/textproto"
	"os"
	"path/filepath"
	"rokomferi-backend/config"
	"rokomferi-backend/internal/domain"
	postgresPkg "rokomferi-backend/pkg/postgres"
	"rokomferi-backend/pkg/storage"
	"rokomferi-backend/pkg/utils"
	"time"

	"gorm.io/gorm"
)

func main() {
	cfg := config.LoadConfig()
	db, err := postgresPkg.NewClient(cfg.DBUrl)
	if err != nil {
		log.Fatal(err)
	}

	// Initialize R2
	r2, err := storage.NewR2Storage(
		cfg.R2AccountID,
		cfg.R2AccessKeyID,
		cfg.R2AccessKeySecret,
		cfg.R2BucketName,
		cfg.R2PublicURL,
	)
	if err != nil {
		log.Printf("Warning: Failed to init R2: %v", err)
	}

	log.Println("--- Starting Seeder with R2 Uploads ---")
	catMap := seedCategories(db)
	users := seedUsers(db)
	products := seedProducts(db, catMap, r2)
	seedReviews(db, users, products)
	log.Println("--- Seeding Completed ---")
}

func uploadImage(r2 *storage.R2Storage, relativePath string) string {
	if r2 == nil {
		return relativePath // R2 not configured
	}

	// Path relative to rokomferi-backend/
	localPath := filepath.Join("../rokomferi-frontend/public", relativePath)

	file, err := os.Open(localPath)
	if err != nil {
		log.Printf("Failed to open file %s: %v", localPath, err)
		return relativePath
	}
	defer file.Close()

	// Mock file header
	fileInfo, _ := file.Stat()
	header := &multipart.FileHeader{
		Filename: filepath.Base(localPath),
		Size:     fileInfo.Size(),
		Header:   make(textproto.MIMEHeader),
	}
	// Simple content type detection
	ext := filepath.Ext(localPath)
	if ext == ".png" {
		header.Header.Set("Content-Type", "image/png")
	} else if ext == ".jpg" || ext == ".jpeg" {
		header.Header.Set("Content-Type", "image/jpeg")
	} else if ext == ".svg" {
		header.Header.Set("Content-Type", "image/svg+xml") // S3 might need this
	}

	url, err := r2.UploadFile(file, header)
	if err != nil {
		log.Printf("Failed to upload %s to R2: %v", localPath, err)
		return relativePath
	}

	log.Printf("Uploaded %s -> %s", relativePath, url)
	return url
}

func seedCategories(db *gorm.DB) map[string]string {
	log.Println("Seeding Categories...")
	catMap := make(map[string]string) // Slug -> ID

	// 1. Top Level
	cats := []domain.Category{
		{Name: "Women", Slug: "women", OrderIndex: 1},
		{Name: "Men", Slug: "men", OrderIndex: 2},
		{Name: "Accessories", Slug: "accessories", OrderIndex: 3},
	}
	for _, c := range cats {
		var existing domain.Category
		if err := db.Where("slug = ?", c.Slug).First(&existing).Error; err == nil {
			catMap[c.Slug] = existing.ID
		} else {
			c.ID = utils.GenerateUUID()
			db.Create(&c)
			catMap[c.Slug] = c.ID
		}
	}

	// 2. Sub Categories
	subCats := []struct {
		Name   string
		Slug   string
		Parent string
	}{
		{Name: "Sarees", Slug: "sarees", Parent: "women"},
		{Name: "Kurtis", Slug: "kurtis", Parent: "women"},
		{Name: "Three Piece", Slug: "three-piece", Parent: "women"},
		{Name: "Panjabi", Slug: "panjabi", Parent: "men"},
		{Name: "Jewelry", Slug: "jewelry", Parent: "accessories"},
		{Name: "Bags & Potlis", Slug: "bags", Parent: "accessories"},
	}

	for _, sc := range subCats {
		var existing domain.Category
		if err := db.Where("slug = ?", sc.Slug).First(&existing).Error; err == nil {
			catMap[sc.Slug] = existing.ID
		} else {
			parentID := catMap[sc.Parent]
			newCat := domain.Category{
				ID:       utils.GenerateUUID(),
				Name:     sc.Name,
				Slug:     sc.Slug,
				ParentID: &parentID,
			}
			db.Create(&newCat)
			catMap[sc.Slug] = newCat.ID
		}
	}
	return catMap
}

func seedUsers(db *gorm.DB) []domain.User {
	log.Println("Seeding Dummy Users...")
	usersData := []domain.User{
		{Email: "alice@example.com", FirstName: "Alice", LastName: "Wonder", Role: "customer"},
		{Email: "bob@example.com", FirstName: "Bob", LastName: "Builder", Role: "customer"},
		{Email: "charlie@example.com", FirstName: "Charlie", LastName: "Chaplin", Role: "customer"},
		{Email: "diana@example.com", FirstName: "Diana", LastName: "Ross", Role: "customer"},
	}

	var users []domain.User
	for _, u := range usersData {
		var existing domain.User
		if err := db.Where("email = ?", u.Email).First(&existing).Error; err == nil {
			users = append(users, existing)
		} else {
			u.ID = utils.GenerateUUID()
			db.Create(&u)
			users = append(users, u)
		}
	}
	return users
}

func seedProducts(db *gorm.DB, catMap map[string]string, r2 *storage.R2Storage) []string {
	log.Println("Seeding Products...")
	// Helper to get ID safely
	getCatID := func(slug string) string {
		if id, ok := catMap[slug]; ok {
			return id
		}
		return "" // Functionally shouldn't happen
	}

	products := []domain.Product{
		// Men - Panjabi
		{
			Name: "Indigo Silk Panjabi", Categories: []domain.Category{{ID: getCatID("panjabi")}}, BasePrice: 4500,
			Description: "Premium indigo silk panjabi with intricate embroidery.",
			StockStatus: "in_stock", Stock: 50, IsFeatured: true,
			Media: domain.JSONB{"images": []string{"/assets/panjabi-indigo-silk.png"}},
		},
		{
			Name: "Platinum White Panjabi", Categories: []domain.Category{{ID: getCatID("panjabi")}}, BasePrice: 3500,
			Description: "Classic platinum white cotton panjabi suitable for summer.",
			StockStatus: "in_stock", Stock: 100, IsFeatured: false,
			Media: domain.JSONB{"images": []string{"/assets/panjabi-platinum-white.png"}},
		},
		{
			Name: "Sage Green Embroidered Panjabi", Categories: []domain.Category{{ID: getCatID("panjabi")}}, BasePrice: 5200,
			Description: "Elegant sage green panjabi for festive occasions.",
			StockStatus: "in_stock", Stock: 30, IsFeatured: true,
			Media: domain.JSONB{"images": []string{"/assets/panjabi-sage-green.png"}},
		},

		// Women - Sarees
		{
			Name: "Crimson Bridal Banarasi", Categories: []domain.Category{{ID: getCatID("sarees")}}, BasePrice: 25000,
			Description: "Authentic crimson Banarasi saree with gold zari work.",
			StockStatus: "in_stock", Stock: 5, IsFeatured: true,
			Media: domain.JSONB{"images": []string{"/assets/saree-crimson-bridal.png"}},
		},
		{
			Name: "Midnight Blue Katan", Categories: []domain.Category{{ID: getCatID("sarees")}}, BasePrice: 12500,
			Description: "Stunning midnight blue katan saree with silver motifs.",
			StockStatus: "in_stock", Stock: 15, IsFeatured: true,
			Media: domain.JSONB{"images": []string{"/assets/saree-blue-katan.png"}},
		},

		// Women - Kurtis
		{
			Name: "Emerald Green Kurti", Categories: []domain.Category{{ID: getCatID("kurtis")}}, BasePrice: 2500,
			Description: "Comfortable emerald green cotton kurti.",
			StockStatus: "in_stock", Stock: 60, IsFeatured: false,
			Media: domain.JSONB{"images": []string{"/assets/kurti-emerald.png"}},
		},
		{
			Name: "Ivory Khadi Kurti", Categories: []domain.Category{{ID: getCatID("kurtis")}}, BasePrice: 2200,
			Description: "Minimalist ivory khadi kurti with wooden buttons.",
			StockStatus: "in_stock", Stock: 45, IsFeatured: false,
			Media: domain.JSONB{"images": []string{"/assets/kurti-ivory-khadi.png"}},
		},
		{
			Name: "Ruby Red Cotton Kurti", Categories: []domain.Category{{ID: getCatID("kurtis")}}, BasePrice: 1800,
			Description: "Vibrant ruby red kurti for daily wear.",
			StockStatus: "in_stock", Stock: 80, IsFeatured: false,
			Media: domain.JSONB{"images": []string{"/assets/kurti-ruby-cotton.png"}},
		},

		// Women - Three Piece
		{
			Name: "Black Georgette Set", Categories: []domain.Category{{ID: getCatID("three-piece")}}, BasePrice: 4800,
			Description: "Stylish black georgette three-piece suit.",
			StockStatus: "in_stock", Stock: 25, IsFeatured: true,
			Media: domain.JSONB{"images": []string{"/assets/threepiece-black-georgette.png"}},
		},
		{
			Name: "Lilac Chiffon Suit", Categories: []domain.Category{{ID: getCatID("three-piece")}}, BasePrice: 5500,
			Description: "Soft lilac chiffon suit with digital print.",
			StockStatus: "in_stock", Stock: 20, IsFeatured: true,
			Media: domain.JSONB{"images": []string{"/assets/threepiece-lilac-chiffon.png"}},
		},
		{
			Name: "Peach Cotton Set", Categories: []domain.Category{{ID: getCatID("three-piece")}}, BasePrice: 3200,
			Description: "Breathable peach cotton three-piece.",
			StockStatus: "in_stock", Stock: 40, IsFeatured: false,
			Media: domain.JSONB{"images": []string{"/assets/threepiece-peach.png"}},
		},

		// Accessories
		{
			Name: "Pearl Choker Set", Categories: []domain.Category{{ID: getCatID("jewelry")}}, BasePrice: 1500,
			Description: "Elegant imitation pearl choker with earrings.",
			StockStatus: "in_stock", Stock: 100, IsFeatured: false,
			Media: domain.JSONB{"images": []string{"/assets/accessory-pearl-choker.png"}},
		},
		{
			Name: "Silver Oxidized Jhumka", Categories: []domain.Category{{ID: getCatID("jewelry")}}, BasePrice: 850,
			Description: "Traditional silver oxidized jhumka earrings.",
			StockStatus: "in_stock", Stock: 150, IsFeatured: false,
			Media: domain.JSONB{"images": []string{"/assets/accessory-silver-jhumka.png"}},
		},
		{
			Name: "Velvet Potli Bag", Categories: []domain.Category{{ID: getCatID("bags")}}, BasePrice: 1200,
			Description: "Embroidered velvet potli bag for weddings.",
			StockStatus: "in_stock", Stock: 50, IsFeatured: false,
			Media: domain.JSONB{"images": []string{"/assets/accessory-velvet-potli.png"}},
		},
	}

	var productIDs []string
	for _, p := range products {
		if len(p.Categories) == 0 || p.Categories[0].ID == "" {
			continue
		}

		p.ID = utils.GenerateUUID()
		p.Slug = utils.GenerateSlug(p.Name)
		p.SKU = utils.GenerateSlug(p.Name) + "-SKU"
		p.CreatedAt = time.Now()
		p.UpdatedAt = time.Now()
		p.IsActive = true

		// UPLOAD TO R2 IF NEEDED
		// Check if we already have this product to skip re-uploading if possible,
		// but for seeding let's just do it or check if R2 URL is already there?
		// To be safe, let's always uploading for now or do a quick check.
		// Use the first image path from the strict
		images := p.Media["images"].([]string)
		if len(images) > 0 {
			uploadedURL := uploadImage(r2, images[0])
			p.Media["images"] = []string{uploadedURL}
		}

		var count int64
		db.Model(&domain.Product{}).Where("name = ?", p.Name).Count(&count)
		if count == 0 {
			db.Create(&p)
			productIDs = append(productIDs, p.ID)
		} else {
			var existing domain.Product
			db.Where("name = ?", p.Name).First(&existing)
			// OPTIONAL: Update the image to the R2 one if we want to "fix" existing data
			db.Model(&existing).UpdateColumn("media", p.Media)

			productIDs = append(productIDs, existing.ID)
		}
	}
	return productIDs
}

func seedReviews(db *gorm.DB, users []domain.User, productIDs []string) {
	log.Println("Seeding Reviews...")
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
		// Add 1-3 reviews per product randomly
		numReviews := rand.Intn(3) + 1
		for i := 0; i < numReviews; i++ {
			user := users[rand.Intn(len(users))]
			rating := rand.Intn(3) + 3 // 3 to 5 stars

			review := domain.Review{
				ID:        utils.GenerateUUID(),
				ProductID: pid,
				UserID:    user.ID,
				Rating:    rating,
				Comment:   comments[rand.Intn(len(comments))],
				CreatedAt: time.Now().Add(-time.Duration(rand.Intn(1000)) * time.Hour),
			}

			var count int64
			db.Model(&domain.Review{}).Where("product_id = ? AND user_id = ?", pid, user.ID).Count(&count)
			if count == 0 {
				db.Create(&review)
			}
		}
	}
}

func stringPtr(s string) *string {
	return &s
}
