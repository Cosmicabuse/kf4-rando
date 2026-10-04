package runner

import (
	"fmt"
	"io"
	"maps"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"kf4randomizer/internal/features/containers"
	"kf4randomizer/internal/features/itemmodel"
	"kf4randomizer/internal/features/itemrespawn"
	"kf4randomizer/internal/features/mods"
	"kf4randomizer/internal/features/paramdomizer"
	"kf4randomizer/internal/features/randomize"
	"kf4randomizer/internal/features/shops"
	"kf4randomizer/internal/features/startingconfig"
	"kf4randomizer/internal/features/titlelogo"
	"kf4randomizer/internal/isopatch"
	"kf4randomizer/internal/kfconst"
	"kf4randomizer/internal/kfmap"
	"kf4randomizer/internal/logic"
	"kf4randomizer/internal/report"
	"kf4randomizer/internal/resources"
)

var mapNameRE = regexp.MustCompile(`(?i)^\d{3}\.map$`)

const (
	itemNamDatPath     = "param/item_nam.prm"
	objDatPath         = "param/obj.prm"
	chrDatPath         = "param/chr.prm"
	itemWpnDatPath     = "param/item_wpn.prm"
	itemPctDatPath     = "param/item_pct.prm"
	startingIniDatPath = "ini/0611.ini"
	titleLogo1DatPath  = "title/title_logo1.tm2"
	titleLogo2DatPath  = "title/title_logo2.tm2"
)

// gameData is everything both subcommands need: parsed maps, the DAT's file
// table, and display-name lookups for reporting.
type gameData struct {
	maps       []*kfmap.Map
	mapEntries []isopatch.DatEntry // parallel to maps, same order
	dat        map[string]isopatch.DatEntry
	itemNames  kfmap.ItemNames // nil if item_nam.prm wasn't found
	mapNames   kfmap.MapNames
}

// read pulls one DAT member's bytes out of f.
func (d *gameData) read(f *os.File, path string) ([]byte, isopatch.DatEntry, error) {
	entry, ok := d.dat[path]
	if !ok {
		return nil, entry, fmt.Errorf("%s not found in DAT", path)
	}
	buf := make([]byte, entry.Length)
	if _, err := f.ReadAt(buf, entry.AbsOffset); err != nil {
		return nil, entry, fmt.Errorf("reading %s: %w", path, err)
	}
	return buf, entry, nil
}

// loadGameData locates KF4.DAT inside f (an already-open ISO) and loads
// every map file plus item_nam.prm's name table out of it.
func loadGameData(w io.Writer, f *os.File) (*gameData, error) {
	datEntry, err := isopatch.FindPath(f, kfconst.IsoDatPath)
	if err != nil {
		return nil, fmt.Errorf("locating KF4.DAT: %w", err)
	}
	datAbsOffset := datEntry.ByteOffset()

	_, _ = fmt.Fprintf(w, "Reading KF4.DAT file table at ISO offset 0x%X ...\n", datAbsOffset)
	fileTable, err := isopatch.ReadDatFileTable(f, datAbsOffset)
	if err != nil {
		return nil, fmt.Errorf("reading DAT file table: %w", err)
	}
	_, _ = fmt.Fprintf(w, "  %d entries in embedded DAT.\n", len(fileTable))

	data := &gameData{dat: fileTable}

	objParams, _, err := data.read(f, objDatPath)
	if err != nil {
		_, _ = fmt.Fprintf(w, "!! %v, containers won't be detected\n", err)
	}

	for _, entry := range fileTable {
		if !mapNameRE.MatchString(entry.Name) {
			continue
		}
		buf := make([]byte, entry.Length)
		if _, err := f.ReadAt(buf, entry.AbsOffset); err != nil {
			return nil, fmt.Errorf("reading %s from DAT: %w", entry.Name, err)
		}
		m, err := kfmap.Load(entry.Name, buf, objParams)
		if err != nil {
			return nil, fmt.Errorf("parsing %s: %w", entry.Name, err)
		}
		data.maps = append(data.maps, m)
		data.mapEntries = append(data.mapEntries, entry)
	}

	if itemNamBuf, _, err := data.read(f, itemNamDatPath); err != nil {
		_, _ = fmt.Fprintf(w, "!! %v, item names won't be available\n", err)
	} else {
		data.itemNames = kfmap.LoadClassIDToName(itemNamBuf)
	}

	data.mapNames, err = kfmap.LoadMapNames(resources.MapNamesCSV)
	if err != nil {
		return nil, fmt.Errorf("loading map names: %w", err)
	}
	return data, nil
}

// validateInput checks Input's MD5 against the original ISO, and
// that ini/0611.ini exists in its DAT, before any byte of Output is touched.
func validateInput(inputPath string) error {
	f, err := os.Open(inputPath)
	if err != nil {
		return fmt.Errorf("opening input ISO: %w", err)
	}
	defer func() {
		_ = f.Close()
	}()

	sum, err := isopatch.MD5(f)
	if err != nil {
		return fmt.Errorf("computing input ISO MD5: %w", err)
	}
	if !strings.EqualFold(sum, kfconst.KnownGoodISOMD5) {
		return fmt.Errorf("input ISO MD5 is %s, expected %s (King's Field IV, USA), other versions aren't supported", sum, kfconst.KnownGoodISOMD5)
	}

	datEntry, err := isopatch.FindPath(f, kfconst.IsoDatPath)
	if err != nil {
		return fmt.Errorf("locating KF4.DAT: %w", err)
	}
	fileTable, err := isopatch.ReadDatFileTable(f, datEntry.ByteOffset())
	if err != nil {
		return fmt.Errorf("reading DAT file table: %w", err)
	}
	if _, ok := fileTable[startingIniDatPath]; !ok {
		return fmt.Errorf("%s wasn't found in this ISO's DAT", startingIniDatPath)
	}
	return nil
}

// RunDebugData reads inputPath read-only and dumps raw item/map data into
// outDir/items_and_maps.txt. No ISO is written.
func RunDebugData(w io.Writer, inputPath, outDir string) error {
	if err := validateInput(inputPath); err != nil {
		return fmt.Errorf("input ISO failed a pre-flight check: %w", err)
	}

	f, err := os.Open(inputPath)
	if err != nil {
		return fmt.Errorf("opening input ISO: %w", err)
	}
	defer func() {
		_ = f.Close()
	}()

	data, err := loadGameData(w, f)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return fmt.Errorf("creating %s: %w", outDir, err)
	}

	path := filepath.Join(outDir, "items_and_maps.txt")
	dump := report.DebugDump(data.itemNames, data.maps, data.mapNames)
	if err := os.WriteFile(path, []byte(dump), 0o644); err != nil {
		return err
	}

	_, _ = fmt.Fprintf(w, "Wrote debug data to %s\n", path)
	return nil
}

// Run copies cfg.Input to cfg.Output, patches SLUS_203.18, applies mods,
// shuffles item classIDs, and writes the randomized maps + spoiler log.
func Run(w io.Writer, cfg *Config) error {
	resolvedMods, err := mods.Resolve(mods.BuiltinCategories, cfg.Mods)
	if err != nil {
		return fmt.Errorf("mods selection error: %w", err)
	}

	// Always vanilla + overrides, never the input ISO's own ini as-is.
	newStartingIni, err := startingconfig.ApplyOverrides(startingconfig.BaseINI, cfg.StartingConfig)
	if err != nil {
		return err
	}

	if err := validateInput(cfg.Input); err != nil {
		return fmt.Errorf("input ISO failed a pre-flight check: %w", err)
	}

	if cfg.StripDummyFiles {
		_, _ = fmt.Fprintf(w, "Copying %s -> %s (stripping DUMMY*.OUT padding files) ...\n", cfg.Input, cfg.Output)
		if err := isopatch.StripDummyFiles(cfg.Input, cfg.Output); err != nil {
			return fmt.Errorf("stripping dummy files: %w", err)
		}
	} else {
		_, _ = fmt.Fprintf(w, "Copying %s -> %s ...\n", cfg.Input, cfg.Output)
		if err := isopatch.CopyFile(cfg.Input, cfg.Output); err != nil {
			return fmt.Errorf("copying ISO: %w", err)
		}
	}

	f, err := os.OpenFile(cfg.Output, os.O_RDWR, 0)
	if err != nil {
		return fmt.Errorf("opening output ISO: %w", err)
	}
	defer func() {
		_ = f.Close()
	}()

	slusEntry, err := isopatch.FindPath(f, []string{kfconst.IsoExecutableName})
	if err != nil {
		return fmt.Errorf("locating SLUS_203.18: %w", err)
	}

	if cfg.DisableRandomizer {
		_, _ = fmt.Fprintln(w, "Randomizer disabled, skipping the whitelist-bypass patch")
	} else {
		if err := isopatch.PatchWhitelistBypass(f, slusEntry.ByteOffset()); err != nil {
			return fmt.Errorf("patching SLUS_203.18: %w", err)
		}
		_, _ = fmt.Fprintln(w, "Patched SLUS_203.18 whitelist bypass")
	}

	// always, see kfconst.SlusBlackVoidFixVAddr
	if err := isopatch.PatchBlackVoidFix(f, slusEntry.ByteOffset()); err != nil {
		return err
	}

	if cfg.WandsAnywhere {
		if err := isopatch.WriteSlusWord(f, slusEntry.ByteOffset(), kfconst.SlusWandPadCheckVAddr, kfconst.SlusWandPadCheckSkip); err != nil {
			return fmt.Errorf("letting wands work anywhere: %w", err)
		}
		_, _ = fmt.Fprintln(w, "Wands now work anywhere, not just on teleport pads")
	}

	if err := isopatch.PatchPickupPopup(f, slusEntry.ByteOffset(), cfg.PickupPopupShowItemIndex); err != nil {
		return fmt.Errorf("patching pickup-name popup: %w", err)
	}

	if cfg.FullDebugMode {
		if err := isopatch.PatchDebugHUD(f, slusEntry.ByteOffset()); err != nil {
			return fmt.Errorf("patching debug HUD: %w", err)
		}
		_, _ = fmt.Fprintln(w, "Patched SLUS_203.18 debug HUD (L3+R3 to toggle)")
	}

	for _, p := range mods.AlwaysOnPatches {
		if err := isopatch.WriteSlusWord(f, slusEntry.ByteOffset(), p.Address, p.Value); err != nil {
			return fmt.Errorf("applying always-on patch: %w", err)
		}
	}

	for _, m := range resolvedMods {
		for _, p := range m.Patches {
			if err := isopatch.WriteSlusWord(f, slusEntry.ByteOffset(), p.Address, p.Value); err != nil {
				return fmt.Errorf("applying mod %q: %w", m.ID, err)
			}
		}
		_, _ = fmt.Fprintf(w, "Applied mod: %s (%s)\n", m.ID, m.Label)
	}

	data, err := loadGameData(w, f)
	if err != nil {
		return err
	}

	iniEntry, ok := data.dat[startingIniDatPath]
	if !ok {
		panic(fmt.Sprintf("%s missing from Output's DAT despite passing the input pre-flight check", startingIniDatPath))
	}
	// only when something is overridden, no need to rewrite the ini otherwise
	if len(cfg.StartingConfig) > 0 {
		if err := isopatch.WriteDatMemberResized(f, iniEntry, newStartingIni); err != nil {
			return fmt.Errorf("applying starting_config to %s: %w", startingIniDatPath, err)
		}
		_, _ = fmt.Fprintf(w, "Applied %d starting_config override(s) to %s\n", len(cfg.StartingConfig), startingIniDatPath)
	}

	if !cfg.DisableRandomizer {
		if err := applyTitleBranding(w, f, data); err != nil {
			return err
		}
	}

	if cfg.RemoveEmptyContainers {
		containerResult := containers.RemoveEmpty(data.maps)
		_, _ = fmt.Fprint(w, containerResult.Summary(data.mapNames))
	}

	// always on, it fixes a vanilla bug
	respawnResult := itemrespawn.Normalize(data.maps)
	_, _ = fmt.Fprint(w, respawnResult.Summary())
	if n := itemrespawn.MoveStuck(data.maps); n > 0 {
		_, _ = fmt.Fprintf(w, "Moved %d ground item(s) out of spots where they can't be picked up\n", n)
	}

	if !cfg.DisableRandomizer || cfg.EnableParamdomizer || cfg.RandomizeShops {
		if err := randomizeAll(w, cfg, f, slusEntry.ByteOffset(), data, *cfg.Seed); err != nil {
			return err
		}
	} else {
		_, _ = fmt.Fprintln(w, "Nothing to randomize")
	}

	if !cfg.DisableRandomizer {
		if err := applySharedItemModel(w, cfg, data); err != nil {
			return err
		}
	}

	entryByName := make(map[string]isopatch.DatEntry, len(data.mapEntries))
	for _, e := range data.mapEntries {
		entryByName[e.Name] = e
	}

	written := 0
	for _, m := range data.maps {
		entry, ok := entryByName[m.Name]
		if !ok {
			panic(fmt.Sprintf("no DAT entry recorded for %s", m.Name))
		}
		if err := isopatch.WriteDatMember(f, entry, m.Buf); err != nil {
			return fmt.Errorf("writing %s back into DAT: %w", m.Name, err)
		}
		written++
	}

	_, _ = fmt.Fprintf(w, "\nWrote %d map file(s) into %s\n", written, cfg.Output)
	return nil
}

// applyTitleBranding only warns if it fails, the logo isn't worth failing a
// build over.
func applyTitleBranding(w io.Writer, f *os.File, data *gameData) error {
	logo1, logo1Entry, err := data.read(f, titleLogo1DatPath)
	if err != nil {
		_, _ = fmt.Fprintf(w, "!! %v, skipping the randomized title logo\n", err)
		return nil
	}
	logo2, logo2Entry, err := data.read(f, titleLogo2DatPath)
	if err != nil {
		_, _ = fmt.Fprintf(w, "!! %v, skipping the randomized title logo\n", err)
		return nil
	}
	newLogo1, newLogo2, err := titlelogo.ApplyBranding(logo1, logo2, resources.TitleRandomizedOverlayPNG)
	if err != nil {
		_, _ = fmt.Fprintf(w, "!! couldn't build the randomized title logo, leaving it alone: %v\n", err)
		return nil
	}
	if err := isopatch.WriteDatMemberResized(f, logo1Entry, newLogo1); err != nil {
		return fmt.Errorf("applying randomized title logo (%s): %w", titleLogo1DatPath, err)
	}
	if err := isopatch.WriteDatMemberResized(f, logo2Entry, newLogo2); err != nil {
		return fmt.Errorf("applying randomized title logo (%s): %w", titleLogo2DatPath, err)
	}
	_, _ = fmt.Fprintln(w, "Applied \"Randomized\" title screen branding")
	return nil
}

// applySharedItemModel repoints every map's item-model chunk at one model, so
// pickups look the same wherever they ended up.
func applySharedItemModel(w io.Writer, cfg *Config, data *gameData) error {
	const scale = 3.0
	model, err := itemmodel.Scale(resources.SharedItemModel, scale)
	if err != nil {
		return fmt.Errorf("scaling the shared item model: %w", err)
	}
	texture := resources.SharedItemTexture
	if cfg.SharedItemModelColor != "" {
		r, g, b, err := itemmodel.ParseColor(cfg.SharedItemModelColor)
		if err != nil {
			return fmt.Errorf("shared_item_model_color: %w", err)
		}
		if texture, err = itemmodel.Recolor(texture, r, g, b); err != nil {
			return fmt.Errorf("recolouring the shared item texture: %w", err)
		}
		_, _ = fmt.Fprintf(w, "Shared item model colour: %s\n", cfg.SharedItemModelColor)
	}
	modelResult := itemmodel.Run(data.maps, model, texture)
	_, _ = fmt.Fprintf(w, "Shared item model scale: %gx\n", scale)
	_, _ = fmt.Fprint(w, modelResult.Summary(data.mapNames))
	return nil
}

// randomizeAll runs the seed-dependent steps: item shuffling, paramdomizer,
// spoiler log. It mutates data.maps and writes into f.
func randomizeAll(w io.Writer, cfg *Config, f *os.File, slus int64, data *gameData, seed uint64) error {
	_, _ = fmt.Fprintf(w, "Using seed %d\n", seed)

	spoilerPath := cfg.Spoiler
	if spoilerPath == "" {
		spoilerPath = filepath.Join(filepath.Dir(cfg.Output), fmt.Sprintf("spoiler_%d.log", seed))
	}

	var result randomize.Result
	var sphereLog string
	switch {
	case cfg.DisableRandomizer:
		_, _ = fmt.Fprintln(w, "Randomizer disabled, items stay vanilla")
	case cfg.UseLogicFill():
		chr, chrEntry, err := data.read(f, chrDatPath)
		if err != nil {
			_, _ = fmt.Fprintf(w, "!! %v, the Fire Lord keeps its drop\n", err)
		}

		var stock []byte
		if cfg.Pool().Shops {
			if stock, err = isopatch.ReadSlusBytes(f, slus, kfconst.SlusShopStockVAddr, kfconst.ShopCount*kfconst.ShopStockSize); err != nil {
				return err
			}
		}
		filled, placements, err := randomize.RunLogic(data.maps, chr, stock, randomize.NewRand(seed), cfg.Pool())
		if err != nil {
			return fmt.Errorf("logic fill: %w", err)
		}
		if stock != nil {
			if err := isopatch.WriteSlusBytes(f, slus, kfconst.SlusShopStockVAddr, stock); err != nil {
				return err
			}
		}
		if chr != nil {
			if err := isopatch.WriteDatMember(f, chrEntry, chr); err != nil {
				return fmt.Errorf("writing %s back into DAT: %w", chrDatPath, err)
			}
		}
		result = filled
		sphereLog = report.Spheres(placements, cfg.Pool(), data.itemNames, data.mapNames)
		_, _ = fmt.Fprint(w, result.Summary(data.mapNames))
	default:
		result = randomize.RunNoLogic(data.maps, randomize.NewRand(seed), cfg.Pool())
		_, _ = fmt.Fprint(w, result.Summary(data.mapNames))
	}

	if !cfg.DisableRandomizer {
		if err := scaleRewards(w, cfg, f, data); err != nil {
			return err
		}
	}

	if result.StartItem != 0 {
		if err := swapStartingIdol(w, cfg, f, data, result.StartItem); err != nil {
			return err
		}
	}

	if cfg.EnableParamdomizer {
		if err := runParamdomizer(w, cfg, f, data, seed); err != nil {
			return err
		}
	}

	shopList, err := runShops(w, cfg, f, slus, seed)
	if err != nil {
		return err
	}

	spoiler := report.Spoiler(seed, result, data.itemNames, data.mapNames)
	if shopList != nil {
		spoiler += report.Shops(*shopList, data.itemNames)
	}
	if err := os.WriteFile(spoilerPath, []byte(spoiler), 0o644); err != nil {
		return fmt.Errorf("writing spoiler log: %w", err)
	}
	_, _ = fmt.Fprintf(w, "Wrote spoiler log to %s\n", spoilerPath)

	if sphereLog != "" {
		path := strings.TrimSuffix(spoilerPath, filepath.Ext(spoilerPath)) + "_spheres.log"
		if err := os.WriteFile(path, []byte(sphereLog), 0o644); err != nil {
			return fmt.Errorf("writing sphere log: %w", err)
		}
		_, _ = fmt.Fprintf(w, "Wrote sphere log to %s\n", path)
	}
	return nil
}

// scaleRewards multiplies the gold and experience monsters give, in chr.prm.
func scaleRewards(w io.Writer, cfg *Config, f *os.File, data *gameData) error {
	gold := cfg.GoldMultiplier > 0 && cfg.GoldMultiplier != 1
	exp := cfg.ExpMultiplier > 0 && cfg.ExpMultiplier != 1
	if !gold && !exp {
		return nil
	}
	chr, entry, err := data.read(f, chrDatPath)
	if err != nil {
		_, _ = fmt.Fprintf(w, "!! %v, gold and experience stay vanilla\n", err)
		return nil
	}
	if gold {
		n := kfmap.ScaleChrGold(chr, cfg.GoldMultiplier)
		_, _ = fmt.Fprintf(w, "Gold drops x%g on %d monster types\n", cfg.GoldMultiplier, n)
	}
	if exp {
		n := kfmap.ScaleChrExp(chr, cfg.ExpMultiplier)
		_, _ = fmt.Fprintf(w, "Experience x%g on %d monster types\n", cfg.ExpMultiplier, n)
	}
	if err := isopatch.WriteDatMember(f, entry, chr); err != nil {
		return fmt.Errorf("writing %s back into DAT: %w", chrDatPath, err)
	}
	return nil
}

// swapStartingIdol rewrites the starting ini so the new game starts with give
// instead of the Idol of Sorrow, on top of any starting_config overrides.
func swapStartingIdol(w io.Writer, cfg *Config, f *os.File, data *gameData, give uint16) error {
	overrides := maps.Clone(cfg.StartingConfig)
	if overrides == nil {
		overrides = map[string]int{}
	}
	overrides[fmt.Sprintf("item_%03d", logic.IdolOfSorrow)] = 0
	withUser, err := startingconfig.ApplyOverrides(startingconfig.BaseINI, overrides)
	if err != nil {
		return err
	}
	overrides[fmt.Sprintf("item_%03d", give)] = min(startingconfig.ItemQuantity(withUser, int(give))+1, 99)
	ini, err := startingconfig.ApplyOverrides(startingconfig.BaseINI, overrides)
	if err != nil {
		return err
	}
	if err := isopatch.WriteDatMemberResized(f, data.dat[startingIniDatPath], ini); err != nil {
		return fmt.Errorf("swapping the starting Idol of Sorrow: %w", err)
	}
	name := fmt.Sprint(give)
	if data.itemNames != nil {
		name = data.itemNames.NameOrDefault(give, name)
	}
	_, _ = fmt.Fprintf(w, "Starting inventory gets %s instead of the Idol of Sorrow\n", name)
	return nil
}

// runShops shuffles the gold shops when asked, and stocks them with ammo
// whenever items move at all, since the logic counts any bow as usable.
func runShops(w io.Writer, cfg *Config, f *os.File, slus int64, seed uint64) (*shops.Result, error) {
	if cfg.DisableRandomizer && !cfg.RandomizeShops {
		return nil, nil
	}
	stock, err := isopatch.ReadSlusBytes(f, slus, kfconst.SlusShopStockVAddr, kfconst.ShopCount*kfconst.ShopStockSize)
	if err != nil {
		return nil, err
	}
	prices, err := isopatch.ReadSlusBytes(f, slus, kfconst.SlusShopPriceVAddr, kfconst.ShopCount*kfconst.ShopPriceSize)
	if err != nil {
		return nil, err
	}

	// With logic on, the fill already stocked the shops.
	if cfg.RandomizeShops && !cfg.UseLogicFill() {
		if err := shops.Shuffle(stock, shops.NewRand(seed)); err != nil {
			return nil, fmt.Errorf("shuffling shops: %w", err)
		}
		_, _ = fmt.Fprintln(w, "Shuffled the gold shops' stock")
	}
	if err := shops.AddAmmo(stock, prices); err != nil {
		return nil, err
	}
	_, _ = fmt.Fprintln(w, "Stocked every gold shop with arrows, bolts and Widda's arrows")

	if err := isopatch.WriteSlusBytes(f, slus, kfconst.SlusShopStockVAddr, stock); err != nil {
		return nil, err
	}
	if err := isopatch.WriteSlusBytes(f, slus, kfconst.SlusShopPriceVAddr, prices); err != nil {
		return nil, err
	}
	list := shops.List(stock, prices)
	return &list, nil
}

func runParamdomizer(w io.Writer, cfg *Config, f *os.File, data *gameData, seed uint64) error {
	wpnBuf, wpnEntry, err := data.read(f, itemWpnDatPath)
	if err != nil {
		_, _ = fmt.Fprintf(w, "!! %v, skipping paramdomizer\n", err)
		return nil
	}
	pctBuf, pctEntry, err := data.read(f, itemPctDatPath)
	if err != nil {
		_, _ = fmt.Fprintf(w, "!! %v, skipping paramdomizer\n", err)
		return nil
	}

	result, err := paramdomizer.Run(wpnBuf, pctBuf, paramdomizer.NewRand(seed), paramdomizer.Options{
		PercentRange:             cfg.ParamdomizerPercent,
		AllowNegativeWeaponStats: cfg.ParamdomizerAllowNegative,
		AllowWorseWeaponLevels:   cfg.ParamdomizerAllowWorseLevels,
	})
	if err != nil {
		return fmt.Errorf("paramdomizer: %w", err)
	}
	if err := isopatch.WriteDatMember(f, wpnEntry, wpnBuf); err != nil {
		return fmt.Errorf("writing %s back into DAT: %w", itemWpnDatPath, err)
	}
	if err := isopatch.WriteDatMember(f, pctEntry, pctBuf); err != nil {
		return fmt.Errorf("writing %s back into DAT: %w", itemPctDatPath, err)
	}
	_, _ = fmt.Fprint(w, result.Summary())
	return nil
}
