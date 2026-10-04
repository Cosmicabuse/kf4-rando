// Package randomize shuffles ground items, containers and monster drops.
package randomize

import (
	"encoding/binary"
	"fmt"
	"math/rand/v2"
	"sort"

	"kf4randomizer/internal/features/shops"
	"kf4randomizer/internal/kfconst"
	"kf4randomizer/internal/kfmap"
	"kf4randomizer/internal/logic"
)

// NewRand salts the seed so each pass gets its own stream.
func NewRand(seed uint64) *rand.Rand {
	return rand.New(rand.NewPCG(seed, seed^0x9e3779b97f4a7c15))
}

type Kind int

const (
	GroundItem Kind = iota
	ContainerItem
	MapDrop
	// PrmDrop lives in chr.prm rather than a map, keyed by creature class.
	PrmDrop
	// StartItem is the starting inventory's Idol of Sorrow.
	StartItem
)

// Change records one shuffled slot's before/after classID, for the spoiler log.
type Change struct {
	MapName     string
	Kind        Kind
	Index       int // Item.Index or Container.Index, per Kind
	FromClassID uint16
	ToClassID   uint16
}

type Result struct {
	TotalMaps     int
	EligibleSlots int
	ChangedPerMap map[string]int // map name -> number of slots changed
	Changes       []Change
	StartItem     uint16 // what replaces the Idol of Sorrow at the start, 0 if it stays
}

// Summary counts changes per map. mapNames may be nil.
func (r Result) Summary(mapNames kfmap.MapNames) string {
	s := fmt.Sprintf("Loaded %d maps, %d eligible item slots.\n", r.TotalMaps, r.EligibleSlots)
	names := make([]string, 0, len(r.ChangedPerMap))
	for name := range r.ChangedPerMap {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		s += fmt.Sprintf("%s: %d slots changed\n", mapNames.Header(name), r.ChangedPerMap[name])
	}
	return s
}

type slot struct {
	mapIdx int
	kind   Kind
	idx    int // index into m.Items, m.Containers or m.Creatures, per kind
}

// RunNoLogic shuffles classIDs across every eligible slot, with no
// reachability check, so a seed can be unwinnable.
func RunNoLogic(maps []*kfmap.Map, rng *rand.Rand, kinds logic.Pool) Result {
	sort.Slice(maps, func(i, j int) bool { return maps[i].Name < maps[j].Name })
	if kinds.Drops {
		moveForgeDwarf(maps)
	}
	rules := logic.Rules()
	// A kept slot keeps every slot sharing its pickup too.
	pinned := map[string]bool{}
	for where, r := range rules {
		if !kinds.Has(where.Slot) && r.Dupe != "" {
			pinned[r.Dupe] = true
		}
	}
	shares := func(name string, kind logic.Slot, index int) bool {
		return pinned[rules[logic.Where{Map: name, Slot: kind, Index: index}].Dupe]
	}

	var slots []slot
	var originalClassIDs []uint16
	for mi, m := range maps {
		for _, item := range m.Items {
			if !kinds.Ground {
				break
			}
			if item.ClassID == 0 || !kfconst.IsKnownClassID(item.ClassID) || kfconst.IsExcludedFromRandomization(item.ClassID) {
				continue
			}
			// An item shuffled here would be unreachable.
			if item.Unspawnable() || !writable(rules, m.Name, logic.Ground, item.Index) || shares(m.Name, logic.Ground, item.Index) {
				continue
			}
			slots = append(slots, slot{mapIdx: mi, kind: GroundItem, idx: item.Index})
			originalClassIDs = append(originalClassIDs, item.ClassID)
		}
		if kinds.Containers {
			for ci, c := range m.Containers {
				if c.ItemClassID == 0 || !kfconst.IsKnownClassID(c.ItemClassID) || kfconst.IsExcludedFromRandomization(c.ItemClassID) {
					continue
				}
				if !writable(rules, m.Name, logic.Container, c.Index) || shares(m.Name, logic.Container, c.Index) {
					continue
				}
				slots = append(slots, slot{mapIdx: mi, kind: ContainerItem, idx: ci})
				originalClassIDs = append(originalClassIDs, c.ItemClassID)
			}
		}
		// No two drops share a creature, so each shuffles independently.
		for ri, c := range m.Creatures {
			if !kinds.Drops {
				break
			}
			if !kfconst.IsKnownClassID(c.DropClassID) || kfconst.IsExcludedFromRandomization(c.DropClassID) {
				continue
			}
			if !writable(rules, m.Name, logic.MapDrop, c.Index) {
				continue
			}
			slots = append(slots, slot{mapIdx: mi, kind: MapDrop, idx: ri})
			originalClassIDs = append(originalClassIDs, c.DropClassID)
		}
	}

	// Slots the game shares draw one item between them, so only the first of
	// each group goes into the pool and the rest copy it afterwards.
	twin := twins(maps, slots)
	if kinds.Idol {
		slots = append(slots, slot{kind: StartItem})
		originalClassIDs = append(originalClassIDs, logic.IdolOfSorrow)
		twin = append(twin, -1)
	}
	var own []int
	var pool []uint16
	for i := range slots {
		if twin[i] < 0 {
			own = append(own, i)
			pool = append(pool, originalClassIDs[i])
		}
	}

	rng.Shuffle(len(pool), func(i, j int) { pool[i], pool[j] = pool[j], pool[i] })

	shuffled := make([]uint16, len(slots))
	for n, i := range own {
		shuffled[i] = pool[n]
	}
	for i, t := range twin {
		if t >= 0 {
			shuffled[i] = shuffled[t]
		}
	}

	changedPerMap := make(map[string]int)
	var changes []Change
	var startItem uint16
	for i, s := range slots {
		if s.kind == StartItem {
			startItem = shuffled[i]
			continue
		}
		m := maps[s.mapIdx]
		from := originalClassIDs[i]
		to := shuffled[i]
		var index int
		switch s.kind {
		case GroundItem:
			m.SetItemClassID(s.idx, to)
			index = m.Items[s.idx].Index
		case ContainerItem:
			c := m.Containers[s.idx]
			m.SetContainerItemClassID(c, to)
			index = c.Index
		case MapDrop:
			c := m.Creatures[s.idx]
			m.SetCreatureDropClassID(c, to)
			index = c.Index
		}
		if from == to {
			continue
		}
		changedPerMap[m.Name]++
		changes = append(changes, Change{MapName: m.Name, Kind: s.kind, Index: index, FromClassID: from, ToClassID: to})
	}

	changes = append(changes, followTwinChest(maps)...)

	sortChanges(changes)

	return Result{
		TotalMaps:     len(maps),
		EligibleSlots: len(slots),
		ChangedPerMap: changedPerMap,
		Changes:       changes,
		StartItem:     startItem,
	}
}

// An unspawnable slot whose vanilla item also sat in a chest in the same map
// follows that chest, so both hand over the same thing. Only map 023 matches.
func followTwinChest(maps []*kfmap.Map) []Change {
	var changes []Change
	for _, m := range maps {
		for i, item := range m.Items {
			if !item.Unspawnable() {
				continue
			}
			for _, c := range m.Containers {
				if c.VanillaItemClassID != item.ClassID {
					continue
				}
				to := m.ContainerItemClassID(c)
				if to == item.ClassID {
					break
				}
				from := item.ClassID
				m.SetItemClassID(i, to)
				changes = append(changes, Change{
					MapName: m.Name, Kind: GroundItem, Index: item.Index,
					FromClassID: from, ToClassID: to,
				})
				break
			}
		}
	}
	return changes
}

// writable reports whether a slot can take a shuffled item. Some slots never
// spawn in game, and some have to keep whatever they already hold.
func writable(rules map[logic.Where]logic.Rule, mapName string, kind logic.Slot, index int) bool {
	r, ok := rules[logic.Where{Map: mapName, Slot: kind, Index: index}]
	return !ok || r.Fill == logic.Shuffle || r.Fill == logic.Copy
}

// twins maps each slot to the slot it must copy, or -1 when it stands alone,
// for pickups the game shares between maps like the forge's.
func twins(maps []*kfmap.Map, slots []slot) []int {
	rules := logic.Rules()
	kinds := map[Kind]logic.Slot{
		GroundItem:    logic.Ground,
		ContainerItem: logic.Container,
		MapDrop:       logic.MapDrop,
	}

	out := make([]int, len(slots))
	first := map[string]int{}
	for i, s := range slots {
		out[i] = -1
		m := maps[s.mapIdx]
		r, ok := rules[logic.Where{Map: m.Name, Slot: kinds[s.kind], Index: slotIndex(m, s)}]
		if !ok || r.Dupe == "" {
			continue
		}
		if j, seen := first[r.Dupe]; seen {
			out[i] = j
		} else {
			first[r.Dupe] = i
		}
	}
	return out
}

// slotIndex is the position in the map's own array. Containers and Creatures
// are filtered views, so their slice position is not the same thing.
func slotIndex(m *kfmap.Map, s slot) int {
	switch s.kind {
	case GroundItem:
		return m.Items[s.idx].Index
	case ContainerItem:
		return m.Containers[s.idx].Index
	case MapDrop:
		return m.Creatures[s.idx].Index
	}
	return s.idx
}

func sortChanges(changes []Change) {
	sort.SliceStable(changes, func(i, j int) bool {
		if changes[i].MapName != changes[j].MapName {
			return changes[i].MapName < changes[j].MapName
		}
		if changes[i].Kind != changes[j].Kind {
			return changes[i].Kind < changes[j].Kind
		}
		return changes[i].Index < changes[j].Index
	})
}

// RunLogic places every item somewhere reachable without already holding it.
func RunLogic(maps []*kfmap.Map, chr, stock []byte, rng *rand.Rand, pool logic.Pool) (Result, []logic.Placement, error) {
	clearCrownPedestals(maps)

	// The forge boss only moves so his drop can be shuffled.
	if pool.Drops {
		moveForgeDwarf(maps)
	}
	if pool.Shops && stock == nil {
		return Result{}, nil, fmt.Errorf("shops are in the pool but no shop stock was given")
	}
	w, start := logic.ForSeed(pool)

	placements, err := w.Fill(start, rng)
	if err != nil {
		return Result{}, nil, err
	}
	if res := w.Apply(placements).Solve(start); len(res.Unreached) > 0 {
		return Result{}, nil, fmt.Errorf("fill produced %d unreachable checks, first %s",
			len(res.Unreached), res.Unreached[0].ID())
	}

	byName := make(map[string]*kfmap.Map, len(maps))
	for _, m := range maps {
		byName[m.Name] = m
	}

	out := Result{TotalMaps: len(maps), ChangedPerMap: map[string]int{}}
	var missed []string
	if pool.Shops {
		shops.Clear(stock)
	}
	for _, p := range placements {
		// The pedestal crowns land on map 020's ground even when ground items
		// stay vanilla, or they would be gone.
		crown := p.Check.Fill == logic.Fixed && p.Check.Slot == logic.Ground && p.Item == logic.BindingCrown
		if p.Check.Fill != logic.Shuffle && p.Check.Fill != logic.Copy && p.Check.Fill != logic.Filler && !crown {
			continue
		}
		if p.Check.Slot == logic.Shop {
			shops.Add(stock, logic.ShopNumber(p.Check), p.Item)
			continue
		}
		if p.Check.Slot == logic.Start {
			out.StartItem = p.Item
			continue
		}
		if p.Check.Slot == logic.PrmDrop {
			ch, err := writeToMonsterTable(chr, p)
			if err != nil {
				return out, nil, err
			}
			out.EligibleSlots++
			if ch.FromClassID != ch.ToClassID {
				out.Changes = append(out.Changes, ch)
				out.ChangedPerMap[ch.MapName]++
			}
			continue
		}
		m := byName[logic.MapFile(p.Check.Region)]
		if m == nil {
			missed = append(missed, p.Check.ID())
			continue
		}
		out.EligibleSlots++
		ch, ok := write(m, p)
		if !ok {
			missed = append(missed, p.Check.ID())
			continue
		}
		if ch.FromClassID != ch.ToClassID {
			out.Changes = append(out.Changes, ch)
			out.ChangedPerMap[m.Name]++
		}
	}
	if len(missed) > 0 {
		return out, nil, fmt.Errorf("%d placements had no slot to write to, first %s",
			len(missed), missed[0])
	}

	sortChanges(out.Changes)
	return out, placements, nil
}

func write(m *kfmap.Map, p logic.Placement) (Change, bool) {
	c := p.Check
	switch c.Slot {
	case logic.Ground:
		for _, it := range m.Items {
			if it.Index != c.Index {
				continue
			}
			m.SetItemClassID(it.Index, p.Item)
			return Change{m.Name, GroundItem, it.Index, it.ClassID, p.Item}, true
		}
	case logic.Container:
		for _, ct := range m.Containers {
			if ct.Index != c.Index {
				continue
			}
			from := ct.ItemClassID
			m.SetContainerItemClassID(ct, p.Item)
			return Change{m.Name, ContainerItem, ct.Index, from, p.Item}, true
		}
	case logic.MapDrop:
		for _, cr := range m.Creatures {
			if cr.Index != c.Index {
				continue
			}
			from := cr.DropClassID
			m.SetCreatureDropClassID(cr, p.Item)
			return Change{m.Name, MapDrop, cr.Index, from, p.Item}, true
		}
	}
	return Change{}, false
}

// writeToMonsterTable patches a monster type's drop in chr.prm, the only home
// of the Fire Lord's Fire Mail.
func writeToMonsterTable(chr []byte, p logic.Placement) (Change, error) {
	if len(chr) == 0 {
		return Change{}, fmt.Errorf("%s needs chr.prm and it was not loaded", p.Check.ID())
	}
	from, err := kfmap.ChrDrop(chr, p.Check.Index)
	if err != nil {
		return Change{}, err
	}
	if err := kfmap.SetChrDrop(chr, p.Check.Index, p.Item); err != nil {
		return Change{}, err
	}
	return Change{
		MapName:     "chr.prm",
		Kind:        PrmDrop,
		Index:       p.Check.Index,
		FromClassID: from,
		ToClassID:   p.Item,
	}, nil
}

// Pedestals that start with a Binding Crown, emptied so the crowns join the
// pool. 012's object 22 stays, its golem guards the way on.
var crownPedestals = []struct {
	mapName string
	index   int
}{
	{"015.map", 6},
	{"110.map", 2},
}

// Forge is duplicated in 3 maps (17/18/19), and 19 doesn't have the icon of
// ice boss. To avoid duplicates, the loot moves from 18's boss to 19.
const (
	forgeLitMap, forgeDwarfEntity  = "018.map", 0
	forgeIconMap, forgeDwarfSlotIn = "019.map", logic.ForgeBossSlot
)

func moveForgeDwarf(maps []*kfmap.Map) {
	var lit, icon *kfmap.Map
	for _, m := range maps {
		switch m.Name {
		case forgeLitMap:
			lit = m
		case forgeIconMap:
			icon = m
		}
	}
	if lit == nil || icon == nil {
		return
	}

	copy(icon.Entity(forgeDwarfSlotIn), lit.Entity(forgeDwarfEntity))
	binary.LittleEndian.PutUint16(
		lit.Entity(forgeDwarfEntity)[kfconst.NodeEntityDropClassIDOffset:],
		kfconst.NodeEntityNoDropSentinel)

	lit.RefreshCreatures()
	icon.RefreshCreatures()
}

func clearCrownPedestals(maps []*kfmap.Map) {
	byName := make(map[string]*kfmap.Map, len(maps))
	for _, m := range maps {
		byName[m.Name] = m
	}
	for _, p := range crownPedestals {
		m := byName[p.mapName]
		if m == nil || len(m.Containers) == 0 {
			continue
		}
		// Containers is a filtered view, so the array base comes off a known
		// entry rather than the header.
		base := m.Containers[0].Offset - m.Containers[0].Index*kfconst.NodeObjectEntrySize
		off := base + p.index*kfconst.NodeObjectEntrySize
		if off < 0 || off+kfconst.NodeObjectEntrySize > len(m.Buf) {
			continue
		}
		if m.Buf[off+kfconst.NodeObjectPrefilledOffset] == 1 {
			m.Buf[off+kfconst.NodeObjectPrefilledOffset] = 0
		}
	}
}
