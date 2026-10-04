package logic

import (
	"fmt"
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
		w.Edges = append(w.Edges, Edge{From: g.from, To: g.to, Need: g.need})
		if !g.oneWay {
			w.Edges = append(w.Edges, Edge{From: g.to, To: g.from, Need: g.need})
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

	// A Skip slot never spawns or is not understood yet, so don't rando
	for _, p := range placements {
		if p.fill == Skip {
			continue
		}
		w.Checks = append(w.Checks, Check{
			Region: p.region, Slot: p.slot, Index: p.index,
			Item: p.item, Need: p.need, Fill: p.fill, Dupe: p.dupe,
			Also: errands[Where{Map: p.region, Slot: p.slot, Index: p.index}],
		})
	}

	inv := Inventory{}
	for _, id := range startItems {
		inv[id]++
	}
	return w, inv
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

// Pool says which kinds of slot the randomizer shuffles.
type Pool struct {
	Ground, Containers, Drops, Idol, Shops bool
}

// Has reports whether slots of this kind are shuffled.
func (p Pool) Has(s Slot) bool {
	switch s {
	case Ground:
		return p.Ground
	case Container:
		return p.Containers
	case Start:
		return p.Idol
	case Shop:
		return p.Shops
	}
	return p.Drops
}

// ForSeed is the world a fill solves against. Kinds left out of the pool keep
// their items, along with any slot sharing a pickup with them, and kept drops
// leave the forge boss in 018.
func ForSeed(pool Pool) (World, Inventory) {
	w, start := Default()
	if pool.Drops {
		w, start = Randomized()
	}
	shared := map[string]bool{}
	for i := range w.Checks {
		c := &w.Checks[i]
		if !pool.Has(c.Slot) && c.Fill == Shuffle {
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

	if pool.Shops {
		w.Checks = append(w.Checks, shopChecks()...)
	}
	// randomize idol
	if pool.Idol && start[IdolOfSorrow] > 0 {
		delete(start, IdolOfSorrow)
		w.Checks = append(w.Checks, Check{Region: w.Start, Slot: Start, Item: IdolOfSorrow, Fill: Shuffle})
	}
	return w, start
}

// shopChecks gives each shop a share of the distinct items the shops sell,
// sized like its vanilla stock. The Caretaker's Map is a copy of the one found
// in 2A, so its slot takes filler only.
func shopChecks() []Check {
	var items []uint16
	seen := map[uint16]bool{}
	counts := make([]int, len(shops))
	for s, sh := range shops {
		counts[s] = len(sh.items)
		for _, id := range sh.items {
			if !seen[id] {
				seen[id] = true
				items = append(items, id)
			}
		}
	}
	var out []Check
	next := 0
	for s, n := range share(counts, len(items)) {
		sh := shops[s]
		for k := range n {
			c := Check{Region: sh.region, Slot: Shop, Index: s*100 + k, Item: items[next],
				Need: sh.need, Fill: Shuffle, Also: sh.also, Lock: sh.lock}
			if c.Item == CaretakersMap {
				c.Item, c.Fill = Herb, Filler
			}
			out = append(out, c)
			next++
		}
	}
	return out
}

// ShopNumber is the gold shop a Shop check belongs to.
func ShopNumber(c Check) int { return c.Index / 100 }

// share splits n over counts proportionally, leftovers to the largest
// remainders.
func share(counts []int, n int) []int {
	total := 0
	for _, c := range counts {
		total += c
	}
	sizes := make([]int, len(counts))
	if total == 0 {
		return sizes
	}
	left := n
	rest := make([]int, len(counts))
	for i, c := range counts {
		sizes[i], rest[i] = c*n/total, c*n%total
		left -= sizes[i]
	}
	for ; left > 0; left-- {
		best := 0
		for i := range rest {
			if rest[i] > rest[best] {
				best = i
			}
		}
		sizes[best]++
		rest[best] = -1
	}
	return sizes
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
