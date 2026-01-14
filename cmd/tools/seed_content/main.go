package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"rokomferi-backend/config"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// --- Shared Structs ---

type HeroSlide struct {
	ID             int    `json:"id"`
	Image          string `json:"image"`
	Title          string `json:"title"`
	Subtitle       string `json:"subtitle"`
	Description    string `json:"description"`
	Alignment      string `json:"alignment,omitempty"`
	TextColor      string `json:"textColor,omitempty"`
	OverlayOpacity int    `json:"overlayOpacity,omitempty"`
	ButtonText     string `json:"buttonText,omitempty"`
	ButtonLink     string `json:"buttonLink,omitempty"`
	ButtonStyle    string `json:"buttonStyle,omitempty"`
}

type FooterLink struct {
	Label string `json:"label"`
	Href  string `json:"href"`
}

type FooterSection struct {
	Title string       `json:"title"`
	Links []FooterLink `json:"links"`
}

// --- New Phase 4 Structs ---

type GlobalSettings struct {
	Branding struct {
		SiteName     string `json:"siteName"`
		Tagline      string `json:"tagline"`
		LogoUrl      string `json:"logoUrl"`
		FaviconUrl   string `json:"faviconUrl"`
		PrimaryColor string `json:"primaryColor"`
	} `json:"branding"`
	Contact struct {
		SupportEmail   string `json:"supportEmail"`
		SalesEmail     string `json:"salesEmail"`
		PhonePrimary   string `json:"phonePrimary"`
		PhoneSecondary string `json:"phoneSecondary"`
		Address        struct {
			Line1   string `json:"line1"`
			Line2   string `json:"line2"`
			City    string `json:"city"`
			Zip     string `json:"zip"`
			Country string `json:"country"`
			MapUrl  string `json:"mapUrl"`
		} `json:"address"`
	} `json:"contact"`
	Socials struct {
		Facebook  string `json:"facebook"`
		Instagram string `json:"instagram"`
		Linkedin  string `json:"linkedin"`
		Youtube   string `json:"youtube"`
	} `json:"socials"`
	Seo struct {
		DefaultMetaTitle       string `json:"defaultMetaTitle"`
		DefaultMetaDescription string `json:"defaultMetaDescription"`
		DefaultOgImage         string `json:"defaultOgImage"`
	} `json:"seo"`
}

type AboutBlock struct {
	Type     string           `json:"type"` // text, image_split, stats
	Heading  string           `json:"heading,omitempty"`
	Body     string           `json:"body,omitempty"`
	Position string           `json:"position,omitempty"` // left, right
	ImageUrl string           `json:"imageUrl,omitempty"`
	Caption  string           `json:"caption,omitempty"`
	Items    []AboutStatsItem `json:"items,omitempty"`
}

type AboutStatsItem struct {
	Label string `json:"label"`
	Value string `json:"value"`
}

type AboutPage struct {
	Hero struct {
		Title    string `json:"title"`
		Subtitle string `json:"subtitle"`
		ImageUrl string `json:"imageUrl"`
	} `json:"hero"`
	Blocks []AboutBlock `json:"blocks"`
}

type PolicySection struct {
	Heading   string   `json:"heading"`
	Content   string   `json:"content,omitempty"`
	ListItems []string `json:"listItems,omitempty"`
}

type PolicyPage struct {
	Sections    []PolicySection `json:"sections"`
	LastUpdated string          `json:"lastUpdated"`
}

type FAQItem struct {
	Question string `json:"question"`
	Answer   string `json:"answer"`
	Category string `json:"category"`
}

type FAQPage struct {
	Items []FAQItem `json:"items"`
}

func main() {
	cfg := config.LoadConfig()

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, cfg.DBUrl)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer pool.Close()

	log.Println("🌱 Seeding Content Blocks...")

	if err := seedHero(ctx, pool); err != nil {
		log.Printf("❌ Failed to seed hero: %v", err)
	}
	if err := seedFooter(ctx, pool); err != nil {
		log.Printf("❌ Failed to seed footer: %v", err)
	}
	if err := seedGlobalSettings(ctx, pool); err != nil {
		log.Printf("❌ Failed to seed global settings: %v", err)
	}
	if err := seedAboutPage(ctx, pool); err != nil {
		log.Printf("❌ Failed to seed about page: %v", err)
	}
	if err := seedPolicies(ctx, pool); err != nil {
		log.Printf("❌ Failed to seed policies: %v", err)
	}
	if err := seedFAQ(ctx, pool); err != nil {
		log.Printf("❌ Failed to seed FAQ: %v", err)
	}

	log.Println("✅ Content Seeding Completed!")
}

// --- Seeder Functions ---

func upsertContent(ctx context.Context, pool *pgxpool.Pool, key string, data interface{}) error {
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return err
	}

	sql := `
		INSERT INTO content_blocks (section_key, content, updated_at)
		VALUES ($1, $2, NOW())
		ON CONFLICT (section_key) 
		DO UPDATE SET content = $2, updated_at = NOW()
	`
	_, err = pool.Exec(ctx, sql, key, jsonBytes)
	if err != nil {
		return err
	}
	fmt.Printf("    ✅ Seeded %s\n", key)
	return nil
}

func seedHero(ctx context.Context, pool *pgxpool.Pool) error {
	slides := []HeroSlide{
		{
			ID: 1, Image: "/assets/eid-hero.png", Title: "Moonlit Silence", Subtitle: "The Eid 2026 Edit",
			Description: "In the stillness of the crescent moon, find the luxury of connection.", Alignment: "center", TextColor: "white", OverlayOpacity: 40, ButtonText: "Explore", ButtonStyle: "primary",
		},
		{
			ID: 2, Image: "/assets/eid-hero-group.png", Title: "Legacy of Loom", Subtitle: "The Eid Heritage Edit",
			Description: "Celebrate the hands that weave history. Authentic Katan, Muslin, and Silk.", Alignment: "center", TextColor: "white", OverlayOpacity: 40, ButtonText: "View Collection", ButtonStyle: "outline",
		},
	}
	return upsertContent(ctx, pool, "home_hero", map[string]interface{}{"slides": slides})
}

func seedFooter(ctx context.Context, pool *pgxpool.Pool) error {
	sections := []FooterSection{
		{Title: "Shop", Links: []FooterLink{{Label: "New Arrivals", Href: "/new-arrivals"}, {Label: "Ready to Wear", Href: "/category/ready-to-wear"}}},
		{Title: "Support", Links: []FooterLink{{Label: "Contact Us", Href: "/contact"}, {Label: "Shipping & Returns", Href: "/shipping"}}},
		{Title: "Legal", Links: []FooterLink{{Label: "Privacy Policy", Href: "/privacy"}}},
	}
	return upsertContent(ctx, pool, "home_footer", map[string]interface{}{"sections": sections})
}

func seedGlobalSettings(ctx context.Context, pool *pgxpool.Pool) error {
	settings := GlobalSettings{}

	settings.Branding.SiteName = "Rokomferi"
	settings.Branding.Tagline = "Opulence Minimal"
	settings.Branding.PrimaryColor = "#000000"

	settings.Contact.SupportEmail = "support@rokomferi.com"
	settings.Contact.PhonePrimary = "+880 17 0000 0000"
	settings.Contact.Address.Line1 = "House 12, Road 5"
	settings.Contact.Address.City = "Dhaka"
	settings.Contact.Address.Country = "Bangladesh"

	settings.Socials.Facebook = "https://facebook.com/rokomferi"
	settings.Socials.Instagram = "https://instagram.com/rokomferi"

	settings.Seo.DefaultMetaTitle = "Rokomferi | Premium Fashion"
	settings.Seo.DefaultMetaDescription = "Shop the latest in premium minimalist fashion."

	return upsertContent(ctx, pool, "settings_global", settings)
}

func seedAboutPage(ctx context.Context, pool *pgxpool.Pool) error {
	page := AboutPage{}
	page.Hero.Title = "Our Story"
	page.Hero.Subtitle = "Redefining Minimalist Luxury"

	page.Blocks = []AboutBlock{
		{Type: "text", Heading: "The Beginning", Body: "Founded in 2024, Rokomferi began with a simple idea: luxury shouldn't shout. It should whisper."},
		{Type: "stats", Items: []AboutStatsItem{{Label: "Years", Value: "10+"}, {Label: "Outlets", Value: "5"}}},
	}
	return upsertContent(ctx, pool, "content_about", page)
}

func seedPolicies(ctx context.Context, pool *pgxpool.Pool) error {
	now := time.Now().Format(time.RFC3339)

	shipping := PolicyPage{
		LastUpdated: now,
		Sections: []PolicySection{
			{Heading: "Delivery Areas", Content: "We deliver nationwide via Pathao and RedX."},
			{Heading: "Estimated Times", Content: "Inside Dhaka: 2-3 Days. Outside Dhaka: 3-5 Days."},
		},
	}
	if err := upsertContent(ctx, pool, "policy_shipping", shipping); err != nil {
		return err
	}

	returns := PolicyPage{
		LastUpdated: now,
		Sections: []PolicySection{
			{Heading: "Return Window", Content: "You can return items within 7 days of delivery."},
			{Heading: "Conditions", ListItems: []string{"Product must be unused.", "Tags must be attached."}},
		},
	}
	return upsertContent(ctx, pool, "policy_return", returns)
}

func seedFAQ(ctx context.Context, pool *pgxpool.Pool) error {
	faq := FAQPage{
		Items: []FAQItem{
			{Question: "Do you ship internationally?", Answer: "Currently only within Bangladesh.", Category: "Shipping"},
			{Question: "What payment methods do you accept?", Answer: "Visa, Mastercard, bKash.", Category: "Payment"},
		},
	}
	return upsertContent(ctx, pool, "content_faq", faq)
}
