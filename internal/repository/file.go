package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"pot-play-storage/internal/model"
)

type FileRepository struct {
	db     *pgxpool.Pool
	cache  *redis.Client
	logger *zap.Logger
}

// Ensure FileRepository implements FileRepositoryInterface
var _ FileRepositoryInterface = (*FileRepository)(nil)

func NewFileRepository(db *pgxpool.Pool, cache *redis.Client, logger *zap.Logger) *FileRepository {
	return &FileRepository{db: db, cache: cache, logger: logger}
}

// Storage-related methods

// GetStorageByHash retrieves storage record by hash
func (r *FileRepository) GetStorageByHash(ctx context.Context, hash string) (*model.Storage, error) {
	var storage model.Storage
	err := r.db.QueryRow(ctx, 
		"SELECT hash, size, content_type, storage_path, reference_count, created_at, updated_at FROM storage WHERE hash = $1",
		hash).Scan(&storage.Hash, &storage.Size, &storage.ContentType, &storage.StoragePath, 
		&storage.ReferenceCount, &storage.CreatedAt, &storage.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &storage, nil
}

// CreateStorage creates a new storage record
func (r *FileRepository) CreateStorage(ctx context.Context, storage *model.Storage) error {
	_, err := r.db.Exec(ctx,
		"INSERT INTO storage (hash, size, content_type, storage_path, reference_count, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6, $7)",
		storage.Hash, storage.Size, storage.ContentType, storage.StoragePath, 
		storage.ReferenceCount, storage.CreatedAt, storage.UpdatedAt)
	return err
}

// IncrementStorageRef increments the reference count for a storage record
func (r *FileRepository) IncrementStorageRef(ctx context.Context, hash string) error {
	_, err := r.db.Exec(ctx,
		"UPDATE storage SET reference_count = reference_count + 1, updated_at = CURRENT_TIMESTAMP WHERE hash = $1",
		hash)
	return err
}

// DecrementStorageRef decrements the reference count and returns the new count
func (r *FileRepository) DecrementStorageRef(ctx context.Context, hash string) (int, error) {
	var newCount int
	err := r.db.QueryRow(ctx,
		"UPDATE storage SET reference_count = reference_count - 1, updated_at = CURRENT_TIMESTAMP WHERE hash = $1 RETURNING reference_count",
		hash).Scan(&newCount)
	return newCount, err
}

// DeleteStorage removes a storage record (should only be called when reference_count = 0)
func (r *FileRepository) DeleteStorage(ctx context.Context, hash string) error {
	_, err := r.db.Exec(ctx, "DELETE FROM storage WHERE hash = $1 AND reference_count = 0", hash)
	return err
}

// Legacy method for backwards compatibility
func (r *FileRepository) GetByChecksum(ctx context.Context, checksum string) (*model.File, error) {
	// First try new schema
	file, err := r.GetByStorageHash(ctx, checksum)
	if err == nil {
		return file, nil
	}
	
	// Fallback to legacy schema
	var legacyFile model.File
	err = r.db.QueryRow(ctx, "SELECT id, name, size, content_type, checksum, created_at, storage_path FROM files WHERE checksum = $1", checksum).
		Scan(&legacyFile.ID, &legacyFile.Name, &legacyFile.Size, &legacyFile.ContentType, &legacyFile.Checksum, &legacyFile.CreatedAt, &legacyFile.StoragePath)
	if err != nil {
		return nil, err
	}
	return &legacyFile, nil
}

// GetByStorageHash retrieves a file by storage hash (new method)
func (r *FileRepository) GetByStorageHash(ctx context.Context, hash string) (*model.File, error) {
	var file model.File
	var storage model.Storage
	
	err := r.db.QueryRow(ctx, `
		SELECT f.id, f.name, f.storage_hash, f.created_at,
		       s.hash, s.size, s.content_type, s.storage_path, s.reference_count, s.created_at, s.updated_at
		FROM files f 
		JOIN storage s ON f.storage_hash = s.hash 
		WHERE f.storage_hash = $1 
		LIMIT 1`, hash).
		Scan(&file.ID, &file.Name, &file.StorageHash, &file.CreatedAt,
			&storage.Hash, &storage.Size, &storage.ContentType, &storage.StoragePath,
			&storage.ReferenceCount, &storage.CreatedAt, &storage.UpdatedAt)
	
	if err != nil {
		return nil, err
	}
	
	file.Storage = &storage
	// Set legacy fields for backwards compatibility
	file.Size = storage.Size
	file.ContentType = storage.ContentType
	file.StoragePath = storage.StoragePath
	file.Checksum = storage.Hash
	
	return &file, nil
}

// CreateFileReference creates a new file record referencing existing storage
func (r *FileRepository) CreateFileReference(ctx context.Context, file *model.File) error {
	_, err := r.db.Exec(ctx, 
		"INSERT INTO files (id, name, storage_hash, created_at) VALUES ($1, $2, $3, $4)",
		file.ID, file.Name, file.StorageHash, file.CreatedAt)
	// Invalidate list cache
	if err == nil {
		r.cache.Del(ctx, "file_list")
	}
	return err
}

// Legacy Create method for backwards compatibility
func (r *FileRepository) Create(ctx context.Context, file *model.File) error {
	// If we have a storage hash, use the new method
	if file.StorageHash != "" {
		return r.CreateFileReference(ctx, file)
	}
	
	// Otherwise use legacy method
	_, err := r.db.Exec(ctx, "INSERT INTO files (id, name, size, content_type, storage_path, checksum, created_at) VALUES ($1, $2, $3, $4, $5, $6, $7)",
		file.ID, file.Name, file.Size, file.ContentType, file.StoragePath, file.Checksum, file.CreatedAt)
	// Invalidate list cache
	if err == nil {
		r.cache.Del(ctx, "file_list")
	}
	return err
}

func (r *FileRepository) GetByID(ctx context.Context, id string) (*model.File, error) {
	cacheKey := fmt.Sprintf("file:%s", id)
	
	// Try to get from cache
	val, err := r.cache.Get(ctx, cacheKey).Result()
	if err == nil {
		var file model.File
		if err := json.Unmarshal([]byte(val), &file); err == nil {
			return &file, nil
		}
	}
	
	// Try new schema first
	var file model.File
	var storage model.Storage
	
	err = r.db.QueryRow(ctx, `
		SELECT f.id, f.name, f.storage_hash, f.created_at,
		       s.hash, s.size, s.content_type, s.storage_path, s.reference_count, s.created_at, s.updated_at
		FROM files f 
		JOIN storage s ON f.storage_hash = s.hash 
		WHERE f.id = $1`, id).
		Scan(&file.ID, &file.Name, &file.StorageHash, &file.CreatedAt,
			&storage.Hash, &storage.Size, &storage.ContentType, &storage.StoragePath,
			&storage.ReferenceCount, &storage.CreatedAt, &storage.UpdatedAt)
	
	if err == nil {
		file.Storage = &storage
		// Set legacy fields for backwards compatibility
		file.Size = storage.Size
		file.ContentType = storage.ContentType
		file.StoragePath = storage.StoragePath
		file.Checksum = storage.Hash
		
		// Cache the result
		if data, err := json.Marshal(file); err == nil {
			r.cache.Set(ctx, cacheKey, data, 1*time.Hour)
		}
		
		return &file, nil
	}
	
	// Fallback to legacy schema
	err = r.db.QueryRow(ctx, "SELECT id, name, size, content_type, checksum, created_at, storage_path FROM files WHERE id = $1", id).
		Scan(&file.ID, &file.Name, &file.Size, &file.ContentType, &file.Checksum, &file.CreatedAt, &file.StoragePath)
	if err != nil {
		return nil, err
	}
	
	// Cache the result
	if data, err := json.Marshal(file); err == nil {
		r.cache.Set(ctx, cacheKey, data, 1*time.Hour)
	}
	
	return &file, nil
}

// GetStorageHashByFileID gets the storage hash for a file (needed for deletion)
func (r *FileRepository) GetStorageHashByFileID(ctx context.Context, id string) (string, error) {
	var storageHash string
	
	// Try new schema first
	err := r.db.QueryRow(ctx, "SELECT storage_hash FROM files WHERE id = $1", id).Scan(&storageHash)
	if err == nil && storageHash != "" {
		return storageHash, nil
	}
	
	// Fallback to legacy schema
	var checksum string
	err = r.db.QueryRow(ctx, "SELECT checksum FROM files WHERE id = $1", id).Scan(&checksum)
	return checksum, err
}

func (r *FileRepository) DeleteByID(ctx context.Context, id string) error {
	_, err := r.db.Exec(ctx, "DELETE FROM files WHERE id = $1", id)
	// Invalidate caches
	r.cache.Del(ctx, fmt.Sprintf("file:%s", id))
	r.cache.Del(ctx, "file_list")
	return err
}

func (r *FileRepository) List(ctx context.Context) ([]model.File, error) {
	cacheKey := "file_list"
	
	// Try to get from cache
	val, err := r.cache.Get(ctx, cacheKey).Result()
	if err == nil {
		var files []model.File
		if err := json.Unmarshal([]byte(val), &files); err == nil {
			return files, nil
		}
	}
	
	// Try new schema first
	rows, err := r.db.Query(ctx, `
		SELECT f.id, f.name, f.storage_hash, f.created_at,
		       s.hash, s.size, s.content_type, s.storage_path, s.reference_count, s.created_at, s.updated_at
		FROM files f 
		JOIN storage s ON f.storage_hash = s.hash 
		ORDER BY f.created_at DESC`)
	
	if err == nil {
		defer rows.Close()
		var files []model.File
		
		for rows.Next() {
			var file model.File
			var storage model.Storage
			
			if err := rows.Scan(&file.ID, &file.Name, &file.StorageHash, &file.CreatedAt,
				&storage.Hash, &storage.Size, &storage.ContentType, &storage.StoragePath,
				&storage.ReferenceCount, &storage.CreatedAt, &storage.UpdatedAt); err != nil {
				return nil, err
			}
			
			file.Storage = &storage
			// Set legacy fields for backwards compatibility
			file.Size = storage.Size
			file.ContentType = storage.ContentType
			file.StoragePath = storage.StoragePath
			file.Checksum = storage.Hash
			
			files = append(files, file)
		}
		
		// Cache the result
		if data, err := json.Marshal(files); err == nil {
			r.cache.Set(ctx, cacheKey, data, 10*time.Minute)
		}
		
		return files, nil
	}
	
	// Fallback to legacy schema
	rows, err = r.db.Query(ctx, "SELECT id, name, size, content_type, checksum, created_at, storage_path FROM files ORDER BY created_at DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var files []model.File
	for rows.Next() {
		var file model.File
		if err := rows.Scan(&file.ID, &file.Name, &file.Size, &file.ContentType, &file.Checksum, &file.CreatedAt, &file.StoragePath); err != nil {
			return nil, err
		}
		files = append(files, file)
	}
	
	// Cache the result
	if data, err := json.Marshal(files); err == nil {
		r.cache.Set(ctx, cacheKey, data, 10*time.Minute)
	}
	
	return files, nil
}