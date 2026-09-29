package node

import "testing"

func TestSanity(t *testing.T) {
	good := Node{ID: "n1", Name: "n1", Role: RoleDevice}
	if err := good.Sanity(); err != nil {
		t.Fatalf("good.Sanity: %v", err)
	}
	cases := []Node{
		{ID: "", Name: "x", Role: RoleDevice},                 // missing id
		{ID: "n1", Name: "", Role: RoleDevice},                // missing name
		{ID: "n1", Name: "n1", Role: Role("martian")},         // invalid role
	}
	wants := []error{ErrMissingID, ErrMissingName, ErrInvalidRole}
	for i, c := range cases {
		if err := c.Sanity(); err != wants[i] {
			t.Fatalf("case %d: got %v, want %v", i, err, wants[i])
		}
	}
}

func TestHasCapability(t *testing.T) {
	n := Node{Capabilities: []Capability{CapSystem, CapCollection}}
	if !n.HasCapability(CapSystem) {
		t.Fatal("expected HasCapability(CapSystem)")
	}
	if !n.HasCapability(CapCollection) {
		t.Fatal("expected HasCapability(CapCollection)")
	}
	if n.HasCapability(CapCascade) {
		t.Fatal("did not expect HasCapability(CapCascade)")
	}
}