package wire

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCatalog_APEs_and_VIIDServers_Paths(t *testing.T) {
	var paths []string
	var methods []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		methods = append(methods, r.Method)
		w.Header().Set("Content-Type", "application/VIID+JSON")
		switch {
		case r.URL.Path == "/VIID/APEs" && r.Method == http.MethodGet:
			_, _ = w.Write([]byte(`{"APEList":{"APEObject":[{"APEID":"a1"}]}}`))
		case r.URL.Path == "/VIID/APEs" && r.Method == http.MethodPost:
			_, _ = w.Write([]byte(`{"ResponseStatus":{"StatusCode":0}}`))
		case r.URL.Path == "/VIID/APEs/a1" && r.Method == http.MethodGet:
			_, _ = w.Write([]byte(`{"APE":{"APEID":"a1"}}`))
		case r.URL.Path == "/VIID/APEs/a1" && r.Method == http.MethodDelete:
			_, _ = w.Write([]byte(`{"ResponseStatus":{"StatusCode":0}}`))
		case r.URL.Path == "/VIID/VIIDServers" && r.Method == http.MethodGet:
			_, _ = w.Write([]byte(`{"VIIDServerList":{"VIIDServerObject":[{"VIIDServerID":"s1"}]}}`))
		case r.URL.Path == "/VIID/VIIDServers" && r.Method == http.MethodPost:
			_, _ = w.Write([]byte(`{"ResponseStatus":{"StatusCode":0}}`))
		case r.URL.Path == "/VIID/Tollgates" && r.Method == http.MethodGet:
			_, _ = w.Write([]byte(`{"TollgateList":{"TollgateObject":[{"TollgateID":"t1"}]}}`))
		default:
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"ResponseStatus":{"StatusCode":0}}`))
		}
	}))
	defer srv.Close()

	cat := NewClient(nil, nil).Catalog()
	ctx := t.Context()
	dev := "DEV1"

	apes, err := cat.ListAPEs(ctx, srv.URL, dev)
	if err != nil || len(apes) != 1 {
		t.Fatalf("ListAPEs: %v %#v", err, apes)
	}
	if err := cat.UpsertAPEs(ctx, srv.URL, dev, map[string]any{
		"APEList": map[string]any{"APEObject": []map[string]any{{"APEID": "a1"}}},
	}); err != nil {
		t.Fatalf("UpsertAPEs: %v", err)
	}
	ape, err := cat.GetAPE(ctx, srv.URL, dev, "a1")
	if err != nil || ape["APEID"] != "a1" {
		t.Fatalf("GetAPE: %v %#v", err, ape)
	}
	if err := cat.DeleteAPE(ctx, srv.URL, dev, "a1"); err != nil {
		t.Fatalf("DeleteAPE: %v", err)
	}
	servers, err := cat.ListVIIDServers(ctx, srv.URL, dev)
	if err != nil || len(servers) != 1 {
		t.Fatalf("ListVIIDServers: %v %#v", err, servers)
	}
	if err := cat.UpsertVIIDServers(ctx, srv.URL, dev, map[string]any{
		"VIIDServerList": map[string]any{"VIIDServerObject": []map[string]any{{"VIIDServerID": "s1"}}},
	}); err != nil {
		t.Fatalf("UpsertVIIDServers: %v", err)
	}
	tg, err := cat.ListTollgates(ctx, srv.URL, dev)
	if err != nil || len(tg) != 1 {
		t.Fatalf("ListTollgates: %v %#v", err, tg)
	}

	want := []string{"/VIID/APEs", "/VIID/APEs", "/VIID/APEs/a1", "/VIID/APEs/a1", "/VIID/VIIDServers", "/VIID/VIIDServers", "/VIID/Tollgates"}
	if len(paths) < len(want) {
		t.Fatalf("paths = %#v", paths)
	}
	for i, p := range want {
		if paths[i] != p {
			t.Fatalf("path[%d]=%q want %q", i, paths[i], p)
		}
	}
}
