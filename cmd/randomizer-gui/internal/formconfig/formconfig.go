// Package formconfig turns the GUI form into a runner.Config, and holds the
// lists the form is built from.
package formconfig

import (
	"fmt"
	"net/http"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"

	"kf4randomizer/internal/features/mods"
	"kf4randomizer/internal/kfconst"
	"kf4randomizer/internal/kfmap"
	"kf4randomizer/internal/runner"
)

// config mirrors runner.Config's YAML, zero values meaning unset.
type config struct {
	Input                    string         `yaml:"input" json:"input"`
	Output                   string         `yaml:"output" json:"output"`
	Seed                     *uint64        `yaml:"seed" json:"seed"`
	Spoiler                  string         `yaml:"spoiler" json:"spoiler"`
	Mods                     mods.Selection `yaml:"mods" json:"mods"`
	StartingConfig           map[string]int `yaml:"starting_config" json:"starting_config"`
	StripDummyFiles          bool           `yaml:"strip_dummy_files" json:"strip_dummy_files"`
	RemoveEmptyContainers    bool           `yaml:"remove_empty_containers" json:"remove_empty_containers"`
	RandomizeGroundItems     *bool          `yaml:"randomize_ground_items" json:"randomize_ground_items"`
	RandomizeContainerItems  *bool          `yaml:"randomize_container_items" json:"randomize_container_items"`
	RandomizeMonsterDrops    *bool          `yaml:"randomize_monster_drops" json:"randomize_monster_drops"`
	RandomizeIdolOfSorrow    *bool          `yaml:"randomize_idol_of_sorrow" json:"randomize_idol_of_sorrow"`
	RandomizeShops           bool           `yaml:"randomize_shops" json:"randomize_shops"`
	GoldMultiplier           float64        `yaml:"gold_multiplier" json:"gold_multiplier"`
	ExpMultiplier            float64        `yaml:"exp_multiplier" json:"exp_multiplier"`
	LogicFill                *bool          `yaml:"logic_fill" json:"logic_fill"`
	PickupPopupShowItemIndex bool           `yaml:"pickup_popup_show_item_index" json:"pickup_popup_show_item_index"`
	DisableRandomizer        bool           `yaml:"disable_randomizer" json:"disable_randomizer"`
	FullDebugMode            bool           `yaml:"full_debug_mode" json:"full_debug_mode"`
	WandsAnywhere            bool           `yaml:"wands_anywhere" json:"wands_anywhere"`
}

// FromForm goes through YAML so the GUI is validated like the CLI.
func FromForm(r *http.Request) (*runner.Config, error) {
	startingConfig, err := buildStartingConfig(r)
	if err != nil {
		return nil, err
	}

	seed, err := resolveSeed(r.FormValue("seed"))
	if err != nil {
		return nil, err
	}

	input := strings.TrimSpace(r.FormValue("input"))
	output := strings.TrimSpace(r.FormValue("output"))
	if output == "" && input != "" {
		output = filepath.Join(filepath.Dir(input), fmt.Sprintf("KF4_%d.iso", seed))
	}

	// the page writes <seed> before the seed is known
	seedText := strconv.FormatUint(seed, 10)
	output = strings.ReplaceAll(output, "<seed>", seedText)
	spoiler := strings.ReplaceAll(strings.TrimSpace(r.FormValue("spoiler")), "<seed>", seedText)

	off := r.FormValue("randomizer") == "off"
	logicFill := r.FormValue("logic") != "none"
	ground := r.FormValue("randomize_ground_items") != ""
	containers := r.FormValue("randomize_container_items") != ""
	drops := r.FormValue("randomize_monster_drops") != ""
	idol := r.FormValue("randomize_idol_of_sorrow") != ""
	gold, err := multiplier(r, "gold_multiplier")
	if err != nil {
		return nil, err
	}
	exp, err := multiplier(r, "exp_multiplier")
	if err != nil {
		return nil, err
	}

	// cheat fields are hidden when cheats are off, ignore them
	if r.FormValue("enable_cheats") == "" {
		startingConfig = nil
	}

	raw := config{
		Input:                    input,
		Output:                   output,
		Seed:                     &seed,
		Spoiler:                  spoiler,
		Mods:                     modsSelectionFromForm(r),
		StartingConfig:           startingConfig,
		StripDummyFiles:          r.FormValue("strip_dummy_files") != "",
		RemoveEmptyContainers:    r.FormValue("remove_empty_containers") != "",
		RandomizeGroundItems:     &ground,
		RandomizeContainerItems:  &containers,
		RandomizeMonsterDrops:    &drops,
		RandomizeIdolOfSorrow:    &idol,
		RandomizeShops:           !off && r.FormValue("randomize_shops") != "",
		GoldMultiplier:           gold,
		ExpMultiplier:            exp,
		LogicFill:                &logicFill,
		PickupPopupShowItemIndex: r.FormValue("pickup_popup_show_item_index") != "",
		DisableRandomizer:        off,
		FullDebugMode:            r.FormValue("full_debug_mode") != "",
		WandsAnywhere:            r.FormValue("wands_anywhere") != "",
	}

	yamlBytes, err := yaml.Marshal(raw)
	if err != nil {
		return nil, fmt.Errorf("internal error building config: %w", err)
	}
	cfg, err := runner.ParseConfig(yamlBytes)
	if err != nil {
		return nil, err
	}

	return cfg, nil
}

// ParseUploadedYAML skips ParseConfig, which would invent a seed. Seed goes
// out as a string, JS can't hold a uint64.
func ParseUploadedYAML(body []byte) (any, error) {
	var cfg config
	if err := yaml.Unmarshal(body, &cfg); err != nil {
		return nil, fmt.Errorf("that doesn't look like a valid config.yaml: %w", err)
	}

	type jsonOut struct {
		config
		Seed *string `json:"seed"`
	}
	out := jsonOut{config: cfg}
	if cfg.Seed != nil {
		s := strconv.FormatUint(*cfg.Seed, 10)
		out.Seed = &s
	}
	return out, nil
}

// resolveSeed parses seedStr, or generates a fresh random seed if empty.
func resolveSeed(seedStr string) (uint64, error) {
	seedStr = strings.TrimSpace(seedStr)
	if seedStr == "" {
		return runner.RandomSeed()
	}
	seed, err := strconv.ParseUint(seedStr, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("seed: %q is not a valid number: %w", seedStr, err)
	}
	return seed, nil
}

// multiplier reads one of the reward dropdowns, 1 when blank.
func multiplier(r *http.Request, key string) (float64, error) {
	v := strings.TrimSpace(r.FormValue(key))
	if v == "" {
		return 1, nil
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil || f < 0 {
		return 0, fmt.Errorf("%s %q isn't a number of 0 or more", key, v)
	}
	return f, nil
}

// modsSelectionFromForm reads one mode choice per category, "mod_<id>".
func modsSelectionFromForm(r *http.Request) mods.Selection {
	var sel mods.Selection
	for _, cat := range mods.BuiltinCategories {
		modeID := r.FormValue("mod_" + cat.ID)
		if modeID == "" {
			continue
		}
		sel = append(sel, map[string]bool{modeID: true})
	}
	return sel
}

// buildStartingConfig reads the cheat fields. Blank means vanilla.
func buildStartingConfig(r *http.Request) (map[string]int, error) {
	result := make(map[string]int)

	for _, f := range CoreStats {
		if err := addIntField(result, r, f.Key, f.Min, f.Max); err != nil {
			return nil, err
		}
	}

	for _, elem := range MagicSpells {
		for _, spell := range elem.Spells {
			if err := addIntField(result, r, spell.Key, 0, 3); err != nil {
				return nil, err
			}
		}
	}

	for _, slot := range EquipSlots {
		v := strings.TrimSpace(r.FormValue(slot.Key))
		if v == "" {
			continue
		}
		classID, err := strconv.Atoi(v)
		if err != nil {
			return nil, fmt.Errorf("%s: %q is not valid", slot.Key, v)
		}
		result[slot.Key] = classID
	}

	invItems := r.PostForm["inv_item"]
	invQtys := r.PostForm["inv_qty"]
	for i, classIDStr := range invItems {
		classIDStr = strings.TrimSpace(classIDStr)
		if classIDStr == "" {
			continue
		}
		classID, err := strconv.Atoi(classIDStr)
		if err != nil || classID < 1 {
			return nil, fmt.Errorf("starting inventory: %q is not a valid item", classIDStr)
		}
		qtyStr := ""
		if i < len(invQtys) {
			qtyStr = strings.TrimSpace(invQtys[i])
		}
		if qtyStr == "" {
			continue // item picked, no quantity typed
		}
		qty, err := strconv.Atoi(qtyStr)
		if err != nil || qty < 0 || qty > 99 {
			return nil, fmt.Errorf("starting inventory: quantity %q must be 0-99", qtyStr)
		}
		// The ini's item_NNN keys index by classID directly
		result[fmt.Sprintf("item_%03d", classID)] = qty
	}

	if len(result) == 0 {
		return nil, nil
	}
	return result, nil
}

// addIntField parses and range-checks formKey into dst, skipping it if blank.
func addIntField(dst map[string]int, r *http.Request, formKey string, min, max int) error {
	v := strings.TrimSpace(r.FormValue(formKey))
	if v == "" {
		return nil
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fmt.Errorf("%s: %q is not a whole number", formKey, v)
	}
	if n < min || n > max {
		return fmt.Errorf("%s: %d is out of range (%d-%d)", formKey, n, min, max)
	}
	dst[formKey] = n
	return nil
}

type StatField struct {
	Key     string
	Label   string
	Default int
	Min     int
	Max     int
}

// debug_dash, debug_auto_recv and debug_magic_rapid_fire are left out.
var CoreStats = []StatField{
	{"hp", "HP", 50, 0, 999},
	{"mp", "MP", 0, 0, 999},
	{"power", "Strength", 5, 0, 999},
	{"magic", "Magic Power", 0, 0, 999},
	{"level", "Level", 1, 1, 99},
	{"exp", "EXP", 0, 0, 999999},
	{"gold", "Gold", 0, 0, 999999},
}

type SpellField struct {
	Key  string
	Name string
}

// ElementSpells groups one element's 5 spells (level 0-3 each).
type ElementSpells struct {
	Element string
	Spells  []SpellField
}

// Spell names come from the ini's own comments.
var MagicSpells = []ElementSpells{
	{"Fire", []SpellField{{"magic_fire_1", "Fire Ball"}, {"magic_fire_2", "Fire Wall"}, {"magic_fire_3", "Flame Breath"}, {"magic_fire_4", "Immolate"}, {"magic_fire_5", "Flame Strike"}}},
	{"Earth", []SpellField{{"magic_earth_1", "Earth Heal"}, {"magic_earth_2", "Upheaval"}, {"magic_earth_3", "Earth Leach"}, {"magic_earth_4", "Volcannon"}, {"magic_earth_5", "Meteor Cascade"}}},
	{"Wind", []SpellField{{"magic_wind_1", "Wind Cutter"}, {"magic_wind_2", "Life Shield"}, {"magic_wind_3", "Ice Clad"}, {"magic_wind_4", "Tornado"}, {"magic_wind_5", "Freeze"}}},
	{"Water", []SpellField{{"magic_water_1", "Purity"}, {"magic_water_2", "Endure Fire"}, {"magic_water_3", "Vortex"}, {"magic_water_4", "Remedy"}, {"magic_water_5", "Hydrosphere"}}},
	{"Holy", []SpellField{{"magic_holy_1", "Divine Light"}, {"magic_holy_2", "Flash"}, {"magic_holy_3", "Retribution"}, {"magic_holy_4", "Invoke Lightning"}, {"magic_holy_5", "Judgement"}}},
	{"Dark", []SpellField{{"magic_dark_1", "Shadow Skin"}, {"magic_dark_2", "Eclipse"}, {"magic_dark_3", "Reckless Strength"}, {"magic_dark_4", "Drifting Death"}, {"magic_dark_5", "Dark Matter"}}},
}

// EquipSlot values are item classIDs, -1 for nothing.
type EquipSlot struct {
	Key      string
	Label    string
	Category string // matches kfconst.ItemCategories[].Name
}

// equip_magic and the equip_shortcut fields number things differently, so
// they're left to config.yaml.
var EquipSlots = []EquipSlot{
	{"equip_weapon", "Weapon", "Weapons"},
	{"equip_shield", "Shield", "Shields"},
	{"equip_helm", "Helm", "Helmets"},
	{"equip_armor", "Armor", "Cuirass"},
	{"equip_guntlet", "Gauntlets", "Gauntlets"},
	{"equip_boot", "Greaves", "Greaves"},
	{"equip_ring", "Ring", "Rings"},
	{"equip_bracelet", "Bracelet", "Bracelets"},
	{"equip_amulet", "Amulet", "Amulets"},
}

type ItemOption struct {
	ClassID uint16
	Name    string
}

// ItemCategoryOptions groups items by item_nam.prm page, for <optgroup>.
type ItemCategoryOptions struct {
	Category string
	Items    []ItemOption
}

var ItemCatalog = buildItemCatalog()

func buildItemCatalog() []ItemCategoryOptions {
	names := kfmap.VanillaItemNames
	var cats []ItemCategoryOptions
	for _, page := range kfconst.ItemCategories {
		var items []ItemOption
		for cid := page.FirstClass; cid <= page.LastClass; cid++ {
			name, ok := names[uint16(cid)]
			if !ok || name == "" {
				continue
			}
			items = append(items, ItemOption{ClassID: uint16(cid), Name: name})
		}
		if len(items) == 0 {
			continue
		}
		sort.SliceStable(items, func(i, j int) bool { return items[i].ClassID < items[j].ClassID })
		cats = append(cats, ItemCategoryOptions{Category: page.Name, Items: items})
	}
	return cats
}

type EquipSlotView struct {
	Key     string
	Label   string
	Options []ItemOption
}

var EquipSlotViews = buildEquipSlotViews()

func buildEquipSlotViews() []EquipSlotView {
	views := make([]EquipSlotView, 0, len(EquipSlots))
	for _, es := range EquipSlots {
		views = append(views, EquipSlotView{Key: es.Key, Label: es.Label, Options: itemOptionsForCategory(es.Category)})
	}
	return views
}

func itemOptionsForCategory(category string) []ItemOption {
	for _, c := range ItemCatalog {
		if c.Category == category {
			return c.Items
		}
	}
	return nil
}
