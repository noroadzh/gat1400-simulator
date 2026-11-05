package wire

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/noroadzh/gat1400-simulator/internal/domain/resource"
)

func TestCollection_Push_List_Put_Delete(t *testing.T) {
	var gotQuery string
	var methods []string
	var paths []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		methods = append(methods, r.Method)
		paths = append(paths, r.URL.Path)
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/VIID+JSON")
		switch r.Method {
		case http.MethodPost:
			_, _ = w.Write([]byte(`{"ResponseStatus":{"StatusCode":0}}`))
		case http.MethodGet:
			if strings.Contains(r.URL.Path, "/f1") {
				_, _ = w.Write([]byte(`{"Face":{"FaceID":"f1"}}`))
			} else {
				_, _ = w.Write([]byte(`{"TotalNum":1,"FaceList":{"FaceObject":[{"FaceID":"f1"}]}}`))
			}
		case http.MethodPut, http.MethodDelete:
			_, _ = w.Write([]byte(`{"ResponseStatus":{"StatusCode":0}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	col := NewClient(nil, nil).Collection()
	ctx := t.Context()
	dev := "DEV1"

	if err := col.Push(ctx, srv.URL, dev, resource.KindFace, map[string]any{
		"FaceList": map[string]any{"FaceObject": []map[string]any{{"FaceID": "f1"}}},
	}); err != nil {
		t.Fatalf("Push: %v", err)
	}

	res, err := col.List(ctx, srv.URL, dev, resource.KindFace, 10, 1)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if res.TotalNum != 1 || len(res.Items) != 1 {
		t.Fatalf("List result = %#v", res)
	}
	if !strings.Contains(gotQuery, "PageRecordNum=10") || !strings.Contains(gotQuery, "RecordStartNo=1") {
		t.Fatalf("query = %q", gotQuery)
	}

	obj, err := col.GetByID(ctx, srv.URL, dev, resource.KindFace, "f1")
	if err != nil || obj["FaceID"] != "f1" {
		t.Fatalf("GetByID: %v %#v", err, obj)
	}
	if err := col.PutByID(ctx, srv.URL, dev, resource.KindFace, "f1", map[string]any{"FaceID": "f1", "Name": "x"}); err != nil {
		t.Fatalf("PutByID: %v", err)
	}
	if err := col.DeleteByID(ctx, srv.URL, dev, resource.KindFace, "f1"); err != nil {
		t.Fatalf("DeleteByID: %v", err)
	}

	wantMethods := []string{"POST", "GET", "GET", "PUT", "DELETE"}
	for i, m := range wantMethods {
		if methods[i] != m {
			t.Fatalf("method[%d]=%s want %s", i, methods[i], m)
		}
	}
	if paths[0] != "/VIID/Faces" {
		t.Fatalf("push path = %s", paths[0])
	}
}
