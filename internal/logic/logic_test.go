package logic

import "testing"

func TestSolveOrdersSpheresByWhatUnlocks(t *testing.T) {
	w := World{
		Start: "0",
		Edges: []Edge{
			{From: "0", To: "1", Need: Req{Items: map[uint16]int{WiddaKey: 1}}},
			{From: "1", To: "2", Need: Req{Caps: []Cap{CapFire}}},
		},
		Checks: []Check{
			{Region: "0", Item: WiddaKey},
			{Region: "1", Item: fireItems[0]},
			{Region: "2", Item: 6},
		},
		Caps: Capabilities{CapFire: fireItems},
	}

	got := w.Solve(Inventory{})
	if len(got.Spheres) != 3 {
		t.Fatalf("spheres = %d, want 3", len(got.Spheres))
	}
	if len(got.Unreached) != 0 {
		t.Errorf("unreached = %v, want none", got.Unreached)
	}
	for i, want := range []uint16{WiddaKey, fireItems[0], 6} {
		if got.Spheres[i].Checks[0].Item != want {
			t.Errorf("sphere %d item = %d, want %d", i, got.Spheres[i].Checks[0].Item, want)
		}
	}
}

func TestSolveNamesTheGateEachSphereOpened(t *testing.T) {
	w := World{
		Start: "0",
		Edges: []Edge{
			{From: "0", To: "1", Need: Req{Items: map[uint16]int{WiddaKey: 1}}},
			{From: "1", To: "0"},
		},
		Checks: []Check{{Region: "0", Item: WiddaKey}, {Region: "1", Item: 6}},
	}

	got := w.Solve(Inventory{})
	if len(got.Spheres) != 2 {
		t.Fatalf("spheres = %d, want 2", len(got.Spheres))
	}
	if len(got.Spheres[0].Opened) != 0 {
		t.Errorf("sphere 0 opened %v, want nothing; it is where the run starts", got.Spheres[0].Opened)
	}
	// The free 1->0 edge shares the pair, so only the gated direction counts.
	if o := got.Spheres[1].Opened; len(o) != 1 || o[0].To != "1" || o[0].Need.Items[WiddaKey] != 1 {
		t.Errorf("sphere 1 opened %v, want the Widda Key gate into 1", o)
	}
}

// Region 16B in the real tables has an Icon of Ice door from 19 and a free
// link to 105, so holding the Icon is not what gets you in.
func TestRandomizedMovesTheForgeBoss(t *testing.T) {
	vanilla, _ := Default()
	patched, _ := Randomized()

	if len(vanilla.Checks) != len(patched.Checks) {
		t.Fatalf("checks = %d and %d, the boss moves rather than multiplying",
			len(vanilla.Checks), len(patched.Checks))
	}
	find := func(w World, region string, index int) *Check {
		for i := range w.Checks {
			if c := &w.Checks[i]; c.Region == region && c.Slot == MapDrop && c.Index == index {
				return c
			}
		}
		return nil
	}
	if find(vanilla, "18", 0) == nil {
		t.Error("the disc has the boss in 018")
	}
	if find(vanilla, "19", ForgeBossSlot) != nil {
		t.Error("the disc has nothing in 019")
	}
	if find(patched, "18", 0) != nil {
		t.Error("018 should be empty once the boss moves")
	}
	if find(patched, "19", ForgeBossSlot) == nil {
		t.Fatal("the boss did not arrive in 019")
	}
	for _, e := range patched.Edges {
		if e.To == "19" && e.Need.Items[IconOfIce] == 0 {
			t.Errorf("%s -> 19 reaches the boss without the icon", e.From)
		}
	}
}

func TestSolveIgnoresAGateYouCanWalkAround(t *testing.T) {
	w := World{
		Start: "0",
		Edges: []Edge{
			{From: "0", To: "1", Need: Req{Items: map[uint16]int{WiddaKey: 1}}},
			{From: "0", To: "2"},
			{From: "2", To: "1"},
		},
		Checks: []Check{{Region: "0", Item: WiddaKey}, {Region: "1", Item: 6}},
	}

	got := w.Solve(Inventory{})
	for i, s := range got.Spheres {
		if len(s.Opened) != 0 {
			t.Errorf("sphere %d opened %v, want nothing; 1 is reachable through 2 for free", i, s.Opened)
		}
	}
}

func TestSolveReportsStrandedRegions(t *testing.T) {
	w := World{Start: "0", Checks: []Check{{Region: "0"}, {Region: "41"}}}
	got := w.Solve(Inventory{})
	if len(got.Stranded) != 1 || got.Stranded[0] != "41" {
		t.Errorf("stranded = %v, want [41]", got.Stranded)
	}
}

func TestSolveHonoursCounts(t *testing.T) {
	w := World{
		Start: "0",
		Edges: []Edge{{From: "0", To: "1", Need: Req{Items: map[uint16]int{BindingCrown: 2}}}},
		Checks: []Check{
			{Region: "0", Item: BindingCrown},
			{Region: "1", Item: 6},
		},
	}
	if got := w.Solve(Inventory{}); len(got.Unreached) != 1 {
		t.Errorf("one crown must not open a two-crown gate, got %v", got.Unreached)
	}
	if got := w.Solve(Inventory{BindingCrown: 1}); len(got.Unreached) != 0 {
		t.Errorf("a second crown should open it, got %v", got.Unreached)
	}
}

func TestDefaultWorldIsConsistent(t *testing.T) {
	w, start := Default()

	if len(w.Checks) < 500 {
		t.Errorf("checks = %d, want every known placement", len(w.Checks))
	}
	if len(w.Edges) == 0 || len(start) == 0 {
		t.Fatalf("edges = %d, start = %v", len(w.Edges), start)
	}

	// Every region an edge names must be somewhere in the graph, or a gate
	// points at a region that does not exist.
	known := map[string]bool{w.Start: true}
	for _, e := range w.Edges {
		known[e.From], known[e.To] = true, true
	}
	for _, c := range w.Checks {
		if !known[c.Region] {
			t.Errorf("region %s holds checks but no edge reaches it", c.Region)
			break
		}
	}
}

func TestDefaultSolveMatchesTheKnownEarlyGame(t *testing.T) {
	w, start := Default()
	res := w.Solve(start)

	// Without a fire source these are the only regions in reach.
	want := map[string]bool{
		"0": true, "2": true, "4": true, "8": true, "9": true,
		"29": true, "96": true, "98": true, "99": true,
	}
	got := map[string]bool{}
	for _, r := range res.Spheres[0].Regions {
		got[r] = true
	}
	for r := range want {
		if !got[r] {
			t.Errorf("sphere 0 is missing region %s", r)
		}
	}
	for r := range got {
		if !want[r] {
			t.Errorf("sphere 0 unexpectedly reaches %s", r)
		}
	}
}

func TestDefaultReachesEverything(t *testing.T) {
	w, start := Default()
	res := w.Solve(start)

	if len(res.Unreached) != 0 {
		for _, c := range res.Unreached {
			t.Errorf("%s (%s) is unreachable", c.ID(), c.Need.Describe(nil))
		}
	}
	if len(res.Stranded) != 0 {
		t.Errorf("stranded regions: %v", res.Stranded)
	}
	// The crowns are what open the late game, so it cannot be one flat sphere.
	if len(res.Spheres) < 5 {
		t.Errorf("spheres = %d, want a real progression", len(res.Spheres))
	}
}

func TestGateOnASubRegionSuppressesTheWholeMapsCrossing(t *testing.T) {
	// The Icon of Ice door leads into 16B, but the crossing table only knows
	// that link as 16<->19. Keying on the full region name left 16->19 free.
	w, start := Default()
	for _, e := range w.Edges {
		if e.From == "16" && e.To == "19" && e.Need.empty() {
			t.Fatal("free 16->19 edge walks around the Icon of Ice gate")
		}
		if e.From == "16" && e.To == "105" && e.Need.empty() {
			t.Fatal("free 16->105 edge walks around 16B")
		}
		if e.From == "6" && e.To == "42" && e.Need.empty() {
			t.Fatal("free 6->42 edge walks around the Black Eye door")
		}
	}
	// The forge branch hangs off map 16, so it must not open in sphere 1.
	res := w.Solve(start)
	for _, c := range res.Spheres[1].Checks {
		switch c.Region {
		case "16", "16A", "17", "18", "19", "20", "24", "25":
			t.Errorf("%s is reachable too early", c.ID())
		}
	}
}
