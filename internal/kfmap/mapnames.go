package kfmap

import (
	"encoding/csv"
	"fmt"
	"strings"
)

// MapNames is a .map filename -> human display name lookup, parsed from
// map_names.csv by LoadMapNames.
type MapNames map[string]string

// Header formats mapName for display, appending its human name in
// parentheses when known. A nil MapNames returns mapName unchanged.
func (n MapNames) Header(mapName string) string {
	if dn, ok := n[mapName]; ok && dn != "" {
		return fmt.Sprintf("%s (%s)", mapName, dn)
	}
	return mapName
}

// LoadMapNames parses a two-column "game_name,human_name" CSV into a
// lookup map, skipping the header row.
func LoadMapNames(data []byte) (MapNames, error) {
	r := csv.NewReader(strings.NewReader(string(data)))
	records, err := r.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("parsing map names CSV: %w", err)
	}

	names := make(MapNames, len(records))
	for i, rec := range records {
		// header row
		if i == 0 {
			continue
		}
		names[rec[0]] = rec[1]
	}
	return names, nil
}
