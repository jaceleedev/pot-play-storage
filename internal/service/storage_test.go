package service

import (
	"bytes"
	"context"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"

	"pot-play-storage/internal/model"
)

// MockStorage implements the storage.Storage interface for testing
type MockStorage struct {
	mock.Mock
	files map[string][]byte
}

func NewMockStorage() *MockStorage {
	return &MockStorage{
		files: make(map[string][]byte),
	}
}

func (m *MockStorage) Put(ctx context.Context, path string, reader io.Reader, size int64) (int64, error) {
	args := m.Called(ctx, path, reader, size)
	
	// Store the content for verification
	data, _ := io.ReadAll(reader)
	m.files[path] = data
	
	return int64(len(data)), args.Error(1)
}

func (m *MockStorage) Get(ctx context.Context, path string) (io.ReadCloser, error) {
	args := m.Called(ctx, path)
	if data, exists := m.files[path]; exists {
		return io.NopCloser(bytes.NewReader(data)), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockStorage) Delete(ctx context.Context, path string) error {
	args := m.Called(ctx, path)
	delete(m.files, path)
	return args.Error(0)
}

func (m *MockStorage) List(ctx context.Context, prefix string) ([]string, error) {
	args := m.Called(ctx, prefix)
	var keys []string
	for k := range m.files {
		keys = append(keys, k)
	}
	return keys, args.Error(1)
}

// MockFileRepository implements repository methods for testing
type MockFileRepository struct {
	mock.Mock
	storages map[string]*model.Storage
	files    map[string]*model.File
}

func NewMockFileRepository() *MockFileRepository {
	return &MockFileRepository{
		storages: make(map[string]*model.Storage),
		files:    make(map[string]*model.File),
	}
}

func (m *MockFileRepository) GetStorageByHash(ctx context.Context, hash string) (*model.Storage, error) {
	args := m.Called(ctx, hash)
	if storage, exists := m.storages[hash]; exists {
		return storage, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockFileRepository) CreateStorage(ctx context.Context, storage *model.Storage) error {
	args := m.Called(ctx, storage)
	m.storages[storage.Hash] = storage
	return args.Error(0)
}

func (m *MockFileRepository) IncrementStorageRef(ctx context.Context, hash string) error {
	args := m.Called(ctx, hash)
	if storage, exists := m.storages[hash]; exists {
		storage.ReferenceCount++
		storage.UpdatedAt = time.Now()
	}
	return args.Error(0)
}

func (m *MockFileRepository) DecrementStorageRef(ctx context.Context, hash string) (int, error) {
	args := m.Called(ctx, hash)
	if storage, exists := m.storages[hash]; exists {
		storage.ReferenceCount--
		storage.UpdatedAt = time.Now()
		return storage.ReferenceCount, args.Error(1)
	}
	return 0, args.Error(1)
}

func (m *MockFileRepository) DeleteStorage(ctx context.Context, hash string) error {
	args := m.Called(ctx, hash)
	delete(m.storages, hash)
	return args.Error(0)
}

func (m *MockFileRepository) CreateFileReference(ctx context.Context, file *model.File) error {
	args := m.Called(ctx, file)
	m.files[file.ID] = file
	return args.Error(0)
}

func (m *MockFileRepository) GetByID(ctx context.Context, id string) (*model.File, error) {
	args := m.Called(ctx, id)
	if file, exists := m.files[id]; exists {
		return file, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockFileRepository) GetStorageHashByFileID(ctx context.Context, id string) (string, error) {
	args := m.Called(ctx, id)
	if file, exists := m.files[id]; exists {
		return file.StorageHash, args.Error(1)
	}
	return "", args.Error(1)
}

func (m *MockFileRepository) DeleteByID(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	delete(m.files, id)
	return args.Error(0)
}

func (m *MockFileRepository) List(ctx context.Context) ([]model.File, error) {
	args := m.Called(ctx)
	var files []model.File
	for _, file := range m.files {
		files = append(files, *file)
	}
	return files, args.Error(1)
}

// Implement remaining interface methods as no-ops for testing
func (m *MockFileRepository) GetByChecksum(ctx context.Context, checksum string) (*model.File, error) {
	return nil, nil
}

func (m *MockFileRepository) GetByStorageHash(ctx context.Context, hash string) (*model.File, error) {
	return nil, nil
}

func (m *MockFileRepository) Create(ctx context.Context, file *model.File) error {
	return nil
}

func TestStorageService_Upload_NewFile(t *testing.T) {
	// Setup
	mockStorage := NewMockStorage()
	mockRepo := NewMockFileRepository()
	logger := zap.NewNop()
	
	service := NewStorageService(mockStorage, mockRepo, logger)
	
	// Test data
	content := "Hello, World!"
	header := &model.FileHeader{
		Name:        "test.txt",
		Size:        int64(len(content)),
		ContentType: "text/plain",
	}
	
	// Expected hash for "Hello, World!"
	expectedHash := "dffd6021bb2bd5b0af676290809ec3a53191dd81c7f70a4b28688a362182986f"
	
	// Setup mocks
	mockStorage.On("Put", mock.Anything, mock.AnythingOfType("string"), mock.Anything, header.Size).Return(header.Size, nil)
	mockRepo.On("GetStorageByHash", mock.Anything, expectedHash).Return(nil, assert.AnError)
	mockRepo.On("CreateStorage", mock.Anything, mock.MatchedBy(func(s *model.Storage) bool {
		return s.Hash == expectedHash && s.Size == header.Size && s.ContentType == header.ContentType
	})).Return(nil)
	mockRepo.On("CreateFileReference", mock.Anything, mock.MatchedBy(func(f *model.File) bool {
		return f.Name == header.Name && f.StorageHash == expectedHash
	})).Return(nil)
	
	// Execute
	result, err := service.Upload(context.Background(), strings.NewReader(content), header)
	
	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, header.Name, result.Name)
	assert.Equal(t, expectedHash, result.StorageHash)
	assert.Equal(t, header.Size, result.GetSize())
	assert.Equal(t, header.ContentType, result.GetContentType())
	
	mockStorage.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
}

func TestStorageService_Upload_DuplicateFile(t *testing.T) {
	// Setup
	mockStorage := NewMockStorage()
	mockRepo := NewMockFileRepository()
	logger := zap.NewNop()
	
	service := NewStorageService(mockStorage, mockRepo, logger)
	
	// Test data
	content := "Hello, World!"
	header := &model.FileHeader{
		Name:        "test-duplicate.txt",
		Size:        int64(len(content)),
		ContentType: "text/plain",
	}
	
	// Expected hash for "Hello, World!"
	expectedHash := "dffd6021bb2bd5b0af676290809ec3a53191dd81c7f70a4b28688a362182986f"
	
	// Existing storage record
	existingStorage := &model.Storage{
		Hash:           expectedHash,
		Size:           int64(len(content)),
		ContentType:    "text/plain",
		StoragePath:    "existing-path",
		ReferenceCount: 1,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
	
	// Setup mocks - first call uploads, second call finds existing storage
	mockStorage.On("Put", mock.Anything, mock.AnythingOfType("string"), mock.Anything, header.Size).Return(header.Size, nil)
	mockStorage.On("Delete", mock.Anything, mock.AnythingOfType("string")).Return(nil) // Delete the duplicate upload
	mockRepo.On("GetStorageByHash", mock.Anything, expectedHash).Return(existingStorage, nil)
	mockRepo.On("IncrementStorageRef", mock.Anything, expectedHash).Return(nil)
	mockRepo.On("CreateFileReference", mock.Anything, mock.MatchedBy(func(f *model.File) bool {
		return f.Name == header.Name && f.StorageHash == expectedHash
	})).Return(nil)
	
	// Execute
	result, err := service.Upload(context.Background(), strings.NewReader(content), header)
	
	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, header.Name, result.Name)
	assert.Equal(t, expectedHash, result.StorageHash)
	assert.Equal(t, existingStorage.Size, result.GetSize())
	assert.Equal(t, existingStorage.ContentType, result.GetContentType())
	
	mockStorage.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
}

func TestStorageService_Delete_LastReference(t *testing.T) {
	// Setup
	mockStorage := NewMockStorage()
	mockRepo := NewMockFileRepository()
	logger := zap.NewNop()
	
	service := NewStorageService(mockStorage, mockRepo, logger)
	
	fileID := "test-file-id"
	storageHash := "test-hash"
	storagePath := "test-path"
	
	storageRecord := &model.Storage{
		Hash:           storageHash,
		Size:           100,
		ContentType:    "text/plain",
		StoragePath:    storagePath,
		ReferenceCount: 1,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
	
	// Setup mocks
	mockRepo.On("GetStorageHashByFileID", mock.Anything, fileID).Return(storageHash, nil)
	mockRepo.On("DeleteByID", mock.Anything, fileID).Return(nil)
	mockRepo.On("DecrementStorageRef", mock.Anything, storageHash).Return(0, nil) // Last reference
	mockRepo.On("GetStorageByHash", mock.Anything, storageHash).Return(storageRecord, nil)
	mockStorage.On("Delete", mock.Anything, storagePath).Return(nil)
	mockRepo.On("DeleteStorage", mock.Anything, storageHash).Return(nil)
	
	// Execute
	err := service.Delete(context.Background(), fileID)
	
	// Assert
	assert.NoError(t, err)
	mockStorage.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
}

func TestStorageService_Delete_NotLastReference(t *testing.T) {
	// Setup
	mockStorage := NewMockStorage()
	mockRepo := NewMockFileRepository()
	logger := zap.NewNop()
	
	service := NewStorageService(mockStorage, mockRepo, logger)
	
	fileID := "test-file-id"
	storageHash := "test-hash"
	
	// Setup mocks
	mockRepo.On("GetStorageHashByFileID", mock.Anything, fileID).Return(storageHash, nil)
	mockRepo.On("DeleteByID", mock.Anything, fileID).Return(nil)
	mockRepo.On("DecrementStorageRef", mock.Anything, storageHash).Return(1, nil) // Still has references
	
	// Storage should NOT be deleted when there are still references
	// mockStorage.AssertNotCalled(t, "Delete")
	// mockRepo.AssertNotCalled(t, "DeleteStorage")
	
	// Execute
	err := service.Delete(context.Background(), fileID)
	
	// Assert
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}