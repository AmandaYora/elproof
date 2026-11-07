package presentation

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func slideRequest(t *testing.T, path, mime string, raw []byte) *httptest.ResponseRecorder {
	t.Helper()
	body, _ := json.Marshal(map[string]string{"mimeType": mime, "base64Data": base64.StdEncoding.EncodeToString(raw)})
	req := httptest.NewRequest(http.MethodPut, path, bytes.NewReader(body))
	rec := httptest.NewRecorder()
	// A nil storage client is fine: every case below must be refused before
	// anything is saved; reaching storage would panic.
	NewLoginSlidesHandler(nil).Replace(rec, req)
	return rec
}

func TestReplaceSlideRefusesBadInput(t *testing.T) {
	notAnImage := []byte("<html>not a photo</html>")
	cases := []struct {
		name string
		path string
		mime string
		raw  []byte
		want int
	}{
		{"unknown slot", "/api/v1/console/login-slides/9", "image/jpeg", []byte{1}, http.StatusNotFound},
		{"negative slot", "/api/v1/console/login-slides/-1", "image/jpeg", []byte{1}, http.StatusNotFound},
		{"slot is not a number", "/api/v1/console/login-slides/abc", "image/jpeg", []byte{1}, http.StatusNotFound},
		{"pdf is not a photo", "/api/v1/console/login-slides/0", "application/pdf", []byte{1}, http.StatusUnprocessableEntity},
		{"empty file", "/api/v1/console/login-slides/0", "image/jpeg", nil, http.StatusUnprocessableEntity},
		{"declared jpeg, really html", "/api/v1/console/login-slides/0", "image/jpeg", notAnImage, http.StatusUnprocessableEntity},
		{"over the size limit", "/api/v1/console/login-slides/0", "image/png", bytes.Repeat([]byte{0}, maxSlideUploadBytes+1), http.StatusUnprocessableEntity},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			rec := slideRequest(t, c.path, c.mime, c.raw)
			if rec.Code != c.want {
				t.Fatalf("status %d, want %d: %s", rec.Code, c.want, strings.TrimSpace(rec.Body.String()))
			}
		})
	}
}
