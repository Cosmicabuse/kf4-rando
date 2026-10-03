// Package mods holds the QoL patch catalog, one choice per category.
// catalog_data.go is generated from qol.pnach by cmd/gencatalog.
package mods

import "fmt"

// Patch is one raw memory poke: Value at Address, a SLUS_203.18 vaddr.
type Patch struct {
	Address uint32
	Value   uint32
}

// A Mode's patches apply in order, later ones can overwrite earlier ones.
type Mode struct {
	ID      string
	Label   string
	Notes   string
	Patches []Patch

	// full_analog_y_axis needs full_analog_controls, for example
	Requires []string
}

// Category is a set of Modes where at most one may be enabled at once.
type Category struct {
	ID          string
	Description string
	Modes       []Mode
}

// Selection is the mods: list in config.yaml, one `mode_id: true/false` each,
// so two modes of one category both set to true can be caught.
type Selection []map[string]bool

// Resolve validates a Selection and returns the enabled modes in category
// order, since a later category can overwrite an earlier one's addresses.
func Resolve(categories []Category, sel Selection) ([]Mode, error) {
	enabled := make(map[string]bool)
	seen := make(map[string]bool)
	for _, entry := range sel {
		for id, isEnabled := range entry {
			if seen[id] {
				return nil, fmt.Errorf("mods: %q is listed more than once", id)
			}
			seen[id] = true
			if isEnabled {
				enabled[id] = true
			}
		}
	}

	for id := range seen {
		if _, _, ok := findMode(categories, id); !ok {
			return nil, fmt.Errorf("mods: unknown mode ID %q, check it against -list_mods", id)
		}
	}

	var resolved []Mode
	for _, cat := range categories {
		var picked *Mode
		for i := range cat.Modes {
			m := &cat.Modes[i]
			if !enabled[m.ID] {
				continue
			}
			if picked != nil {
				return nil, fmt.Errorf(
					"category %q allows only one enabled mode at a time, but your mods config has both %q and %q set to true (%s). Set one of them back to false.",
					cat.ID, picked.ID, m.ID, cat.Description)
			}
			picked = m
		}
		if picked != nil {
			for _, reqID := range picked.Requires {
				if !enabled[reqID] {
					return nil, fmt.Errorf("mods: %q requires %q to also be set to true", picked.ID, reqID)
				}
			}
			resolved = append(resolved, *picked)
		}
	}

	return resolved, nil
}

func findMode(categories []Category, modeID string) (Category, Mode, bool) {
	for _, cat := range categories {
		for _, m := range cat.Modes {
			if m.ID == modeID {
				return cat, m, true
			}
		}
	}
	return Category{}, Mode{}, false
}
