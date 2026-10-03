package logic

import (
	"math/rand/v2"
	"testing"
)

func seeded(n uint64) *rand.Rand { return rand.New(rand.NewPCG(n, n^0x9e3779b9)) }

func TestFillNeverPutsAnItemBehindItself(t *testing.T) {
	w, start := Default()
	for seed := uint64(1); seed <= 25; seed++ {
		ps, err := w.Fill(start, seeded(seed))
		if err != nil {
			t.Fatalf("seed %d: %v", seed, err)
		}
		res := w.Apply(ps).Solve(start)
		if len(res.Unreached) != 0 {
			t.Errorf("seed %d: %d checks unreachable, first %s",
				seed, len(res.Unreached), res.Unreached[0].ID())
		}
	}
}

func TestFillKeepsEveryItemAndSlot(t *testing.T) {
	w, start := Default()
	ps, err := w.Fill(start, seeded(7))
	if err != nil {
		t.Fatal(err)
	}

	var want int
	for _, c := range w.Checks {
		if c.Fill != Skip {
			want++
		}
	}
	if len(ps) != want {
		t.Errorf("placements = %d, want %d", len(ps), want)
	}

	before, after := map[uint16]int{}, map[uint16]int{}
	gs, fixed := w.groups()
	for _, g := range gs {
		before[g.item]++
	}
	for _, i := range fixed {
		before[w.Checks[i].Item]++
	}
	seen := map[string]bool{}
	for _, p := range ps {
		if seen[p.Check.ID()] {
			t.Fatalf("%s filled twice", p.Check.ID())
		}
		seen[p.Check.ID()] = true
		if p.Check.Dupe == "" || firstOfDupe(ps, p.Check.Dupe) == p.Check.ID() {
			after[p.Item]++
		}
	}
	for id, n := range before {
		if after[id] != n {
			t.Errorf("item %d: %d in, %d out", id, n, after[id])
		}
	}
}

func firstOfDupe(ps []Placement, dupe string) string {
	for _, p := range ps {
		if p.Check.Dupe == dupe {
			return p.Check.ID()
		}
	}
	return ""
}

func TestFillGivesDuplicateSlotsTheSameItem(t *testing.T) {
	w, start := Default()
	ps, err := w.Fill(start, seeded(3))
	if err != nil {
		t.Fatal(err)
	}
	byDupe := map[string]uint16{}
	for _, p := range ps {
		if p.Check.Dupe == "" {
			continue
		}
		if got, ok := byDupe[p.Check.Dupe]; ok && got != p.Item {
			t.Errorf("%s: %s has both %d and %d", p.Check.Dupe, p.Check.ID(), got, p.Item)
		}
		byDupe[p.Check.Dupe] = p.Item
	}
	if len(byDupe) == 0 {
		t.Fatal("no duplicate groups in the world at all")
	}
}

func TestFillLeavesFixedSlotsAlone(t *testing.T) {
	w, start := Default()
	ps, err := w.Fill(start, seeded(11))
	if err != nil {
		t.Fatal(err)
	}
	n := 0
	for _, p := range ps {
		if p.Check.Fill == Fixed {
			n++
			if p.Item != p.Check.Item {
				t.Errorf("%s changed from %d to %d", p.Check.ID(), p.Check.Item, p.Item)
			}
		}
	}
	if n == 0 {
		t.Fatal("no fixed slots in the world at all")
	}
}

func TestFillSpreadsProgressionAcrossSeeds(t *testing.T) {
	w, start := Default()
	where := map[string]bool{}
	for seed := uint64(1); seed <= 10; seed++ {
		ps, err := w.Fill(start, seeded(seed))
		if err != nil {
			t.Fatal(err)
		}
		for _, p := range ps {
			if p.Item == KingsKey {
				where[p.Check.ID()] = true
			}
		}
	}
	// The King's Key is a gate item that moves. The Moonlight Sword is pinned
	// to its altar, so it would land in the same slot every seed.
	if len(where) < 3 {
		t.Errorf("King's Key landed in only %d places over 10 seeds", len(where))
	}
}
