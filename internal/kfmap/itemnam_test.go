package kfmap

import "testing"

// buildItemNamName encodes a record whose name is the direct-ASCII bytes
// in s, terminated by 0xFFFF, matching decodeItemName's decoder.
func buildItemNamName(s string) []byte {
	rec := make([]byte, itemNamRecordSize)
	i := 0
	for _, c := range []byte(s) {
		rec[i] = c
		rec[i+1] = 0x08
		i += 2
	}
	rec[i] = 0xFF
	rec[i+1] = 0xFF
	return rec
}

func TestLoadClassIDToName(t *testing.T) {
	rec0 := buildItemNamName("AB")
	rec1 := make([]byte, itemNamRecordSize)
	for i := range rec1 {
		rec1[i] = 0xFF // empty slot, should be skipped
	}

	data := make([]byte, itemNamHeaderOffset+2*itemNamRecordSize)
	copy(data[itemNamHeaderOffset:], rec0)
	copy(data[itemNamHeaderOffset+itemNamRecordSize:], rec1)

	names := LoadClassIDToName(data)

	if got, want := names[1], "AB"; got != want {
		t.Errorf("classID 1: got %q, want %q", got, want)
	}
	if _, ok := names[2]; ok {
		t.Errorf("classID 2 should be absent (empty record), got %q", names[2])
	}
}

func TestItemNames_NameOrDefault(t *testing.T) {
	names := ItemNames{1: "Herb"}

	if got := names.NameOrDefault(1, "?"); got != "Herb" {
		t.Errorf("got %q, want %q", got, "Herb")
	}
	if got := names.NameOrDefault(2, "Unknown"); got != "Unknown" {
		t.Errorf("got %q, want fallback %q", got, "Unknown")
	}

	var nilNames ItemNames
	if got := nilNames.NameOrDefault(1, "Unknown"); got != "Unknown" {
		t.Errorf("nil ItemNames: got %q, want fallback %q", got, "Unknown")
	}
}

func TestDecodeItemName(t *testing.T) {
	for _, c := range []struct {
		why  string
		raw  []byte
		want string
	}{
		{"offset scheme: 0 is a space, 7 an apostrophe, 0x20-0x3F letters",
			[]byte{0x21, 0x08, 0x00, 0x08, 0x07, 0x08}, "A '"},
		{"a high byte that isn't 0x08", []byte{0x41, 0x00, 0xFF, 0xFF}, "?"},
	} {
		if got := decodeItemName(c.raw); got != c.want {
			t.Errorf("%s: got %q, want %q", c.why, got, c.want)
		}
	}
}
