package containers

import (
	"encoding/binary"
	"math"
	"testing"

	"kf4randomizer/internal/kfconst"
	"kf4randomizer/internal/kfmap"
)

func newContainerMap(t *testing.T, itemClassIDs []uint16) *kfmap.Map {
	t.Helper()
	const stride = 256
	buf := make([]byte, (len(itemClassIDs)+1)*stride)
	m := &kfmap.Map{Name: "000.map", Buf: buf}
	for i, cid := range itemClassIDs {
		off := i * stride
		binary.LittleEndian.PutUint32(buf[off+kfconst.NodeObjectPositionXOffset:], math.Float32bits(float32(i)*1000))
		binary.LittleEndian.PutUint32(buf[off+kfconst.NodeObjectPositionYOffset:], math.Float32bits(0))
		m.Containers = append(m.Containers, kfmap.Container{Index: i, Offset: off, ClassID: 2, ItemClassID: cid})
	}
	return m
}

func TestRemoveEmpty(t *testing.T) {
	m := newContainerMap(t, []uint16{0, 5, 0})

	result := RemoveEmpty([]*kfmap.Map{m})

	if result.TotalContainers != 3 {
		t.Errorf("TotalContainers = %d, want 3", result.TotalContainers)
	}
	if len(result.Removed) != 2 {
		t.Fatalf("Removed = %d entries, want 2 (the two empty containers)", len(result.Removed))
	}

	yOf := func(c kfmap.Container) float32 {
		return math.Float32frombits(binary.LittleEndian.Uint32(m.Buf[c.Offset+kfconst.NodeObjectPositionYOffset:]))
	}

	if got := yOf(m.Containers[0]); got != kfconst.ContainerRemovalYOffset {
		t.Errorf("empty container 0: Position.Y = %v, want %v", got, kfconst.ContainerRemovalYOffset)
	}
	if got := yOf(m.Containers[1]); got != 0 {
		t.Errorf("non-empty container 1 should be untouched: Position.Y = %v, want 0", got)
	}
	if got := yOf(m.Containers[2]); got != kfconst.ContainerRemovalYOffset {
		t.Errorf("empty container 2: Position.Y = %v, want %v", got, kfconst.ContainerRemovalYOffset)
	}

	if n := result.RemovedPerMap["000.map"]; n != 2 {
		t.Errorf(`RemovedPerMap["000.map"] = %d, want 2`, n)
	}
}

// scenery classes (planks, signs) stay put
func TestRemoveEmpty_SkipsClassIDsThatNeverCarryAnItem(t *testing.T) {
	m := newContainerMap(t, []uint16{0, 7})
	m.Containers[0].ClassID = 91 // never carries an item in this map set

	result := RemoveEmpty([]*kfmap.Map{m})

	if len(result.Removed) != 0 {
		t.Errorf("Removed = %v, want none (classID 91 is scenery)", result.Removed)
	}
	y := math.Float32frombits(binary.LittleEndian.Uint32(m.Buf[m.Containers[0].Offset+kfconst.NodeObjectPositionYOffset:]))
	if y != 0 {
		t.Errorf("scenery object was relocated: Position.Y = %v, want 0", y)
	}
}

// map 116's button chest and the forge chests stay put
func TestRemoveEmpty_KeepsScriptDrivenContainers(t *testing.T) {
	m := newContainerMap(t, []uint16{0, 0, 5})
	m.Containers[1].ScriptRef = 1130

	result := RemoveEmpty([]*kfmap.Map{m})

	if len(result.Removed) != 1 {
		t.Fatalf("Removed = %d entries, want 1 (only the unwired empty one)", len(result.Removed))
	}
	if result.Removed[0].Index != 0 {
		t.Errorf("removed container %d, want 0", result.Removed[0].Index)
	}
	if result.KeptScripted != 1 {
		t.Errorf("KeptScripted = %d, want 1", result.KeptScripted)
	}
	y := math.Float32frombits(binary.LittleEndian.Uint32(m.Buf[m.Containers[1].Offset+kfconst.NodeObjectPositionYOffset:]))
	if y != 0 {
		t.Errorf("script-driven container was relocated: Position.Y = %v, want 0", y)
	}
}

// Moving the empty one of a stacked pair made Myu's Necklace unobtainable in
// map 032.
func TestRemoveEmpty_KeepsContainerStackedOnFilledOne(t *testing.T) {
	m := newContainerMap(t, []uint16{377, 0})
	// put the empty one exactly on top of the filled one
	binary.LittleEndian.PutUint32(m.Buf[m.Containers[1].Offset+kfconst.NodeObjectPositionXOffset:],
		math.Float32bits(0))

	result := RemoveEmpty([]*kfmap.Map{m})

	if len(result.Removed) != 0 {
		t.Fatalf("Removed = %d entries, want 0 (the empty one delivers the item)", len(result.Removed))
	}
	if result.KeptStacked != 1 {
		t.Errorf("KeptStacked = %d, want 1", result.KeptStacked)
	}
}

func TestRemoveEmpty_KeepsContainerWithEventID(t *testing.T) {
	// a filled one of the same class, so the scenery check doesn't skip it
	m := newContainerMap(t, []uint16{5, 0})
	m.Containers[1].EventID = 151

	result := RemoveEmpty([]*kfmap.Map{m})

	if len(result.Removed) != 0 {
		t.Fatalf("Removed = %d entries, want 0 (a script can address it)", len(result.Removed))
	}
	if result.KeptAddressable != 1 {
		t.Errorf("KeptAddressable = %d, want 1", result.KeptAddressable)
	}
}
