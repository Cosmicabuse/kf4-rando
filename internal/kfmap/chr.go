package kfmap

import (
	"encoding/binary"
	"fmt"
	"math"
)

// chr.prm has one 0x450 record per monster type, with two drop slots and a
// percent each. The Fire Lord's Fire Mail is the only one at 100.
const (
	chrStride    = 0x450
	chrExp       = 0x0E
	chrGold      = 0x230
	chrDropItem  = 0x232
	chrDropItem2 = 0x234
)

// ScaleChrGold multiplies every monster type's gold drop. A drop is a random
// amount between half the stored value and the value itself.
func ScaleChrGold(buf []byte, factor float64) int { return scaleChr(buf, chrGold, factor) }

// ScaleChrExp multiplies the experience every monster type gives.
func ScaleChrExp(buf []byte, factor float64) int { return scaleChr(buf, chrExp, factor) }

func scaleChr(buf []byte, field int, factor float64) int {
	changed := 0
	for o := 0; o+chrStride <= len(buf); o += chrStride {
		v := binary.LittleEndian.Uint16(buf[o+field:])
		if v == 0 {
			continue
		}
		scaled := math.Round(float64(v) * factor)
		scaled = math.Max(1, math.Min(scaled, math.MaxUint16))
		binary.LittleEndian.PutUint16(buf[o+field:], uint16(scaled))
		changed++
	}
	return changed
}

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
