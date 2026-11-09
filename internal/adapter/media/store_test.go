package media

import "testing"

func TestStore_PutGetDelete(t *testing.T) {
	s := NewStore()
	s.Put("a", []byte("hello"))
	got, ok := s.Get("a")
	if !ok || string(got) != "hello" {
		t.Fatalf("Get = %q ok=%v", got, ok)
	}
	got[0] = 'x'
	got2, _ := s.Get("a")
	if string(got2) != "hello" {
		t.Fatal("Get must return copy")
	}
	s.Delete("a")
	if _, ok := s.Get("a"); ok {
		t.Fatal("expected missing after delete")
	}
}

func TestPlaceholderJPEGBase64(t *testing.T) {
	b64 := PlaceholderJPEGBase64()
	if b64 == "" {
		t.Fatal("empty base64")
	}
	if len(PlaceholderJPEG) < 4 || PlaceholderJPEG[0] != 0xff || PlaceholderJPEG[1] != 0xd8 {
		t.Fatal("not a JPEG SOI")
	}
}
