package kfmap

import (
	"encoding/binary"
	"fmt"
)

// chr.prm has one 0x450 record per monster type, with two drop slots and a
// percent each. The Fire Lord's Fire Mail is the only one at 100.
const (
	chrStride    = 0x450
	chrDropItem  = 0x232
	chrDropItem2 = 0x234
)

// ChrDrop reads the first drop item of a creature class.
func ChrDrop(buf []byte, class int) (uint16, error) {
	o := class * chrStride
	if o < 0 || o+chrStride > len(buf) {
		return 0, fmt.Errorf("creature class %d is outside chr.prm (%d bytes)", class, len(buf))
	}
	return binary.LittleEndian.Uint16(buf[o+chrDropItem:]), nil
}

// SetChrDrop overwrites it. The percent beside it is left alone, so a class
// that always dropped still always drops.
func SetChrDrop(buf []byte, class int, classID uint16) error {
	o := class * chrStride
	if o < 0 || o+chrStride > len(buf) {
		return fmt.Errorf("creature class %d is outside chr.prm (%d bytes)", class, len(buf))
	}
	binary.LittleEndian.PutUint16(buf[o+chrDropItem:], classID)
	return nil
}
