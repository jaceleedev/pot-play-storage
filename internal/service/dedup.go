package service

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
)

// calculateFileHash computes SHA256 hash of file content
func calculateFileHash(reader io.Reader) (string, error) {
	hasher := sha256.New()
	
	// Read and hash the content
	_, err := io.Copy(hasher, reader)
	if err != nil {
		return "", err
	}
	
	hash := hex.EncodeToString(hasher.Sum(nil))
	return hash, nil
}