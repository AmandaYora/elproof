package application

import (
	"bytes"
	"context"
	"io"
	"sort"
	"strings"
	"testing"
	"time"

	"elproof/internal/modules/projects/domain"
	"elproof/internal/shared/apperror"
)

// objStorage is an in-memory object store with the optional capabilities.
type objStorage struct {
	objs      map[string][]byte
	types     map[string]string
	presign   bool
	deleted   []string
	presigned string
}

func newObjStorage(presign bool) *objStorage {
	return &objStorage{objs: map[string][]byte{}, types: map[string]string{}, presign: presign}
}

func (o *objStorage) Save(_ context.Context, key string, data []byte, ct string) (string, error) {
	o.objs[key], o.types[key] = data, ct
	return key, nil
}
func (o *objStorage) Open(context.Context, string) (io.ReadCloser, error) { return nil, nil }
func (o *objStorage) Delete(_ context.Context, key string) error {
	delete(o.objs, key)
	o.deleted = append(o.deleted, key)
	return nil
}
func (o *objStorage) CanPresign() bool { return o.presign }
func (o *objStorage) PresignPut(_ context.Context, key, ct string, _ time.Duration) (string, error) {
	o.presigned = key
	return "https://s3.example/" + key + "?sig=1&ct=" + ct, nil
}
func (o *objStorage) Stat(_ context.Context, key string) (int64, string, bool, error) {
	b, ok := o.objs[key]
	return int64(len(b)), o.types[key], ok, nil
}

// putAs simulates the browser uploading to a granted URL (object storage does
// not check the size or type, which is why Resolve does).
func (o *objStorage) putAs(key string, data []byte, ct string) { o.objs[key], o.types[key] = data, ct }

// proofFiles is an in-memory evidence repository that enforces the unique
// storage path like the database does.
type proofFiles struct {
	EvidenceRepository
	rows []domain.Evidence
	next int64
	// failOn makes Create fail for the n-th insert (1-based) with a database error.
	failOn int
	calls  int
}

func (p *proofFiles) ExistsByStoragePath(_ context.Context, path string) (bool, error) {
	for _, e := range p.rows {
		if e.StoragePath == path {
			return true, nil
		}
	}
	return false, nil
}
func (p *proofFiles) Create(_ context.Context, e *domain.Evidence) error {
	p.calls++
	if p.failOn == p.calls {
		return apperror.Internal("disk full")
	}
	for _, r := range p.rows {
		if r.StoragePath == e.StoragePath {
			return domain.ErrStoragePathTaken
		}
	}
	p.next++
	e.ID = 500 + p.next
	p.rows = append(p.rows, *e)
	return nil
}

func buildKeyForTests(tenant, project, category, name string) string {
	return "up/" + tenant + "/" + project + "/" + category + "/" + name
}

func newUploads(store *objStorage, files EvidenceRepository) *UploadService {
	s := NewUploadService(testSigner("secret-one"), store, files, buildKeyForTests)
	s.now = func() time.Time { return tokenNow }
	return s
}

func fieldsOf(t *testing.T, err error) map[string][]string {
	t.Helper()
	ae, ok := apperror.As(err)
	if !ok || ae.Kind != apperror.KindValidation {
		t.Fatalf("want a validation error, got %v", err)
	}
	return ae.Fields
}

func sortedKeys(m map[string][]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func TestGrantValidatesTheFileDescription(t *testing.T) {
	u := newUploads(newObjStorage(true), &proofFiles{})
	_, err := u.Grant(context.Background(), 1, 2, UploadInput{FileName: "../etc/passwd", MimeType: "text/html", Size: 16*1024*1024 + 1})
	f := fieldsOf(t, err)
	if got := sortedKeys(f); strings.Join(got, ",") != "fileName,mimeType,size" {
		t.Errorf("every bad part must be reported at once, got %v", got)
	}
	for name, in := range map[string]UploadInput{
		"slash":          {FileName: "a/b.pdf", MimeType: "application/pdf", Size: 1},
		"backslash":      {FileName: `a\b.pdf`, MimeType: "application/pdf", Size: 1},
		"dots":           {FileName: "..", MimeType: "application/pdf", Size: 1},
		"empty":          {FileName: "  ", MimeType: "application/pdf", Size: 1},
		"control":        {FileName: "a\x00b.pdf", MimeType: "application/pdf", Size: 1},
		"too long":       {FileName: strings.Repeat("a", 151), MimeType: "application/pdf", Size: 1},
		"zero size":      {FileName: "a.pdf", MimeType: "application/pdf", Size: 0},
		"svg":            {FileName: "a.svg", MimeType: "image/svg+xml", Size: 1},
		"no type":        {FileName: "a.pdf", MimeType: "", Size: 1},
		"exactly 15MB+1": {FileName: "a.pdf", MimeType: "application/pdf", Size: 15*1024*1024 + 1},
	} {
		if _, err := u.Grant(context.Background(), 1, 2, in); err == nil {
			t.Errorf("%s must be refused", name)
		}
	}
	if _, err := u.Grant(context.Background(), 1, 2, UploadInput{FileName: "bukti transfer.JPG", MimeType: "image/jpeg; charset=binary", Size: 15 * 1024 * 1024}); err != nil {
		t.Errorf("a normal file (15 MB exactly, type with a parameter) must pass: %v", err)
	}
}

func TestGrantDirectOrThroughTheAPI(t *testing.T) {
	ctx := context.Background()
	in := UploadInput{FileName: "bukti.png", MimeType: "image/png", Size: 1234}

	direct := newObjStorage(true)
	g, err := newUploads(direct, &proofFiles{}).Grant(ctx, 7, 42, in)
	if err != nil || !g.Direct || !strings.HasPrefix(g.URL, "https://s3.example/up/7/42/uploads/") || direct.presigned == "" {
		t.Fatalf("a presigning store gives a direct URL under tenant/project: %+v %v", g, err)
	}
	if !g.ExpiresAt.Equal(tokenNow.Add(15 * time.Minute)) {
		t.Errorf("expiry must be 15 minutes: %v", g.ExpiresAt)
	}

	proxy := newObjStorage(false)
	g, err = newUploads(proxy, &proofFiles{}).Grant(ctx, 7, 42, in)
	if err != nil || g.Direct || g.URL != "/api/v1/projects/42/uploads/"+g.Token {
		t.Fatalf("a store that cannot presign gives the API's own route with the same token: %+v %v", g, err)
	}
	claims, err := testSigner("secret-one").Verify(g.Token, 7, 42)
	if err != nil || claims.MimeType != "image/png" || claims.MaxSize != 1234 || claims.FileName != "bukti.png" {
		t.Errorf("the token must carry what was declared: %+v %v", claims, err)
	}
}

func grantAndPut(t *testing.T, u *UploadService, store *objStorage, tenant, project int64, data []byte, mime string) string {
	t.Helper()
	g, err := u.Grant(context.Background(), tenant, project, UploadInput{FileName: "bukti.jpg", MimeType: mime, Size: int64(len(data))})
	if err != nil {
		t.Fatal(err)
	}
	claims, _ := testSigner("secret-one").Verify(g.Token, tenant, project)
	store.putAs(claims.Key, data, mime)
	return g.Token
}

func TestResolveCollectsEveryProblemAtOnce(t *testing.T) {
	ctx := context.Background()
	store := newObjStorage(true)
	files := &proofFiles{}
	u := newUploads(store, files)

	good := grantAndPut(t, u, store, 1, 10, []byte("jpegbytes"), "image/jpeg")
	foreign := grantAndPut(t, u, store, 1, 11, []byte("jpegbytes"), "image/jpeg") // another project
	otherTenant := grantAndPut(t, u, store, 2, 10, []byte("jpegbytes"), "image/jpeg")
	notUploaded, _ := u.Grant(ctx, 1, 10, UploadInput{FileName: "x.jpg", MimeType: "image/jpeg", Size: 5})
	// Declared 5 bytes, 9 sent.
	g2, _ := u.Grant(ctx, 1, 10, UploadInput{FileName: "big.jpg", MimeType: "image/jpeg", Size: 5})
	c2, _ := testSigner("secret-one").Verify(g2.Token, 1, 10)
	store.putAs(c2.Key, []byte("123456789"), "image/jpeg")
	// Declared jpeg, stored as html.
	g3, _ := u.Grant(ctx, 1, 10, UploadInput{FileName: "w.jpg", MimeType: "image/jpeg", Size: 50})
	c3, _ := testSigner("secret-one").Verify(g3.Token, 1, 10)
	store.putAs(c3.Key, []byte("<html>"), "text/html")

	five := make([]ProofItem, 5)
	for i := range five {
		five[i] = ProofItem{UploadToken: good, Type: domain.EvidenceTransferProof}
	}
	_, err := u.Resolve(ctx, 1, 10, five, clientProofTypes)
	// More than 4 items is its own error; the per-item problems follow.
	if f := fieldsOf(t, err); strings.Join(sortedKeys(f), ",") != "evidence" {
		t.Fatalf("more than 4 items: %v", f)
	}

	_, err = u.Resolve(ctx, 1, 10, []ProofItem{
		{UploadToken: foreign, Type: domain.EvidenceTransferProof},
		{UploadToken: "garbage", Type: domain.EvidenceTransferProof},
		{UploadToken: notUploaded.Token, Type: domain.EvidenceTransferProof},
		{UploadToken: good, Type: domain.EvidenceInvoice},
	}, clientProofTypes)
	f := fieldsOf(t, err)
	want := "evidence[0],evidence[0].uploadToken,evidence[1],evidence[1].uploadToken,evidence[2],evidence[2].uploadToken,evidence[3].type"
	if got := strings.Join(sortedKeys(f), ","); got != want {
		t.Errorf("keys: got %s\nwant %s", got, want)
	}

	_, err = u.Resolve(ctx, 1, 10, []ProofItem{{UploadToken: otherTenant, Type: domain.EvidenceTransferProof}, {UploadToken: g2.Token, Type: domain.EvidenceTransferProof}, {UploadToken: g3.Token, Type: domain.EvidenceTransferProof}, {UploadToken: good, Type: domain.EvidenceTransferProof}}, clientProofTypes)
	f = fieldsOf(t, err)
	if got := strings.Join(sortedKeys(f), ","); got != "evidence[0],evidence[0].uploadToken,evidence[1],evidence[1].uploadToken,evidence[2],evidence[2].uploadToken" {
		t.Errorf("tenant/size/type problems: %s", got)
	}
	// An object that can never be used is removed; a good one is left alone.
	if len(store.deleted) != 2 {
		t.Errorf("the oversize and the wrong-type objects must be deleted, got %v", store.deleted)
	}
	if _, ok := store.objs[c2.Key]; ok {
		t.Error("oversize object must be gone")
	}

	out, err := u.Resolve(ctx, 1, 10, []ProofItem{{UploadToken: good, Type: domain.EvidenceTransferProof, Name: "  Bukti DP "}}, clientProofTypes)
	if err != nil || len(out) != 1 || out[0].Name != "Bukti DP" || out[0].Claims.FileName != "bukti.jpg" {
		t.Errorf("a good item resolves: %+v %v", out, err)
	}

	// The same file twice in one request.
	_, err = u.Resolve(ctx, 1, 10, []ProofItem{{UploadToken: good, Type: domain.EvidenceTransferProof}, {UploadToken: good, Type: domain.EvidenceTransferProof}}, clientProofTypes)
	if f := fieldsOf(t, err); len(f["evidence[1]"]) == 0 {
		t.Errorf("the second use of one token in a request is refused: %v", f)
	}

	// Already claimed by a payment.
	c1, _ := testSigner("secret-one").Verify(good, 1, 10)
	files.rows = append(files.rows, domain.Evidence{StoragePath: c1.Key})
	_, err = u.Resolve(ctx, 1, 10, []ProofItem{{UploadToken: good, Type: domain.EvidenceTransferProof}}, clientProofTypes)
	if f := fieldsOf(t, err); len(f["evidence[0].uploadToken"]) == 0 {
		t.Errorf("a spent token is refused: %v", f)
	}
}

func TestResolveExpiredToken(t *testing.T) {
	store := newObjStorage(true)
	u := newUploads(store, &proofFiles{})
	tok := grantAndPut(t, u, store, 1, 10, []byte("x"), "image/jpeg")
	u.signer = u.signer.WithClock(func() time.Time { return tokenNow.Add(16 * time.Minute) })
	_, err := u.Resolve(context.Background(), 1, 10, []ProofItem{{UploadToken: tok, Type: domain.EvidenceTransferProof}}, clientProofTypes)
	f := fieldsOf(t, err)
	if len(f["evidence[0].uploadToken"]) == 0 || !strings.Contains(f["evidence[0]"][0], "kedaluwarsa") {
		t.Errorf("an expired upload says so: %v", f)
	}
}

func TestStoreOnTheAPIRoute(t *testing.T) {
	ctx := context.Background()
	store := newObjStorage(false)
	files := &proofFiles{}
	u := newUploads(store, files)
	g, _ := u.Grant(ctx, 1, 10, UploadInput{FileName: "bukti.pdf", MimeType: "application/pdf", Size: 20})
	claims, _ := testSigner("secret-one").Verify(g.Token, 1, 10)

	if _, err := u.Store(ctx, 1, 11, g.Token, "application/pdf", strings.NewReader("%PDF")); err == nil {
		t.Error("another project's token must be refused")
	} else if f := fieldsOf(t, err); len(f["uploadToken"]) == 0 {
		t.Errorf("on the field that carried it: %v", f)
	}
	if _, err := u.Store(ctx, 1, 10, g.Token, "image/png", strings.NewReader("%PDF")); err == nil {
		t.Error("a different content type must be refused")
	}
	if _, err := u.Store(ctx, 1, 10, g.Token, "application/pdf", bytes.NewReader(nil)); err == nil {
		t.Error("an empty body must be refused")
	}
	if _, err := u.Store(ctx, 1, 10, g.Token, "application/pdf", strings.NewReader(strings.Repeat("a", 21))); err == nil {
		t.Error("more than the declared size must be refused")
	} else if _, ok := err.(*UploadTooLargeError); !ok {
		t.Errorf("want UploadTooLargeError (413), got %T", err)
	}
	if len(store.objs) != 0 {
		t.Fatal("nothing may be stored by a refused PUT")
	}
	n, err := u.Store(ctx, 1, 10, g.Token, "application/pdf; charset=x", strings.NewReader("%PDF-1.4 tiny"))
	if err != nil || n != int64(len("%PDF-1.4 tiny")) || string(store.objs[claims.Key]) != "%PDF-1.4 tiny" {
		t.Fatalf("a good PUT stores the bytes under the token's key: n=%d err=%v", n, err)
	}
	// Retrying the same PUT before a payment used it is fine (a dropped response).
	if _, err := u.Store(ctx, 1, 10, g.Token, "application/pdf", strings.NewReader("%PDF-1.4 tiny")); err != nil {
		t.Errorf("a retry before use must work: %v", err)
	}
	// Once a payment owns the object the token is spent: no overwriting a recorded proof.
	files.rows = append(files.rows, domain.Evidence{StoragePath: claims.Key})
	_, err = u.Store(ctx, 1, 10, g.Token, "application/pdf", strings.NewReader("tampered"))
	if ae, ok := apperror.As(err); !ok || ae.Kind != apperror.KindConflict {
		t.Errorf("a PUT after use must be a conflict, got %v", err)
	}
	if string(store.objs[claims.Key]) != "%PDF-1.4 tiny" {
		t.Error("the recorded proof must not change")
	}
}
