//go:build integration

package isolation

import (
	"bytes"
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"elproof/internal/shared/config"
)

// stack is one throwaway ElProof: its own database, the real server binary,
// a fake Google JWKS and (when configured) a real S3-compatible bucket.
type stack struct {
	t       *testing.T
	base    string
	db      *sql.DB // the throwaway database, for the few facts the API cannot create
	google  *fakeGoogle
	client  *http.Client
	ipCount int
	mu      sync.Mutex // guards ipCount (some scenarios send requests concurrently)
}

type fakeGoogle struct {
	key *rsa.PrivateKey
	srv *httptest.Server
}

const jwksKID = "isolation-1"

func b64(b []byte) string { return base64.RawURLEncoding.EncodeToString(b) }

func newFakeGoogle(t *testing.T) *fakeGoogle {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/jwks", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"keys": []map[string]string{{
			"kid": jwksKID, "kty": "RSA", "alg": "RS256", "use": "sig",
			"n": b64(key.N.Bytes()), "e": b64(big.NewInt(int64(key.E)).Bytes()),
		}}})
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return &fakeGoogle{key: key, srv: srv}
}

// token mints a Google ID token the server will accept for this identity.
func (g *fakeGoogle) token(email, name, sub string) string {
	hdr, _ := json.Marshal(map[string]string{"alg": "RS256", "typ": "JWT", "kid": jwksKID})
	now := time.Now()
	pl, _ := json.Marshal(map[string]any{
		"iss": "https://accounts.google.com", "aud": config.DefaultGoogleClientID, "sub": sub,
		"email": email, "email_verified": true, "name": name,
		"iat": now.Unix(), "exp": now.Add(time.Hour).Unix(),
	})
	in := b64(hdr) + "." + b64(pl)
	h := sha256.Sum256([]byte(in))
	sig, err := rsa.SignPKCS1v15(rand.Reader, g.key, crypto.SHA256, h[:])
	if err != nil {
		panic(err)
	}
	return in + "." + b64(sig)
}

func freePort(t *testing.T) string {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	return fmt.Sprint(l.Addr().(*net.TCPAddr).Port)
}

// startStack builds and starts the server against a fresh database. It skips
// the test when TEST_MYSQL_ADMIN_DSN is not set.
func startStack(t *testing.T) *stack { return startStackWith(t) }

// startStackWith is startStack with extra environment for the server (for
// example "S3_PRESIGN=false" to make uploads go through the API).
func startStackWith(t *testing.T, extraEnv ...string) *stack {
	t.Helper()
	adminDSN := os.Getenv("TEST_MYSQL_ADMIN_DSN")
	if adminDSN == "" {
		t.Skip("TEST_MYSQL_ADMIN_DSN not set (e.g. root:@tcp(127.0.0.1:3399)/): skipping the tenant isolation suite")
	}
	ctx := context.Background()

	admin, err := sql.Open("mysql", adminDSN)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { admin.Close() })
	dbName := fmt.Sprintf("elproof_iso_%d", time.Now().UnixNano())
	if _, err := admin.ExecContext(ctx, "CREATE DATABASE "+dbName+" CHARACTER SET utf8mb4"); err != nil {
		t.Fatalf("create database: %v", err)
	}
	t.Cleanup(func() { _, _ = admin.Exec("DROP DATABASE IF EXISTS " + dbName) })
	dsnBase := strings.TrimSuffix(strings.TrimPrefix(adminDSN, "mysql://"), "/")
	db, err := sql.Open("mysql", dsnBase+"/"+dbName+"?parseTime=true")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })

	tmp := t.TempDir()
	bin := filepath.Join(tmp, "api-server")
	moduleRoot, _ := filepath.Abs("../..")
	build := exec.Command("go", "build", "-o", bin, "./cmd/server")
	build.Dir = moduleRoot
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build server: %v\n%s", err, out)
	}

	g := newFakeGoogle(t)
	port := freePort(t)
	s3Endpoint := envOr("TEST_S3_ENDPOINT", "127.0.0.1:9")
	env := append(os.Environ(),
		"APP_ENV=development", "APP_PORT="+port,
		"DATABASE_URL=mysql://"+dsnBase+"/"+dbName,
		"JWT_SECRET=isolation-jwt-secret-not-for-anything-real-0123456789",
		"JWT_ACCESS_TTL=24h",
		"GOOGLE_JWKS_URL="+g.srv.URL+"/jwks",
		"S3_ENDPOINT="+s3Endpoint, "S3_BUCKET=isolation", "S3_USE_SSL=false", "S3_REGION=us-east-1",
		"S3_ACCESS_KEY="+envOr("TEST_S3_ACCESS_KEY", "x"), "S3_SECRET_KEY="+envOr("TEST_S3_SECRET_KEY", "x"),
		"PAYMENT_ENCRYPTION_KEY=isolation-payment-encryption-key-long-random-0123456789",
	)
	env = append(env, extraEnv...)
	if os.Getenv("TEST_S3_ENDPOINT") != "" {
		ensureBucket(t, "isolation")
	}

	migrate := exec.Command(bin, "migrate", "up")
	migrate.Dir, migrate.Env = tmp, env
	if out, err := migrate.CombinedOutput(); err != nil {
		t.Fatalf("migrate up: %v\n%s", err, out)
	}

	logFile, err := os.Create(filepath.Join(tmp, "server.log"))
	if err != nil {
		t.Fatal(err)
	}
	srv := exec.Command(bin)
	srv.Dir, srv.Env, srv.Stdout, srv.Stderr = tmp, env, logFile, logFile
	if err := srv.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = srv.Process.Kill()
		_, _ = srv.Process.Wait()
		if t.Failed() {
			if b, err := os.ReadFile(logFile.Name()); err == nil {
				t.Logf("server log (tail):\n%s", tail(string(b), 3000))
			}
		}
	})

	s := &stack{t: t, base: "http://127.0.0.1:" + port, db: db, google: g, client: &http.Client{Timeout: 20 * time.Second}}
	deadline := time.Now().Add(30 * time.Second)
	for {
		resp, err := s.client.Get(s.base + "/api/v1/health")
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				break
			}
		}
		if time.Now().After(deadline) {
			t.Fatal("server did not become healthy in time")
		}
		time.Sleep(200 * time.Millisecond)
	}
	return s
}

func ensureBucket(t *testing.T, bucket string) {
	raw, err := minio.New(os.Getenv("TEST_S3_ENDPOINT"), &minio.Options{Creds: credentials.NewStaticV4(os.Getenv("TEST_S3_ACCESS_KEY"), os.Getenv("TEST_S3_SECRET_KEY"), "")})
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if ok, _ := raw.BucketExists(ctx, bucket); !ok {
		if err := raw.MakeBucket(ctx, bucket, minio.MakeBucketOptions{}); err != nil {
			t.Fatalf("make bucket: %v", err)
		}
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func tail(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[len(s)-n:]
}

// ---- request helper ----

type reply struct {
	status int
	body   []byte
	header http.Header
}

func (r reply) String() string { return fmt.Sprintf("%d %s", r.status, tail(string(r.body), 300)) }

// data decodes the response envelope's `data` field.
func (r reply) json() map[string]any {
	var m map[string]any
	_ = json.Unmarshal(r.body, &m)
	return m
}

// dig walks nested maps ("data", "client", "id") and returns the value.
func (r reply) dig(keys ...string) any {
	var cur any = r.json()
	for _, k := range keys {
		m, ok := cur.(map[string]any)
		if !ok {
			return nil
		}
		cur = m[k]
	}
	return cur
}

func (r reply) id(keys ...string) int64 {
	if f, ok := r.dig(keys...).(float64); ok {
		return int64(f)
	}
	return 0
}

// do sends one request. An empty token sends no Authorization header.
func (s *stack) do(token, method, path string, body any) reply {
	s.t.Helper()
	return s.doWith(token, method, path, body, nil)
}

// doWith is do plus extra request headers (for example Idempotency-Key).
func (s *stack) doWith(token, method, path string, body any, headers map[string]string) reply {
	s.t.Helper()
	var rd io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			s.t.Fatal(err)
		}
		rd = bytes.NewReader(b)
	}
	return s.send(token, method, path, rd, body != nil, "application/json", headers)
}

// putRaw sends raw bytes (an upload) with the given Content-Type.
func (s *stack) putRaw(token, path, contentType string, data []byte) reply {
	s.t.Helper()
	return s.send(token, "PUT", path, bytes.NewReader(data), true, contentType, nil)
}

func (s *stack) send(token, method, path string, rd io.Reader, hasBody bool, contentType string, headers map[string]string) reply {
	s.t.Helper()
	req, err := http.NewRequest(method, s.base+path, rd)
	if err != nil {
		s.t.Fatal(err)
	}
	if hasBody {
		req.Header.Set("Content-Type", contentType)
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	// Each call comes "from" a different address so the per-IP limiters on
	// sign-in and sign-up do not turn the suite into a rate-limit test.
	s.mu.Lock()
	s.ipCount++
	n := s.ipCount
	s.mu.Unlock()
	req.Header.Set("X-Forwarded-For", fmt.Sprintf("10.20.%d.%d", n/250, n%250+1))
	resp, err := s.client.Do(req)
	if err != nil {
		s.t.Fatalf("%s %s: %v", method, path, err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return reply{status: resp.StatusCode, body: b, header: resp.Header}
}

// mustOK fails the test unless the reply is a 2xx; it is for fixture setup.
func (s *stack) mustOK(what string, r reply) reply {
	s.t.Helper()
	if r.status < 200 || r.status > 299 {
		s.t.Fatalf("setup %s failed: %s", what, r)
	}
	return r
}
