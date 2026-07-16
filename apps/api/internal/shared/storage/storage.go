// Package storage wraps the S3-compatible object storage client (IDCloudHost
// Object Storage — see ADR-0006). Callers deal in keys and byte slices only;
// nothing outside this package knows it's backed by S3.
package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type Client struct {
	client *minio.Client
	bucket string
}

type Config struct {
	Endpoint  string
	Bucket    string
	AccessKey string
	SecretKey string
	UseSSL    bool
}

func New(cfg Config) (*Client, error) {
	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("create s3 client: %w", err)
	}
	return &Client{client: client, bucket: cfg.Bucket}, nil
}

// Save uploads data under key, returning the same key on success (mirrors the
// shape callers need — the key itself is the "path", computed by the caller
// per ADR-0006's convention, not by this package).
func (c *Client) Save(ctx context.Context, key string, data []byte, contentType string) (string, error) {
	_, err := c.client.PutObject(ctx, c.bucket, key, bytes.NewReader(data), int64(len(data)),
		minio.PutObjectOptions{ContentType: contentType},
	)
	if err != nil {
		return "", fmt.Errorf("upload object %q: %w", key, err)
	}
	return key, nil
}

// Open streams the object back — caller is responsible for closing it.
func (c *Client) Open(ctx context.Context, key string) (io.ReadCloser, error) {
	obj, err := c.client.GetObject(ctx, c.bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("open object %q: %w", key, err)
	}
	return obj, nil
}

// Delete removes an object — used by tests/cleanup and by evidence deletion
// flows (Fase 4).
func (c *Client) Delete(ctx context.Context, key string) error {
	if err := c.client.RemoveObject(ctx, c.bucket, key, minio.RemoveObjectOptions{}); err != nil {
		return fmt.Errorf("delete object %q: %w", key, err)
	}
	return nil
}

// BuildKey constructs the object key per ADR-0006's convention:
// elproof/upload/{tenantId}/{projectId}/{category}/{filename}
func BuildKey(tenantID, projectID, category, filename string) string {
	return fmt.Sprintf("elproof/upload/%s/%s/%s/%s", tenantID, projectID, category, filename)
}
