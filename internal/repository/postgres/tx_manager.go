package postgres

import (
	"context"
	"rokomferi-backend/internal/domain"

	"gorm.io/gorm"
)

type transactionManager struct {
	db *gorm.DB
}

func NewTransactionManager(db *gorm.DB) domain.TransactionManager {
	return &transactionManager{db: db}
}

type contextKey string

const txKey contextKey = "tx"

func (tm *transactionManager) Do(ctx context.Context, fn func(ctx context.Context) error) error {
	return tm.db.Transaction(func(tx *gorm.DB) error {
		// Pass the transaction DB instance in context
		txCtx := context.WithValue(ctx, txKey, tx)
		return fn(txCtx)
	})
}

// Helper to extract DB from context or return default
func getDB(ctx context.Context, defaultDB *gorm.DB) *gorm.DB {
	if tx, ok := ctx.Value(txKey).(*gorm.DB); ok {
		return tx
	}
	return defaultDB.WithContext(ctx)
}
