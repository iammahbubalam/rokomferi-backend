package domain

import (
	"errors"
	"time"
)

type ContentBlock struct {
	ID         string    `json:"id"`
	SectionKey string    `json:"section_key"`
	Content    RawJSON   `json:"content"`
	UpdatedAt  time.Time `json:"updated_at"`
}

var (
	ErrContentNotFound = errors.New("content not found")
)
