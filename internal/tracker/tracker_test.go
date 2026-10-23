package tracker

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTilesUnique(t *testing.T) {
	sections := map[string]bool{}
	for _, s := range Sections {
		sections[s] = true
	}
	seen := map[string]bool{}
	for _, tl := range Tiles {
		if seen[tl.ID] {
			t.Errorf("duplicate tile %s", tl.ID)
		}
		seen[tl.ID] = true
		if !sections[tl.Section] {
			t.Errorf("%s: unknown section %q", tl.ID, tl.Section)
		}
		if (tl.Section == "Keys" || tl.Section == "Icons") && tl.Label == "" {
			t.Errorf("%s: empty label", tl.ID)
		}
	}
	if len(Tiles) != 49 {
		t.Errorf("got %d tiles, want 49", len(Tiles))
	}
}

func TestSaveLoadSeed(t *testing.T) {
	path := filepath.Join(t.TempDir(), "t.json")
	in := &File{Seed: "18446744073709551615", State: map[string]int{"keystone": 3}}
	if err := in.Save(path); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), `"seed": "18446744073709551615"`) {
		t.Errorf("seed not a quoted string:\n%s", raw)
	}
	out, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if out.Seed != in.Seed || out.State["keystone"] != 3 {
		t.Errorf("got seed %q state %v", out.Seed, out.State)
	}
}

func TestLoadNormalizes(t *testing.T) {
	path := filepath.Join(t.TempDir(), "t.json")
	in := &File{
		Seed:  "1",
		Start: map[string]int{"rusty_key": 1, "gone": 2},
		State: map[string]int{"nope": 1, "keystone": 9},
	}
	if err := in.Save(path); err != nil {
		t.Fatal(err)
	}
	f, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if f.State["keystone"] != 5 {
		t.Errorf("keystone = %d, want 5", f.State["keystone"])
	}
	if _, ok := f.State["nope"]; ok {
		t.Error("unknown state id kept")
	}
	if _, ok := f.Start["gone"]; ok {
		t.Error("unknown start id kept")
	}
	if f.State["rusty_key"] != 1 {
		t.Errorf("rusty_key = %d, want its start value 1", f.State["rusty_key"])
	}
}

func TestLoadRejectsOtherJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	if err := os.WriteFile(path, []byte(`{"editor.fontSize": 12}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path); err == nil {
		t.Error("loaded a file that is not a tracker")
	}
}

func TestCapNeverZero(t *testing.T) {
	f := File{Max: map[string]int{"crystal_vial": 0}}
	if got := f.Cap("crystal_vial"); got != 1 {
		t.Errorf("Cap = %d, want 1", got)
	}
}

func TestFreePath(t *testing.T) {
	dir := t.TempDir()
	if got, want := FreePath(dir, 7), filepath.Join(dir, "tracker_7.json"); got != want {
		t.Fatalf("got %s, want %s", got, want)
	}
	a := filepath.Join(dir, "tracker_7.json")
	b := filepath.Join(dir, "tracker_7-1.json")
	for _, p := range []string{a, b} {
		if err := os.WriteFile(p, []byte("keep"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if got, want := FreePath(dir, 7), filepath.Join(dir, "tracker_7-2.json"); got != want {
		t.Errorf("got %s, want %s", got, want)
	}
	for _, p := range []string{a, b} {
		if raw, _ := os.ReadFile(p); string(raw) != "keep" {
			t.Errorf("%s changed", p)
		}
	}
}

func TestSaveKeepsOldOnFailure(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing", "t.json")
	if err := (&File{}).Save(path); err == nil {
		t.Error("save into missing directory succeeded")
	}
}
