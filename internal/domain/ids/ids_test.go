package ids

import (
	"regexp"
	"strings"
	"testing"
)

func TestGenerator_DeviceID_Length20AndDigits(t *testing.T) {
	g := NewGenerator(41000000, 30)
	got := g.DeviceID()
	if len(got) != 20 {
		t.Fatalf("DeviceID length = %d, want 20", len(got))
	}
	if matched, _ := regexp.MatchString(`^\d{20}$`, got); !matched {
		t.Fatalf("DeviceID = %q, want 20 digits", got)
	}
	if !strings.HasPrefix(got, "410000003011") {
		t.Fatalf("DeviceID prefix = %q, want %q", got[:11], "410000003011")
	}
}

func TestGenerator_DeviceID_SequenceIncreases(t *testing.T) {
	g := NewGenerator(41000000, 130)
	a := g.DeviceID()
	b := g.DeviceID()
	if a == b {
		t.Fatalf("expected distinct DeviceIDs across calls, got %q twice", a)
	}
	if a >= b {
		t.Fatalf("expected sequence increase: %q < %q", a, b)
	}
}

func TestGenerator_Nonce_UniqueAndHexLength(t *testing.T) {
	g := NewGenerator(0, 130)
	seen := map[string]bool{}
	// hexRE 提到循环外，避免 1000 次重复编译正则
	hexRE := regexp.MustCompile(`^[0-9a-f]+$`)
	for i := 0; i < 1000; i++ {
		n := g.Nonce()
		if len(n) != 32 {
			t.Fatalf("Nonce length = %d, want 32", len(n))
		}
		if !hexRE.MatchString(n) {
			t.Fatalf("Nonce = %q, want hex", n)
		}
		if seen[n] {
			t.Fatalf("duplicate nonce %q at %d", n, i)
		}
		seen[n] = true
	}
}

func TestGenerator_SubscribeID_Length12AndUppercase(t *testing.T) {
	g := NewGenerator(0, 130)
	id := g.SubscribeID()
	if len(id) != 12 {
		t.Fatalf("SubscribeID length = %d, want 12", len(id))
	}
	if matched, _ := regexp.MatchString(`^[0-9A-F]+$`, id); !matched {
		t.Fatalf("SubscribeID = %q, want uppercase hex", id)
	}
}

func TestGenerator_UUID_NoDashesAndUnique(t *testing.T) {
	g := NewGenerator(0, 130)
	a := g.UUID()
	b := g.UUID()
	if a == b {
		t.Fatalf("UUIDs collided: %q", a)
	}
	if strings.Contains(a, "-") {
		t.Fatalf("UUID should be dashless, got %q", a)
	}
	if len(a) != 32 {
		t.Fatalf("UUID length = %d, want 32", len(a))
	}
}

func TestGenerator_ClampsOutOfRangeSiteCode(t *testing.T) {
	g := NewGenerator(0, 130) // siteCode 0 → clamped to 0
	id := g.DeviceID()
	if !strings.HasPrefix(id, "00000000") {
		t.Fatalf("DeviceID prefix = %q, want zeros", id[:8])
	}
}