package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"pot-play-storage/internal/model"
	"pot-play-storage/internal/repository"
	"pot-play-storage/pkg/storage"
	"pot-play-storage/pkg/validator"
)

type StorageService struct {
	storage storage.Storage
	repo    repository.FileRepositoryInterface
	logger  *zap.Logger
}

func NewStorageService(st storage.Storage, repo repository.FileRepositoryInterface, logger *zap.Logger) *StorageService {
	return &StorageService{storage: st, repo: repo, logger: logger}
}

func (s *StorageService) Upload(ctx context.Context, reader io.Reader, header *model.FileHeader) (*model.File, error) {
	if err := validator.ValidateFile(header); err != nil {
		return nil, err
	}

	// Calculate checksum while reading the file
	hasher := sha256.New()
	teeReader := io.TeeReader(reader, hasher)

	// First, store the file content to calculate the hash
	path := uuid.NewString()
	size, err := s.storage.Put(ctx, path, teeReader, header.Size)
	if err != nil {
		return nil, err
	}

	checksum := hex.EncodeToString(hasher.Sum(nil))

	// Check if storage with this hash already exists
	existingStorage, err := s.repo.GetStorageByHash(ctx, checksum)
	if err == nil && existingStorage != nil {
		// Duplicate content found, delete the newly uploaded file
		s.storage.Delete(ctx, path)
		
		// Increment reference count for existing storage
		if err := s.repo.IncrementStorageRef(ctx, checksum); err != nil {
			s.logger.Error("Failed to increment storage reference", 
				zap.Error(err), zap.String("hash", checksum))
			return nil, err
		}

		// Create a new file record referencing the existing storage
		fileID := uuid.NewString()
		file := &model.File{
			ID:          fileID,
			Name:        header.Name,
			StorageHash: checksum,
			CreatedAt:   time.Now(),
			Storage:     existingStorage,
			// Set legacy fields for backwards compatibility
			Size:        existingStorage.Size,
			ContentType: existingStorage.ContentType,
			StoragePath: existingStorage.StoragePath,
			Checksum:    existingStorage.Hash,
		}

		if err := s.repo.CreateFileReference(ctx, file); err != nil {
			// If file creation fails, decrement the reference count
			s.repo.DecrementStorageRef(ctx, checksum)
			return nil, err
		}

		s.logger.Info("Duplicate file content detected, created reference",
			zap.String("checksum", checksum),
			zap.String("file_id", fileID),
			zap.Int("ref_count", existingStorage.ReferenceCount+1))

		return file, nil
	}

	// No duplicate found, create new storage record
	now := time.Now()
	storageRecord := &model.Storage{
		Hash:           checksum,
		Size:           size,
		ContentType:    header.ContentType,
		StoragePath:    path,
		ReferenceCount: 1,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if err := s.repo.CreateStorage(ctx, storageRecord); err != nil {
		s.storage.Delete(ctx, path)
		return nil, err
	}

	// Create the file record
	fileID := uuid.NewString()
	file := &model.File{
		ID:          fileID,
		Name:        header.Name,
		StorageHash: checksum,
		CreatedAt:   now,
		Storage:     storageRecord,
		// Set legacy fields for backwards compatibility
		Size:        size,
		ContentType: header.ContentType,
		StoragePath: path,
		Checksum:    checksum,
	}

	if err := s.repo.CreateFileReference(ctx, file); err != nil {
		// If file creation fails, clean up storage
		s.repo.DeleteStorage(ctx, checksum)
		s.storage.Delete(ctx, path)
		return nil, err
	}

	s.logger.Info("New file uploaded",
		zap.String("checksum", checksum),
		zap.String("file_id", fileID),
		zap.Int64("size", size))

	return file, nil
}

func (s *StorageService) Download(ctx context.Context, id string) (io.ReadCloser, *model.File, error) {
	file, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, nil, err
	}
	
	// Use the helper method to get storage path
	storagePath := file.GetStoragePath()
	reader, err := s.storage.Get(ctx, storagePath)
	if err != nil {
		return nil, nil, err
	}
	return reader, file, nil
}

func (s *StorageService) Delete(ctx context.Context, id string) error {
	// Get the storage hash for this file
	storageHash, err := s.repo.GetStorageHashByFileID(ctx, id)
	if err != nil {
		return err
	}

	// Delete the file record first
	if err := s.repo.DeleteByID(ctx, id); err != nil {
		return err
	}

	// Decrement the reference count for the storage
	newRefCount, err := s.repo.DecrementStorageRef(ctx, storageHash)
	if err != nil {
		s.logger.Error("Failed to decrement storage reference", 
			zap.Error(err), zap.String("hash", storageHash))
		// Even if we can't decrement, the file is deleted, so continue
	}

	s.logger.Info("File deleted, storage reference decremented",
		zap.String("file_id", id),
		zap.String("hash", storageHash),
		zap.Int("new_ref_count", newRefCount))

	// If reference count is now 0, delete the actual storage
	if newRefCount == 0 {
		// Get the storage record to find the storage path
		storage, err := s.repo.GetStorageByHash(ctx, storageHash)
		if err != nil {
			s.logger.Error("Failed to get storage record for cleanup", 
				zap.Error(err), zap.String("hash", storageHash))
			return nil // File was deleted successfully, storage cleanup is secondary
		}

		// Delete the actual file from storage
		if err := s.storage.Delete(ctx, storage.StoragePath); err != nil {
			s.logger.Error("Failed to delete storage file", 
				zap.Error(err), zap.String("path", storage.StoragePath))
			// Don't return error as file metadata was deleted successfully
		}

		// Delete the storage record
		if err := s.repo.DeleteStorage(ctx, storageHash); err != nil {
			s.logger.Error("Failed to delete storage record", 
				zap.Error(err), zap.String("hash", storageHash))
			// Don't return error as file metadata was deleted successfully
		}

		s.logger.Info("Storage cleaned up",
			zap.String("hash", storageHash),
			zap.String("path", storage.StoragePath))
	}

	return nil
}

func (s *StorageService) List(ctx context.Context) ([]model.File, error) {
	return s.repo.List(ctx)
}