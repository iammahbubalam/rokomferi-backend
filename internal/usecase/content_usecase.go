package usecase

import (
	"context"
	"rokomferi-backend/internal/domain"
	repo "rokomferi-backend/internal/repository/sqlc"
)

type ContentUsecase interface {
	GetContent(ctx context.Context, key string) (*domain.ContentBlock, error)
	UpsertContent(ctx context.Context, key string, content interface{}) (*domain.ContentBlock, error)
}

type contentUsecase struct {
	repo repo.ContentRepository
}

func NewContentUsecase(r repo.ContentRepository) ContentUsecase {
	return &contentUsecase{repo: r}
}

func (u *contentUsecase) GetContent(ctx context.Context, key string) (*domain.ContentBlock, error) {
	return u.repo.GetContentByKey(ctx, key)
}

func (u *contentUsecase) UpsertContent(ctx context.Context, key string, content interface{}) (*domain.ContentBlock, error) {
	// TODO: Add validation if needed, for specific keys
	return u.repo.UpsertContent(ctx, key, content)
}
