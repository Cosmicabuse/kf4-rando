// Package itemrespawn makes ground items one-time pickups. Some respawned on
// room load in vanilla, which breaks tracking what was collected.
//
// Slots with chance 0 are spawned by events (NPC death, quests) and left
// alone, or the Altar Key, Widda Key and Moonlight Sword would just lie there.
package itemrespawn

import (
	"fmt"
	"sort"

	"kf4randomizer/internal/kfmap"
)

type Result struct {
	TotalMaps   int
	TotalItems  int
	EventPlaced int // left alone because their spawn is event-driven
}

func (r Result) Summary() string {
	return fmt.Sprintf("Normalized respawn behavior on %d ground items across %d maps (one-time pickup, no reappearance roll); left %d event-placed slots untouched.\n",
		r.TotalItems, r.TotalMaps, r.EventPlaced)
}

func Normalize(maps []*kfmap.Map) Result {
	sort.Slice(maps, func(i, j int) bool { return maps[i].Name < maps[j].Name })

	total, eventPlaced := 0, 0
	for _, m := range maps {
		for i := range m.Items {
			if m.Items[i].RespawnChance == 0 {
				eventPlaced++
				continue
			}
			m.SetItemRespawnOnce(i)
			total++
		}
	}

	return Result{TotalMaps: len(maps), TotalItems: total, EventPlaced: eventPlaced}
}
