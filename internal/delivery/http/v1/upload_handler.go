package v1

import (
	"log"
	"net/http"
	"rokomferi-backend/pkg/storage"
	"rokomferi-backend/pkg/utils"
)

type UploadHandler struct {
	storage *storage.R2Storage
}

func NewUploadHandler(s *storage.R2Storage) *UploadHandler {
	return &UploadHandler{storage: s}
}

func (h *UploadHandler) UploadFile(w http.ResponseWriter, r *http.Request) {
	// 1. Parse Multipart Form (10 MB limit)
	err := r.ParseMultipartForm(10 << 20)
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, "File too large or invalid format")
		return
	}

	// 2. Get File
	file, header, err := r.FormFile("file")
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, "Invalid file")
		return
	}
	defer file.Close()

	// 3. Upload to R2
	url, err := h.storage.UploadFile(file, header)
	if err != nil {
		log.Printf("R2 Upload Error: %v", err)
		utils.WriteError(w, http.StatusInternalServerError, "Failed to upload file")
		return
	}

	// 4. Return URL
	utils.WriteJSON(w, http.StatusOK, map[string]string{
		"url": url,
	})
}
