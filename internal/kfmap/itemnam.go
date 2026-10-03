package kfmap

// item_nam.prm is a flat table of 0x40-byte records (classID = row+1).
// The name runs from 0 to 0x30 (where the record's flag field starts),
// 2 bytes/char: high byte 0x08, low byte either an offset (0=space,
// 7=apostrophe, 0x20-0x3F=letters via low+0x20) or a raw printable-ASCII
// byte. 0xFFFF terminates. Longest real name is 18 chars.
const (
	itemNamHeaderOffset = 0x40
	itemNamRecordSize   = 0x40
	itemNamNameBytes    = 0x30
)

type ItemNames map[uint16]string

// NameOrDefault looks up classID's display name, falling back when the entry
// is missing or empty, as happens without item_nam.prm or out of range.
func (n ItemNames) NameOrDefault(classID uint16, fallback string) string {
	if name, ok := n[classID]; ok && name != "" {
		return name
	}
	return fallback
}

// LoadClassIDToName parses a raw item_nam.prm buffer into classID -> name.
// Unused/empty records are omitted from the returned map.
func LoadClassIDToName(data []byte) ItemNames {
	names := make(ItemNames)
	if len(data) <= itemNamHeaderOffset {
		return names
	}
	totalRecords := (len(data) - itemNamHeaderOffset) / itemNamRecordSize

	for row := 0; row < totalRecords; row++ {
		off := itemNamHeaderOffset + row*itemNamRecordSize
		rec := data[off : off+itemNamRecordSize]
		if isEmptyItemRecord(rec) {
			continue
		}
		classID := uint16(row + 1)
		names[classID] = decodeItemName(rec[:itemNamNameBytes])
	}
	return names
}

func isEmptyItemRecord(rec []byte) bool {
	for _, b := range rec {
		if b != 0xFF {
			return false
		}
	}
	return true
}

func decodeItemName(raw []byte) string {
	var out []byte
	for i := 0; i+1 < len(raw); i += 2 {
		low, high := raw[i], raw[i+1]
		if (low == 0xFF && high == 0xFF) || (low == 0 && high == 0) {
			break
		}
		if high != 0x08 {
			out = append(out, '?')
			continue
		}
		switch {
		case low == 0:
			out = append(out, ' ')
		case low == 7:
			out = append(out, '\'')
		case low >= 0x20 && low <= 0x3F:
			out = append(out, low+0x20)
		case low >= 32 && low <= 126:
			out = append(out, low)
		default:
			out = append(out, '?')
		}
	}
	return string(out)
}
