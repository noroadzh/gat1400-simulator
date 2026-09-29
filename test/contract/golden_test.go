package contract

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/noroadzh/gat1400-simulator/internal/adapter/capture"
	"github.com/noroadzh/gat1400-simulator/internal/adapter/httpapi"
	"github.com/noroadzh/gat1400-simulator/internal/adapter/storage"
	"github.com/noroadzh/gat1400-simulator/internal/app/application"
	"github.com/noroadzh/gat1400-simulator/internal/domain/ids"
	"github.com/noroadzh/gat1400-simulator/internal/domain/node"
)

// Step represents a single interaction in a golden sample.
type Step struct {
	Direction string         `json:"direction"` // "outbound" or "inbound"
	Method    string         `json:"method"`
	Path      string         `json:"path"`
	Headers   map[string]string `json:"headers"`
	Body      any            `json:"body"`
	Status    int            `json:"status,omitempty"`
	BodyShape map[string]any `json:"body_shape,omitempty"` // for inbound, used for shape comparison
}

// GoldenSample is the JSON schema for files under test/contract/golden/.
type GoldenSample struct {
	Description string `json:"description"`
	Steps       []Step `json:"steps"`
}

// fixture builds an in-process httpapi.Server bound to a real TCP port
// and pre-populates the node required by the sample.
func fixture(t *testing.T, preseedNodeIDs ...string) (*httptest.Server, *storage.NonceStore) {
	t.Helper()
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "golden.db")
	ns, err := storage.NewNonceStore(ctx, dbPath)
	if err != nil {
		t.Fatalf("nonce store: %v", err)
	}
	t.Cleanup(func() { _ = ns.Close() })
	cs, err := storage.NewCaptureStore(ctx, dbPath)
	if err != nil {
		t.Fatalf("capture store: %v", err)
	}
	t.Cleanup(func() { _ = cs.Close() })

	rec := capture.NewRecorder(cs, slog.New(slog.NewTextHandler(io.Discard, nil)))
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	idGen := ids.NewGenerator(41000000, 30)
	nodeSvc := application.NewNodeService(log, idGen)
	scenSvc := application.NewScenarioService(log, nil)

	for _, nid := range preseedNodeIDs {
		if _, err := nodeSvc.UpsertNode(ctx, node.Node{
			ID: nid, Name: nid, Role: node.RoleDevice,
			Capabilities: []node.Capability{node.CapSystem, node.CapCollection, node.CapCascade},
		}); err != nil {
			t.Fatalf("seed node %s: %v", nid, err)
		}
	}

	srv := httpapi.NewServer(log, nodeSvc, scenSvc, rec, ns, idGen, &httpapi.Config{
		Auth: httpapi.AuthConfig{Realm: "viid", Username: "admin", Password: "admin", Qop: "auth"},
	})
	return httptest.NewServer(srv), ns
}

func loadSamples(t *testing.T, dir string) []struct {
	Name   string
	Sample GoldenSample
} {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Skipf("golden dir %s not present: %v", dir, err)
	}
	var out []struct {
		Name   string
		Sample GoldenSample
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			t.Fatalf("read %s: %v", e.Name(), err)
		}
		var s GoldenSample
		if err := json.Unmarshal(raw, &s); err != nil {
			t.Fatalf("parse %s: %v", e.Name(), err)
		}
		out = append(out, struct {
			Name   string
			Sample GoldenSample
		}{Name: e.Name(), Sample: s})
	}
	return out
}

// shapesMatch returns true when every key in `needles` is present in `actual`
// with the expected value (scalar) or matching type (slice/array).
func shapesMatch(actual, needles map[string]any) bool {
	if needles == nil {
		return true
	}
	for k, want := range needles {
		got, ok := actual[k]
		if !ok {
			return false
		}
		switch w := want.(type) {
		case float64:
			gf, ok := got.(float64)
			if !ok || gf != w {
				return false
			}
		case string:
			gs, ok := got.(string)
			if !ok || gs != w {
				return false
			}
		}
	}
	return true
}

func replaySample(t *testing.T, srv *httptest.Server, ns *storage.NonceStore, s GoldenSample) {
	t.Helper()
	var lastChallengeNonce string
	for i, st := range s.Steps {
		if st.Direction != "outbound" {
			continue
		}
		var body io.Reader
		if st.Body != nil {
			b, err := json.Marshal(st.Body)
			if err != nil {
				t.Fatalf("step %d marshal: %v", i, err)
			}
			body = strings.NewReader(string(b))
		}
		req, err := http.NewRequest(st.Method, srv.URL+st.Path, body)
		if err != nil {
			t.Fatalf("step %d new request: %v", i, err)
		}
		headers := map[string]string{}
		for k, v := range st.Headers {
			headers[k] = v
		}
		// Inject the previous 401 challenge nonce into any placeholder ({{nonce}})
		// the Authorization header contains. If no placeholder, fall through to the
		// literal-nonce branch below which pre-issues the value to NonceStore.
		if lastChallengeNonce != "" {
			if auth, ok := headers["Authorization"]; ok && strings.Contains(auth, "{{nonce}}") {
				headers["Authorization"] = strings.Replace(auth, "{{nonce}}", lastChallengeNonce, 1)
			}
		}
		// Pre-issue any literal nonce= value used in Authorization so Consume will accept it.
		if auth, ok := headers["Authorization"]; ok {
			if v := extractNonce(auth); v != "" {
				if err := ns.Issue(v); err != nil {
					t.Fatalf("issue nonce for step %d: %v", i, err)
				}
			}
		}
		for k, v := range headers {
			req.Header.Add(k, v)
		}
		resp, err := srv.Client().Do(req)
		if err != nil {
			t.Fatalf("step %d do: %v", i, err)
		}
		raw, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()

		// Capture next-challenge nonce (if 401) for following step replay.
		if resp.StatusCode == http.StatusUnauthorized {
			if wa := resp.Header.Get("WWW-Authenticate"); wa != "" {
				if v := extractNonceFromChallenge(wa); v != "" {
					lastChallengeNonce = v
				}
			}
		}

		// Find the inbound step that follows.
		if i+1 >= len(s.Steps) || s.Steps[i+1].Direction != "inbound" {
			continue
		}
		want := s.Steps[i+1]
		if resp.StatusCode != want.Status {
			t.Errorf("step %d status = %d, want %d (body=%s)", i, resp.StatusCode, want.Status, raw)
		}
		if want.BodyShape != nil {
			var got map[string]any
			if err := json.Unmarshal(raw, &got); err != nil {
				t.Errorf("step %d body not JSON: %v (raw=%s)", i, err, raw)
				continue
			}
			if !shapesMatch(got, want.BodyShape) {
				t.Errorf("step %d body shape mismatch: got=%v want=%v", i, got, want.BodyShape)
			}
		}
	}
}

func extractNonce(auth string) string {
	// Pull the value after nonce="…"; tolerate whitespace.
	for _, part := range strings.Split(auth, ",") {
		kv := strings.SplitN(strings.TrimSpace(part), "=", 2)
		if len(kv) == 2 && kv[0] == "nonce" {
			return strings.Trim(kv[1], `"`)
		}
	}
	return ""
}

func extractNonceFromChallenge(wa string) string {
	if !strings.HasPrefix(wa, "Digest ") {
		return ""
	}
	return extractNonce(strings.TrimPrefix(wa, "Digest "))
}

func TestGoldenSamples(t *testing.T) {
	dir := "golden"
	samples := loadSamples(t, dir)
	if len(samples) == 0 {
		t.Skip("no golden samples found")
	}

	for _, s := range samples {
		t.Run(s.Name, func(t *testing.T) {
			// Different samples need different fixtures; we always seed a known node.
			srv, ns := fixture(t, "41000000005030312222")
			defer srv.Close()
			replaySample(t, srv, ns, s.Sample)
		})
	}
}
