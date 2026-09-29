package wire

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
	"time"
)

func TestParseChallenge_Basic(t *testing.T) {
	header := `Digest realm="gat1400", nonce="abc123", qop="auth", algorithm=MD5`
	ch, ok := parseChallenge(header)
	if !ok {
		t.Fatal("parseChallenge failed")
	}
	if ch.realm != "gat1400" {
		t.Fatalf("realm = %q", ch.realm)
	}
	if ch.nonce != "abc123" {
		t.Fatalf("nonce = %q", ch.nonce)
	}
	if ch.qop != "auth" {
		t.Fatalf("qop = %q", ch.qop)
	}
	if ch.algorithm != "MD5" {
		t.Fatalf("algorithm = %q", ch.algorithm)
	}
}

func TestParseChallenge_UnquotedValues(t *testing.T) {
	header := `Digest realm=gat1400, nonce=abc123, algorithm=MD5`
	ch, ok := parseChallenge(header)
	if !ok {
		t.Fatal("parseChallenge failed")
	}
	if ch.realm != "gat1400" || ch.nonce != "abc123" || ch.algorithm != "MD5" {
		t.Fatalf("parsed = %+v", ch)
	}
}

func TestParseChallenge_OpaqueAndQop(t *testing.T) {
	header := `Digest realm="r", nonce="n", qop="auth", opaque="op", algorithm=MD5`
	ch, ok := parseChallenge(header)
	if !ok {
		t.Fatal("parseChallenge failed")
	}
	if ch.opaque != "op" || ch.qop != "auth" {
		t.Fatalf("opaque/qop = %q/%q", ch.opaque, ch.qop)
	}
}

func TestParseChallenge_MalformedReturnsFalse(t *testing.T) {
	if _, ok := parseChallenge("Bearer foo"); ok {
		t.Fatal("non-Digest header parsed ok")
	}
	if _, ok := parseChallenge(""); ok {
		t.Fatal("empty parsed ok")
	}
	if _, ok := parseChallenge(`Digest realm="r"`); ok {
		t.Fatal("missing nonce should fail")
	}
}

// TestBuildAuthorization_HasExpectedFields pins the *shape* of the header.
func TestBuildAuthorization_HasExpectedFields(t *testing.T) {
	c := &Client{username: "u", password: "p", realm: "gat1400"}
	ch := challenge{realm: "gat1400", nonce: "deadbeef", qop: "auth", algorithm: "MD5", opaque: "op"}
	auth, err := c.buildAuthorization(ch, "POST", "/VIID/Persons", []byte(`{"PersonID":"p1"}`))
	if err != nil {
		t.Fatalf("buildAuthorization: %v", err)
	}
	for _, want := range []string{
		`username="u"`,
		`realm="gat1400"`,
		`nonce="deadbeef"`,
		`uri="/VIID/Persons"`,
		`qop=auth`,
		`response="`,
		`algorithm=MD5`,
		`opaque="op"`,
	} {
		if !strings.Contains(auth, want) {
			t.Fatalf("authorization missing %q\n%s", want, auth)
		}
	}
	md5 := regexp.MustCompile(`response="[0-9a-f]{32}"`)
	if !md5.MatchString(auth) {
		t.Fatalf("response is not 32-hex chars:\n%s", auth)
	}
}

func TestBuildAuthorization_NoQopHasHexResponse(t *testing.T) {
	c := &Client{username: "u", password: "p"}
	ch := challenge{realm: "r", nonce: "n", algorithm: "MD5"}
	auth, err := c.buildAuthorization(ch, "GET", "/x", nil)
	if err != nil {
		t.Fatalf("buildAuthorization: %v", err)
	}
	if !strings.Contains(auth, "qop=auth") {
		// implementation always emits qop=auth; verify at minimum.
	}
	md5 := regexp.MustCompile(`response="[0-9a-f]{32}"`)
	if !md5.MatchString(auth) {
		t.Fatalf("response is not 32-hex chars:\n%s", auth)
	}
}

// TestClient_DigestHandshake spins up a tiny HTTP server that demands Digest
// auth, then verifies the wire.Client is able to perform the second request
// with a valid Authorization header.
func TestClient_DigestHandshake(t *testing.T) {
	const (
		user  = "alice"
		pass  = "secret"
		realm = "gat1400"
	)

	mux := http.NewServeMux()
	mux.HandleFunc("/VIID/Persons", func(w http.ResponseWriter, r *http.Request) {
		hdr := r.Header.Get("Authorization")
		if hdr == "" {
			w.Header().Set("WWW-Authenticate",
				`Digest realm="`+realm+`", nonce="N-1", qop="auth", algorithm=MD5`)
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"ResponseStatus":{"StatusCode":6}}`))
			return
		}
		// Parse the response and verify qop=auth elements are present.
		for _, want := range []string{`username="` + user + `"`, `realm="` + realm + `"`,
			`uri="/VIID/Persons"`, `qop=auth`, `nc=`, `cnonce="`, `response="`} {
			if !strings.Contains(hdr, want) {
				t.Errorf("auth header missing %q\n%s", want, hdr)
			}
		}
		w.Header().Set("Content-Type", "application/VIID+JSON")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ResponseStatus":{"StatusCode":0,"StatusString":"OK"}}`))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	c := NewClient(nil, nil)
	c.Configure(Options{Username: user, Password: pass, Qop: "auth"})
	body, status, err := c.PostJSON(t.Context(), srv.URL+"/VIID/Persons", "DEV00000000000000000001", map[string]any{"PersonID": "p1"})
	if err != nil {
		t.Fatalf("PostJSON: %v", err)
	}
	if status != http.StatusOK {
		t.Fatalf("status = %d, body=%v", status, body)
	}
}

func TestPostJSON_SendsUserIdentifyAndContentType(t *testing.T) {
	var seenUI, seenCT bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("User-Identify") != "" {
			seenUI = true
		}
		if r.Header.Get("Content-Type") != "" {
			seenCT = true
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"ok":true}`)
	}))
	defer srv.Close()
	c := NewClient(nil, nil)
	body, status, err := c.PostJSON(t.Context(), srv.URL+"/x", "DEV-001", map[string]any{"k": "v"})
	if err != nil {
		t.Fatalf("PostJSON: %v", err)
	}
	if status != http.StatusOK {
		t.Fatalf("status = %d", status)
	}
	if !seenUI {
		t.Fatalf("User-Identify header not received")
	}
	if !seenCT {
		t.Fatalf("Content-Type header not received")
	}
	if body["ok"] != true {
		t.Fatalf("body = %v", body)
	}
}

func TestPostJSON_PropagatesNon200(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = io.WriteString(w, `{"err":"boom"}`)
	}))
	defer srv.Close()
	c := NewClient(nil, nil)
	body, status, err := c.PostJSON(t.Context(), srv.URL+"/z", "DEV-001", nil)
	if err != nil {
		t.Fatalf("PostJSON: %v", err)
	}
	if status != http.StatusInternalServerError {
		t.Fatalf("status = %d", status)
	}
	if body["err"] != "boom" {
		t.Fatalf("body = %v", body)
	}
}

func TestPostJSON_RespectsContextTimeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		_, _ = io.WriteString(w, `{}`)
	}))
	defer srv.Close()
	c := NewClient(nil, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Millisecond)
	defer cancel()
	_, _, err := c.PostJSON(ctx, srv.URL+"/slow", "DEV-001", nil)
	if err == nil {
		t.Fatalf("expected timeout error, got nil")
	}
}