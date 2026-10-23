package tracker

import (
	"fmt"
	"regexp"
	"strconv"
	"time"

	"kf4randomizer/internal/features/shops"
	"kf4randomizer/internal/features/startingconfig"
	"kf4randomizer/internal/kfconst"
	"kf4randomizer/internal/kfmap"
	"kf4randomizer/internal/logic"
)

// Count is how many of item id a build holds: map pickups, shop stock and
// the starting inventory, plus monster drops in chr.prm. Slots that only copy
// or never spawn are left out, and a Dupe group is one pickup.
func Count(maps []*kfmap.Map, stock, chr, ini []byte, id uint16) int {
	rules := logic.Rules()
	seen := map[string]bool{}
	n := 0
	count := func(r logic.Rule) {
		if r.Fill == logic.Copy || r.Fill == logic.Skip {
			return
		}
		if r.Dupe != "" {
			if seen[r.Dupe] {
				return
			}
			seen[r.Dupe] = true
		}
		n++
	}
	add := func(m *kfmap.Map, slot logic.Slot, index int, item uint16) {
		if item == id {
			count(rules[logic.Where{Map: m.Name, Slot: slot, Index: index}])
		}
	}
	for w, r := range rules {
		if w.Slot != logic.PrmDrop {
			continue
		}
		if item, err := kfmap.ChrDrop(chr, w.Index); err == nil && item == id {
			count(r)
		}
	}
	for _, m := range maps {
		for _, it := range m.Items {
			add(m, logic.Ground, it.Index, it.ClassID)
		}
		for _, c := range m.Containers {
			add(m, logic.Container, c.Index, c.ItemClassID)
		}
		for _, c := range m.Creatures {
			add(m, logic.MapDrop, c.Index, c.DropClassID)
		}
	}
	if len(stock) == kfconst.ShopCount*kfconst.ShopStockSize {
		for shop := range kfconst.ShopCount {
			n += shops.Quantity(stock, shop, id)
		}
	}
	return n + startingconfig.ItemQuantity(ini, int(id))
}

// New builds the tracker file for a fresh build.
func New(seed uint64, maps []*kfmap.Map, stock, chr, ini []byte, now time.Time) *File {
	f := &File{
		Seed:      strconv.FormatUint(seed, 10),
		Generated: now,
		// no crown count: pedestal crowns are flags, not item slots, and there are always 3
		Max:   map[string]int{"crystal_vial": Count(maps, stock, chr, ini, 14)},
		Start: map[string]int{},
	}
	for _, t := range Tiles {
		f.Start[t.ID] = min(f.startValue(t, ini), f.Cap(t.ID))
	}
	f.State = make(map[string]int, len(f.Start))
	for id, v := range f.Start {
		f.State[id] = v
	}
	return f
}

func (f *File) startValue(t Tile, ini []byte) int {
	held := func(item uint16) int { return startingconfig.ItemQuantity(ini, int(item)) }
	switch t.ID {
	case "green_stone", "blue_stone":
		return 1
	case "binding_crown":
		return 1 + max(held(107), 0)
	case "sword":
		if held(196) > 0 {
			return 2
		}
		if held(195) > 0 {
			return 1
		}
		return 0
	}
	if t.Section == "Magic" {
		return spells(ini, t.ID)
	}
	if t.Kind == Counter {
		return max(held(t.Items[0]), 0)
	}
	for _, item := range t.Items {
		if held(item) > 0 {
			return 1
		}
	}
	return 0
}

// spells counts the known spells of one element, keys <id>_1 to <id>_5.
func spells(ini []byte, id string) int {
	re := regexp.MustCompile(fmt.Sprintf(`(?m)^%s_[1-5]=(\d+)`, id))
	n := 0
	for _, m := range re.FindAllSubmatch(ini, -1) {
		if v, _ := strconv.Atoi(string(m[1])); v > 0 {
			n++
		}
	}
	return n
}
