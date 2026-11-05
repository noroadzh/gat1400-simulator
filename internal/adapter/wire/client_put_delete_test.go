package wire

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPutJSON_401RetryAndContentType(t *testing.T) {
	const (
		user  = "admin"
		pass  = "admin123"
		realm = "gat1400"
	)
	var attempts int
	var gotCT, gotMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		gotMethod = r.Method
		gotCT = r.Header.Get("Content-Type")
		if r.Header.Get("Authorization") == "" {
			w.Header().Set("WWW-Authenticate",
				`Digest realm="`+realm+`", nonce="N-put", qop="auth", algorithm=MD5`)
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/VIID+JSON")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ResponseStatus":{"StatusCode":0}}`))
	}))
	defer srv.Close()

	c := NewClient(nil, nil)
	c.Configure(Options{Username: user, Password: pass, Qop: "auth"})
	_, status, err := c.PutJSON(t.Context(), srv.URL+"/VIID/Faces/f1", "DEV1", map[string]any{"FaceID": "f1"})
	if err != nil {
		t.Fatalf("PutJSON: %v", err)
	}
	if status != http.StatusOK {
		t.Fatalf("status = %d", status)
	}
	if attempts != 2 {
		t.Fatalf("attempts = %d want 2", attempts)
	}
	if gotMethod != http.MethodPut {
		t.Fatalf("method = %q", gotMethod)
	}
	if gotCT != "application/VIID+JSON" {
		t.Fatalf("Content-Type = %q", gotCT)
	}
}

func TestDeleteJSON_Success(t *testing.T) {
	var gotMethod, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		_, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ResponseStatus":{"StatusCode":0}}`))
	}))
	defer srv.Close()

	c := NewClient(nil, nil)
	_, status, err := c.DeleteJSON(t.Context(), srv.URL+"/VIID/Faces/f1", "DEV1")
	if err != nil {
		t.Fatalf("DeleteJSON: %v", err)
	}
	if status != http.StatusOK {
		t.Fatalf("status = %d", status)
	}
	if gotMethod != http.MethodDelete {
		t.Fatalf("method = %q", gotMethod)
	}
	if !strings.HasSuffix(gotPath, "/VIID/Faces/f1") {
		t.Fatalf("path = %q", gotPath)
	}
}
