// Package itemmodel points every map's item-model chunk at one shared model.
// Relies on the whitelist patch, which makes every lookup resolve to pair 0.
package itemmodel

import (
	"encoding/binary"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"

	"kf4randomizer/internal/kfconst"
	"kf4randomizer/internal/kfmap"
)

// Unused pairs shrink to this; the chain still walks through them.
const fillerBlobSize = 16

type Change struct {
	MapName string
	Pairs   int // whitelist entries
	Before  int // bytes the chunk originally held
	After   int // bytes it holds now
}

type Result struct {
	TotalMaps int
	Changes   []Change
	Skipped   []string // maps the rewrite could not be applied to, with the reason
	Untouched []string // maps with nothing shuffled, left vanilla
}

func chunkOffset(buf []byte) int {
	return int(binary.LittleEndian.Uint32(buf[kfconst.MapHeaderItemChunkOffset:]))
}

// whitelistLen counts model groups: negative-terminated, 32 at most.
func whitelistLen(m *kfmap.Map) int {
	off := m.ItemOffset + len(m.Items)*kfconst.ItemStructSize
	for i := 0; i < kfconst.ItemWhitelistMaxEntries; i++ {
		p := off + i*2
		if p+2 > len(m.Buf) {
			return i
		}
		if int16(binary.LittleEndian.Uint16(m.Buf[p:])) < 0 {
			return i
		}
	}
	return kfconst.ItemWhitelistMaxEntries
}

// chunkLen walks the existing chain and returns the bytes it spans.
func chunkLen(buf []byte, start, pairs int) (int, error) {
	p := start
	for i := 0; i < pairs*2; i++ {
		if p+4 > len(buf) {
			return 0, fmt.Errorf("blob %d starts past the end of the map", i)
		}
		size := int(binary.LittleEndian.Uint32(buf[p:]))
		if size <= 0 || p+size > len(buf) {
			return 0, fmt.Errorf("blob %d has an unusable size (%d)", i, size)
		}
		p += size
	}
	return p - start, nil
}

// Same filters randomize.RunNoLogic applies.
func shuffled(classID uint16) bool {
	return classID != 0 &&
		kfconst.IsKnownClassID(classID) &&
		!kfconst.IsExcludedFromRandomization(classID)
}

// Models are all-or-nothing per map, so a map with nothing shuffled keeps
// its vanilla look.
func holdsShuffledItem(m *kfmap.Map) bool {
	for _, it := range m.Items {
		if shuffled(it.ClassID) {
			return true
		}
	}
	for _, c := range m.Containers {
		if shuffled(c.ItemClassID) {
			return true
		}
	}
	for _, c := range m.Creatures {
		if shuffled(c.DropClassID) {
			return true
		}
	}
	return false
}

// .omd packet layout, used only by Scale.
const (
	omdHeaderSize    = 0x10
	omdMeshSize      = 48
	omdMeshStripsOff = 36 // u32 offTristrips, u32 numTristrips, u32 pad
	packetHeaderSize = 0x90
	packetVertexSize = 32
	packetTailSize   = 16
	packetVertexOff  = 0x70 // u8 count of vertices in this packet
)

// Scale multiplies vertex positions. The mesh header has scale values too,
// untested.
func Scale(model []byte, factor float64) ([]byte, error) {
	if factor == 1 {
		return model, nil
	}
	if factor <= 0 {
		return nil, fmt.Errorf("scale %v must be positive", factor)
	}
	out := make([]byte, len(model))
	copy(out, model)

	numMesh := int(binary.LittleEndian.Uint16(out[4:]))
	for i := 0; i < numMesh; i++ {
		mesh := omdHeaderSize + i*omdMeshSize
		if mesh+omdMeshSize > len(out) {
			return nil, fmt.Errorf("mesh %d runs past the end of the model", i)
		}
		p := int(binary.LittleEndian.Uint32(out[mesh+omdMeshStripsOff:]))
		strips := int(binary.LittleEndian.Uint32(out[mesh+omdMeshStripsOff+4:]))
		for s := 0; s < strips; s++ {
			if p+packetHeaderSize > len(out) {
				return nil, fmt.Errorf("mesh %d strip %d runs past the end", i, s)
			}
			n := int(out[p+packetVertexOff])
			for v := 0; v < n; v++ {
				o := p + packetHeaderSize + v*packetVertexSize
				if o+packetVertexSize > len(out) {
					return nil, fmt.Errorf("mesh %d strip %d vertex %d runs past the end", i, s, v)
				}
				for axis := 0; axis < 3; axis++ {
					c := int16(binary.LittleEndian.Uint16(out[o+axis*2:]))
					scaled := math.Round(float64(c) * factor)
					if scaled > math.MaxInt16 || scaled < math.MinInt16 {
						return nil, fmt.Errorf("scale %v pushes a vertex out of int16 range", factor)
					}
					binary.LittleEndian.PutUint16(out[o+axis*2:], uint16(int16(scaled)))
				}
			}
			p += packetHeaderSize + n*packetVertexSize + packetTailSize
		}
	}
	return out, nil
}

// Texture blobs end with a 256-entry RGBA palette. Alpha 128 is opaque.
const (
	textureCLUTEntries = 256
	textureCLUTSize    = textureCLUTEntries * 4
	textureOpaqueAlpha = 128
)

// Recolor paints the whole palette one colour, no need for the swizzle or UVs.
func Recolor(texture []byte, r, g, b uint8) ([]byte, error) {
	if len(texture) < textureCLUTSize {
		return nil, fmt.Errorf("texture is %d bytes, too small to hold a palette", len(texture))
	}
	out := make([]byte, len(texture))
	copy(out, texture)
	clut := out[len(out)-textureCLUTSize:]
	for i := 0; i < textureCLUTEntries; i++ {
		clut[i*4+0] = r
		clut[i*4+1] = g
		clut[i*4+2] = b
		clut[i*4+3] = textureOpaqueAlpha
	}
	return out, nil
}

// ParseColor reads "#rrggbb" or "rrggbb".
func ParseColor(s string) (uint8, uint8, uint8, error) {
	s = strings.TrimPrefix(strings.TrimSpace(s), "#")
	if len(s) != 6 {
		return 0, 0, 0, fmt.Errorf("colour %q must be six hex digits, optionally led by #", s)
	}
	var v [3]uint8
	for i := 0; i < 3; i++ {
		n, err := strconv.ParseUint(s[i*2:i*2+2], 16, 8)
		if err != nil {
			return 0, 0, 0, fmt.Errorf("colour %q is not hex: %w", s, err)
		}
		v[i] = uint8(n)
	}
	return v[0], v[1], v[2], nil
}

// The chain walks by these size fields, so a wrong one derails every pair
// after it.
func selfSized(blob []byte) bool {
	return len(blob) >= 4 && int(binary.LittleEndian.Uint32(blob)) == len(blob)
}

// Run rewrites every map's chunk so it starts with model and texture.
func Run(maps []*kfmap.Map, model, texture []byte) Result {
	sort.Slice(maps, func(i, j int) bool { return maps[i].Name < maps[j].Name })

	result := Result{TotalMaps: len(maps)}
	for name, blob := range map[string][]byte{"model": model, "texture": texture} {
		if !selfSized(blob) {
			result.Skipped = append(result.Skipped, fmt.Sprintf(
				"every map: %s blob is %d bytes but its size field says otherwise",
				name, len(blob)))
			return result
		}
	}
	for _, m := range maps {
		start := chunkOffset(m.Buf)
		pairs := whitelistLen(m)
		if start == 0 || pairs == 0 {
			continue // no items in this map, so nothing renders from it
		}
		if !holdsShuffledItem(m) {
			result.Untouched = append(result.Untouched, m.Name)
			continue
		}
		have, err := chunkLen(m.Buf, start, pairs)
		if err != nil {
			result.Skipped = append(result.Skipped, fmt.Sprintf("%s: %v", m.Name, err))
			continue
		}
		need := len(model) + len(texture) + (pairs*2-2)*fillerBlobSize
		if need > have {
			result.Skipped = append(result.Skipped,
				fmt.Sprintf("%s: needs %d bytes, chunk only spans %d", m.Name, need, have))
			continue
		}

		p := start
		copy(m.Buf[p:], model)
		p += len(model)
		copy(m.Buf[p:], texture)
		p += len(texture)
		// Stubs are never dereferenced, but their sizes must stay right.
		for i := 0; i < pairs*2-2; i++ {
			binary.LittleEndian.PutUint32(m.Buf[p:], fillerBlobSize)
			for j := 4; j < fillerBlobSize; j++ {
				m.Buf[p+j] = 0
			}
			p += fillerBlobSize
		}
		result.Changes = append(result.Changes, Change{
			MapName: m.Name, Pairs: pairs, Before: have, After: need,
		})
	}
	return result
}

// Summary formats a short report. mapNames is optional (pass nil).
func (r Result) Summary(mapNames kfmap.MapNames) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Shared item model: %d of %d maps repointed.\n",
		len(r.Changes), r.TotalMaps)
	if len(r.Untouched) > 0 {
		fmt.Fprintf(&b, "  left vanilla, nothing in them is shuffled: %s\n",
			strings.Join(r.Untouched, " "))
	}
	for _, s := range r.Skipped {
		fmt.Fprintf(&b, "  !! skipped %s\n", s)
	}
	return b.String()
}
