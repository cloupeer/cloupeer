package storage

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"cloupeer.io/cloupeer/pkg/log"
	"cloupeer.io/cloupeer/pkg/options"
)

type minioProvider struct {
	client     *minio.Client
	bucketName string
}

// NewMinIOProvider 创建基于 S3 协议的存储提供者
func NewMinIOProvider(opts *options.S3Options) (Provider, error) {
	// 初始化 MinIO Client
	// 注意：由于开发环境使用自签名证书，我们需要配置自定义的 Transport 来跳过验证
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	minioOpts := &minio.Options{
		Creds:     credentials.NewStaticV4(opts.AccessKeyID, opts.SecretAccessKey, ""),
		Secure:    opts.UseSSL,
		Transport: transport, // 注入自定义 Transport
	}

	client, err := minio.New(opts.Endpoint, minioOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to create minio client: %w", err)
	}

	return &minioProvider{
		client:     client,
		bucketName: opts.BucketName,
	}, nil
}

func (p *minioProvider) CheckBucket(ctx context.Context) error {
	exists, err := p.client.BucketExists(ctx, p.bucketName)
	if err != nil {
		return fmt.Errorf("failed to check bucket existence: %w", err)
	}
	if !exists {
		// 自动创建桶（仅开发环境便利性，生产环境通常手动管理）
		log.Info("Bucket does not exist, creating...", "bucket", p.bucketName)
		if err := p.client.MakeBucket(ctx, p.bucketName, minio.MakeBucketOptions{}); err != nil {
			return fmt.Errorf("failed to create bucket: %w", err)
		}
	}
	return nil
}

func (p *minioProvider) GeneratePresignedURL(ctx context.Context, objectKey string, expiry time.Duration) (string, error) {
	// 生成预签名 URL
	// Set request parameters for content-disposition.
	reqParams := make(url.Values)
	// reqParams.Set("response-content-disposition", "attachment; filename=\"firmware.bin\"")

	presignedURL, err := p.client.PresignedGetObject(ctx, p.bucketName, objectKey, expiry, reqParams)
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned url: %w", err)
	}

	return presignedURL.String(), nil
}
