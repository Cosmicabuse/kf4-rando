// Command logiccheck prints the sphere-by-sphere progression without building
// an ISO. With no -iso it reports the unmodified game; with one it runs a fill
// against the real maps, checks what it wrote, and reports that instead.
package main

import (
	"flag"
	"fmt"
	"os"
	"regexp"
	"strings"

	"kf4randomizer/internal/features/randomize"
	"kf4randomizer/internal/isopatch"
	"kf4randomizer/internal/kfconst"
	"kf4randomizer/internal/kfmap"
	"kf4randomizer/internal/logic"
	"kf4randomizer/internal/report"
	"kf4randomizer/internal/resources"
)

var mapNameRE = regexp.MustCompile(`^\d{3}\.map$`)

func main() {
	iso := flag.String("iso", "", "source ISO to fill against; omit to report the unmodified game")
	seed := flag.Uint64("seed", 1, "fill seed")
	ground := flag.Bool("ground", true, "shuffle ground items")
	containers := flag.Bool("containers", true, "shuffle containers")
	drops := flag.Bool("drops", true, "shuffle monster drops")
	idol := flag.Bool("idol", true, "shuffle the Idol of Sorrow")
	shopsOn := flag.Bool("shops", false, "put the gold shops in the pool")
	out := flag.String("o", "", "write the report to this file instead of stdout")
	flag.Parse()
	pool := logic.Pool{Ground: *ground, Containers: *containers, Drops: *drops, Idol: *idol, Shops: *shopsOn}

	names := kfmap.VanillaItemNames
	mapNames, err := kfmap.LoadMapNames(resources.MapNamesCSV)
	if err != nil {
		fail(err)
	}

	var placements []logic.Placement
	if *iso != "" {
		maps, err := loadMaps(*iso)
		if err != nil {
			fail(err)
		}
		stock := make([]byte, kfconst.ShopCount*kfconst.ShopStockSize)
		res, ps, err := randomize.RunLogic(maps, make([]byte, 269*0x450), stock, randomize.NewRand(*seed), pool)
		if err != nil {
			fail(fmt.Errorf("fill: %w", err))
		}
		placements = ps
		fmt.Printf("seed %d: %d maps, %d slots written, %d changed\n",
			*seed, res.TotalMaps, res.EligibleSlots, len(res.Changes))
		if pool.Shops {
			for shop := range kfconst.ShopGoldCount {
				var sold []string
				for id := 1; id < kfconst.ShopStockSize; id++ {
					if stock[shop*kfconst.ShopStockSize+id] != 0 {
						sold = append(sold, names.NameOrDefault(uint16(id), fmt.Sprint(id)))
					}
				}
				fmt.Printf("shop %d: %s\n", shop, strings.Join(sold, ", "))
			}
		}
		checkWrites(maps, names)
	}

	text := report.Spheres(placements, pool, names, mapNames)
	if *out == "" {
		fmt.Print(text)
		return
	}
	if err := os.WriteFile(*out, []byte(text), 0o644); err != nil {
		fail(err)
	}
	fmt.Printf("wrote %s\n", *out)
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}

func loadMaps(path string) ([]*kfmap.Map, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	dat, err := isopatch.FindPath(f, kfconst.IsoDatPath)
	if err != nil {
		return nil, err
	}
	table, err := isopatch.ReadDatFileTable(f, dat.ByteOffset())
	if err != nil {
		return nil, err
	}
	read := func(e isopatch.DatEntry) ([]byte, error) {
		buf := make([]byte, e.Length)
		_, err := f.ReadAt(buf, e.AbsOffset)
		return buf, err
	}

	var params kfmap.ObjectParams
	if e, ok := table["param/obj.prm"]; ok {
		if params, err = read(e); err != nil {
			return nil, err
		}
	}
	var maps []*kfmap.Map
	for _, e := range table {
		if !mapNameRE.MatchString(e.Name) {
			continue
		}
		buf, err := read(e)
		if err != nil {
			return nil, err
		}
		m, err := kfmap.Load(e.Name, buf, params)
		if err != nil {
			return nil, err
		}
		maps = append(maps, m)
	}
	return maps, nil
}

// checkWrites re-reads the maps: shared slots hold one item, fixed slots are
// untouched.
func checkWrites(maps []*kfmap.Map, names kfmap.ItemNames) {
	byName := map[string]*kfmap.Map{}
	for _, m := range maps {
		byName[m.Name] = m
	}
	read := func(region string, slot logic.Slot, idx int) (uint16, bool) {
		m := byName[logic.MapFile(region)]
		if m == nil {
			return 0, false
		}
		switch slot {
		case logic.Ground:
			for _, it := range m.Items {
				if it.Index == idx {
					return it.ClassID, true
				}
			}
		case logic.Container:
			for _, c := range m.Containers {
				if c.Index == idx {
					return m.ContainerItemClassID(c), true
				}
			}
		case logic.MapDrop:
			for _, c := range m.Creatures {
				if c.Index == idx {
					return m.CreatureDropClassID(c), true
				}
			}
		}
		return 0, false
	}

	w, _ := logic.Default()
	groups := map[string][]logic.Check{}
	bad, fixedOK := 0, 0
	for _, c := range w.Checks {
		if c.Dupe != "" {
			groups[c.Dupe] = append(groups[c.Dupe], c)
		}
		if c.Fill != logic.Fixed {
			continue
		}
		got, ok := read(c.Region, c.Slot, c.Index)
		if !ok {
			continue
		}
		if got != c.Item {
			fmt.Printf("  FIXED CHANGED %s: %s -> %s\n", c.ID(),
				names.NameOrDefault(c.Item, "?"), names.NameOrDefault(got, "?"))
			bad++
			continue
		}
		fixedOK++
	}
	for dupe, cs := range groups {
		var first uint16
		for i, c := range cs {
			got, ok := read(c.Region, c.Slot, c.Index)
			if !ok {
				continue
			}
			if i == 0 {
				first = got
			} else if got != first {
				fmt.Printf("  DUPE SPLIT %s: %s has %s, want %s\n", dupe, c.ID(),
					names.NameOrDefault(got, "?"), names.NameOrDefault(first, "?"))
				bad++
			}
		}
	}
	fmt.Printf("checked %d duplicate groups, %d fixed slots untouched, %d problems\n",
		len(groups), fixedOK, bad)
}
