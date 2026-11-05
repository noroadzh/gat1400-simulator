package generator

import (
	"testing"

	"github.com/noroadzh/gat1400-simulator/internal/domain/resource"
)

func TestFake_PersonWithImage(t *testing.T) {
	obj := Fake(resource.KindPerson, Options{DeviceID: "D1", Seq: 1, WithImage: true})
	if obj["PersonID"] == nil || obj["DeviceID"] != "D1" {
		t.Fatalf("unexpected: %#v", obj)
	}
	if obj["SubImageList"] == nil {
		t.Fatal("expected SubImageList")
	}
}

func TestFake_MotorVehicle(t *testing.T) {
	obj := Fake(resource.KindMotorVehicle, Options{Seq: 7})
	if obj["MotorVehicleID"] == nil || obj["PlateNo"] == nil {
		t.Fatalf("unexpected: %#v", obj)
	}
}
