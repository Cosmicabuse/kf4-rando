// Package logic answers what is reachable from the start, sphere by sphere.
package logic

import (
	"fmt"
	"sort"
	"strings"
)

// Cap is a requirement no single item satisfies on its own.
type Cap string

// The values are what a report prints, so they read as requirements.
const (
	CapFire       Cap = "a fire spell"
	CapProjectile Cap = "a projectile weapon"
	// A weapon strong enough to break a crate at the stat a new game starts
	// with. Bare-handed, the Wooden Club and both Short Swords are not.
	CapBreak Cap = "a weapon that can break containers"
)

// Req is what an edge or a check asks for. Items are counted because
// keystones and release rocks are consumed on use.
type Req struct {
	Items map[uint16]int
	Caps  []Cap
}

func (r Req) empty() bool { return len(r.Items) == 0 && len(r.Caps) == 0 }

// has asks for one of each item. List an item twice to ask for two.
func has(items ...uint16) Req {
	r := Req{Items: map[uint16]int{}}
	for _, id := range items {
		r.Items[id]++
	}
	return r
}

func can(caps ...Cap) Req { return Req{Caps: caps} }

func (r Req) and(o Req) Req {
	out := Req{Items: map[uint16]int{}, Caps: append(append([]Cap(nil), r.Caps...), o.Caps...)}
	for id, n := range r.Items {
		out.Items[id] += n
	}
	for id, n := range o.Items {
		out.Items[id] += n
	}
	return out
}

// Inventory counts items held.
type Inventory map[uint16]int

// Capabilities lists the items that grant each Cap.
type Capabilities map[Cap][]uint16

// Met reports whether inv satisfies r.
func (r Req) Met(inv Inventory, caps Capabilities) bool {
	for id, n := range r.Items {
		if inv[id] < n {
			return false
		}
	}
	for _, c := range r.Caps {
		ok := false
		for _, id := range caps[c] {
			if inv[id] > 0 {
				ok = true
				break
			}
		}
		if !ok {
			return false
		}
	}
	return true
}

// Describe names a Req's items and capabilities.
func (r Req) Describe(names map[uint16]string) string {
	if r.empty() {
		return ""
	}
	var parts []string
	ids := make([]uint16, 0, len(r.Items))
	for id := range r.Items {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	for _, id := range ids {
		name, ok := names[id]
		if !ok || name == "" {
			name = fmt.Sprintf("item %d", id)
		}
		if n := r.Items[id]; n > 1 {
			parts = append(parts, fmt.Sprintf("%dx %s", n, name))
		} else {
			parts = append(parts, name)
		}
	}
	for _, c := range r.Caps {
		parts = append(parts, string(c))
	}
	return strings.Join(parts, ", ")
}

// Edge is a one-way move between regions. Two-way crossings are two edges.
type Edge struct {
	From, To string
	Need     Req
}

// Slot says which array a check lives in.
type Slot int

const (
	Ground Slot = iota
	Container
	MapDrop
	// A PrmDrop comes off the monster type in chr.prm, not off a map
	// slot, so Index is a monster classID.
	PrmDrop
	// Start is the starting inventory, where the Idol of Sorrow sits.
	Start
	// Shop slots: Index is shop*100 plus the slot within that shop.
	Shop
)

func (s Slot) String() string {
	switch s {
	case Ground:
		return "ground"
	case Container:
		return "container"
	case MapDrop:
		return "monster map drop"
	case PrmDrop:
		return "monster prm drop"
	case Start:
		return "starting inventory"
	case Shop:
		return "shop"
	}
	return fmt.Sprintf("slot%d", int(s))
}

// Fill says whether a slot can take a shuffled item.
type Fill int

const (
	Shuffle Fill = iota
	Fixed        // real, but keeps its vanilla item
	Skip         // the game never puts anything here
	Copy         // shows its twin's item, out of reach from this side
	Filler       // shuffled, but never handed a progression item
)

// Check is one item placement. Region plus Slot plus Index identifies it.
type Check struct {
	Region string
	Slot   Slot
	Index  int
	Item   uint16
	Need   Req
	Fill   Fill
	// Slots sharing a Dupe are one pickup to the game and must end up
	// holding the same item.
	Dupe string
	// Another place that has to be reachable too, for an errand whose
	// ingredient is elsewhere.
	Also string
	// Lock pays into another map's lock, for an NPC who moved.
	Lock string
}

func (c Check) within(reached map[string]bool) bool {
	return reached[c.Region] && (c.Also == "" || reached[c.Also])
}

// ID is a stable name for one check.
func (c Check) ID() string {
	return fmt.Sprintf("%s/%s%d", c.Region, c.Slot, c.Index)
}

// World is a map graph plus everything placed in it.
type World struct {
	Start  string
	Edges  []Edge
	Checks []Check
	Caps   Capabilities
}

// Sphere is one round of the search.
type Sphere struct {
	Regions []string
	Checks  []Check

	// Gates that the previous round's items made passable, which is what
	// makes this round's new regions reachable.
	Opened []Edge
}

// Result is a full solve.
type Result struct {
	Spheres   []Sphere
	Unreached []Check
	// Stranded regions hold checks but have no route in.
	Stranded []string
}

// Keystones, release rocks, rocks of life and strange eggs get used up. A lock costs its count once, shared
// by everything behind it, and only counts as open when what is held covers
// every lock in reach, whatever order they get opened in.
var consumable = map[uint16]bool{Keystone: true, ReleaseRock: true, RockOfLife: true, StrangeEgg: true}

// lockKey names the lock a requirement pays into, "" when nothing is used up.
// Checks in one map behind the same cost share a lock.
func lockKey(site string, r Req) (key string, item uint16, cost int) {
	for id, n := range r.Items {
		if consumable[id] {
			return fmt.Sprintf("%s/%d/%d", site, id, n), id, n
		}
	}
	return "", 0, 0
}

// A two-way gate is one door, so both directions share its lock.
func edgeLock(e Edge) (string, uint16, int) {
	a, b := e.From, e.To
	if a > b {
		a, b = b, a
	}
	return lockKey(a+"|"+b, e.Need)
}

func checkLock(c Check) (string, uint16, int) {
	if c.Lock != "" {
		return lockKey(c.Lock, c.Need)
	}
	n, _ := splitRegion(c.Region)
	return lockKey(fmt.Sprint(n), c.Need)
}

// reach walks the region graph with inv in hand.
func (w World) reach(inv Inventory) map[string]bool {
	return w.reachSkipping(inv, nil)
}

// reachSkipping is reach with some edges pulled out, keyed by index, to tell
// a real gate from one there is a way around.
func (w World) reachSkipping(inv Inventory, skip map[int]bool) map[string]bool {
	reached, _ := w.walk(inv, skip)
	return reached
}

// walk returns the regions in reach and the locks that can be paid for. An
// item whose locks in reach cost more than is held opens none of them, and
// stays that way so the walk can't flip back and forth.
func (w World) walk(inv Inventory, skip map[int]bool) (map[string]bool, map[string]bool) {
	open := map[string]bool{}
	short := map[uint16]bool{}
	for {
		reached := w.flood(inv, skip, open)
		costs := map[string]int{}
		items := map[string]uint16{}
		note := func(key string, item uint16, cost int) {
			if key != "" {
				costs[key], items[key] = cost, item
			}
		}
		for i, e := range w.Edges {
			if !skip[i] && reached[e.From] && e.Need.Met(inv, w.Caps) {
				note(edgeLock(e))
			}
		}
		for _, c := range w.Checks {
			if c.Fill != Copy && c.within(reached) && c.Need.Met(inv, w.Caps) {
				note(checkLock(c))
			}
		}
		total := map[uint16]int{}
		for key, cost := range costs {
			total[items[key]] += cost
		}
		next := map[string]bool{}
		for key, item := range items {
			if inv[item] < total[item] {
				short[item] = true
			}
			if !short[item] {
				next[key] = true
			}
		}
		same := len(next) == len(open)
		for key := range next {
			same = same && open[key]
		}
		if same {
			return reached, open
		}
		open = next
	}
}

// flood is the plain graph walk, with consumable gates passable only when open.
func (w World) flood(inv Inventory, skip map[int]bool, open map[string]bool) map[string]bool {
	out := map[string]bool{w.Start: true}
	for grew := true; grew; {
		grew = false
		for i, e := range w.Edges {
			if skip[i] || !out[e.From] || out[e.To] || !e.Need.Met(inv, w.Caps) {
				continue
			}
			if key, _, _ := edgeLock(e); key != "" && !open[key] {
				continue
			}
			out[e.To] = true
			grew = true
		}
	}
	return out
}

// takeable reports whether c can be picked up given what walk returned.
func (w World) takeable(c Check, inv Inventory, reached, open map[string]bool) bool {
	if !c.within(reached) || !c.Need.Met(inv, w.Caps) {
		return false
	}
	key, _, _ := checkLock(c)
	return key == "" || open[key]
}

// opened lists the gates that made this round's new regions reachable. A
// gate counts only if removing it loses the region.
func (w World) opened(inv Inventory, now, before map[string]bool) []Edge {
	if before == nil {
		return nil
	}

	candidates := map[string][]int{}
	for i, e := range w.Edges {
		if e.Need.empty() || before[e.To] || !now[e.To] || !now[e.From] {
			continue
		}
		if !e.Need.Met(inv, w.Caps) {
			continue
		}
		candidates[e.To] = append(candidates[e.To], i)
	}

	keep := map[int]bool{}
	for region, idx := range candidates {
		all := make(map[int]bool, len(idx))
		for _, i := range idx {
			all[i] = true
		}
		if w.reachSkipping(inv, all)[region] {
			continue // there is a way in that costs nothing
		}
		critical := false
		for _, i := range idx {
			if !w.reachSkipping(inv, map[int]bool{i: true})[region] {
				keep[i] = true
				critical = true
			}
		}
		if !critical {
			// Several gates lead in and any one of them will do, so none is
			// critical on its own. All of them are the answer.
			for _, i := range idx {
				keep[i] = true
			}
		}
	}

	var out []Edge
	seen := map[[2]string]bool{}
	for i, e := range w.Edges {
		if !keep[i] {
			continue
		}
		key := [2]string{e.From, e.To}
		if e.From > e.To {
			key = [2]string{e.To, e.From}
		}
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, e)
	}
	return out
}

// Solve runs the sphere search from start.
func (w World) Solve(start Inventory) Result {
	inv := Inventory{}
	for k, v := range start {
		inv[k] = v
	}

	var res Result
	var before map[string]bool
	taken := make(map[int]bool, len(w.Checks))
	for {
		reached, open := w.walk(inv, nil)

		var got []Check
		for i, c := range w.Checks {
			if taken[i] || c.Fill == Copy || !w.takeable(c, inv, reached, open) {
				continue
			}
			taken[i] = true
			got = append(got, c)
		}
		if len(got) == 0 {
			break
		}

		regions := make([]string, 0, len(reached))
		for r := range reached {
			regions = append(regions, r)
		}
		sort.Strings(regions)
		// inv is still what was held going in, so a gate passable now is one
		// the last round's items paid for.
		res.Spheres = append(res.Spheres, Sphere{
			Regions: regions, Checks: got,
			Opened: w.opened(inv, reached, before),
		})
		before = reached

		for _, c := range got {
			if c.Item != 0 {
				inv[c.Item]++
			}
		}
	}

	final := w.reach(inv)
	for i, c := range w.Checks {
		if !taken[i] && c.Fill != Copy {
			res.Unreached = append(res.Unreached, c)
			for _, r := range []string{c.Region, c.Also} {
				if r != "" && !final[r] {
					res.Stranded = append(res.Stranded, r)
				}
			}
		}
	}
	res.Stranded = dedup(res.Stranded)
	return res
}

func dedup(in []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, s := range in {
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	sort.Strings(out)
	return out
}
