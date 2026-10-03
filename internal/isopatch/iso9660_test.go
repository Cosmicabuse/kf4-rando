package isopatch

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// buildDirRecord builds a minimal valid ISO9660 directory record: length
// byte, LBA at +2, size at +10, name length at +32, name at +33.
func buildDirRecord(lba, size uint32, name string) []byte {
	length := 33 + len(name)
	if length%2 != 0 {
		length++ // ISO9660 records are padded to an even length
	}
	rec := make([]byte, length)
	rec[0] = byte(length)
	binary.LittleEndian.PutUint32(rec[2:], lba)
	binary.LittleEndian.PutUint32(rec[10:], size)
	rec[32] = byte(len(name))
	copy(rec[33:], name)
	return rec
}

func TestParseDirRecord(t *testing.T) {
	rec := buildDirRecord(100, 2048, "TEST.MAP;1")

	entry, length, err := parseDirRecord(rec, 0)
	if err != nil {
		t.Fatalf("parseDirRecord: %v", err)
	}
	if length != len(rec) {
		t.Errorf("length = %d, want %d", length, len(rec))
	}
	if entry.Name != "TEST.MAP" || entry.LBA != 100 || entry.Size != 2048 {
		t.Errorf("entry = %+v", entry)
	}
}

func TestParseDirRecord_ZeroLengthIsPaddingMarker(t *testing.T) {
	entry, length, err := parseDirRecord([]byte{0, 0, 0, 0}, 0)
	if err != nil {
		t.Fatalf("parseDirRecord: %v", err)
	}
	if entry != nil || length != 0 {
		t.Errorf("expected nil, 0 for a zero-length record, got %+v, %d", entry, length)
	}
}

func TestParseDirRecord_Truncated(t *testing.T) {
	// Fixed fields (LBA/size/nameLen) run past the buffer.
	if _, _, err := parseDirRecord([]byte{10, 0, 0, 0}, 0); err == nil {
		t.Error("a record whose fixed fields run past the buffer should fail")
	}
	// Name runs past the buffer.
	rec := buildDirRecord(1, 1, "TOOLONG.MAP")
	if _, _, err := parseDirRecord(rec[:34], 0); err == nil {
		t.Error("a record whose name runs past the buffer should fail")
	}
}

func TestSameFile(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "a.iso")
	if err := os.WriteFile(a, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}

	same, err := SameFile(a, a)
	if err != nil {
		t.Fatalf("SameFile: %v", err)
	}
	if !same {
		t.Error("a path compared with itself should be the same file")
	}

	// Different case, same path. Windows paths are case-insensitive.
	same, err = SameFile(a, strings.ToUpper(a))
	if err != nil {
		t.Fatalf("SameFile: %v", err)
	}
	if !same {
		t.Error("the same path in a different case should still be the same file")
	}

	b := filepath.Join(dir, "b.iso")
	same, err = SameFile(a, b)
	if err != nil {
		t.Fatalf("SameFile: %v", err)
	}
	if same {
		t.Error("two distinct paths should not be the same file")
	}

	// A nonexistent path should report "not the same file", not error.
	notYetCreated := filepath.Join(dir, "does-not-exist.iso")
	same, err = SameFile(a, notYetCreated)
	if err != nil {
		t.Fatalf("SameFile with a nonexistent path: %v", err)
	}
	if same {
		t.Error("a nonexistent path should never compare equal by identity")
	}
}

func TestCopyFile_RefusesSameFile(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "a.iso")
	if err := os.WriteFile(a, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := CopyFile(a, a); err == nil {
		t.Fatal("expected an error when src and dst are the same file")
	}

	// The file must survive untouched.
	got, err := os.ReadFile(a)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "hello" {
		t.Errorf("source file was modified: got %q, want %q", got, "hello")
	}
}

// buildFakeISO builds a minimal ISO9660 image with a PVD at sector 16
// whose root directory (sector 17) contains one file record for fileName.
func buildFakeISO(fileName string, fileLBA, fileSize uint32) []byte {
	const sectorSize = 2048
	img := make([]byte, sectorSize*18) // sectors 0-17

	pvdOff := 16 * sectorSize
	copy(img[pvdOff+1:], []byte("CD001"))
	// Root dir record embedded in the PVD; 0x00 name means "self".
	rootRec := buildDirRecord(17, sectorSize, "\x00")
	copy(img[pvdOff+156:], rootRec)

	rootDirOff := 17 * sectorSize
	fileRec := buildDirRecord(fileLBA, fileSize, fileName+";1")
	copy(img[rootDirOff:], fileRec)

	return img
}

func TestFindPath(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "fake.iso")
	if err := os.WriteFile(path, buildFakeISO("SLUS_203.18", 18, 12345), 0o644); err != nil {
		t.Fatal(err)
	}

	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	entry, err := FindPath(f, []string{"SLUS_203.18"})
	if err != nil {
		t.Fatalf("FindPath: %v", err)
	}
	if entry.Name != "SLUS_203.18" || entry.LBA != 18 || entry.Size != 12345 {
		t.Errorf("entry = %+v", entry)
	}

	// Case-insensitive, and ";1" version suffix already stripped.
	entry, err = FindPath(f, []string{"slus_203.18"})
	if err != nil {
		t.Fatalf("FindPath (lowercase): %v", err)
	}
	if entry.LBA != 18 {
		t.Errorf("case-insensitive lookup: entry = %+v", entry)
	}

	if _, err := FindPath(f, []string{"NOT_THERE"}); err == nil {
		t.Fatal("expected an error for a path that doesn't exist")
	}
}

func TestCopyFile(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.iso")
	dst := filepath.Join(dir, "dst.iso")
	if err := os.WriteFile(src, []byte("hello world"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := CopyFile(src, dst); err != nil {
		t.Fatalf("CopyFile: %v", err)
	}

	got, err := os.ReadFile(dst)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "hello world" {
		t.Errorf("got %q, want %q", got, "hello world")
	}
}
