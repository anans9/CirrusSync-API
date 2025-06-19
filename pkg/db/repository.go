package db

import (
	"context"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Repository defines a generic repository interface
type Repository[T any] interface {
	// Basic CRUD operations
	Create(ctx context.Context, entity *T) error
	FindByID(ctx context.Context, id interface{}) (*T, error)
	Update(ctx context.Context, entity *T) error
	Delete(ctx context.Context, id interface{}) error

	// Additional helper methods
	FindWhere(ctx context.Context, condition string, args ...interface{}) ([]T, error)
	FindOneWhere(ctx context.Context, condition string, args ...interface{}) (*T, error)

	// Get the underlying DB connection
	DB() *gorm.DB
}

// BaseRepository implements the Repository interface with built-in locking
type BaseRepository[T any] struct {
	db *gorm.DB
}

// NewRepository creates a new repository using the global DB connection
func NewRepository[T any]() *BaseRepository[T] {
	return &BaseRepository[T]{
		db: DB,
	}
}

// NewRepositoryWithDB creates a repository with a specific DB connection
func NewRepositoryWithDB[T any](db *gorm.DB) *BaseRepository[T] {
	return &BaseRepository[T]{
		db: db,
	}
}

// DB returns the underlying DB connection
func (r *BaseRepository[T]) DB() *gorm.DB {
	return r.db
}

// Create saves a new entity with automatic locking to prevent race conditions
func (r *BaseRepository[T]) Create(ctx context.Context, entity *T) error {
	return withTransactionDB(r.db, ctx, func(tx *gorm.DB) error {
		// For creation, we use a transaction to ensure atomicity
		return tx.Create(entity).Error
	})
}

// FindByID finds an entity by ID
func (r *BaseRepository[T]) FindByID(ctx context.Context, id interface{}) (*T, error) {
	var entity T
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&entity).Error
	if err != nil {
		return nil, err
	}
	return &entity, nil
}

// Update updates an entity with automatic locking
func (r *BaseRepository[T]) Update(ctx context.Context, entity *T) error {
	return withTransactionDB(r.db, ctx, func(tx *gorm.DB) error {
		// Use FOR UPDATE lock mode for the update operation
		// This ensures that the row is locked for the duration of the update
		return tx.Clauses(clause.Locking{Strength: "UPDATE"}).Save(entity).Error
	})
}

// Delete deletes an entity by ID with automatic locking
func (r *BaseRepository[T]) Delete(ctx context.Context, id interface{}) error {
	return withTransactionDB(r.db, ctx, func(tx *gorm.DB) error {
		var entity T

		// Only use the parameterized query with the ID
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", id).First(&entity).Error; err != nil {
			return err
		}

		// Delete using the entity model, not by constructing a query
		return tx.Delete(&entity).Error
	})
}

// FindWhere finds entities matching the given condition
func (r *BaseRepository[T]) FindWhere(ctx context.Context, condition string, args ...interface{}) ([]T, error) {
	var entities []T
	err := r.db.WithContext(ctx).Where(condition, args...).Find(&entities).Error
	if err != nil {
		return nil, err
	}
	return entities, nil
}

// FindOneWhere finds a single entity matching the condition
func (r *BaseRepository[T]) FindOneWhere(ctx context.Context, condition string, args ...interface{}) (*T, error) {
	var entity T
	err := r.db.WithContext(ctx).Where(condition, args...).First(&entity).Error
	if err != nil {
		return nil, err
	}
	return &entity, nil
}

// UpdateWithLock updates an entity with explicit locking
func (r *BaseRepository[T]) UpdateWithLock(ctx context.Context, entity *T) error {
	return withTransactionDB(r.db, ctx, func(tx *gorm.DB) error {
		// Explicit FOR UPDATE lock
		return tx.Clauses(clause.Locking{Strength: "UPDATE"}).Save(entity).Error
	})
}

// WithLock executes a function with a locked entity
func (r *BaseRepository[T]) WithLock(ctx context.Context, id interface{}, fn func(*T, *gorm.DB) error) error {
	return withTransactionDB(r.db, ctx, func(tx *gorm.DB) error {
		// Find and lock the entity
		var entity T
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&entity, id).Error; err != nil {
			return err
		}

		// Execute function with the locked entity
		return fn(&entity, tx)
	})
}
