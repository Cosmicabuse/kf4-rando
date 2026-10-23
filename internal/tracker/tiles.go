package tracker

import "kf4randomizer/internal/logic"

type Kind int

const (
	Toggle Kind = iota
	Counter
	Sword
)

// MarshalText names the kind for the page script.
func (k Kind) MarshalText() ([]byte, error) {
	return []byte([...]string{"toggle", "counter", "sword"}[k]), nil
}

type Tile struct {
	ID      string   `json:"id"` // also the icon file name: icons/<ID>.png, sword uses <ID>_1/_2
	Name    string   `json:"name"`
	Label   string   `json:"label"` // caps text under keys and icons
	Section string   `json:"section"`
	Kind    Kind     `json:"kind"`
	Max     int      `json:"max"`
	Items   []uint16 `json:"-"` // classIDs that light it from the starting inventory
}

var Sections = []string{"Keys", "Icons", "Quest", "Counters", "Wands & stones", "Magic"}

func toggle(section, id, name, label string, item uint16) Tile {
	return Tile{ID: id, Name: name, Label: label, Section: section, Max: 1, Items: []uint16{item}}
}

func counter(id, name string, max int, item uint16) Tile {
	return Tile{ID: id, Name: name, Section: "Counters", Kind: Counter, Max: max, Items: []uint16{item}}
}

func magic(id, name string) Tile {
	return Tile{ID: id, Name: name, Section: "Magic", Kind: Counter, Max: 5}
}

var Tiles = []Tile{
	toggle("Keys", "zastari_key", "Zastari's Key", "Z", 47),
	toggle("Keys", "expedition_key", "Expedition's Key", "EX", 48),
	toggle("Keys", "rusty_key", "Rusty Key", "R", 49),
	toggle("Keys", "earth_folk_key", "Earth Folk's Key", "EF", 50),
	toggle("Keys", "jail_key", "Jail Key", "J", 51),
	toggle("Keys", "widda_key", "Widda Key", "W", 52),
	toggle("Keys", "ancient_key", "Ancient Key", "A", 53),
	toggle("Keys", "serrak_key", "Serrak's Key", "S", 55),
	toggle("Keys", "engineer_key", "Engineer's Key", "ENG", 56),
	toggle("Keys", "craftsman_key", "Craftsman's Key", "C", 57),
	toggle("Keys", "priest_key", "Priest's Key", "P", 58),
	toggle("Keys", "king_key", "King's Key", "K", 59),
	toggle("Keys", "altar_key", "Altar Key", "ALT", 105),

	toggle("Icons", "icon_healing", "Icon of Healing", "HEAL", 100),
	toggle("Icons", "icon_ice", "Icon of Ice", "ICE", 101),
	toggle("Icons", "icon_fire", "Icon of Fire", "FIR", 102),
	toggle("Icons", "icon_water", "Icon of Water", "WAT", 103),

	toggle("Quest", "forest_dew", "Forest Dew", "", 120),
	toggle("Quest", "idol", "Idol of Sorrow", "", 149),
	toggle("Quest", "water_talisman", "Water Talisman", "", 374),
	toggle("Quest", "ring_of_wisdom", "Ring of Wisdom", "", 352),
	toggle("Quest", "widda_ring", "Widda's Ring", "", 353),
	toggle("Quest", "widda_bracelet", "Widda's Bracelet", "", 362),
	toggle("Quest", "widda_necklace", "Widda's Necklace", "", 372),
	{ID: "sword", Name: "Lawful Blade / Moonlight Sword", Section: "Quest", Kind: Sword, Max: 2, Items: []uint16{195, 196}},
	{ID: "bow", Name: "Bow", Section: "Quest", Max: 1, Items: logic.ProjectileItems},

	counter("crystal_vial", "Crystal Vial", 6, 14),
	counter("rock_of_life", "Rock of Life", 2, 114),
	counter("keystone", "Keystone", 5, 106),
	counter("release_rock", "Release Rock", 6, 104),
	counter("strange_egg", "Strange Egg", 2, 125),
	counter("black_eye", "Black Eye", 4, 130),
	counter("binding_crown", "Binding Crown", 3, 107),

	toggle("Wands & stones", "red_wand", "Red Wand", "", 139),
	toggle("Wands & stones", "green_wand", "Green Wand", "", 140),
	toggle("Wands & stones", "blue_wand", "Blue Wand", "", 141),
	toggle("Wands & stones", "yellow_wand", "Yellow Wand", "", 142),
	toggle("Wands & stones", "violet_wand", "Violet Wand", "", 143),
	toggle("Wands & stones", "red_stone", "Red Guide Stone", "", 134),
	toggle("Wands & stones", "green_stone", "Green Guide Stone", "", 135),
	toggle("Wands & stones", "blue_stone", "Blue Guide Stone", "", 136),
	toggle("Wands & stones", "yellow_stone", "Yellow Guide Stone", "", 137),
	toggle("Wands & stones", "violet_stone", "Violet Guide Stone", "", 138),

	magic("magic_fire", "Fire magic"),
	magic("magic_earth", "Earth magic"),
	magic("magic_wind", "Wind magic"),
	magic("magic_water", "Water magic"),
	magic("magic_holy", "Light magic"),
	magic("magic_dark", "Dark magic"),
}
