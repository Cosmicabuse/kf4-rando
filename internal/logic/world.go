package logic

import (
	"fmt"
	"strings"
)

// Default is the unmodified game: every crossing, every gate, and every item
// where it sits in vanilla, plus what a new game starts with.
func Default() (World, Inventory) {
	w := World{Start: startRegion, Caps: Capabilities{}}

	w.Caps[CapFire] = fireItems
	w.Caps[CapProjectile] = projectileItems
	w.Caps[CapBreak] = breakItems

	gated := map[[2]string]bool{}
	for _, g := range gates {
		gated[pair(g.from, g.to)] = true
		w.Edges = append(w.Edges, Edge{From: g.from, To: g.to, Need: need(g.req)})
		if !g.oneWay {
			w.Edges = append(w.Edges, Edge{From: g.to, To: g.from, Need: need(g.req)})
		}
	}

	cut := map[[2]string]bool{}
	for _, b := range blocked {
		cut[pair(b[0], b[1])] = true
	}

	for _, c := range crossings {
		p := pair(c.from, c.to)
		if gated[p] || cut[p] {
			continue
		}
		w.Edges = append(w.Edges, Edge{From: c.from, To: c.to})
		if !c.oneWay {
			w.Edges = append(w.Edges, Edge{From: c.to, To: c.from})
		}
	}

	// A Skip slot never spawns, so counting its item would hand out something
	// that can never be picked up.
	for _, p := range placements {
		if p.fill == Skip {
			continue
		}
		w.Checks = append(w.Checks, Check{
			Region: p.region, Slot: p.slot, Index: p.index,
			Item: p.item, Need: need(p.req), Fill: p.fill, Dupe: p.dupe,
			Also: errands[Where{Map: p.region, Slot: p.slot, Index: p.index}],
		})
	}

	inv := Inventory{}
	for _, id := range startItems {
		inv[id]++
	}
	return w, inv
}

// need looks up a requirement string. Capitalisation varies between rows, and
// anything with no entry at all is treated as free.
func need(req string) Req {
	return requirements[strings.ToLower(strings.TrimSpace(req))]
}

// ForgeBossSlot is the entity index in 019 the randomizer copies 018's boss
// into. The map edit and Randomized below have to name the same slot.
const ForgeBossSlot = 3

// Randomized is Default plus the randomizer's own map edits, so a fill solves
// against the game being built. For now that is the forge boss moving to 019.
func Randomized() (World, Inventory) {
	w, start := Default()
	for i := range w.Checks {
		c := &w.Checks[i]
		if c.Region == "18" && c.Slot == MapDrop && c.Index == 0 {
			c.Region, c.Index = "19", ForgeBossSlot
		}
	}
	return w, start
}

// ForSeed is the world a fill solves against. Kept drops leave the forge boss
// in 018 and pin every drop, along with any slot sharing its pickup.
func ForSeed(drops bool) (World, Inventory) {
	if drops {
		return Randomized()
	}
	w, start := Default()
	shared := map[string]bool{}
	for i := range w.Checks {
		c := &w.Checks[i]
		if (c.Slot == MapDrop || c.Slot == PrmDrop) && c.Fill == Shuffle {
			c.Fill = Fixed
			if c.Dupe != "" {
				shared[c.Dupe] = true
			}
		}
	}
	for i := range w.Checks {
		if c := &w.Checks[i]; shared[c.Dupe] && c.Fill == Shuffle {
			c.Fill = Fixed
		}
	}
	return w, start
}

// Rule constrains what may be placed in one slot.
type Rule struct {
	Fill Fill
	Dupe string
}

// Where identifies a slot by map file. Some maps are split into lettered
// regions for logic, but they are still one file.
type Where struct {
	Map   string
	Slot  Slot
	Index int
}

var rulesBySlot map[Where]Rule

// Rules maps every known slot to its constraints.
func Rules() map[Where]Rule {
	if rulesBySlot == nil {
		rulesBySlot = make(map[Where]Rule, len(placements))
		for _, p := range placements {
			w := Where{Map: MapFile(p.region), Slot: p.slot, Index: p.index}
			rulesBySlot[w] = Rule{Fill: p.fill, Dupe: p.dupe}
		}
	}
	return rulesBySlot
}

// MapFile is the .map a region lives in. "16B" and "16" are the same file.
func MapFile(region string) string {
	digits, _ := splitRegion(region)
	if digits < 0 {
		return ""
	}
	return fmt.Sprintf("%03d.map", digits)
}

// splitRegion breaks "16B" into 16 and "B". A region with no leading digits
// returns -1.
func splitRegion(region string) (int, string) {
	n := 0
	for n < len(region) && region[n] >= '0' && region[n] <= '9' {
		n++
	}
	if n == 0 {
		return -1, region
	}
	id := 0
	for _, c := range region[:n] {
		id = id*10 + int(c-'0')
	}
	return id, region[n:]
}

// pair keys a crossing by map id, dropping any part letter, since the
// crossing table only knows whole maps.
func pair(a, b string) [2]string {
	na, _ := splitRegion(a)
	nb, _ := splitRegion(b)
	if na > nb {
		na, nb = nb, na
	}
	return [2]string{fmt.Sprint(na), fmt.Sprint(nb)}
}
