package response

import (
	"encoding/json"
	"testing"
)

func TestOK_HasExpectedFields(t *testing.T) {
	rs := OK("p1")
	if rs.StatusCode != CodeOK {
		t.Fatalf("StatusCode = %d", rs.StatusCode)
	}
	if rs.Id != "p1" {
		t.Fatalf("Id = %q", rs.Id)
	}
	if rs.Time.IsZero() {
		t.Fatal("Time not set")
	}
}

func TestError_Variants(t *testing.T) {
	cases := []struct {
		name string
		code StatusCode
		why  string
	}{
		{"invalid", CodeInvalid, "bad payload"},
		{"notfound", CodeNotFound, "missing"},
		{"unauthorized", CodeUnauthorized, "no creds"},
		{"conflict", CodeConflict, "already exists"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rs := Error("x", tc.code, tc.why)
			if rs.StatusCode != tc.code {
				t.Fatalf("code = %d", rs.StatusCode)
			}
			if rs.StatusString != tc.why {
				t.Fatalf("StatusString = %q", rs.StatusString)
			}
		})
	}
}

func TestResponseStatus_JSON_Roundtrip(t *testing.T) {
	rs := OK("entity-42")
	raw, err := json.Marshal(rs)
	if err != nil {
		t.Fatal(err)
	}
	var got ResponseStatus
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}
	if got.StatusCode != CodeOK || got.Id != "entity-42" {
		t.Fatalf("roundtrip lost data: %+v", got)
	}
}

func TestError_JSONIncludesReason(t *testing.T) {
	rs := Error("x", CodeFailure, "kaboom")
	raw, err := json.Marshal(rs)
	if err != nil {
		t.Fatal(err)
	}
	if !containsBytes(raw, []byte("kaboom")) {
		t.Fatalf("JSON missing reason: %s", raw)
	}
}

func containsBytes(haystack, needle []byte) bool {
	if len(needle) == 0 {
		return true
	}
	for i := 0; i+len(needle) <= len(haystack); i++ {
		match := true
		for j := 0; j < len(needle); j++ {
			if haystack[i+j] != needle[j] {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}