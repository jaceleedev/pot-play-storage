package storage

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSeaweedFSSimpleStorage(t *testing.T) {
	// Mock SeaweedFS filer server
	filerServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "POST" && strings.HasPrefix(r.URL.Path, "/test-file"):
			w.WriteHeader(http.StatusCreated)
		case r.Method == "GET" && r.URL.Path == "/test-file":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("test content"))
		case r.Method == "DELETE" && r.URL.Path == "/test-file":
			w.WriteHeader(http.StatusOK)
		case r.Method == "GET" && r.URL.Path == "/?pretty=y":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"Path": "/",
				"Entries": [
					{
						"FullPath": "/test-file",
						"Mode": 33188,
						"Size": 12,
						"name": "test-file"
					}
				]
			}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer filerServer.Close()

	// Mock SeaweedFS master server
	masterServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/cluster/status" {
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer masterServer.Close()

	// Create storage instance with mock URLs
	storage := &SeaweedFSSimpleStorage{
		masterURL: masterServer.URL,
		filerURL:  filerServer.URL,
		client:    http.DefaultClient,
	}

	ctx := context.Background()

	// Test Put
	content := strings.NewReader("test content")
	size, err := storage.Put(ctx, "test-file", content, 12)
	if err != nil {
		t.Fatalf("Put failed: %v", err)
	}
	if size != 12 {
		t.Errorf("Expected size 12, got %d", size)
	}

	// Test Get
	reader, err := storage.Get(ctx, "test-file")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	defer reader.Close()

	// Test List
	files, err := storage.List(ctx, "")
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(files) != 1 || files[0] != "test-file" {
		t.Errorf("Expected [test-file], got %v", files)
	}

	// Test Delete
	err = storage.Delete(ctx, "test-file")
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}
}

func TestSeaweedFSSimpleStorage_HealthCheck(t *testing.T) {
	// Test successful health check
	masterServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/cluster/status" {
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer masterServer.Close()

	storage, err := NewSeaweedFSSimpleStorage(masterServer.URL)
	if err != nil {
		t.Fatalf("Expected successful connection, got error: %v", err)
	}
	if storage == nil {
		t.Fatal("Expected storage instance, got nil")
	}

	// Test failed health check
	failServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer failServer.Close()

	_, err = NewSeaweedFSSimpleStorage(failServer.URL)
	if err == nil {
		t.Fatal("Expected error for failed health check, got nil")
	}
}