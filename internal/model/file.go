package model

import "time"

// Storage represents a unique file content with reference counting
type Storage struct {
	Hash           string    `json:"hash"`
	Size           int64     `json:"size"`
	ContentType    string    `json:"content_type"`
	StoragePath    string    `json:"-"`
	ReferenceCount int       `json:"reference_count"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// File represents a file record that references stored content
type File struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	StorageHash string    `json:"storage_hash"`
	CreatedAt   time.Time `json:"created_at"`
	
	// Embedded storage information for API responses
	Storage *Storage `json:"storage,omitempty"`
	
	// Legacy fields for backwards compatibility - can be removed in future
	Size        int64  `json:"size,omitempty"`
	ContentType string `json:"content_type,omitempty"`
	StoragePath string `json:"-"`
	Checksum    string `json:"checksum,omitempty"`
}

// FileHeader represents file upload metadata
type FileHeader struct {
	Name        string
	Size        int64
	ContentType string
}

// GetSize returns the file size from storage or legacy field
func (f *File) GetSize() int64 {
	if f.Storage != nil {
		return f.Storage.Size
	}
	return f.Size
}

// GetContentType returns the content type from storage or legacy field
func (f *File) GetContentType() string {
	if f.Storage != nil {
		return f.Storage.ContentType
	}
	return f.ContentType
}

// GetStoragePath returns the storage path from storage or legacy field
func (f *File) GetStoragePath() string {
	if f.Storage != nil {
		return f.Storage.StoragePath
	}
	return f.StoragePath
}

// GetChecksum returns the checksum from storage hash or legacy field
func (f *File) GetChecksum() string {
	if f.StorageHash != "" {
		return f.StorageHash
	}
	return f.Checksum
}