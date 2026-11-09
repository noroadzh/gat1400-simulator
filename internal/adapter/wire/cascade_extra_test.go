package wire

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCascade_DispositionList_Delete_Notification(t *testing.T) {
	const dispID = "91000000001000000001"
	const notifID = "notif-1"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/VIID+JSON")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/VIID/Dispositions":
			_, _ = w.Write([]byte(`{"DispositionList":{"DispositionObject":[{"DispositionID":"` + dispID + `"}]}}`))
		case r.Method == http.MethodDelete && r.URL.Path == "/VIID/Dispositions/"+dispID:
			_, _ = w.Write([]byte(`{"ResponseStatus":{"StatusCode":0}}`))
		case r.Method == http.MethodPost && r.URL.Path == "/VIID/DispositionNotifications":
			body, _ := io.ReadAll(r.Body)
			var m map[string]any
			_ = json.Unmarshal(body, &m)
			if m["DispositionNotificationList"] == nil && m["NotifyInfo"] == nil {
				t.Errorf("unexpected body: %s", body)
			}
			_, _ = w.Write([]byte(`{"ResponseStatus":{"StatusCode":0}}`))
		case r.Method == http.MethodGet && r.URL.Path == "/VIID/DispositionNotifications":
			_, _ = w.Write([]byte(`{"DispositionNotificationList":{"DispositionNotificationObject":[{"DispositionNotificationID":"` + notifID + `","DispositionID":"` + dispID + `"}]}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	c := NewClient(nil, nil)
	cas := c.Cascade()
	ctx := t.Context()
	dev := "DEV1"

	list, err := cas.DispositionList(ctx, srv.URL, dev)
	if err != nil {
		t.Fatalf("DispositionList: %v", err)
	}
	if len(list) != 1 || list[0]["DispositionID"] != dispID {
		t.Fatalf("list = %#v", list)
	}

	if err := cas.DispositionNotificationPush(ctx, srv.URL, dev, map[string]any{
		"DispositionNotificationList": []map[string]any{
			{"DispositionNotificationID": notifID, "DispositionID": dispID},
		},
	}); err != nil {
		t.Fatalf("DispositionNotificationPush: %v", err)
	}

	notifs, err := cas.DispositionNotificationList(ctx, srv.URL, dev, dispID)
	if err != nil {
		t.Fatalf("DispositionNotificationList: %v", err)
	}
	if len(notifs) != 1 {
		t.Fatalf("notifs len = %d", len(notifs))
	}

	if err := cas.DispositionDelete(ctx, srv.URL, dev, dispID); err != nil {
		t.Fatalf("DispositionDelete: %v", err)
	}
}
