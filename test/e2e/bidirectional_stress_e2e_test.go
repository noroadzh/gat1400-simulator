package e2e

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/noroadzh/gat1400-simulator/internal/adapter/wire"
	"github.com/noroadzh/gat1400-simulator/internal/domain/resource"
)

// TestUAC_Bidirectional_FullProtocolLoopback covers Register → Collection push →
// Catalog upsert → DispositionNotification on a real TCP UAS.
func TestUAC_Bidirectional_FullProtocolLoopback(t *testing.T) {
	fx := startRealServer(t, realListener(t))
	ctx := context.Background()
	deviceID := "41000000001320000099"
	seedNode(t, fx, deviceID)

	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	client := wire.NewClient(log, fx.nonce)
	client.Configure(wire.Options{
		Username: "admin",
		Password: "admin",
		Realm:    "com.gat1400.simulator",
		Qop:      "auth",
		Timeout:  5 * time.Second,
	})

	base := fx.ts.URL

	if err := client.System().Register(ctx, base, wire.RegisterObject{
		DeviceID: deviceID, Status: "ONLINE", Keepalive: 60,
	}); err != nil {
		t.Fatalf("Register: %v", err)
	}

	faceID := "41000000005030312999"
	if err := client.Collection().Push(ctx, base, deviceID, resource.KindFace, map[string]any{
		"FaceList": map[string]any{"FaceObject": []map[string]any{
			{"FaceID": faceID, "Name": "e2e-stress"},
		}},
	}); err != nil {
		t.Fatalf("Collection.Push: %v", err)
	}

	listed, err := client.Collection().List(ctx, base, deviceID, resource.KindFace, 10, 1)
	if err != nil {
		t.Fatalf("Collection.List: %v", err)
	}
	if listed.TotalNum < 1 {
		t.Fatalf("expected TotalNum>=1 after push, got %#v", listed)
	}

	apeID := "41000000001310000999"
	if err := client.Catalog().UpsertAPEs(ctx, base, deviceID, map[string]any{
		"APEList": map[string]any{"APEObject": []map[string]any{
			{"APEID": apeID, "Name": "e2e-ape", "IsOnline": "1"},
		}},
	}); err != nil {
		t.Fatalf("Catalog.UpsertAPEs: %v", err)
	}
	apes, err := client.Catalog().ListAPEs(ctx, base, deviceID)
	if err != nil {
		t.Fatalf("Catalog.ListAPEs: %v", err)
	}
	found := false
	for _, a := range apes {
		if a["APEID"] == apeID {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("APE %s not found in list %#v", apeID, apes)
	}

	dispID := "91000000001000000099"
	if _, err := client.Cascade().DispositionCreate(ctx, base, deviceID, map[string]any{
		"DispositionList": []map[string]any{
			{"DispositionID": dispID, "DispositionCode": "010000", "Title": "e2e"},
		},
	}); err != nil {
		t.Fatalf("DispositionCreate: %v", err)
	}
	if err := client.Cascade().DispositionNotificationPush(ctx, base, deviceID, map[string]any{
		"DispositionID": dispID,
		"TriggerTime":   time.Now().UTC().Format(time.RFC3339),
	}); err != nil {
		t.Fatalf("DispositionNotificationPush: %v", err)
	}
	notifs, err := client.Cascade().DispositionNotificationList(ctx, base, deviceID, dispID)
	if err != nil {
		t.Fatalf("DispositionNotificationList: %v", err)
	}
	if len(notifs) < 1 {
		t.Fatalf("expected disposition notifications, got %d", len(notifs))
	}
}
