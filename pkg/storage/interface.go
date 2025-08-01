package storage

import (
	"context"
	"io"
)

type Storage interface {
	Put(ctx context.Context, path string, reader io.Reader, size int64) (int64, error)
	Get(ctx context.Context, path string) (io.ReadCloser, error)
	Delete(ctx context.Context, path string) error
	List(ctx context.Context, prefix string) ([]string, error)
}