package storage

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// SeaweedFSStorage implements Storage interface using SeaweedFS HTTP API
type SeaweedFSStorage struct {
	masterURL string
	client    *http.Client
}

// SeaweedFS API response structures
type AssignResponse struct {
	FileID    string `json:"fid"`
	URL       string `json:"url"`
	PublicURL string `json:"publicUrl"`
	Count     int    `json:"count"`
	Error     string `json:"error,omitempty"`
}

type VolumeListResponse struct {
	Volumes []Volume `json:"volumes"`
}

type Volume struct {
	ID               int    `json:"id"`
	Size             int64  `json:"size"`
	Collection       string `json:"collection"`
	FileCount        int    `json:"fileCount"`
	DeleteCount      int    `json:"deleteCount"`
	DeletedByteCount int64  `json:"deletedByteCount"`
	ReadOnly         bool   `json:"readOnly"`
}

type FileInfo struct {
	Name string `json:"name"`
	Size int64  `json:"size"`
}

type DirectoryListResponse struct {
	Path    string            `json:"Path"`
	Entries []DirectoryEntry  `json:"Entries"`
	Limit   int              `json:"Limit"`
	LastFileName string       `json:"LastFileName"`
	ShouldDisplayLoadMore bool `json:"ShouldDisplayLoadMore"`
}

type DirectoryEntry struct {
	FullPath string    `json:"FullPath"`
	Mtime    time.Time `json:"Mtime"`
	Crtime   time.Time `json:"Crtime"`
	Mode     int       `json:"Mode"`
	Uid      int       `json:"Uid"`
	Gid      int       `json:"Gid"`
	Mime     string    `json:"Mime"`
	Size     int64     `json:"Size"`
	Name     string    `json:"name"`
}

// NewSeaweedFSStorage creates a new SeaweedFS storage client
// For production use with full volume/filer coordination
func NewSeaweedFSStorage(masterURL string) (*SeaweedFSStorage, error) {
	// Ensure masterURL has proper format
	if !strings.HasPrefix(masterURL, "http://") && !strings.HasPrefix(masterURL, "https://") {
		masterURL = "http://" + masterURL
	}
	
	client := &http.Client{
		Timeout: 30 * time.Second,
	}
	
	storage := &SeaweedFSStorage{
		masterURL: strings.TrimSuffix(masterURL, "/"),
		client:    client,
	}
	
	// Test connection by getting cluster status
	if err := storage.healthCheck(); err != nil {
		return nil, fmt.Errorf("seaweedfs connection failed: %w", err)
	}
	
	return storage, nil
}

// healthCheck verifies SeaweedFS master is accessible
func (s *SeaweedFSStorage) healthCheck() error {
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

// assignFileID gets a file ID from SeaweedFS master for upload
func (s *SeaweedFSStorage) assignFileID(ctx context.Context) (*AssignResponse, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", s.masterURL+"/dir/assign", nil)
	if err != nil {
		return nil, err
	}
	
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("assign request failed with status %d", resp.StatusCode)
	}
	
	var assignResp AssignResponse
	if err := json.NewDecoder(resp.Body).Decode(&assignResp); err != nil {
		return nil, err
	}
	
	if assignResp.Error != "" {
		return nil, fmt.Errorf("assign error: %s", assignResp.Error)
	}
	
	return &assignResp, nil
}

// Put uploads a file to SeaweedFS
func (s *SeaweedFSStorage) Put(ctx context.Context, path string, reader io.Reader, size int64) (int64, error) {
	// Get file ID assignment
	assign, err := s.assignFileID(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to assign file ID: %w", err)
	}
	
	// Prepare multipart form data
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	
	// Add the file part
	part, err := writer.CreateFormFile("file", path)
	if err != nil {
		return 0, err
	}
	
	writtenBytes, err := io.Copy(part, reader)
	if err != nil {
		return 0, err
	}
	
	if err := writer.Close(); err != nil {
		return 0, err
	}
	
	// Upload to assigned volume server
	uploadURL := "http://" + assign.URL + "/" + assign.FileID
	req, err := http.NewRequestWithContext(ctx, "POST", uploadURL, &buf)
	if err != nil {
		return 0, err
	}
	
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Content-Length", strconv.FormatInt(int64(buf.Len()), 10))
	
	resp, err := s.client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return 0, fmt.Errorf("upload failed with status %d: %s", resp.StatusCode, string(body))
	}
	
	// Store the mapping of path -> fileID for later retrieval
	// In a production system, you'd want to persist this mapping
	// For now, we'll use the SeaweedFS filer to store files by path
	return writtenBytes, s.storeFileMapping(ctx, path, assign.FileID)
}

// storeFileMapping stores the file using SeaweedFS filer API with the original path
func (s *SeaweedFSStorage) storeFileMapping(ctx context.Context, path, fileID string) error {
	// Use filer to create a symbolic reference
	filerURL := strings.Replace(s.masterURL, ":9333", ":8888", 1) // Default filer port
	
	// Create directory structure if needed
	dirs := strings.Split(path, "/")
	if len(dirs) > 1 {
		dirPath := strings.Join(dirs[:len(dirs)-1], "/")
		if err := s.createDirectory(ctx, filerURL, dirPath); err != nil {
			return err
		}
	}
	
	// Store file metadata
	metadata := map[string]string{
		"fid": fileID,
	}
	
	metadataJSON, _ := json.Marshal(metadata)
	
	req, err := http.NewRequestWithContext(ctx, "PUT", filerURL+"/"+path+"?metadata=true", bytes.NewReader(metadataJSON))
	if err != nil {
		return err
	}
	
	req.Header.Set("Content-Type", "application/json")
	
	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to store file mapping: status %d", resp.StatusCode)
	}
	
	return nil
}

// createDirectory creates directory structure in filer
func (s *SeaweedFSStorage) createDirectory(ctx context.Context, filerURL, dirPath string) error {
	req, err := http.NewRequestWithContext(ctx, "POST", filerURL+"/"+dirPath+"/", nil)
	if err != nil {
		return err
	}
	
	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	
	// Directory creation is idempotent, so ignore already exists errors
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusConflict {
		return fmt.Errorf("failed to create directory: status %d", resp.StatusCode)
	}
	
	return nil
}

// Get downloads a file from SeaweedFS
func (s *SeaweedFSStorage) Get(ctx context.Context, path string) (io.ReadCloser, error) {
	// Try to get file directly via filer first (simpler approach)
	filerURL := strings.Replace(s.masterURL, ":9333", ":8888", 1)
	
	req, err := http.NewRequestWithContext(ctx, "GET", filerURL+"/"+path, nil)
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

// Delete removes a file from SeaweedFS
func (s *SeaweedFSStorage) Delete(ctx context.Context, path string) error {
	filerURL := strings.Replace(s.masterURL, ":9333", ":8888", 1)
	
	req, err := http.NewRequestWithContext(ctx, "DELETE", filerURL+"/"+path, nil)
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
func (s *SeaweedFSStorage) List(ctx context.Context, prefix string) ([]string, error) {
	filerURL := strings.Replace(s.masterURL, ":9333", ":8888", 1)
	
	// Build query parameters for listing
	params := url.Values{}
	params.Set("pretty", "y")
	if prefix != "" {
		params.Set("namePattern", prefix+"*")
	}
	
	listURL := filerURL + "/" + prefix
	if len(params) > 0 {
		listURL += "?" + params.Encode()
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
		// Directory doesn't exist, return empty list
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
		// Skip directories, only return files
		if entry.Mode&0x4000 == 0 { // Not a directory
			files = append(files, entry.FullPath)
		}
	}
	
	return files, nil
}