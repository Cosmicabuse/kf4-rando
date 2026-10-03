// Package containers moves every empty barrel, crate and pot far out of its
// map, so a container still visible in game is one that holds something.
package containers

import (
	"fmt"
	"sort"

	"kf4randomizer/internal/kfmap"
)

type Change struct {
	MapName string
	Index   int
	ClassID uint16
}

type Result struct {
	TotalMaps       int
	TotalContainers int            // containers seen, empty or not
	RemovedPerMap   map[string]int // map name -> number of containers removed
	Removed         []Change
	KeptScripted    int // empty containers left alone because a map script drives them
	KeptAddressable int // empty containers a map script can address by event ID
	KeptStacked     int // empty containers sharing a spot with a container that holds an item
}

func RemoveEmpty(maps []*kfmap.Map) Result {
	sort.Slice(maps, func(i, j int) bool { return maps[i].Name < maps[j].Name })

	// Some container classes are scenery (planks, signs), so only classes that
	// hold an item somewhere get moved.
	carriesItem := make(map[uint16]bool)
	for _, m := range maps {
		for _, c := range m.Containers {
			if c.ItemClassID != 0 {
				carriesItem[c.ClassID] = true
			}
		}
	}

	removedPerMap := make(map[string]int)
	var removed []Change
	total, kept, keptAddressable, keptStacked := 0, 0, 0, 0
	for _, m := range maps {
		total += len(m.Containers)
		for _, c := range m.Containers {
			if c.ItemClassID != 0 || !carriesItem[c.ClassID] {
				continue
			}
			// script-driven lids are puzzle switches
			if c.ScriptRef != 0 {
				kept++
				continue
			}
			// a script can target it by event ID
			if c.EventID != 0 {
				keptAddressable++
				continue
			}
			if m.StackedOnFilledContainer(c) {
				keptStacked++
				continue
			}
			// it delivers the ground item it sits on (forge chests)
			if m.PairedWithItem(c) {
				continue
			}
			m.RelocateContainerOutOfBounds(c)
			removedPerMap[m.Name]++
			removed = append(removed, Change{MapName: m.Name, Index: c.Index, ClassID: c.ClassID})
		}
	}

	return Result{TotalMaps: len(maps), TotalContainers: total, RemovedPerMap: removedPerMap, Removed: removed, KeptScripted: kept,
		KeptAddressable: keptAddressable, KeptStacked: keptStacked}
}

// Summary formats a short report. mapNames may be nil.
func (r Result) Summary(mapNames kfmap.MapNames) string {
	s := fmt.Sprintf("Removed %[3]d empty containers out of %[1]d, %[2]d maps", r.TotalContainers, r.TotalMaps, len(r.Removed))
	if r.KeptScripted > 0 {
		s += fmt.Sprintf("; kept %d driven by a map script", r.KeptScripted)
	}
	if r.KeptAddressable > 0 {
		s += fmt.Sprintf("; kept %d a script can address by event ID", r.KeptAddressable)
	}
	if r.KeptStacked > 0 {
		s += fmt.Sprintf("; kept %d sharing a spot with a filled container", r.KeptStacked)
	}
	s += ".\n"
	names := make([]string, 0, len(r.RemovedPerMap))
	for name := range r.RemovedPerMap {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		s += fmt.Sprintf("%s: %d empty container(s) removed\n", mapNames.Header(name), r.RemovedPerMap[name])
	}
	return s
}
