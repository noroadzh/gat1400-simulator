package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefault_HasExpectedShape(t *testing.T) {
	c := Default()
	if c == nil {
		t.Fatal("Default() returned nil")
	}
	if c.Node.SiteCode == 0 {
		t.Errorf("expected non-zero default SiteCode")
	}
	if c.Protocol.Listen == "" {
		t.Errorf("expected default Protocol.Listen")
	}
	if c.Control.Listen == "" {
		t.Errorf("expected default Control.Listen")
	}
	if c.Storage.Path == "" {
		t.Errorf("expected default Storage.Path")
	}
	if c.Auth.Realm == "" || c.Auth.Username == "" || c.Auth.Password == "" {
		t.Errorf("expected default auth credentials, got %+v", c.Auth)
	}
	if c.ScenariosDir == "" {
		t.Errorf("expected default ScenariosDir")
	}
}

func TestLoad_NoFilesReturnsDefaults(t *testing.T) {
	c, err := Load()
	if err != nil {
		t.Fatalf("Load() with no files: %v", err)
	}
	def := Default()
	if c.Node.SiteCode != def.Node.SiteCode {
		t.Errorf("expected default SiteCode, got %d", c.Node.SiteCode)
	}
}

func TestLoad_MissingFilesAreSilentlySkipped(t *testing.T) {
	c, err := Load("/nonexistent/foo.yaml", "/also/missing.yaml")
	if err != nil {
		t.Fatalf("Load() with missing files: %v", err)
	}
	if c.Protocol.Listen != Default().Protocol.Listen {
		t.Errorf("expected defaults retained, got %s", c.Protocol.Listen)
	}
}

func TestLoad_OverlayOverwritesDefaults(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	content := []byte(`
node:
  siteCode: 12345678
  industryCode: 130
protocol:
  listen: ":9999"
auth:
  realm: "test.realm"
  username: "u1"
  password: "p1"
  qop: "auth"
`)
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	c, err := Load(path)
	if err != nil {
		t.Fatalf("Load(): %v", err)
	}
	if c.Node.SiteCode != 12345678 {
		t.Errorf("SiteCode = %d, want 12345678", c.Node.SiteCode)
	}
	if c.Node.IndustryCode != 130 {
		t.Errorf("IndustryCode = %d, want 130", c.Node.IndustryCode)
	}
	if c.Protocol.Listen != ":9999" {
		t.Errorf("Protocol.Listen = %s, want :9999", c.Protocol.Listen)
	}
	if c.Auth.Username != "u1" || c.Auth.Password != "p1" || c.Auth.Realm != "test.realm" {
		t.Errorf("auth not overlaid: %+v", c.Auth)
	}
	// Unchanged fields must remain at default.
	if c.Control.Listen != Default().Control.Listen {
		t.Errorf("Control.Listen unexpectedly changed: %s", c.Control.Listen)
	}
}

func TestLoad_UnreadableFileReturnsError(t *testing.T) {
	// A path that exists but cannot be read (a directory) makes os.ReadFile
	// return an error which Load() must propagate.
	dir := t.TempDir()
	if _, err := Load(dir); err == nil {
		t.Fatal("expected error when path is a directory")
	}
}

func TestLoad_MultipleFilesLastWins(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "a.yaml")
	b := filepath.Join(dir, "b.yaml")
	if err := os.WriteFile(a, []byte("protocol:\n  listen: \":1111\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(b, []byte("protocol:\n  listen: \":2222\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	c, err := Load(a, b)
	if err != nil {
		t.Fatalf("Load(): %v", err)
	}
	if c.Protocol.Listen != ":2222" {
		t.Errorf("Protocol.Listen = %s, want :2222", c.Protocol.Listen)
	}
}