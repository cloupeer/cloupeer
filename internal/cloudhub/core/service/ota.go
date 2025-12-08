package service

import (
	"context"
	"fmt"
	"time"
)

// GetFirmwareDownloadURL generates a secure, temporary URL for the vehicle to download firmware.
// This decouples the vehicle from the underlying storage details (S3/MinIO).
func (s *Service) GetFirmwareDownloadURL(ctx context.Context, firmwarePath string) (string, error) {
	// Hardcoded bucket name for now, in production this could come from config or the Firmware CRD.
	const urlExpiry = 1 * time.Hour

	if firmwarePath == "" {
		return "", fmt.Errorf("firmware path is empty")
	}

	url, err := s.storage.GeneratePresignedURL(ctx, firmwarePath, urlExpiry)
	if err != nil {
		return "", fmt.Errorf("failed to generate firmware URL: %w", err)
	}

	return url, nil
}
