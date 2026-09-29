package resource

import "testing"

func TestCollectionAndIDFields(t *testing.T) {
	cases := map[Kind]struct {
		coll string
		id   string
	}{
		KindPerson:        {"Persons", "PersonID"},
		KindFace:          {"Faces", "FaceID"},
		KindMotorVehicle:  {"MotorVehicles", "MotorVehicleID"},
		KindNonMotorVehicle: {"NonMotorVehicles", "NonMotorVehicleID"},
		KindThing:         {"Things", "ThingID"},
		KindScene:         {"Scenes", "SceneID"},
		KindVideoSlice:    {"VideoSlices", "VideoSliceID"},
		KindImage:         {"Images", "ImageID"},
		KindFile:          {"Files", "FileID"},
		KindCase:          {"Cases", "CaseID"},
	}
	for k, want := range cases {
		if got := CollectionOf(k); got != want.coll {
			t.Fatalf("CollectionOf(%s) = %s, want %s", k, got, want.coll)
		}
		if got := IDOf(k); got != want.id {
			t.Fatalf("IDOf(%s) = %s, want %s", k, got, want.id)
		}
	}
}

func TestAllKindsUnique(t *testing.T) {
	seen := map[Kind]bool{}
	for _, k := range AllKinds {
		if seen[k] {
			t.Fatalf("duplicate kind %s", k)
		}
		seen[k] = true
	}
	if len(seen) != len(AllKinds) {
		t.Fatalf("AllKinds length mismatch")
	}
}

func TestIsValid(t *testing.T) {
	if !IsValid(KindPerson) {
		t.Fatal("KindPerson must be valid")
	}
	if IsValid(Kind("Garbage")) {
		t.Fatal("Garbage must not be valid")
	}
}