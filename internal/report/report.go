// Package report renders the text files a run produces: the spoiler log, the
// sphere-by-sphere progression, and the raw data dump behind debug_data.
package report

import (
	"fmt"
	"sort"
	"strings"

	"kf4randomizer/internal/features/randomize"
	"kf4randomizer/internal/features/shops"
	"kf4randomizer/internal/kfmap"
	"kf4randomizer/internal/logic"
)

// Spoiler lists every slot whose item changed, grouped by map then by kind.
// mapNames may be nil.
func Spoiler(seed uint64, result randomize.Result, names kfmap.ItemNames, mapNames kfmap.MapNames) string {
	type section struct {
		ground    []randomize.Change
		container []randomize.Change
		mapDrop   []randomize.Change
		prmDrop   []randomize.Change
	}

	var b strings.Builder
	fmt.Fprintf(&b, "King's Field IV item randomizer: spoiler log\n")
	fmt.Fprintf(&b, "Seed: %d\n", seed)
	fmt.Fprintf(&b, "%d maps, %d slots randomized\n", result.TotalMaps, result.EligibleSlots)
	if result.StartItem != 0 {
		fmt.Fprintf(&b, "Starting inventory: %s replaces Idol of Sorrow\n", names.NameOrDefault(result.StartItem, "(unknown)"))
	}
	b.WriteString("\n")

	sections := map[string]*section{}
	var order []string
	for _, c := range result.Changes {
		s := sections[c.MapName]
		if s == nil {
			s = &section{}
			sections[c.MapName] = s
			order = append(order, c.MapName)
		}
		switch c.Kind {
		case randomize.ContainerItem:
			s.container = append(s.container, c)
		case randomize.MapDrop:
			s.mapDrop = append(s.mapDrop, c)
		case randomize.PrmDrop:
			s.prmDrop = append(s.prmDrop, c)
		default:
			s.ground = append(s.ground, c)
		}
	}
	sort.Strings(order)

	for i, name := range order {
		if i > 0 {
			b.WriteString("\n")
		}
		fmt.Fprintf(&b, "=== %s ===\n", mapNames.Header(name))

		s := sections[name]
		for _, group := range []struct {
			heading string
			label   string
			changes []randomize.Change
		}{
			{"Ground items", "slot", s.ground},
			{"Containers", "container", s.container},
			{"Monster map drops", "monster", s.mapDrop},
			{"Monster prm drops", "class", s.prmDrop},
		} {
			if len(group.changes) == 0 {
				continue
			}
			fmt.Fprintf(&b, "  %s:\n", group.heading)
			for _, c := range group.changes {
				fmt.Fprintf(&b, "    %s %2d: %s -> %s\n", group.label, c.Index,
					names.NameOrDefault(c.FromClassID, "(unknown)"),
					names.NameOrDefault(c.ToClassID, "(unknown)"))
			}
		}
	}

	return b.String()
}

// Shops lists everything each gold shop sells, changed or not.
func Shops(res shops.Result, names kfmap.ItemNames) string {
	var b strings.Builder
	b.WriteString("\n=== Shops ===\n")
	for i, sales := range res.Shops {
		fmt.Fprintf(&b, "  Shop %d:\n", i)
		for _, s := range sales {
			qty := fmt.Sprint(s.Quantity)
			if s.Quantity == 255 {
				qty = "unlimited"
			}
			fmt.Fprintf(&b, "    %-20s %5d gold  x%s\n", names.NameOrDefault(s.ID, "(unknown)"), s.Buy, qty)
		}
	}
	return b.String()
}

// Spheres writes what each sphere opens and holds. Nil placements report the
// unmodified game.
func Spheres(placements []logic.Placement, pool logic.Pool, names kfmap.ItemNames, mapNames kfmap.MapNames) string {
	w, start := logic.Default()
	if placements != nil {
		// Placements name slots as the randomizer leaves them, not as the
		// disc has them.
		w, start = logic.ForSeed(pool)
		w = w.Apply(placements)
	}
	res := w.Solve(start)

	var b strings.Builder
	total := len(res.Unreached)
	for _, s := range res.Spheres {
		total += len(s.Checks)
	}
	fmt.Fprintf(&b, "%d checks, %d reachable, %d spheres\n\n",
		total, total-len(res.Unreached), len(res.Spheres))

	for i, s := range res.Spheres {
		var opened []string
		if i > 0 {
			opened = addedRegions(res.Spheres[i-1].Regions, s.Regions)
		}
		fmt.Fprintf(&b, "%s\n=== SPHERE %d: %d checks, %d areas reachable",
			strings.Repeat("-", 72), i, len(s.Checks), len(s.Regions))
		if len(opened) > 0 {
			fmt.Fprintf(&b, ", %d new", len(opened))
		}
		b.WriteString("\n")
		if len(opened) > 0 {
			labels := make([]string, len(opened))
			for j, r := range opened {
				labels[j] = regionLabel(r, mapNames)
			}
			fmt.Fprintf(&b, "opens: %s\n", strings.Join(labels, ", "))
		}
		for _, line := range unlockLines(s.Opened, names) {
			fmt.Fprintf(&b, "using: %s\n", line)
		}
		b.WriteString("\n")

		byRegion := map[string][]logic.Check{}
		var order []string
		for _, c := range s.Checks {
			if _, ok := byRegion[c.Region]; !ok {
				order = append(order, c.Region)
			}
			byRegion[c.Region] = append(byRegion[c.Region], c)
		}
		sort.Slice(order, func(a, b int) bool { return regionLess(order[a], order[b]) })

		for _, region := range order {
			cs := byRegion[region]
			sort.Slice(cs, func(a, b int) bool {
				if cs[a].Slot != cs[b].Slot {
					return cs[a].Slot < cs[b].Slot
				}
				return cs[a].Index < cs[b].Index
			})
			fmt.Fprintf(&b, "  %s\n", regionLabel(region, mapNames))
			for _, c := range cs {
				note := ""
				if d := c.Need.Describe(names); d != "" {
					note = "   [" + d + "]"
				}
				if c.Fill == logic.Fixed {
					note += "   (not randomized)"
				}
				fmt.Fprintf(&b, "      %-10s %-3d  %-18s%s\n",
					c.Slot, c.Index, names.NameOrDefault(c.Item, unknownItem(c.Item)), note)
			}
		}
		b.WriteString("\n")
	}

	if len(res.Unreached) > 0 {
		fmt.Fprintf(&b, "%s\n=== NEVER REACHED (%d)\n\n", strings.Repeat("-", 72), len(res.Unreached))
		for _, c := range res.Unreached {
			fmt.Fprintf(&b, "  %-22s %-18s needs %s\n", regionLabel(c.Region, mapNames),
				names.NameOrDefault(c.Item, unknownItem(c.Item)), c.Need.Describe(names))
		}
	}
	return b.String()
}

func unknownItem(id uint16) string { return fmt.Sprintf("item %d", id) }

// unlockLines names what paid for each gate this sphere opened, one line per
// requirement with the crossings it covers.
func unlockLines(gates []logic.Edge, names kfmap.ItemNames) []string {
	crossings := map[string][]string{}
	var order []string
	for _, g := range gates {
		req := g.Need.Describe(names)
		if req == "" {
			continue
		}
		if _, ok := crossings[req]; !ok {
			order = append(order, req)
		}
		from, fromSuffix := splitRegion(g.From)
		to, toSuffix := splitRegion(g.To)
		crossings[req] = append(crossings[req],
			fmt.Sprintf("%03d%s->%03d%s", from, fromSuffix, to, toSuffix))
	}
	sort.Strings(order)

	out := make([]string, 0, len(order))
	for _, req := range order {
		out = append(out, fmt.Sprintf("%-26s %s", req, strings.Join(crossings[req], ", ")))
	}
	return out
}

func addedRegions(before, after []string) []string {
	had := map[string]bool{}
	for _, r := range before {
		had[r] = true
	}
	var out []string
	for _, r := range after {
		if !had[r] {
			out = append(out, r)
		}
	}
	sort.Slice(out, func(a, b int) bool { return regionLess(out[a], out[b]) })
	return out
}

// regionLabel turns "16B" into "016B South Cliffs". A letter suffix is a slice
// of one map, so the name comes off the number.
func regionLabel(region string, mapNames kfmap.MapNames) string {
	id, suffix := splitRegion(region)
	label := fmt.Sprintf("%03d%s", id, suffix)
	if name := mapNames[logic.MapFile(region)]; name != "" {
		return label + " " + name
	}
	return label
}

func regionLess(a, b string) bool {
	ida, sa := splitRegion(a)
	idb, sb := splitRegion(b)
	if ida != idb {
		return ida < idb
	}
	return sa < sb
}

func splitRegion(region string) (int, string) {
	n := 0
	for n < len(region) && region[n] >= '0' && region[n] <= '9' {
		n++
	}
	id := 0
	for _, c := range region[:n] {
		id = id*10 + int(c-'0')
	}
	return id, region[n:]
}

// DebugDump lists every known item, then every map's item, container and drop
// slots. mapNames may be nil.
func DebugDump(names kfmap.ItemNames, maps []*kfmap.Map, mapNames kfmap.MapNames) string {
	var b strings.Builder

	b.WriteString("=== All Items ===\n")
	ids := make([]uint16, 0, len(names))
	for id, name := range names {
		if name != "" {
			ids = append(ids, id)
		}
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	for _, id := range ids {
		fmt.Fprintf(&b, "%4d: %s\n", id, names[id])
	}

	sorted := make([]*kfmap.Map, len(maps))
	copy(sorted, maps)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Name < sorted[j].Name })

	b.WriteString("\n=== Maps ===\n")
	for _, m := range sorted {
		fmt.Fprintf(&b, "\n=== %s ===\n", mapNames.Header(m.Name))
		for _, item := range m.Items {
			fmt.Fprintf(&b, "  slot %2d: id %4d: %s\n", item.Index, item.ClassID, slotItemName(names, item.ClassID))
		}
		for _, c := range m.Containers {
			if c.ItemClassID == 0 {
				continue
			}
			fmt.Fprintf(&b, "  container obj %3d (classID %3d): id %4d: %s\n",
				c.Index, c.ClassID, c.ItemClassID, slotItemName(names, c.ItemClassID))
		}
		for _, c := range m.Creatures {
			fmt.Fprintf(&b, "  monster drop entity %3d: id %4d: %s\n",
				c.Index, c.DropClassID, slotItemName(names, c.DropClassID))
		}
	}

	return b.String()
}

func slotItemName(names kfmap.ItemNames, classID uint16) string {
	if classID == 0 {
		return "(empty)"
	}
	return names.NameOrDefault(classID, "(unknown)")
}
