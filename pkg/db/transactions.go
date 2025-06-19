package db

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// LockMode defines the type of lock to acquire
type LockMode string

const (
	// LockForUpdate locks the selected rows for update (exclusive lock)
	LockForUpdate LockMode = "FOR UPDATE"

	// LockForShare locks the selected rows for shared access (shared lock)
	LockForShare LockMode = "FOR SHARE"

	// NoWait specifies that the lock operation should not wait
	NoWait LockMode = "NOWAIT"

	// SkipLocked specifies that locked rows should be skipped
	SkipLocked LockMode = "SKIP LOCKED"
)

var (
	// ErrRowLocked indicates that a row is already locked
	ErrRowLocked = errors.New("row is locked by another transaction")
)

// TxFn is a function that executes in a transaction
type TxFn func(tx *gorm.DB) error

// WithTransaction runs the given function in a database transaction
func WithTransaction(ctx context.Context, fn TxFn) error {
	return withTransactionDB(DB, ctx, fn)
}

// withTransactionDB runs a transaction on the provided DB instance
func withTransactionDB(db *gorm.DB, ctx context.Context, fn TxFn) error {
	tx := db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return fmt.Errorf("failed to begin transaction: %w", tx.Error)
	}

	defer func() {
		// Handle panic
		if r := recover(); r != nil {
			tx.Rollback()
			panic(r) // re-throw panic after rollback
		}
	}()

	if err := fn(tx); err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// ApplyLock applies the specified lock mode to a GORM query
func ApplyLock(query *gorm.DB, lockMode LockMode) *gorm.DB {
	switch {
	case string(lockMode) == string(LockForUpdate):
		return query.Clauses(clause.Locking{Strength: "UPDATE"})
	case string(lockMode) == string(LockForShare):
		return query.Clauses(clause.Locking{Strength: "SHARE"})
	case string(lockMode) == string(LockForUpdate)+" "+string(NoWait):
		return query.Clauses(clause.Locking{Strength: "UPDATE", Options: "NOWAIT"})
	case string(lockMode) == string(LockForShare)+" "+string(NoWait):
		return query.Clauses(clause.Locking{Strength: "SHARE", Options: "NOWAIT"})
	case string(lockMode) == string(LockForUpdate)+" "+string(SkipLocked):
		return query.Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"})
	case string(lockMode) == string(LockForShare)+" "+string(SkipLocked):
		return query.Clauses(clause.Locking{Strength: "SHARE", Options: "SKIP LOCKED"})
	default:
		return query
	}
}

// WithLock executes a function with row locking
func WithLock(ctx context.Context, model interface{}, lockMode LockMode, condition string, args []interface{}, fn TxFn) error {
	return WithTransaction(ctx, func(tx *gorm.DB) error {
		// Apply locking based on the specified mode
		query := ApplyLock(tx, lockMode)

		// Find the record with locking
		if err := query.Where(condition, args...).First(model).Error; err != nil {
			// Check for NOWAIT errors
			if err.Error() == "could not obtain lock on row" {
				return ErrRowLocked
			}
			return err
		}

		// Execute the function with the locked record
		return fn(tx)
	})
}

// IsLockError checks if an error is related to locking
func IsLockError(err error) bool {
	if err == nil {
		return false
	}

	errorMsg := err.Error()
	lockErrorIndicators := []string{
		"could not obtain lock on row",
		"deadlock detected",
		"lock timeout",
		"lock not available",
	}

	for _, indicator := range lockErrorIndicators {
		if errorMsg == indicator {
			return true
		}
	}

	return false
}
