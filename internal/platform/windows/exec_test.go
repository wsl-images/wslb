package windows

import "testing"

func TestDecodePossiblyUTF16(t *testing.T) {
	utf16le := []byte{0xff, 0xfe, 'H', 0x00, 'i', 0x00}
	got := DecodePossiblyUTF16(utf16le)
	if got != "Hi" {
		t.Fatalf("decoded=%q", got)
	}

	utf8 := []byte("hello\n")
	got = DecodePossiblyUTF16(utf8)
	if got != "hello" {
		t.Fatalf("decoded utf8=%q", got)
	}
}
