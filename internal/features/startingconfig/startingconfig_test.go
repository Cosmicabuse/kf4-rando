package startingconfig

import (
	"strings"
	"testing"
)

const fixtureINI = "hp=50\nmp=0\ngold=0\n; a comment line\nlevel=1\n"

func TestApplyOverridesRewritesOnlyTheGivenKeys(t *testing.T) {
	out, err := ApplyOverrides([]byte(fixtureINI), map[string]int{"gold": 100, "hp": 999, "level": 5})
	if err != nil {
		t.Fatalf("ApplyOverrides: %v", err)
	}
	for _, want := range []string{"gold=100", "hp=999", "level=5", "mp=0", "; a comment line"} {
		if !strings.Contains(string(out), want) {
			t.Errorf("missing %q, got:\n%s", want, out)
		}
	}

	out, err = ApplyOverrides([]byte(fixtureINI), nil)
	if err != nil {
		t.Fatalf("ApplyOverrides: %v", err)
	}
	if string(out) != fixtureINI {
		t.Errorf("no overrides should leave the ini byte-identical, got:\n%s", out)
	}

	if _, err := ApplyOverrides([]byte(fixtureINI), map[string]int{"not_a_real_key": 1}); err == nil {
		t.Error("an unknown ini key should be rejected rather than ignored")
	}
}

// Equipping something with a quantity of 0 crashes the equipment menu, which
// is how the game's own fix_9393.ini ships.
func TestApplyOverridesGrantsAnythingEquipped(t *testing.T) {
	for _, c := range []struct {
		why       string
		base      string
		overrides map[string]int
		want      string
	}{
		{"an override equips something unowned",
			"equip_bracelet=-1\nitem_361=0\nitem_195=1\n",
			map[string]int{"equip_bracelet": 361}, "item_361=1"},
		{"the base already equips something unowned",
			"equip_weapon=195\nhp=50\nitem_195=0\n",
			map[string]int{"hp": 987}, "item_195=1"},
		{"an owned item keeps the quantity it had",
			"equip_weapon=195\nitem_195=7\n", nil, "item_195=7"},
		{"-1 means nothing equipped",
			"equip_weapon=195\nitem_000=0\n",
			map[string]int{"equip_weapon": -1}, "equip_weapon=-1"},
	} {
		out, err := ApplyOverrides([]byte(c.base), c.overrides)
		if err != nil {
			t.Errorf("%s: %v", c.why, err)
			continue
		}
		if !strings.Contains(string(out), c.want) {
			t.Errorf("%s: want %q, got:\n%s", c.why, c.want, out)
		}
	}
}
