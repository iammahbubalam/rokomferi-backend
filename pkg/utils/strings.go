package utils

import (
	"regexp"
	"strings"
)

// GenerateSlug converts a string into a URL-friendly slug.
// e.g. "Men's T-Shirt!" -> "mens-t-shirt"
func GenerateSlug(input string) string {
	// Convert to lower case
	s := strings.ToLower(input)

	// Remove invalid chars (keep a-z, 0-9, space, hyphen)
	reg := regexp.MustCompile("[^a-z0-9 -]+")
	s = reg.ReplaceAllString(s, "")

	// Replace spaces with hyphens
	s = strings.ReplaceAll(s, " ", "-")

	// Collapse multiple hyphens
	reg2 := regexp.MustCompile("-+")
	s = reg2.ReplaceAllString(s, "-")

	// Trim hyphens
	s = strings.Trim(s, "-")

	return s
}
