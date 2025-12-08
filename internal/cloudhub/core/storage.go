package core

import (
	"context"
	"time"
)

// Storage defines the interface for object storage operations.
type Storage interface {
	// GeneratePresignedURL generates a temporary URL for downloading a file (firmware).
	GeneratePresignedURL(ctx context.Context, key string, expiry time.Duration) (string, error)

	// CheckBucket for initial
	CheckBucket(ctx context.Context) error
}
