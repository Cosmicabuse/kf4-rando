package isopatch

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"

	"kf4randomizer/internal/kfconst"
)

func TestReadDatFileTable_RejectsImplausibleNumFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "fake.dat")

	// A 64-byte header claiming an absurd number of file-table entries,
	// backed by a file far too small to actually hold them.
	hdr := make([]byte, kfconst.DatHeaderSize)
	binary.LittleEndian.PutUint32(hdr[4:8], 0xFFFFFFFF)
	if err := os.WriteFile(path, hdr, 0o644); err != nil {
		t.Fatal(err)
	}

	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	if _, err := ReadDatFileTable(f, 0); err == nil {
		t.Fatal("expected an error for a numFile that runs past the end of the file")
	}
}

func TestReadDatFileTable_EmptyTable(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "fake.dat")

	hdr := make([]byte, kfconst.DatHeaderSize)
	binary.LittleEndian.PutUint32(hdr[4:8], 0) // zero entries
	if err := os.WriteFile(path, hdr, 0o644); err != nil {
		t.Fatal(err)
	}

	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	entries, err := ReadDatFileTable(f, 0)
	if err != nil {
		t.Fatalf("ReadDatFileTable: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("got %d entries, want 0", len(entries))
	}
}

func buildDatFileRow(name string, length, paddedLength, relOffset uint32) []byte {
	row := make([]byte, kfconst.DatFileEntrySize)
	copy(row[:kfconst.DatNameFieldSize], name)
	binary.LittleEndian.PutUint32(row[52:], length)
	binary.LittleEndian.PutUint32(row[56:], paddedLength)
	binary.LittleEndian.PutUint32(row[60:], relOffset)
	return row
}

// buildFakeDat builds a one-member DAT and returns the content's offset.
func buildFakeDat(t *testing.T, name string, content []byte, paddedLength uint32) ([]byte, int64) {
	t.Helper()
	const contentOff = kfconst.DatHeaderSize + kfconst.DatFileEntrySize

	hdr := make([]byte, kfconst.DatHeaderSize)
	binary.LittleEndian.PutUint32(hdr[4:8], 1)
	row := buildDatFileRow(name, uint32(len(content)), paddedLength, contentOff)

	img := make([]byte, contentOff+int(paddedLength))
	copy(img, hdr)
	copy(img[kfconst.DatHeaderSize:], row)
	copy(img[contentOff:], content)
	return img, contentOff
}

func openFakeDat(t *testing.T, img []byte) *os.File {
	t.Helper()
	path := filepath.Join(t.TempDir(), "fake.dat")
	if err := os.WriteFile(path, img, 0o644); err != nil {
		t.Fatal(err)
	}
	f, err := os.OpenFile(path, os.O_RDWR, 0)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { f.Close() })
	return f
}

func TestReadDatFileTable_RoundTrip(t *testing.T) {
	img, contentOff := buildFakeDat(t, "ini/0611.ini", []byte("HELLO"), 32768)
	f := openFakeDat(t, img)

	entries, err := ReadDatFileTable(f, 0)
	if err != nil {
		t.Fatalf("ReadDatFileTable: %v", err)
	}
	entry, ok := entries["ini/0611.ini"]
	if !ok {
		t.Fatalf("entry not found (lowercased lookup), got %v", entries)
	}
	if entry.Name != "ini/0611.ini" || entry.Length != 5 || entry.PaddedLength != 32768 || entry.AbsOffset != contentOff {
		t.Errorf("entry = %+v", entry)
	}
}

func TestWriteDatMember(t *testing.T) {
	img, _ := buildFakeDat(t, "ini/0611.ini", []byte("HELLO"), 32768)
	f := openFakeDat(t, img)

	entries, err := ReadDatFileTable(f, 0)
	if err != nil {
		t.Fatalf("ReadDatFileTable: %v", err)
	}
	entry := entries["ini/0611.ini"]

	if err := WriteDatMember(f, entry, []byte("WORLD")); err != nil {
		t.Fatalf("WriteDatMember (exact length): %v", err)
	}
	got := make([]byte, 5)
	if _, err := f.ReadAt(got, entry.AbsOffset); err != nil {
		t.Fatal(err)
	}
	if string(got) != "WORLD" {
		t.Errorf("content after write = %q, want %q", got, "WORLD")
	}

	if err := WriteDatMember(f, entry, []byte("TOO LONG")); err == nil {
		t.Error("expected an error when content length doesn't exactly match entry.Length")
	}
	if err := WriteDatMember(f, entry, []byte("HI")); err == nil {
		t.Error("expected an error when content is shorter than entry.Length too")
	}
}

func TestWriteDatMemberResized(t *testing.T) {
	img, _ := buildFakeDat(t, "ini/0611.ini", []byte("HELLO"), 32) // tiny padded slot for an easy overflow test
	f := openFakeDat(t, img)

	entries, err := ReadDatFileTable(f, 0)
	if err != nil {
		t.Fatalf("ReadDatFileTable: %v", err)
	}
	entry := entries["ini/0611.ini"]

	// Grows within PaddedLength: succeeds and updates the Length field.
	if err := WriteDatMemberResized(f, entry, []byte("HELLO WORLD")); err != nil {
		t.Fatalf("WriteDatMemberResized (within padding): %v", err)
	}
	got := make([]byte, 11)
	if _, err := f.ReadAt(got, entry.AbsOffset); err != nil {
		t.Fatal(err)
	}
	if string(got) != "HELLO WORLD" {
		t.Errorf("content = %q, want %q", got, "HELLO WORLD")
	}
	reread, err := ReadDatFileTable(f, 0)
	if err != nil {
		t.Fatalf("re-reading file table: %v", err)
	}
	if got := reread["ini/0611.ini"].Length; got != 11 {
		t.Errorf("Length field after resize = %d, want 11", got)
	}

	// past PaddedLength (32 bytes), has to fail
	tooBig := make([]byte, 64)
	if err := WriteDatMemberResized(f, entry, tooBig); err == nil {
		t.Error("expected an error when new content exceeds PaddedLength")
	}
}
