package storage

import (
	"context"
	"time"
)

// Provider 定义了固件仓库的通用接口
type Provider interface {
	// GeneratePresignedURL 生成一个带签名的临时下载链接
	GeneratePresignedURL(ctx context.Context, objectKey string, expiry time.Duration) (string, error)

	// CheckBucket 确保存储桶存在（用于初始化检查）
	CheckBucket(ctx context.Context) error
}
