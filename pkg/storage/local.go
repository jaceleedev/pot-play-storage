package storage

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type LocalStorage struct {
	basePath string
}

// sanitizePath prevents path traversal attacks by cleaning the path and ensuring it's within the base directory
func (s *LocalStorage) sanitizePath(userPath string) (string, error) {
	// Clean the path to resolve any . or .. components
	cleanPath := filepath.Clean(userPath)
	
	// Remove any leading slashes to prevent absolute path access
	cleanPath = strings.TrimPrefix(cleanPath, "/")
	
	// Check for path traversal attempts
	if strings.Contains(cleanPath, "..") {
		return "", fmt.Errorf("path traversal detected: %s", userPath)
	}
	
	// Ensure the path doesn't start with sensitive directories
	if strings.HasPrefix(cleanPath, "etc/") || strings.HasPrefix(cleanPath, "var/") || 
	   strings.HasPrefix(cleanPath, "usr/") || strings.HasPrefix(cleanPath, "home/") ||
	   strings.HasPrefix(cleanPath, "root/") || strings.HasPrefix(cleanPath, "sys/") ||
	   strings.HasPrefix(cleanPath, "proc/") {
		return "", fmt.Errorf("access to system directories not allowed: %s", cleanPath)
	}
	
	// Create the full path and ensure it's within the base directory
	fullPath := filepath.Join(s.basePath, cleanPath)
	absFullPath, err := filepath.Abs(fullPath)
	if err != nil {
		return "", fmt.Errorf("failed to resolve absolute path: %w", err)
	}
	
	absBasePath, err := filepath.Abs(s.basePath)
	if err != nil {
		return "", fmt.Errorf("failed to resolve base path: %w", err)
	}
	
	// Ensure the resolved path is still within the base directory
	if !strings.HasPrefix(absFullPath, absBasePath+string(filepath.Separator)) && absFullPath != absBasePath {
		return "", fmt.Errorf("path outside base directory: %s", userPath)
	}
	
	return cleanPath, nil
}

func NewLocalStorage(basePath string) (*LocalStorage, error) {
	if err := os.MkdirAll(basePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create base path: %w", err)
	}
	return &LocalStorage{basePath: basePath}, nil
}

func (s *LocalStorage) Put(ctx context.Context, path string, reader io.Reader, size int64) (int64, error) {
	cleanPath, err := s.sanitizePath(path)
	if err != nil {
		return 0, fmt.Errorf("invalid path: %w", err)
	}
	
	fullPath := filepath.Join(s.basePath, cleanPath)
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return 0, fmt.Errorf("failed to create directory: %w", err)
	}
	
	file, err := os.Create(fullPath)
	if err != nil {
		return 0, fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()
	
	return io.Copy(file, reader)
}

func (s *LocalStorage) Get(ctx context.Context, path string) (io.ReadCloser, error) {
	cleanPath, err := s.sanitizePath(path)
	if err != nil {
		return nil, fmt.Errorf("invalid path: %w", err)
	}
	
	fullPath := filepath.Join(s.basePath, cleanPath)
	file, err := os.Open(fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	
	return file, nil
}

func (s *LocalStorage) Delete(ctx context.Context, path string) error {
	cleanPath, err := s.sanitizePath(path)
	if err != nil {
		return fmt.Errorf("invalid path: %w", err)
	}
	
	fullPath := filepath.Join(s.basePath, cleanPath)
	err = os.Remove(fullPath)
	if err != nil {
		return fmt.Errorf("failed to delete file: %w", err)
	}
	
	return nil
}

func (s *LocalStorage) List(ctx context.Context, prefix string) ([]string, error) {
	cleanPrefix, err := s.sanitizePath(prefix)
	if err != nil {
		return nil, fmt.Errorf("invalid prefix: %w", err)
	}
	
	var files []string
	searchPath := filepath.Join(s.basePath, cleanPrefix)
	
	err = filepath.Walk(searchPath, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			// Skip files that can't be accessed instead of failing
			return nil
		}
		if !info.IsDir() {
			rel, err := filepath.Rel(s.basePath, p)
			if err != nil {
				return nil // Skip files with path issues
			}
			files = append(files, rel)
		}
		return nil
	})
	
	if err != nil {
		return nil, fmt.Errorf("failed to list files: %w", err)
	}
	
	return files, nil
}