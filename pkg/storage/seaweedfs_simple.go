package storage

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
	"time"
)

// SeaweedFSSimpleStorage implements Storage interface using SeaweedFS HTTP API
// This is a simplified version that stores files directly without complex directory management
type SeaweedFSSimpleStorage struct {
	masterURL string
	filerURL  string
	client    *http.Client
}

// Simple response structures
type SimpleAssignResponse struct {
	FileID    string `json:"fid"`
	URL       string `json:"url"`
	PublicURL string `json:"publicUrl"`
	Error     string `json:"error,omitempty"`
}

// NewSeaweedFSSimpleStorage creates a simplified SeaweedFS storage client
func NewSeaweedFSSimpleStorage(masterURL string) (*SeaweedFSSimpleStorage, error) {
	// Ensure masterURL has proper format
	if !strings.HasPrefix(masterURL, "http://") && !strings.HasPrefix(masterURL, "https://") {
		masterURL = "http://" + masterURL
	}
	
	// Derive filer URL from master URL (assume standard ports)
	// Replace both port and hostname for Docker compose setup
	filerURL := strings.Replace(masterURL, ":9333", ":8888", 1)
	filerURL = strings.Replace(filerURL, "seaweedfs-master", "seaweedfs-filer", 1)
	
	client := &http.Client{
		Timeout: 30 * time.Second,
	}
	
	storage := &SeaweedFSSimpleStorage{
		masterURL: strings.TrimSuffix(masterURL, "/"),
		filerURL:  strings.TrimSuffix(filerURL, "/"),
		client:    client,
	}
	
	// Test connection
	if err := storage.healthCheck(); err != nil {
		return nil, fmt.Errorf("seaweedfs connection failed: %w", err)
	}
	
	return storage, nil
}

// healthCheck verifies SeaweedFS master is accessible
func (s *SeaweedFSSimpleStorage) healthCheck() error {
	resp, err := s.client.Get(s.masterURL + "/cluster/status")
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("master server returned status %d", resp.StatusCode)
	}
	
	return nil
}

// Put uploads a file to SeaweedFS using the filer
func (s *SeaweedFSSimpleStorage) Put(ctx context.Context, path string, reader io.Reader, size int64) (int64, error) {
	// Read all data into buffer to get actual size
	buf := new(bytes.Buffer)
	writtenBytes, err := io.Copy(buf, reader)
	if err != nil {
		return 0, err
	}
	
	// Create multipart form
	var formBuf bytes.Buffer
	writer := multipart.NewWriter(&formBuf)
	
	part, err := writer.CreateFormFile("file", path)
	if err != nil {
		return 0, err
	}
	
	_, err = io.Copy(part, buf)
	if err != nil {
		return 0, err
	}
	
	if err := writer.Close(); err != nil {
		return 0, err
	}
	
	// Upload directly to filer
	uploadURL := s.filerURL + "/" + path
	req, err := http.NewRequestWithContext(ctx, "POST", uploadURL, &formBuf)
	if err != nil {
		return 0, err
	}
	
	req.Header.Set("Content-Type", writer.FormDataContentType())
	
	resp, err := s.client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return 0, fmt.Errorf("upload failed with status %d: %s", resp.StatusCode, string(body))
	}
	
	return writtenBytes, nil
}

// Get downloads a file from SeaweedFS filer
func (s *SeaweedFSSimpleStorage) Get(ctx context.Context, path string) (io.ReadCloser, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", s.filerURL+"/"+path, nil)
	if err != nil {
		return nil, err
	}
	
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	
	if resp.StatusCode == http.StatusNotFound {
		resp.Body.Close()
		return nil, fmt.Errorf("file not found: %s", path)
	}
	
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("failed to get file: status %d", resp.StatusCode)
	}
	
	return resp.Body, nil
}

// Delete removes a file from SeaweedFS filer
func (s *SeaweedFSSimpleStorage) Delete(ctx context.Context, path string) error {
	req, err := http.NewRequestWithContext(ctx, "DELETE", s.filerURL+"/"+path, nil)
	if err != nil {
		return err
	}
	
	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode == http.StatusNotFound {
		// File already doesn't exist, consider it successful
		return nil
	}
	
	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to delete file: status %d", resp.StatusCode)
	}
	
	return nil
}

// List returns a list of files with the given prefix
func (s *SeaweedFSSimpleStorage) List(ctx context.Context, prefix string) ([]string, error) {
	// For simplicity, we'll list the root directory and filter by prefix
	// In production, you'd want more sophisticated directory traversal
	
	listURL := s.filerURL + "/?pretty=y"
	if prefix != "" {
		// Try to list the directory containing the prefix
		if strings.Contains(prefix, "/") {
			dir := prefix[:strings.LastIndex(prefix, "/")]
			listURL = s.filerURL + "/" + dir + "/?pretty=y"
		}
	}
	
	req, err := http.NewRequestWithContext(ctx, "GET", listURL, nil)
	if err != nil {
		return nil, err
	}
	
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode == http.StatusNotFound {
		return []string{}, nil
	}
	
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to list files: status %d", resp.StatusCode)
	}
	
	var listResp DirectoryListResponse
	if err := json.NewDecoder(resp.Body).Decode(&listResp); err != nil {
		return nil, err
	}
	
	var files []string
	for _, entry := range listResp.Entries {
		// Skip directories, only return files that match prefix
		if entry.Mode&0x4000 == 0 { // Not a directory
			fullPath := strings.TrimPrefix(entry.FullPath, "/")
			if prefix == "" || strings.HasPrefix(fullPath, prefix) {
				files = append(files, fullPath)
			}
		}
	}
	
	return files, nil
}