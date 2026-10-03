package isopatch

import (
	"encoding/binary"
	"fmt"
	"os"
	"strings"

	"kf4randomizer/internal/kfconst"
)

// DatEntry is one member of KF4.DAT's file table, offsets resolved to
// absolute ISO byte offsets.
type DatEntry struct {
	Name         string
	Length       uint32
	PaddedLength uint32
	AbsOffset    int64 // absolute byte offset of this member's content
	RowAbsOffset int64 // absolute byte offset of this member's file-table row
}

// ReadDatFileTable reads KF4.DAT's header and file table, given KF4.DAT's
// own absolute byte offset within the ISO.
func ReadDatFileTable(f *os.File, datAbsOffset int64) (map[string]DatEntry, error) {
	hdr := make([]byte, kfconst.DatHeaderSize)
	if _, err := f.ReadAt(hdr, datAbsOffset); err != nil {
		return nil, fmt.Errorf("reading DAT header: %w", err)
	}
	numFile := binary.LittleEndian.Uint32(hdr[4:8])

	// Check against the ISO's actual size before allocating.
	info, err := f.Stat()
	if err != nil {
		return nil, fmt.Errorf("stat: %w", err)
	}
	tableSize := int64(numFile) * kfconst.DatFileEntrySize
	if datAbsOffset+kfconst.DatHeaderSize+tableSize > info.Size() {
		return nil, fmt.Errorf("DAT header claims %d file-table entries (%d bytes), past the end of the ISO (%d bytes)", numFile, tableSize, info.Size())
	}

	table := make([]byte, tableSize)
	if _, err := f.ReadAt(table, datAbsOffset+kfconst.DatHeaderSize); err != nil {
		return nil, fmt.Errorf("reading DAT file table (%d entries): %w", numFile, err)
	}

	entries := make(map[string]DatEntry, numFile)
	for i := uint32(0); i < numFile; i++ {
		o := int(i) * kfconst.DatFileEntrySize
		raw := table[o : o+kfconst.DatNameFieldSize]
		name := string(raw)
		if idx := strings.IndexByte(name, 0); idx >= 0 {
			name = name[:idx]
		}

		length := binary.LittleEndian.Uint32(table[o+52:])
		paddedLength := binary.LittleEndian.Uint32(table[o+56:])
		relOffset := binary.LittleEndian.Uint32(table[o+60:])

		entries[strings.ToLower(name)] = DatEntry{
			Name:         name,
			Length:       length,
			PaddedLength: paddedLength,
			AbsOffset:    datAbsOffset + int64(relOffset),
			RowAbsOffset: datAbsOffset + kfconst.DatHeaderSize + int64(o),
		}
	}
	return entries, nil
}

func writeDatMemberContent(f *os.File, entry DatEntry, data []byte) error {
	if _, err := f.WriteAt(data, entry.AbsOffset); err != nil {
		return fmt.Errorf("writing %s content: %w", entry.Name, err)
	}
	return nil
}

// WriteDatMember overwrites a DAT member's content. data must be
// exactly entry.Length bytes; WriteDatMemberResized changes the size.
func WriteDatMember(f *os.File, entry DatEntry, data []byte) error {
	if uint32(len(data)) != entry.Length {
		return fmt.Errorf("%s: refusing to write %d bytes into a %d-byte DAT entry (size must match exactly)", entry.Name, len(data), entry.Length)
	}
	return writeDatMemberContent(f, entry, data)
}

// WriteDatMemberSlack writes data into a DAT member's padding slack
// (beyond Length, within PaddedLength), without touching the file table.
func WriteDatMemberSlack(f *os.File, entry DatEntry, byteOffset uint32, data []byte) error {
	if byteOffset+uint32(len(data)) > entry.PaddedLength {
		return fmt.Errorf("%s: writing %d bytes at slack offset %d would exceed this DAT entry's padded slot of %d bytes",
			entry.Name, len(data), byteOffset, entry.PaddedLength)
	}
	if byteOffset < entry.Length {
		return fmt.Errorf("%s: slack offset %d overlaps this member's own declared content (Length=%d)",
			entry.Name, byteOffset, entry.Length)
	}
	if _, err := f.WriteAt(data, entry.AbsOffset+int64(byteOffset)); err != nil {
		return fmt.Errorf("writing %s slack content: %w", entry.Name, err)
	}
	return nil
}

// WriteDatMemberResized overwrites a DAT member's content and updates its
// Length field, for content whose size differs from what's there now.
func WriteDatMemberResized(f *os.File, entry DatEntry, data []byte) error {
	if uint32(len(data)) > entry.PaddedLength {
		return fmt.Errorf("%s: new content is %d bytes, the slot only has %d", entry.Name, len(data), entry.PaddedLength)
	}
	if err := writeDatMemberContent(f, entry, data); err != nil {
		return err
	}
	if uint32(len(data)) != entry.Length {
		lengthBuf := make([]byte, 4)
		binary.LittleEndian.PutUint32(lengthBuf, uint32(len(data)))
		if _, err := f.WriteAt(lengthBuf, entry.RowAbsOffset+52); err != nil {
			return fmt.Errorf("updating %s's length field: %w", entry.Name, err)
		}
	}
	return nil
}
