package randomize

import (
	"encoding/binary"
	"math/rand/v2"
	"reflect"
	"sort"
	"testing"

	"kf4randomizer/internal/kfconst"
	"kf4randomizer/internal/kfmap"
	"kf4randomizer/internal/logic"
)

func newTestRNG(seed uint64) *rand.Rand {
	return rand.New(rand.NewPCG(seed, seed))
}

// newTestMap places the items, plus one class 2 container per container item.
func newTestMap(name string, itemClassIDs []uint16, containerItemClassIDs []uint16) *kfmap.Map {
	const stride = 256
	total := len(itemClassIDs) + len(containerItemClassIDs)
	buf := make([]byte, (total+1)*stride)

	m := &kfmap.Map{Name: name, Buf: buf}
	off := 0
	for i, cid := range itemClassIDs {
		m.Items = append(m.Items, kfmap.Item{
			Index: i, Offset: off, ClassID: cid,
			// 0 would make it unspawnable and empty the pool
			RespawnChance: 100,
		})
		off += stride
	}
	for i, cid := range containerItemClassIDs {
		m.Containers = append(m.Containers, kfmap.Container{Index: i, Offset: off, ClassID: 2, ItemClassID: cid})
		off += stride
	}
	return m
}

func classIDMultiset(ids []uint16) map[uint16]int {
	out := make(map[uint16]int)
	for _, id := range ids {
		out[id]++
	}
	return out
}

func TestRunNoLogic_EligibilityFiltering(t *testing.T) {
	m := newTestMap("900.map", []uint16{0, 9999, 10, 20}, nil) // 0=empty, 9999=out of range, 10/20=valid
	result := RunNoLogic([]*kfmap.Map{m}, newTestRNG(1), logic.Pool{Ground: true, Containers: false, Drops: true})

	if result.EligibleSlots != 2 {
		t.Fatalf("EligibleSlots = %d, want 2 (only classID 10 and 20 are eligible)", result.EligibleSlots)
	}
	if m.Items[0].ClassID != 0 {
		t.Errorf("empty slot (classID 0) should stay untouched, got %d", m.Items[0].ClassID)
	}
	if m.Items[1].ClassID != 9999 {
		t.Errorf("out-of-range classID should stay untouched, got %d", m.Items[1].ClassID)
	}
	got := classIDMultiset([]uint16{m.Items[2].ClassID, m.Items[3].ClassID})
	want := classIDMultiset([]uint16{10, 20})
	if len(got) != len(want) || got[10] != want[10] || got[20] != want[20] {
		t.Errorf("eligible items' classIDs became %v, want the same multiset %v", got, want)
	}
}

func TestRunNoLogic_ContainersOnlyIncludedWhenRequested(t *testing.T) {
	newMap := func() *kfmap.Map { return newTestMap("900.map", []uint16{10, 20}, []uint16{99}) }

	excluded := newMap()
	result := RunNoLogic([]*kfmap.Map{excluded}, newTestRNG(1), logic.Pool{Ground: true, Containers: false, Drops: true})
	if result.EligibleSlots != 2 {
		t.Fatalf("includeContainers=false: EligibleSlots = %d, want 2", result.EligibleSlots)
	}
	if excluded.Containers[0].ItemClassID != 99 {
		t.Errorf("includeContainers=false: container should be untouched, got classID %d", excluded.Containers[0].ItemClassID)
	}

	included := newMap()
	result = RunNoLogic([]*kfmap.Map{included}, newTestRNG(1), logic.Pool{Ground: true, Containers: true, Drops: true})
	if result.EligibleSlots != 3 {
		t.Fatalf("includeContainers=true: EligibleSlots = %d, want 3", result.EligibleSlots)
	}
	// Containers is a copy, read the result from Changes
	containerFinal := included.Containers[0].ItemClassID // original value, 99
	for i := range result.Changes {
		c := &result.Changes[i]
		if c.FromClassID == c.ToClassID {
			t.Errorf("Changes should never contain a no-op entry, got %+v", c)
		}
		if c.Kind == ContainerItem {
			containerFinal = c.ToClassID
		}
	}
	got := classIDMultiset([]uint16{included.Items[0].ClassID, included.Items[1].ClassID, containerFinal})
	want := classIDMultiset([]uint16{10, 20, 99})
	for id, n := range want {
		if got[id] != n {
			t.Errorf("includeContainers=true: classID multiset after shuffle = %v, want %v", got, want)
		}
	}
	if got := included.Containers[0].ClassID; got != 2 {
		t.Errorf("container ClassID (the object type, never shuffled) changed to %d, want 2", got)
	}
}

func TestRunNoLogic_NoOpSlotsAreNotReportedAsChanges(t *testing.T) {
	// A single-element pool always shuffles to itself.
	m := newTestMap("900.map", []uint16{10}, nil)
	result := RunNoLogic([]*kfmap.Map{m}, newTestRNG(1), logic.Pool{Ground: true, Containers: false, Drops: true})

	if result.EligibleSlots != 1 {
		t.Fatalf("EligibleSlots = %d, want 1", result.EligibleSlots)
	}
	if len(result.Changes) != 0 {
		t.Errorf("Changes = %v, want empty (the only slot didn't move)", result.Changes)
	}
	if n := result.ChangedPerMap["900.map"]; n != 0 {
		t.Errorf("ChangedPerMap[\"000.map\"] = %d, want 0", n)
	}
	if m.Items[0].ClassID != 10 {
		t.Errorf("item's classID should still be 10, got %d", m.Items[0].ClassID)
	}
}

func TestRunNoLogic_ExcludedClassIDsNeverMove(t *testing.T) {
	// 196 = Moonlight Sword, kfconst.ExcludedFromRandomizationClassIDs.
	for seed := uint64(0); seed < 20; seed++ {
		m := newTestMap("901.map", []uint16{196, 10, 20, 30}, []uint16{196, 40})
		result := RunNoLogic([]*kfmap.Map{m}, newTestRNG(seed), logic.Pool{Ground: true, Containers: true, Drops: true})

		if result.EligibleSlots != 4 {
			t.Fatalf("seed %d: EligibleSlots = %d, want 4 (both classID-196 slots excluded)", seed, result.EligibleSlots)
		}
		if m.Items[0].ClassID != 196 {
			t.Errorf("seed %d: excluded item slot moved to classID %d, want 196", seed, m.Items[0].ClassID)
		}
		if m.Containers[0].ItemClassID != 196 {
			t.Errorf("seed %d: excluded container slot moved to classID %d, want 196", seed, m.Containers[0].ItemClassID)
		}
		for i := 1; i < len(m.Items); i++ {
			if m.Items[i].ClassID == 196 {
				t.Errorf("seed %d: classID 196 shuffled into item slot %d, should never appear outside its own slot", seed, i)
			}
		}
		if m.Containers[1].ItemClassID == 196 {
			t.Errorf("seed %d: classID 196 shuffled into the other container slot, should never appear outside its own slot", seed)
		}
		for _, c := range result.Changes {
			if c.FromClassID == 196 || c.ToClassID == 196 {
				t.Errorf("seed %d: Changes should never reference the excluded classID, got %+v", seed, c)
			}
		}
	}
}

func TestRunNoLogic_SameSeedIsDeterministic(t *testing.T) {
	build := func() *kfmap.Map {
		return newTestMap("900.map", []uint16{10, 20, 30, 40, 50}, nil)
	}

	m1 := build()
	r1 := RunNoLogic([]*kfmap.Map{m1}, newTestRNG(42), logic.Pool{Ground: true, Containers: false, Drops: true})

	m2 := build()
	r2 := RunNoLogic([]*kfmap.Map{m2}, newTestRNG(42), logic.Pool{Ground: true, Containers: false, Drops: true})

	if len(r1.Changes) != len(r2.Changes) {
		t.Fatalf("got %d and %d changes for the same seed", len(r1.Changes), len(r2.Changes))
	}
	sort.Slice(r1.Changes, func(i, j int) bool { return r1.Changes[i].Index < r1.Changes[j].Index })
	sort.Slice(r2.Changes, func(i, j int) bool { return r2.Changes[i].Index < r2.Changes[j].Index })
	for i := range r1.Changes {
		if r1.Changes[i].ToClassID != r2.Changes[i].ToClassID {
			t.Errorf("slot %d: got ToClassID %d and %d for the same seed", i, r1.Changes[i].ToClassID, r2.Changes[i].ToClassID)
		}
	}
}

// newTestMapWithCreatures adds monster drops after the items and containers.
func newTestMapWithCreatures(name string, itemClassIDs, creatureDropClassIDs []uint16) *kfmap.Map {
	const stride = 256
	total := len(itemClassIDs) + len(creatureDropClassIDs)
	buf := make([]byte, (total+1)*stride)

	m := &kfmap.Map{Name: name, Buf: buf}
	off := 0
	for i, cid := range itemClassIDs {
		m.Items = append(m.Items, kfmap.Item{
			Index: i, Offset: off, ClassID: cid,
			// 0 would make it unspawnable and empty the pool
			RespawnChance: 100,
		})
		off += stride
	}
	for i, cid := range creatureDropClassIDs {
		m.Creatures = append(m.Creatures, kfmap.Creature{Index: i, Offset: off, DropClassID: cid})
		off += stride
	}
	return m
}

func TestRunNoLogic_CreatureDropsAlwaysShuffled(t *testing.T) {
	m := newTestMapWithCreatures("900.map", []uint16{10, 20}, []uint16{30, 40})

	// includeContainers is false: drops must still be in the pool.
	result := RunNoLogic([]*kfmap.Map{m}, newTestRNG(7), logic.Pool{Ground: true, Containers: false, Drops: true})

	if result.EligibleSlots != 4 {
		t.Fatalf("EligibleSlots = %d, want 4 (2 ground items + 2 creature drops)", result.EligibleSlots)
	}

	got := []uint16{m.Items[0].ClassID, m.Items[1].ClassID}
	for _, c := range m.Creatures {
		got = append(got, binary.LittleEndian.Uint16(m.Buf[c.Offset+kfconst.NodeEntityDropClassIDOffset:]))
	}
	want := classIDMultiset([]uint16{10, 20, 30, 40})
	if diff := classIDMultiset(got); !reflect.DeepEqual(diff, want) {
		t.Errorf("shuffled pool = %v, want the same multiset %v", diff, want)
	}
}

func TestRunNoLogic_CreatureDropExclusions(t *testing.T) {
	// 9999 is outside item_nam.prm, 196 is on the exclusion list.
	m := newTestMapWithCreatures("900.map", nil, []uint16{9999, 196, 30})
	result := RunNoLogic([]*kfmap.Map{m}, newTestRNG(1), logic.Pool{Ground: true, Containers: false, Drops: true})

	if result.EligibleSlots != 1 {
		t.Fatalf("EligibleSlots = %d, want 1 (only classID 30 is eligible)", result.EligibleSlots)
	}
	for _, c := range m.Creatures[:2] {
		if got := binary.LittleEndian.Uint16(m.Buf[c.Offset+kfconst.NodeEntityDropClassIDOffset:]); got != 0 {
			t.Errorf("ineligible drop %d was written to Buf as %d, want untouched", c.DropClassID, got)
		}
	}
}

// newSlotMap has one ground item whose spawnability the test picks.
func newSlotMap(name string, classID uint16, respawn, event byte) *kfmap.Map {
	buf := make([]byte, 512)
	return &kfmap.Map{
		Name: name,
		Buf:  buf,
		Items: []kfmap.Item{{
			Index: 0, Offset: 0, ClassID: classID,
			RespawnChance: respawn, EventID: event,
		}},
	}
}

func TestRunNoLogic_SkipsSlotsNothingCanEverSpawn(t *testing.T) {
	// Three real slots are like this: a Binding Crown under map 015's floor,
	// a Widda Key in 023, a Long Bow in 031.
	dead := newSlotMap("015.map", 10, 0, 0)
	if got := RunNoLogic([]*kfmap.Map{dead}, newTestRNG(1), logic.Pool{Ground: true, Containers: false, Drops: true}).EligibleSlots; got != 0 {
		t.Errorf("EligibleSlots = %d, want 0; an item put here would be lost", got)
	}
	if dead.Items[0].ClassID != 10 {
		t.Errorf("an excluded slot should keep its vanilla item, got %d", dead.Items[0].ClassID)
	}
}

func TestRunNoLogic_KeepsEventRevealedAndNaturalSlots(t *testing.T) {
	for _, c := range []struct {
		name           string
		respawn, event byte
	}{
		{"natural", 100, 0},
		{"revealed by an event", 0, 201},
	} {
		m := newSlotMap("900.map", 10, c.respawn, c.event)
		if got := RunNoLogic([]*kfmap.Map{m}, newTestRNG(1), logic.Pool{Ground: true, Containers: false, Drops: true}).EligibleSlots; got != 1 {
			t.Errorf("%s: EligibleSlots = %d, want 1", c.name, got)
		}
	}
}

func TestUnspawnableSlotFollowsItsTwinChest(t *testing.T) {
	// Map 023's Widda Key: a chest and an unspawnable slot both stocked it.
	m := newSlotMap("023.map", 52, 0, 0)
	m.Containers = []kfmap.Container{{
		Index: 71, Offset: 256, ClassID: 98,
		ItemClassID: 34, VanillaItemClassID: 52,
	}}
	// the chest is read back from Buf, not the struct
	binary.LittleEndian.PutUint16(
		m.Buf[256+kfconst.NodeObjectItemClassIDOffset:], 34)

	changes := followTwinChest([]*kfmap.Map{m})
	if len(changes) != 1 {
		t.Fatalf("changes = %d, want 1", len(changes))
	}
	if m.Items[0].ClassID != 34 {
		t.Errorf("slot holds %d, want 34, should match its twin chest", m.Items[0].ClassID)
	}
}

func TestUnspawnableSlotWithNoTwinChestIsLeftAlone(t *testing.T) {
	// Map 015's Binding Crown has no chest holding one, so it has no twin.
	m := newSlotMap("015.map", 107, 0, 0)
	m.Containers = []kfmap.Container{{
		Index: 3, Offset: 256, ClassID: 2,
		ItemClassID: 34, VanillaItemClassID: 6,
	}}

	if changes := followTwinChest([]*kfmap.Map{m}); len(changes) != 0 {
		t.Errorf("changes = %v, want none", changes)
	}
	if m.Items[0].ClassID != 107 {
		t.Errorf("slot holds %d, want its vanilla 107", m.Items[0].ClassID)
	}
}

func TestRunNoLogicSkipsSlotsTheTablesRuleOut(t *testing.T) {
	// 000.map ground slot 3 is a shop item that never spawns, and slot 6 is
	// half of the Rusty Key group. Neither may take a shuffled item on its own.
	m := newTestMap("000.map", []uint16{10, 20, 30, 40, 50, 60, 70, 80}, nil)
	before := m.Items[3].ClassID
	RunNoLogic([]*kfmap.Map{m}, newTestRNG(1), logic.Pool{Ground: true, Containers: false, Drops: true})

	if m.Items[3].ClassID != before {
		t.Errorf("slot 3 should never be written to, changed from %d to %d", before, m.Items[3].ClassID)
	}
}

func TestRunNoLogicGivesSharedSlotsOneItem(t *testing.T) {
	// 017 and 018 are the forge in two states; their slot 0 and slot 1 are the
	// same Crusader Gloves pickup, so both have to end up holding one item.
	a := newTestMap("017.map", []uint16{10, 20, 30, 40, 50, 60, 70, 80}, nil)
	b := newTestMap("018.map", []uint16{10, 20, 30, 40, 50, 60, 70, 80}, nil)
	RunNoLogic([]*kfmap.Map{a, b}, newTestRNG(3), logic.Pool{Ground: true, Containers: false, Drops: true})

	if a.Items[0].ClassID != b.Items[1].ClassID {
		t.Errorf("017 slot 0 = %d but 018 slot 1 = %d, want the same item",
			a.Items[0].ClassID, b.Items[1].ClassID)
	}
}
