package report

import (
	"strings"
	"testing"

	"kf4randomizer/internal/features/randomize"
	"kf4randomizer/internal/kfmap"
)

func TestSpoilerGroupsByMapThenKind(t *testing.T) {
	result := randomize.Result{
		TotalMaps:     2,
		EligibleSlots: 4,
		Changes: []randomize.Change{
			{MapName: "001.map", Kind: randomize.GroundItem, Index: 3, FromClassID: 1, ToClassID: 2},
			{MapName: "001.map", Kind: randomize.ContainerItem, Index: 9, FromClassID: 2, ToClassID: 1},
			{MapName: "000.map", Kind: randomize.MapDrop, Index: 4, FromClassID: 1, ToClassID: 3},
			{MapName: "chr.prm", Kind: randomize.PrmDrop, Index: 40, FromClassID: 3, ToClassID: 1},
		},
	}
	names := kfmap.ItemNames{1: "Herb", 2: "Antidote"}
	mapNames := kfmap.MapNames{"000.map": "Village Ruins"}

	got := Spoiler(7, result, names, mapNames)

	for _, want := range []string{
		"Seed: 7",
		"=== 000.map (Village Ruins) ===",
		"=== 001.map ===",
		"    slot  3: Herb -> Antidote",
		"    container  9: Antidote -> Herb",
		"    monster  4: Herb -> (unknown)",
		"    class 40: (unknown) -> Herb",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q, got:\n%s", want, got)
		}
	}
	if strings.Index(got, "000.map") > strings.Index(got, "001.map") {
		t.Error("maps should come out in name order")
	}
}

func TestSpheresReportsTheVanillaGame(t *testing.T) {
	got := Spheres(nil, true, kfmap.ItemNames{49: "RUSTY KEY"}, kfmap.MapNames{"002.map": "Village Ruins"})

	if !strings.Contains(got, "=== SPHERE 0:") {
		t.Fatalf("no spheres in the report:\n%s", got)
	}
	if strings.Contains(got, "NEVER REACHED") {
		t.Error("the unmodified game should reach everything")
	}
	// Region 2A is behind the Rusty Key, and its label carries map 002's name.
	if !strings.Contains(got, "002A Village Ruins") {
		t.Error("a lettered region should be labelled with its map's name")
	}
	if !strings.Contains(got, "[RUSTY KEY]") {
		t.Error("a check's requirement should be named")
	}
	if !strings.Contains(got, "using: RUSTY KEY                  002->002A") {
		t.Error("a sphere should name the gate its predecessor's items opened")
	}
}
