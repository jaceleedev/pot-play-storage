package service

import (
	"bytes"
	"context"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"

	"pot-play-storage/internal/model"
)

// SimpleInMemoryStorage for testing
type SimpleInMemoryStorage struct {
	files map[string][]byte
}

func NewSimpleInMemoryStorage() *SimpleInMemoryStorage {
	return &SimpleInMemoryStorage{
		files: make(map[string][]byte),
	}
}

func (s *SimpleInMemoryStorage) Put(ctx context.Context, path string, reader io.Reader, size int64) (int64, error) {
	data, err := io.ReadAll(reader)
	if err != nil {
		return 0, err
	}
	s.files[path] = data
	return int64(len(data)), nil
}

func (s *SimpleInMemoryStorage) Get(ctx context.Context, path string) (io.ReadCloser, error) {
	if data, exists := s.files[path]; exists {
		return io.NopCloser(bytes.NewReader(data)), nil
	}
	return nil, assert.AnError
}

func (s *SimpleInMemoryStorage) Delete(ctx context.Context, path string) error {
	delete(s.files, path)
	return nil
}

func (s *SimpleInMemoryStorage) List(ctx context.Context, prefix string) ([]string, error) {
	var keys []string
	for k := range s.files {
		keys = append(keys, k)
	}
	return keys, nil
}

// SimpleInMemoryRepository for testing
type SimpleInMemoryRepository struct {
	storages map[string]*model.Storage
	files    map[string]*model.File
}

func NewSimpleInMemoryRepository() *SimpleInMemoryRepository {
	return &SimpleInMemoryRepository{
		storages: make(map[string]*model.Storage),
		files:    make(map[string]*model.File),
	}
}

func (r *SimpleInMemoryRepository) GetStorageByHash(ctx context.Context, hash string) (*model.Storage, error) {
	if storage, exists := r.storages[hash]; exists {
		return storage, nil
	}
	return nil, assert.AnError
}

func (r *SimpleInMemoryRepository) CreateStorage(ctx context.Context, storage *model.Storage) error {
	r.storages[storage.Hash] = storage
	return nil
}

func (r *SimpleInMemoryRepository) IncrementStorageRef(ctx context.Context, hash string) error {
	if storage, exists := r.storages[hash]; exists {
		storage.ReferenceCount++
	}
	return nil
}

func (r *SimpleInMemoryRepository) DecrementStorageRef(ctx context.Context, hash string) (int, error) {
	if storage, exists := r.storages[hash]; exists {
		storage.ReferenceCount--
		return storage.ReferenceCount, nil
	}
	return 0, assert.AnError
}

func (r *SimpleInMemoryRepository) DeleteStorage(ctx context.Context, hash string) error {
	delete(r.storages, hash)
	return nil
}

func (r *SimpleInMemoryRepository) CreateFileReference(ctx context.Context, file *model.File) error {
	r.files[file.ID] = file
	return nil
}

func (r *SimpleInMemoryRepository) GetByID(ctx context.Context, id string) (*model.File, error) {
	if file, exists := r.files[id]; exists {
		return file, nil
	}
	return nil, assert.AnError
}

func (r *SimpleInMemoryRepository) GetStorageHashByFileID(ctx context.Context, id string) (string, error) {
	if file, exists := r.files[id]; exists {
		return file.StorageHash, nil
	}
	return "", assert.AnError
}

func (r *SimpleInMemoryRepository) DeleteByID(ctx context.Context, id string) error {
	delete(r.files, id)
	return nil
}

func (r *SimpleInMemoryRepository) List(ctx context.Context) ([]model.File, error) {
	var files []model.File
	for _, file := range r.files {
		files = append(files, *file)
	}
	return files, nil
}

// Legacy methods (not used in this test)
func (r *SimpleInMemoryRepository) GetByChecksum(ctx context.Context, checksum string) (*model.File, error) {
	return nil, nil
}

func (r *SimpleInMemoryRepository) GetByStorageHash(ctx context.Context, hash string) (*model.File, error) {
	return nil, nil
}

func (r *SimpleInMemoryRepository) Create(ctx context.Context, file *model.File) error {
	return nil
}

func TestDeduplicationIntegration(t *testing.T) {
	// Setup
	storage := NewSimpleInMemoryStorage()
	repo := NewSimpleInMemoryRepository()
	logger := zap.NewNop()
	service := NewStorageService(storage, repo, logger)

	ctx := context.Background()
	content := "Hello, World! This is a test file."
	
	// Upload first file
	header1 := &model.FileHeader{
		Name:        "file1.txt",
		Size:        int64(len(content)),
		ContentType: "text/plain",
	}
	
	file1, err := service.Upload(ctx, strings.NewReader(content), header1)
	assert.NoError(t, err)
	assert.NotNil(t, file1)
	assert.Equal(t, "file1.txt", file1.Name)
	
	// Verify storage was created
	storage1, err := repo.GetStorageByHash(ctx, file1.StorageHash)
	assert.NoError(t, err)
	assert.Equal(t, 1, storage1.ReferenceCount)
	
	// Upload identical content with different name
	header2 := &model.FileHeader{
		Name:        "file2.txt",
		Size:        int64(len(content)),
		ContentType: "text/plain",
	}
	
	file2, err := service.Upload(ctx, strings.NewReader(content), header2)
	assert.NoError(t, err)
	assert.NotNil(t, file2)
	assert.Equal(t, "file2.txt", file2.Name)
	
	// Files should have same storage hash (deduplication worked)
	assert.Equal(t, file1.StorageHash, file2.StorageHash)
	
	// Verify reference count increased
	storage2, err := repo.GetStorageByHash(ctx, file2.StorageHash)
	assert.NoError(t, err)
	assert.Equal(t, 2, storage2.ReferenceCount)
	
	// Verify only one physical file was stored
	assert.Equal(t, 1, len(storage.files))
	
	// Test downloading both files returns same content
	reader1, retrievedFile1, err := service.Download(ctx, file1.ID)
	assert.NoError(t, err)
	defer reader1.Close()
	
	reader2, retrievedFile2, err := service.Download(ctx, file2.ID)
	assert.NoError(t, err)
	defer reader2.Close()
	
	// Both files should have same content but different names
	assert.Equal(t, retrievedFile1.GetContentType(), retrievedFile2.GetContentType())
	assert.Equal(t, retrievedFile1.GetSize(), retrievedFile2.GetSize())
	assert.NotEqual(t, retrievedFile1.Name, retrievedFile2.Name)
	
	// Test deletion - delete first file
	err = service.Delete(ctx, file1.ID)
	assert.NoError(t, err)
	
	// Storage should still exist with ref count 1
	storageAfterDelete, err := repo.GetStorageByHash(ctx, file1.StorageHash)
	assert.NoError(t, err)
	assert.Equal(t, 1, storageAfterDelete.ReferenceCount)
	
	// Physical file should still exist
	assert.Equal(t, 1, len(storage.files))
	
	// Second file should still be downloadable
	reader3, _, err := service.Download(ctx, file2.ID)
	assert.NoError(t, err)
	reader3.Close()
	
	// Delete second file
	err = service.Delete(ctx, file2.ID)
	assert.NoError(t, err)
	
	// Now storage should be cleaned up
	_, err = repo.GetStorageByHash(ctx, file2.StorageHash)
	assert.Error(t, err) // Should not exist
	
	// Physical file should be deleted
	assert.Equal(t, 0, len(storage.files))
}

func TestDeduplicationDifferentContent(t *testing.T) {
	// Setup
	storage := NewSimpleInMemoryStorage()
	repo := NewSimpleInMemoryRepository()
	logger := zap.NewNop()
	service := NewStorageService(storage, repo, logger)

	ctx := context.Background()
	
	// Upload two different files
	content1 := "Hello, World!"
	header1 := &model.FileHeader{
		Name:        "file1.txt",
		Size:        int64(len(content1)),
		ContentType: "text/plain",
	}
	
	content2 := "Hello, Universe!"
	header2 := &model.FileHeader{
		Name:        "file2.txt",
		Size:        int64(len(content2)),
		ContentType: "text/plain",
	}
	
	file1, err := service.Upload(ctx, strings.NewReader(content1), header1)
	assert.NoError(t, err)
	
	file2, err := service.Upload(ctx, strings.NewReader(content2), header2)
	assert.NoError(t, err)
	
	// Files should have different storage hashes (no deduplication)
	assert.NotEqual(t, file1.StorageHash, file2.StorageHash)
	
	// Two physical files should be stored
	assert.Equal(t, 2, len(storage.files))
	
	// Each storage should have reference count 1
	storage1, err := repo.GetStorageByHash(ctx, file1.StorageHash)
	assert.NoError(t, err)
	assert.Equal(t, 1, storage1.ReferenceCount)
	
	storage2, err := repo.GetStorageByHash(ctx, file2.StorageHash)
	assert.NoError(t, err)
	assert.Equal(t, 1, storage2.ReferenceCount)
}