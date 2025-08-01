package repository

import (
	"context"
	"pot-play-storage/internal/model"
)

// FileRepositoryInterface defines the interface for file repository operations
type FileRepositoryInterface interface {
	// Storage operations
	GetStorageByHash(ctx context.Context, hash string) (*model.Storage, error)
	CreateStorage(ctx context.Context, storage *model.Storage) error
	IncrementStorageRef(ctx context.Context, hash string) error
	DecrementStorageRef(ctx context.Context, hash string) (int, error)
	DeleteStorage(ctx context.Context, hash string) error
	
	// File operations
	CreateFileReference(ctx context.Context, file *model.File) error
	GetByID(ctx context.Context, id string) (*model.File, error)
	GetStorageHashByFileID(ctx context.Context, id string) (string, error)
	DeleteByID(ctx context.Context, id string) error
	List(ctx context.Context) ([]model.File, error)
	
	// Legacy operations for backwards compatibility
	GetByChecksum(ctx context.Context, checksum string) (*model.File, error)
	GetByStorageHash(ctx context.Context, hash string) (*model.File, error)
	Create(ctx context.Context, file *model.File) error
}