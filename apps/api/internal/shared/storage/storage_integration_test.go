package storage

import (
	"context"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// Runs only against a real S3-compatible server (a local MinIO is enough):
//
//	TEST_S3_ENDPOINT=127.0.0.1:9099 TEST_S3_ACCESS_KEY=... TEST_S3_SECRET_KEY=... go test ./internal/shared/storage
func TestPresignAgainstRealStorage(t *testing.T) {
	endpoint, ak, sk := os.Getenv("TEST_S3_ENDPOINT"), os.Getenv("TEST_S3_ACCESS_KEY"), os.Getenv("TEST_S3_SECRET_KEY")
	if endpoint == "" {
		t.Skip("TEST_S3_ENDPOINT not set")
	}
	ctx := context.Background()
	bucket := "presign-test"
	raw, err := minio.New(endpoint, &minio.Options{Creds: credentials.NewStaticV4(ak, sk, "")})
	if err != nil {
		t.Fatal(err)
	}
	if ok, _ := raw.BucketExists(ctx, bucket); !ok {
		if err := raw.MakeBucket(ctx, bucket, minio.MakeBucketOptions{}); err != nil {
			t.Fatal(err)
		}
	}

	c, err := New(Config{Endpoint: endpoint, Bucket: bucket, AccessKey: ak, SecretKey: sk, Region: "us-east-1", Presign: true})
	if err != nil {
		t.Fatal(err)
	}
	key := "elproof/upload/1/2/evidence/nota.pdf"
	body := "%PDF-1.4 fake"
	if _, err := c.Save(ctx, key, []byte(body), "application/pdf"); err != nil {
		t.Fatal(err)
	}

	get := func(link string) (*http.Response, string) {
		resp, err := http.Get(link) // no Authorization header: the link alone must do it
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		b, _ := io.ReadAll(resp.Body)
		return resp, string(b)
	}

	link, err := c.PresignGet(ctx, key, "Nota April.pdf", false, 3*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	resp, got := get(link)
	if resp.StatusCode != http.StatusOK || got != body {
		t.Fatalf("link must serve the object: %d %q", resp.StatusCode, got)
	}
	if cd := resp.Header.Get("Content-Disposition"); !strings.HasPrefix(cd, "inline;") || !strings.Contains(cd, "Nota April.pdf") {
		t.Errorf("inline disposition not applied: %q", cd)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "application/pdf" {
		t.Errorf("content type %q", ct)
	}

	dl, _ := c.PresignGet(ctx, key, "Nota April.pdf", true, 3*time.Second)
	if resp, _ := get(dl); !strings.HasPrefix(resp.Header.Get("Content-Disposition"), "attachment;") {
		t.Errorf("download link must force attachment: %q", resp.Header.Get("Content-Disposition"))
	}

	// A link for one object cannot be pointed at another.
	other := strings.Replace(link, "nota.pdf", "lain.pdf", 1)
	if resp, _ := get(other); resp.StatusCode == http.StatusOK {
		t.Error("a link must not work for a different key")
	}

	time.Sleep(4 * time.Second)
	if resp, _ := get(link); resp.StatusCode != http.StatusForbidden {
		t.Errorf("an expired link must be refused, got %d", resp.StatusCode)
	}
	_ = c.Delete(ctx, key)
}
