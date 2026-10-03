// Package isopatch finds and patches files inside the ISO: SLUS_203.18 and
// the members of KF4.DAT.
package isopatch

import (
	"crypto/md5"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"kf4randomizer/internal/kfconst"
)

// DirEntry is one parsed ISO9660 directory record.
type DirEntry struct {
	Name string
	LBA  uint32 // logical block address, in IsoSectorSize-byte sectors
	Size uint32 // byte size
}

// ByteOffset returns this entry's absolute byte offset within the ISO.
func (e DirEntry) ByteOffset() int64 {
	return int64(e.LBA) * kfconst.IsoSectorSize
}

func readSectors(f *os.File, lba uint32, count uint32) ([]byte, error) {
	buf := make([]byte, int64(count)*kfconst.IsoSectorSize)
	_, err := f.ReadAt(buf, int64(lba)*kfconst.IsoSectorSize)
	return buf, err
}

// parseDirRecord parses one ISO9660 directory record starting at off.
// Returns nil, 0, nil at a zero-length (sector-padding) record.
func parseDirRecord(data []byte, off int) (*DirEntry, int, error) {
	length := int(data[off])
	if length == 0 {
		return nil, 0, nil
	}
	const fixedFieldsSize = 33 // through the name-length byte at off+32
	if off+fixedFieldsSize > len(data) {
		return nil, 0, fmt.Errorf("directory record at offset %d is truncated (buffer is %d bytes)", off, len(data))
	}
	lba := binary.LittleEndian.Uint32(data[off+2:])
	size := binary.LittleEndian.Uint32(data[off+10:])
	nameLen := int(data[off+32])
	if off+fixedFieldsSize+nameLen > len(data) {
		return nil, 0, fmt.Errorf("directory record at offset %d claims a %d-byte name that runs past the end of the buffer (%d bytes)", off, nameLen, len(data))
	}
	nameRaw := data[off+33 : off+33+nameLen]

	var name string
	switch {
	case nameLen == 1 && nameRaw[0] == 0x00:
		name = "."
	case nameLen == 1 && nameRaw[0] == 0x01:
		name = ".."
	default:
		name = string(nameRaw)
		if i := strings.IndexByte(name, ';'); i >= 0 {
			name = name[:i] // strip ";1" version suffix
		}
	}

	return &DirEntry{Name: name, LBA: lba, Size: size}, length, nil
}

func listDir(f *os.File, lba, size uint32) ([]DirEntry, error) {
	nSectors := (size + kfconst.IsoSectorSize - 1) / kfconst.IsoSectorSize
	data, err := readSectors(f, lba, nSectors)
	if err != nil {
		return nil, err
	}

	var entries []DirEntry
	off := 0
	for off < len(data) {
		rec, recLen, err := parseDirRecord(data, off)
		if err != nil {
			return nil, err
		}
		if rec == nil {
			// Zero-length record: padding to the next sector boundary.
			off = ((off / kfconst.IsoSectorSize) + 1) * kfconst.IsoSectorSize
			continue
		}
		entries = append(entries, *rec)
		off += recLen
	}
	return entries, nil
}

// FindPath walks the ISO9660 tree from the root directory (Primary Volume
// Descriptor at sector 16) following pathParts in order, case-insensitively.
func FindPath(f *os.File, pathParts []string) (*DirEntry, error) {
	pvd, err := readSectors(f, 16, 1)
	if err != nil {
		return nil, err
	}
	if string(pvd[1:6]) != "CD001" {
		return nil, fmt.Errorf("sector 16 is not a valid Primary Volume Descriptor (magic mismatch)")
	}
	root, _, err := parseDirRecord(pvd, 156)
	if err != nil {
		return nil, fmt.Errorf("parsing root directory record: %w", err)
	}
	if root == nil {
		return nil, fmt.Errorf("could not parse root directory record")
	}

	cur := *root
	for _, part := range pathParts {
		entries, err := listDir(f, cur.LBA, cur.Size)
		if err != nil {
			return nil, err
		}
		var match *DirEntry
		for i := range entries {
			if strings.EqualFold(entries[i].Name, part) {
				match = &entries[i]
				break
			}
		}
		if match == nil {
			return nil, fmt.Errorf("%q not found while walking path %v", part, pathParts)
		}
		cur = *match
	}
	return &cur, nil
}

// CopyFile copies src over dst.
func CopyFile(src, dst string) error {
	if same, err := SameFile(src, dst); err != nil {
		return err
	} else if same {
		return fmt.Errorf("input and output are the same file (%s)", dst)
	}

	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Close()
}

// SameFile compares paths case-insensitively, then with os.SameFile. A path
// that doesn't exist yet just means not the same.
func SameFile(a, b string) (bool, error) {
	absA, err := filepath.Abs(a)
	if err != nil {
		return false, fmt.Errorf("resolving %s: %w", a, err)
	}
	absB, err := filepath.Abs(b)
	if err != nil {
		return false, fmt.Errorf("resolving %s: %w", b, err)
	}
	if strings.EqualFold(absA, absB) {
		return true, nil
	}

	infoA, errA := os.Stat(absA)
	infoB, errB := os.Stat(absB)
	if errA != nil || errB != nil {
		return false, nil
	}
	return os.SameFile(infoA, infoB), nil
}

// MD5 streams r, the ISO is too big to hold in memory.
func MD5(r io.Reader) (string, error) {
	h := md5.New()
	if _, err := io.Copy(h, r); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
