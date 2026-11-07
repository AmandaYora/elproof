package storage

import (
	"context"
	"net/url"
	"strings"
	"testing"
	"time"
)

func newTestClient(t *testing.T, presign bool) *Client {
	t.Helper()
	// Region set: signing needs no network round trip to discover it.
	c, err := New(Config{Endpoint: "s3.example.test", Bucket: "bkt", AccessKey: "AK", SecretKey: "SK", UseSSL: true, Region: "us-east-1", Presign: presign})
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestPresignGetBuildsASignedExpiringLink(t *testing.T) {
	c := newTestClient(t, true)
	link, err := c.PresignGet(context.Background(), "elproof/upload/3/9/evidence/nota.pdf", `Nota "April".pdf`, false, 2*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	u, err := url.Parse(link)
	if err != nil {
		t.Fatal(err)
	}
	q := u.Query()
	if u.Scheme != "https" || u.Host != "s3.example.test" || !strings.HasSuffix(u.Path, "/bkt/elproof/upload/3/9/evidence/nota.pdf") {
		t.Errorf("unexpected target: %s", link)
	}
	if q.Get("X-Amz-Signature") == "" || q.Get("X-Amz-Expires") != "120" {
		t.Errorf("link must be signed and last 120s: %v", q)
	}
	if got := q.Get("response-content-disposition"); !strings.HasPrefix(got, "inline;") || !strings.Contains(got, "Nota") {
		t.Errorf("inline disposition missing: %q", got)
	}
	if strings.Contains(link, "SK") && strings.Contains(link, "SecretKey") {
		t.Error("the secret key must never appear in a link")
	}

	dl, err := c.PresignGet(context.Background(), "k", "a.pdf", true, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(dl, "attachment") {
		t.Errorf("download link must force attachment: %s", dl)
	}
}

func TestPresignGetRefusedWhenDisabled(t *testing.T) {
	c := newTestClient(t, false)
	if c.CanPresign() {
		t.Fatal("presign was switched off")
	}
	if _, err := c.PresignGet(context.Background(), "k", "a.pdf", false, time.Minute); err == nil {
		t.Fatal("a disabled client must not hand out links")
	}
}
