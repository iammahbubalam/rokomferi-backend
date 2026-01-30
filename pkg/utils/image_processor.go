package utils

import (
	"bytes"
	"image"
	_ "image/gif" // Register GIF decoder
	"image/jpeg"
	_ "image/png" // Register PNG decoder
	"log"
	"mime/multipart"

	"github.com/chai2010/webp"
	"github.com/disintegration/imaging"
)

// ProcessImage Resize and Convert to WebP
func ProcessImage(file multipart.File, filename string) ([]byte, string, error) {
	// 1. Decode generic image
	img, format, err := image.Decode(file)
	if err != nil {
		// If standard decode fails, try decoding explicitly if needed, but standard usually works for jpg/png
		return nil, "", err
	}
	log.Printf("Processing image: %s (format: %s)", filename, format)

	// 2. Resize if too large (Max Width 2000px)
	bounds := img.Bounds()
	if bounds.Dx() > 2000 {
		img = imaging.Resize(img, 2000, 0, imaging.Lanczos)
	}

	// 3. Prepare Buffer
	var buf bytes.Buffer

	// 4. Encode as WebP
	// Quality: 85 is excellent sweet spot. Lossless: false.
	err = webp.Encode(&buf, img, &webp.Options{
		Lossless: false,
		Quality:  85,
	})
	if err != nil {
		// If WebP fails, fallback to JPEG
		log.Printf("WebP encoding failed, falling back to JPEG: %v", err)
		err = jpeg.Encode(&buf, img, &jpeg.Options{Quality: 85})
		if err != nil {
			return nil, "", err
		}
		return buf.Bytes(), "image/jpeg", nil
	}

	return buf.Bytes(), "image/webp", nil
}

// IsImage verifies simple content type
func IsImage(contentType string) bool {
	return contentType == "image/jpeg" || contentType == "image/png" || contentType == "image/webp" || contentType == "image/jpg"
}
