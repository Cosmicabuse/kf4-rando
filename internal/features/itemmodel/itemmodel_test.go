package itemmodel

import (
	"encoding/binary"
	"reflect"
	"testing"

	"kf4randomizer/internal/kfconst"
	"kf4randomizer/internal/kfmap"
)

const (
	shuffledClassID = 6   // an ordinary item, in the pool
	excludedClassID = 196 // Moonlight Sword, never shuffled
)

// newTestMap has `pairs` blobs and a whitelist right after the item array.
func newTestMap(name string, itemClassIDs []uint16, pairs, blobSize, chunkAt int) *kfmap.Map {
	buf := make([]byte, chunkAt+pairs*2*blobSize+64)
	binary.LittleEndian.PutUint32(buf[kfconst.MapHeaderItemChunkOffset:], uint32(chunkAt))

	itemOffset := 0x100
	items := make([]kfmap.Item, len(itemClassIDs))
	for i, cid := range itemClassIDs {
		items[i] = kfmap.Item{Index: i, ClassID: cid}
	}

	wl := itemOffset + len(items)*kfconst.ItemStructSize
	for i := 0; i < pairs; i++ {
		binary.LittleEndian.PutUint16(buf[wl+i*2:], uint16(i+1))
	}
	binary.LittleEndian.PutUint16(buf[wl+pairs*2:], 0xFFFF) // negative terminator

	// Each blob states its own length, which is what the chain walks by.
	for i := 0; i < pairs*2; i++ {
		binary.LittleEndian.PutUint32(buf[chunkAt+i*blobSize:], uint32(blobSize))
	}

	return &kfmap.Map{Name: name, Buf: buf, ItemOffset: itemOffset, Items: items}
}

// blobOf's first u32 is its own length, like a real .omd or texture.
func blobOf(n int) []byte {
	b := make([]byte, n)
	binary.LittleEndian.PutUint32(b, uint32(n))
	return b
}

// walk follows the blob chain the way the engine does.
func walk(t *testing.T, m *kfmap.Map, pairs int) int {
	t.Helper()
	p := chunkOffset(m.Buf)
	for i := 0; i < pairs*2; i++ {
		size := int(binary.LittleEndian.Uint32(m.Buf[p:]))
		if size <= 0 || p+size > len(m.Buf) {
			t.Fatalf("blob %d has an unwalkable size %d at %d", i, size, p)
		}
		p += size
	}
	return p
}

func TestRun_ChainStaysWalkable(t *testing.T) {
	const pairs, blob, chunkAt = 4, 2048, 0x400
	m := newTestMap("000.map", []uint16{shuffledClassID}, pairs, blob, chunkAt)
	model, tex := blobOf(900), blobOf(700)

	result := Run([]*kfmap.Map{m}, model, tex)
	if len(result.Changes) != 1 {
		t.Fatalf("Changes = %d, want 1 (skipped: %v)", len(result.Changes), result.Skipped)
	}
	if got := whitelistLen(m); got != pairs {
		t.Fatalf("whitelistLen = %d, want %d", got, pairs)
	}

	end := walk(t, m, pairs)
	want := chunkAt + len(model) + len(tex) + (pairs*2-2)*fillerBlobSize
	if end != want {
		t.Errorf("chain ends at %d, want %d", end, want)
	}
	if got := int(binary.LittleEndian.Uint32(m.Buf[chunkAt:])); got != len(model) {
		t.Errorf("first blob size = %d, want the model's %d", got, len(model))
	}
}

func TestRun_SkipsWhenChunkTooSmall(t *testing.T) {
	m := newTestMap("000.map", []uint16{shuffledClassID}, 2, 64, 0x400)
	result := Run([]*kfmap.Map{m}, blobOf(4096), blobOf(4096))

	if len(result.Changes) != 0 {
		t.Error("rewrote a map whose chunk cannot hold the model")
	}
	if len(result.Skipped) != 1 {
		t.Fatalf("Skipped = %v, want one entry explaining the skip", result.Skipped)
	}
	if got := int(binary.LittleEndian.Uint32(m.Buf[0x400:])); got != 64 {
		t.Errorf("first blob size = %d, want the original 64", got)
	}
}

func TestRun_RejectsABlobThatMisstatesItsSize(t *testing.T) {
	m := newTestMap("000.map", []uint16{shuffledClassID}, 2, 2048, 0x400)
	bad := make([]byte, 100) // size field left at zero

	result := Run([]*kfmap.Map{m}, bad, blobOf(100))
	if len(result.Changes) != 0 {
		t.Error("wrote a blob whose size field would derail the chain")
	}
	if len(result.Skipped) != 1 {
		t.Fatalf("Skipped = %v, want one entry naming the bad blob", result.Skipped)
	}
}

func TestRun_IgnoresMapsWithoutAChunk(t *testing.T) {
	m := newTestMap("039.map", []uint16{shuffledClassID}, 2, 512, 0x400)
	binary.LittleEndian.PutUint32(m.Buf[kfconst.MapHeaderItemChunkOffset:], 0)

	result := Run([]*kfmap.Map{m}, blobOf(16), blobOf(16))
	if len(result.Changes) != 0 || len(result.Skipped) != 0 {
		t.Errorf("a map with no chunk should be passed over silently, got %+v", result)
	}
}

func TestRun_LeavesAnAllVanillaMapAlone(t *testing.T) {
	// Map 042 in the real game: its only pickup is the Moonlight Sword, which
	// never moves, so disguising it would hide something that isn't randomized.
	m := newTestMap("042.map", []uint16{excludedClassID}, 2, 2048, 0x400)

	result := Run([]*kfmap.Map{m}, blobOf(900), blobOf(700))
	if len(result.Changes) != 0 {
		t.Error("disguised an item that never gets shuffled")
	}
	if len(result.Untouched) != 1 || result.Untouched[0] != "042.map" {
		t.Fatalf("Untouched = %v, want [042.map]", result.Untouched)
	}
	if got := int(binary.LittleEndian.Uint32(m.Buf[0x400:])); got != 2048 {
		t.Errorf("first blob size = %d, want the original 2048", got)
	}
}

func TestRun_OneShuffledItemIsEnough(t *testing.T) {
	m := newTestMap("000.map", []uint16{excludedClassID, shuffledClassID}, 2, 2048, 0x400)

	result := Run([]*kfmap.Map{m}, blobOf(900), blobOf(700))
	if len(result.Changes) != 1 {
		t.Fatalf("Changes = %d, want 1 (untouched: %v, skipped: %v)",
			len(result.Changes), result.Untouched, result.Skipped)
	}
}

func TestRun_CountsCreatureDropsAsShuffled(t *testing.T) {
	m := newTestMap("105.map", nil, 2, 2048, 0x400)
	m.Creatures = []kfmap.Creature{{Index: 0, DropClassID: shuffledClassID}}

	result := Run([]*kfmap.Map{m}, blobOf(900), blobOf(700))
	if len(result.Changes) != 1 {
		t.Fatalf("a map whose only pickup is a monster drop should be repointed, got %+v", result)
	}
}

// vertexPositions reads positions back the way Scale walks them.
func vertexPositions(t *testing.T, model []byte) [][3]int16 {
	t.Helper()
	var out [][3]int16
	numMesh := int(binary.LittleEndian.Uint16(model[4:]))
	for i := 0; i < numMesh; i++ {
		mesh := omdHeaderSize + i*omdMeshSize
		p := int(binary.LittleEndian.Uint32(model[mesh+omdMeshStripsOff:]))
		strips := int(binary.LittleEndian.Uint32(model[mesh+omdMeshStripsOff+4:]))
		for s := 0; s < strips; s++ {
			n := int(model[p+packetVertexOff])
			for v := 0; v < n; v++ {
				o := p + packetHeaderSize + v*packetVertexSize
				out = append(out, [3]int16{
					int16(binary.LittleEndian.Uint16(model[o:])),
					int16(binary.LittleEndian.Uint16(model[o+2:])),
					int16(binary.LittleEndian.Uint16(model[o+4:])),
				})
			}
			p += packetHeaderSize + n*packetVertexSize + packetTailSize
		}
	}
	return out
}

// oneTriangleModel is one mesh with one three-vertex strip.
func oneTriangleModel(p0, p1, p2 [3]int16) []byte {
	const strip = packetHeaderSize + 3*packetVertexSize + packetTailSize
	b := make([]byte, omdHeaderSize+omdMeshSize+strip)
	binary.LittleEndian.PutUint32(b, uint32(len(b)))
	binary.LittleEndian.PutUint16(b[4:], 1) // numMesh
	binary.LittleEndian.PutUint16(b[6:], 1) // numTriangle
	binary.LittleEndian.PutUint32(b[8:], 3) // numVertex
	mesh := omdHeaderSize
	binary.LittleEndian.PutUint32(b[mesh+omdMeshStripsOff:], uint32(omdHeaderSize+omdMeshSize))
	binary.LittleEndian.PutUint32(b[mesh+omdMeshStripsOff+4:], 1)
	pkt := omdHeaderSize + omdMeshSize
	b[pkt+packetVertexOff] = 3
	for i, p := range [][3]int16{p0, p1, p2} {
		o := pkt + packetHeaderSize + i*packetVertexSize
		for axis := 0; axis < 3; axis++ {
			binary.LittleEndian.PutUint16(b[o+axis*2:], uint16(p[axis]))
		}
	}
	return b
}

func TestScale_MultipliesPositionsOnly(t *testing.T) {
	src := oneTriangleModel([3]int16{10, -20, 30}, [3]int16{0, 5, -5}, [3]int16{-7, 7, 0})
	out, err := Scale(src, 2.5)
	if err != nil {
		t.Fatalf("Scale: %v", err)
	}
	if len(out) != len(src) {
		t.Fatalf("Scale changed the model length: %d -> %d", len(src), len(out))
	}
	want := [][3]int16{{25, -50, 75}, {0, 13, -13}, {-18, 18, 0}}
	if got := vertexPositions(t, out); !reflect.DeepEqual(got, want) {
		t.Errorf("positions = %v, want %v", got, want)
	}
	if !reflect.DeepEqual(vertexPositions(t, src), [][3]int16{{10, -20, 30}, {0, 5, -5}, {-7, 7, 0}}) {
		t.Error("Scale mutated its input instead of copying")
	}
}

func TestScale_RejectsOverflowAndBadFactors(t *testing.T) {
	src := oneTriangleModel([3]int16{30000, 0, 0}, [3]int16{}, [3]int16{})
	if _, err := Scale(src, 4); err == nil {
		t.Error("scaling a vertex past int16 should fail loudly, not wrap")
	}
	if _, err := Scale(src, 0); err == nil {
		t.Error("a zero scale should be rejected")
	}
	if _, err := Scale(src, -2); err == nil {
		t.Error("a negative scale should be rejected")
	}
}

func TestRecolor_FlattensThePaletteAndLeavesPixelsAlone(t *testing.T) {
	src := make([]byte, 2048)
	for i := range src {
		src[i] = byte(i % 251)
	}
	pixels := append([]byte(nil), src[:len(src)-textureCLUTSize]...)

	out, err := Recolor(src, 0x20, 0xC8, 0xD8)
	if err != nil {
		t.Fatalf("Recolor: %v", err)
	}
	if len(out) != len(src) {
		t.Fatalf("Recolor changed the blob length: %d -> %d", len(src), len(out))
	}
	if !reflect.DeepEqual(out[:len(out)-textureCLUTSize], pixels) {
		t.Error("Recolor touched the pixel data, which it must not")
	}
	clut := out[len(out)-textureCLUTSize:]
	for i := 0; i < textureCLUTEntries; i++ {
		e := clut[i*4 : i*4+4]
		want := []byte{0x20, 0xC8, 0xD8, textureOpaqueAlpha}
		if !reflect.DeepEqual(e, want) {
			t.Fatalf("palette entry %d = %v, want %v", i, e, want)
		}
	}
	if src[len(src)-1] == out[len(out)-1] && src[len(src)-1] == textureOpaqueAlpha {
		t.Skip("input happened to match; nothing proven about copying")
	}
	if reflect.DeepEqual(src, out) {
		t.Error("Recolor mutated its input instead of copying")
	}
}

func TestParseColor(t *testing.T) {
	for _, in := range []string{"#20C8D8", "20c8d8", "  #20C8D8  "} {
		r, g, b, err := ParseColor(in)
		if err != nil || r != 0x20 || g != 0xC8 || b != 0xD8 {
			t.Errorf("ParseColor(%q) = %d,%d,%d,%v", in, r, g, b, err)
		}
	}
	for _, bad := range []string{"", "#fff", "zzzzzz", "20C8D8FF"} {
		if _, _, _, err := ParseColor(bad); err == nil {
			t.Errorf("ParseColor(%q) should have failed", bad)
		}
	}
}
