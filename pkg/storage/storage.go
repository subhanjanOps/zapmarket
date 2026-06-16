// Package storage wraps an S3-compatible object store (MinIO today; any real
// AWS S3-compatible service later, since minio-go speaks the same API) for
// services that need to store user-uploaded files such as product images.
package storage

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/zapmarket/zapmarket/pkg/config"
)

// Client wraps a MinIO/S3 client bound to a single bucket.
type Client struct {
	mc        *minio.Client
	bucket    string
	publicURL string
}

// New creates a Client from cfg, verifying the configured bucket exists.
// It does not create the bucket — that's done once at infra-setup time (see
// the `minio-init` job in docker-compose.yml), not on every service boot.
func New(ctx context.Context, cfg *config.Config) (*Client, error) {
	mc, err := minio.New(cfg.MinIOEndpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.MinIOAccessKey, cfg.MinIOSecretKey, ""),
		Secure: cfg.MinIOUseSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("create minio client: %w", err)
	}

	exists, err := mc.BucketExists(ctx, cfg.MinIOBucket)
	if err != nil {
		return nil, fmt.Errorf("check bucket %q: %w", cfg.MinIOBucket, err)
	}
	if !exists {
		return nil, fmt.Errorf("bucket %q does not exist (expected minio-init to have created it)", cfg.MinIOBucket)
	}

	return &Client{
		mc:        mc,
		bucket:    cfg.MinIOBucket,
		publicURL: strings.TrimRight(cfg.MinIOPublicURL, "/"),
	}, nil
}

// Upload writes r (size bytes, of contentType) to the bucket under key,
// returning key unchanged for convenience at call sites.
func (c *Client) Upload(ctx context.Context, key string, r io.Reader, size int64, contentType string) (string, error) {
	_, err := c.mc.PutObject(ctx, c.bucket, key, r, size, minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return "", fmt.Errorf("upload object %q: %w", key, err)
	}
	return key, nil
}

// Delete removes the object at key. Deleting a key that doesn't exist is not
// an error (MinIO's RemoveObject is idempotent in that sense).
func (c *Client) Delete(ctx context.Context, key string) error {
	if err := c.mc.RemoveObject(ctx, c.bucket, key, minio.RemoveObjectOptions{}); err != nil {
		return fmt.Errorf("delete object %q: %w", key, err)
	}
	return nil
}

// PublicURL builds the externally-reachable URL for key. Requires the
// bucket to have an anonymous-download policy set (see `minio-init` in
// docker-compose.yml) — this is a plain URL, not a presigned/signed one.
func (c *Client) PublicURL(key string) string {
	return fmt.Sprintf("%s/%s/%s", c.publicURL, c.bucket, key)
}
