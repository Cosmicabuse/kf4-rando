// Package startingconfig edits ini/0611.ini, the New Game starting character
package startingconfig

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"

	"kf4randomizer/internal/resources"
)

var BaseINI = resources.StartingConfigBaseINI

// key=integer at line start
func keyValueRe(key string) *regexp.Regexp {
	return regexp.MustCompile(`(?m)^(` + regexp.QuoteMeta(key) + `=)(-?\d+)`)
}

// Equipment slots, each taking an item classID.
var equipKeys = map[string]bool{
	"equip_weapon": true, "equip_shield": true, "equip_helm": true,
	"equip_armor": true, "equip_guntlet": true, "equip_boot": true,
	"equip_ring": true, "equip_bracelet": true, "equip_amulet": true,
}

// ItemQuantity is how many of classID the ini grants.
func ItemQuantity(ini []byte, classID int) int {
	m := regexp.MustCompile(
		`(?m)^item_0*` + fmt.Sprint(classID) + `=(-?\d+)`).FindSubmatch(ini)
	if m == nil {
		return 0
	}
	n, err := strconv.Atoi(string(m[1]))
	if err != nil {
		return 0
	}
	return n
}

// grantEquipped adds one of anything an equip slot points at that the
// inventory does not already hold.
func grantEquipped(ini []byte) []byte {
	slots := make([]string, 0, len(equipKeys))
	for k := range equipKeys {
		slots = append(slots, k)
	}
	sort.Strings(slots)

	for _, key := range slots {
		m := keyValueRe(key).FindSubmatch(ini)
		if m == nil {
			continue
		}
		classID, err := strconv.Atoi(string(m[2]))
		if err != nil || classID < 0 {
			continue // -1 is "nothing equipped"
		}
		if ItemQuantity(ini, classID) > 0 {
			continue
		}
		re := keyValueRe(fmt.Sprintf("item_%d", classID))
		if re.Match(ini) {
			ini = re.ReplaceAll(ini, []byte("${1}1"))
		}
	}
	return ini
}

// ApplyOverrides rewrites the given keys and leaves the rest alone. An unknown
// key is an error, it's usually a typo. Equipped items are also granted,
// equipping something you don't have crashes the menu.
func ApplyOverrides(base []byte, overrides map[string]int) ([]byte, error) {
	out := append([]byte(nil), base...)

	// Sort for deterministic error ordering.
	keys := make([]string, 0, len(overrides))
	for k := range overrides {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, key := range keys {
		re := keyValueRe(key)
		if !re.Match(out) {
			return nil, fmt.Errorf("starting_config: ini key %q not found in 0611.ini", key)
		}
		replacement := fmt.Sprintf("${1}%d", overrides[key])
		out = re.ReplaceAll(out, []byte(replacement))
	}

	return grantEquipped(out), nil
}
