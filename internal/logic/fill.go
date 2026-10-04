package logic

import (
	"fmt"
	"math/rand/v2"
	"sort"
)

// Placement is one slot and what ends up in it.
type Placement struct {
	Check Check
	Item  uint16
}

// group is a set of slots the game treats as one pickup, so they all hold the
// same item and between them they contribute one item to the pool.
type group struct {
	checks []int
	item   uint16
}

// groups collapses duplicate slots and leaves every other slot on its own.
func (w World) groups() ([]group, []int) {
	var out []group
	byDupe := map[string]int{}
	var fixed, copies []int
	for i, c := range w.Checks {
		switch c.Fill {
		case Skip:
			continue
		case Copy:
			copies = append(copies, i)
			continue
		case Fixed:
			fixed = append(fixed, i)
			continue
		}
		if c.Dupe != "" {
			if g, ok := byDupe[c.Dupe]; ok {
				out[g].checks = append(out[g].checks, i)
				continue
			}
			byDupe[c.Dupe] = len(out)
		}
		out = append(out, group{checks: []int{i}, item: c.Item})
	}
	// A copy joins its twin's group last, so it never decides where the
	// group counts as reachable.
	for _, i := range copies {
		if g, ok := byDupe[w.Checks[i].Dupe]; ok {
			out[g].checks = append(out[g].checks, i)
		}
	}
	return out, fixed
}

// progression is every item some rule asks for, by name or through a
// capability. Everything else can go anywhere.
func (w World) progression() map[uint16]bool {
	out := map[uint16]bool{}
	for _, e := range w.Edges {
		for id := range e.Need.Items {
			out[id] = true
		}
		for _, c := range e.Need.Caps {
			for _, id := range w.Caps[c] {
				out[id] = true
			}
		}
	}
	for _, c := range w.Checks {
		for id := range c.Need.Items {
			out[id] = true
		}
		for _, cap := range c.Need.Caps {
			for _, id := range w.Caps[cap] {
				out[id] = true
			}
		}
	}
	return out
}

// collect adds the fixed slots' items once they are within reach, rather
// than up front, so the fill can't hide their prerequisites behind them.
func (w World) collect(inv Inventory, fixed []int) Inventory {
	out := Inventory{}
	for k, v := range inv {
		out[k] = v
	}
	taken := make(map[int]bool, len(fixed))
	for {
		reached, open := w.walk(out, nil)
		grew := false
		for _, i := range fixed {
			c := w.Checks[i]
			if taken[i] || c.Item == 0 {
				continue
			}
			if w.takeable(c, out, reached, open) {
				out[c.Item]++
				taken[i] = true
				grew = true
			}
		}
		if !grew {
			return out
		}
	}
}

// open lists the still-empty groups whose slot is reachable holding inv.
func (w World) open(inv Inventory, gs []group, filled []bool, fixed []int) []int {
	inv = w.collect(inv, fixed)
	reached, locks := w.walk(inv, nil)
	var out []int
	for i, g := range gs {
		if filled[i] {
			continue
		}
		c := w.Checks[g.checks[0]]
		if c.Fill != Filler && w.takeable(c, inv, reached, locks) {
			out = append(out, i)
		}
	}
	return out
}

// A run can dead-end when the last progression items only have taken slots
// left; a fresh shuffle almost always clears it.
const fillAttempts = 40

// Fill places every item somewhere reachable without it: progression first,
// from a pool that shrinks as it goes, then filler into what is left.
func (w World) Fill(start Inventory, rng *rand.Rand) ([]Placement, error) {
	var err error
	for i := 0; i < fillAttempts; i++ {
		var ps []Placement
		if ps, err = w.fillOnce(start, rng); err != nil {
			continue
		}
		// The fill can't plan for keystones and rocks being used up, so a run
		// only counts once the full solve agrees.
		if res := w.Apply(ps).Solve(start); len(res.Unreached) > 0 {
			err = fmt.Errorf("%d checks unreachable once locks are paid for, first %s",
				len(res.Unreached), res.Unreached[0].ID())
			continue
		}
		return ps, nil
	}
	return nil, fmt.Errorf("%d attempts failed, last: %w", fillAttempts, err)
}

func (w World) fillOnce(start Inventory, rng *rand.Rand) ([]Placement, error) {
	gs, fixed := w.groups()
	prog := w.progression()

	base := Inventory{}
	for k, v := range start {
		base[k] += v
	}

	var pool, filler []uint16
	for _, g := range gs {
		if prog[g.item] {
			pool = append(pool, g.item)
		} else {
			filler = append(filler, g.item)
		}
	}
	sort.Slice(pool, func(i, j int) bool { return pool[i] < pool[j] })
	sort.Slice(filler, func(i, j int) bool { return filler[i] < filler[j] })
	rng.Shuffle(len(pool), func(i, j int) { pool[i], pool[j] = pool[j], pool[i] })
	rng.Shuffle(len(filler), func(i, j int) { filler[i], filler[j] = filler[j], filler[i] })

	filled := make([]bool, len(gs))
	assumed := Inventory{}
	for k, v := range base {
		assumed[k] = v
	}
	for _, id := range pool {
		assumed[id]++
	}

	for len(pool) > 0 {
		id := pool[len(pool)-1]
		pool = pool[:len(pool)-1]
		assumed[id]--

		spots := w.open(assumed, gs, filled, fixed)
		if len(spots) == 0 {
			return nil, fmt.Errorf("nowhere left to put item %d with %d still unplaced", id, len(pool))
		}
		g := spots[rng.IntN(len(spots))]
		gs[g].item = id
		filled[g] = true
	}

	for i := range gs {
		if filled[i] {
			continue
		}
		if len(filler) == 0 {
			return nil, fmt.Errorf("ran out of filler with %d slots empty", len(gs)-i)
		}
		gs[i].item = filler[len(filler)-1]
		filler = filler[:len(filler)-1]
		filled[i] = true
	}

	var out []Placement
	for _, g := range gs {
		for _, i := range g.checks {
			out = append(out, Placement{Check: w.Checks[i], Item: g.item})
		}
	}
	for _, i := range fixed {
		out = append(out, Placement{Check: w.Checks[i], Item: w.Checks[i].Item})
	}
	return out, nil
}

// Apply returns a copy of w with these placements in it, for re-solving.
func (w World) Apply(ps []Placement) World {
	at := make(map[string]uint16, len(ps))
	for _, p := range ps {
		at[p.Check.ID()] = p.Item
	}
	out := w
	out.Checks = make([]Check, len(w.Checks))
	copy(out.Checks, w.Checks)
	for i := range out.Checks {
		if id, ok := at[out.Checks[i].ID()]; ok {
			out.Checks[i].Item = id
		}
	}
	return out
}
