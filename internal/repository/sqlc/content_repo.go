package sqlcrepo

import (
	"context"
	"encoding/json"
	"rokomferi-backend/db/sqlc"
	"rokomferi-backend/internal/domain"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

type ContentRepository interface {
	GetContentByKey(ctx context.Context, key string) (*domain.ContentBlock, error)
	UpsertContent(ctx context.Context, key string, content interface{}) (*domain.ContentBlock, error)
}

type contentRepository struct {
	q *sqlc.Queries
}

func NewContentRepository(db sqlc.DBTX) ContentRepository {
	return &contentRepository{
		q: sqlc.New(db),
	}
}

func (r *contentRepository) GetContentByKey(ctx context.Context, key string) (*domain.ContentBlock, error) {
	content, err := r.q.GetContentByKey(ctx, key)
	if err != nil {
		return nil, err
	}

	return &domain.ContentBlock{
		ID:         uuidToString(content.ID),
		SectionKey: content.SectionKey,
		Content:    domain.RawJSON(content.Content),
		UpdatedAt:  pgtimestamptzToTime(content.UpdatedAt),
	}, nil
}

func (r *contentRepository) UpsertContent(ctx context.Context, key string, data interface{}) (*domain.ContentBlock, error) {
	bytes, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	content, err := r.q.UpsertContent(ctx, sqlc.UpsertContentParams{
		SectionKey: key,
		Content:    bytes,
	})
	if err != nil {
		return nil, err
	}

	return &domain.ContentBlock{
		ID:         uuidToString(content.ID),
		SectionKey: content.SectionKey,
		Content:    domain.RawJSON(content.Content),
		UpdatedAt:  pgtimestamptzToTime(content.UpdatedAt),
	}, nil
}

func pgtimestamptzToTime(t pgtype.Timestamptz) time.Time {
	if t.Valid {
		return t.Time
	}
	return time.Time{}
}
