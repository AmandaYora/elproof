// Package storage wraps the S3-compatible object storage client (IDCloudHost
// Object Storage — see ADR-0006). Callers deal in keys and byte slices only;
// nothing outside this package knows it's backed by S3.
package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime"
	"net/url"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type Client struct {
	client  *minio.Client
	bucket  string
	presign bool
}

type Config struct {
	Endpoint  string
	Bucket    string
	AccessKey string
	SecretKey string
	UseSSL    bool
	// Region is the bucket's region; empty lets the client discover it. Set
	// it to presign without a network round trip.
	Region string
	// Presign allows handing the browser a short-lived direct link to an
	// object instead of streaming it through the API. Off when the storage
	// endpoint is not reachable from a browser (a local dev bucket).
	Presign bool
}

func New(cfg Config) (*Client, error) {
	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
		Region: cfg.Region,
	})
	if err != nil {
		return nil, fmt.Errorf("create s3 client: %w", err)
	}
	return &Client{client: client, bucket: cfg.Bucket, presign: cfg.Presign}, nil
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

// CanPresign reports whether direct, time-limited links may be handed out.
func (c *Client) CanPresign() bool { return c.presign }

// PresignGet returns a signed GET link for key that works for ttl (keep it to
// minutes). fileName is what the browser calls the file when saving it;
// attachment forces a download, otherwise the browser may show it inline.
// Whoever holds the link can read the object until it expires, so it is only
// ever created after the caller was authorised for that exact object.
func (c *Client) PresignGet(ctx context.Context, key, fileName string, attachment bool, ttl time.Duration) (string, error) {
	if !c.presign {
		return "", fmt.Errorf("presigned links are disabled")
	}
	kind := "inline"
	if attachment {
		kind = "attachment"
	}
	params := url.Values{}
	if disposition := mime.FormatMediaType(kind, map[string]string{"filename": fileName}); disposition != "" {
		params.Set("response-content-disposition", disposition)
	}
	u, err := c.client.PresignedGetObject(ctx, c.bucket, key, ttl, params)
	if err != nil {
		return "", fmt.Errorf("presign object %q: %w", key, err)
	}
	return u.String(), nil
}

// PresignPut returns a signed PUT link for key that works for ttl. The browser
// uploads the file straight to object storage with it, so the bytes never pass
// through this server. The link names exactly one object, so it can be handed
// out only after the caller was authorised to add a file to that place. The
// content type is not part of the signature: callers verify it afterwards with
// Stat, and refuse an object that is not what was declared.
func (c *Client) PresignPut(ctx context.Context, key, contentType string, ttl time.Duration) (string, error) {
	if !c.presign {
		return "", fmt.Errorf("presigned links are disabled")
	}
	u, err := c.client.PresignedPutObject(ctx, c.bucket, key, ttl)
	if err != nil {
		return "", fmt.Errorf("presign upload %q: %w", key, err)
	}
	return u.String(), nil
}

// Stat reports the size and content type of a stored object without reading
// it. found is false (and err nil) when there is no such object.
func (c *Client) Stat(ctx context.Context, key string) (size int64, contentType string, found bool, err error) {
	info, err := c.client.StatObject(ctx, c.bucket, key, minio.StatObjectOptions{})
	if err != nil {
		if minio.ToErrorResponse(err).Code == "NoSuchKey" {
			return 0, "", false, nil
		}
		return 0, "", false, fmt.Errorf("stat object %q: %w", key, err)
	}
	return info.Size, info.ContentType, true, nil
}

// BuildKey constructs the object key per ADR-0006's convention:
// elproof/upload/{tenantId}/{projectId}/{category}/{filename}
func BuildKey(tenantID, projectID, category, filename string) string {
	return fmt.Sprintf("elproof/upload/%s/%s/%s/%s", tenantID, projectID, category, filename)
}
