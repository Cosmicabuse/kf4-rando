package logic

// Item classIDs some requirement asks for.
const (
	AltarKey       uint16 = 105
	AncientKey     uint16 = 53
	BindingCrown   uint16 = 107
	CaretakersMap  uint16 = 1
	BlackEye       uint16 = 130
	CraftsmansKey  uint16 = 57
	CrystalVial    uint16 = 14
	EarthFolksKey  uint16 = 50
	EngineersKey   uint16 = 56
	ExpeditionsKey uint16 = 48
	ForestDew      uint16 = 120
	GreenWand      uint16 = 140
	Herb           uint16 = 6
	IconOfIce      uint16 = 101
	IconOfWater    uint16 = 103
	IdolOfSorrow   uint16 = 149
	JailKey        uint16 = 51
	Keystone       uint16 = 106
	KingsKey       uint16 = 59
	LawfulBlade    uint16 = 195
	MoonlightSword uint16 = 196
	PriestsKey     uint16 = 58
	ReleaseRock    uint16 = 104
	RingOfWisdom   uint16 = 352
	RockOfLife     uint16 = 114
	RustyKey       uint16 = 49
	SerraksKey     uint16 = 55
	StrangeEgg     uint16 = 125
	WiddaKey       uint16 = 52
	WiddasBracelet uint16 = 362
	WiddasNecklace uint16 = 372
	WaterTalisman  uint16 = 374
	WiddasRing     uint16 = 353
	ZastarisKey    uint16 = 47
)

// Every crossing between two maps; true means one-way.
var crossings = []struct {
	from, to string
	oneWay   bool
}{
	{"0", "96", false},
	{"0", "29", false},
	{"0", "99", false},
	{"2", "96", false},
	{"4", "99", false},
	{"5", "95", false},
	{"5", "98", false},
	{"5", "12", false},
	{"6", "42", false},
	{"6", "7", false},
	{"6", "13", false},
	{"6", "30", false},
	{"7", "41", false},
	{"7", "6", false},
	{"7", "100", false},
	{"8", "9", false},
	{"8", "99", false},
	{"12", "5", false},
	{"12", "15", false},
	{"12", "11", false},
	{"13", "14", false},
	{"13", "103", false},
	{"13", "6", false},
	{"15", "100", false},
	{"15", "12", false},
	{"16", "19", false},
	{"16", "20", false},
	{"16", "105", false},
	{"16", "102", false},
	{"16", "18", false},
	{"17", "20", false},
	{"17", "16", true},
	{"17", "24", false},
	{"18", "20", false},
	{"18", "16", false},
	{"18", "24", false},
	{"19", "20", false},
	{"19", "16", false},
	{"19", "24", false},
	{"20", "17", false},
	{"20", "16", false},
	{"20", "18", false},
	{"20", "19", false},
	{"22", "23", false},
	{"22", "43", false},
	{"22", "26", false},
	{"22", "32", true},
	{"23", "32", false},
	{"23", "22", false},
	{"24", "17", false},
	{"24", "18", false},
	{"24", "19", false},
	{"24", "25", false},
	{"26", "22", false},
	{"26", "27", false},
	{"26", "28", false},
	{"26", "31", false},
	{"27", "21", false},
	{"27", "26", false},
	{"27", "28", false},
	{"28", "26", false},
	{"28", "27", false},
	{"29", "0", false},
	{"29", "96", false},
	{"31", "26", false},
	{"31", "111", false},
	{"32", "113", false},
	{"32", "44", false},
	{"32", "23", false},
	{"33", "34", false},
	{"33", "115", false},
	{"35", "37", false},
	{"35", "112", false},
	{"37", "38", false},
	{"37", "36", false},
	{"37", "35", false},
	{"39", "40", false},
	{"95", "5", false},
	{"95", "96", false},
	{"95", "97", false},
	{"96", "0", false},
	{"96", "29", false},
	{"96", "2", false},
	{"96", "95", false},
	{"98", "5", false},
	{"98", "99", false},
	{"99", "98", false},
	{"99", "8", false},
	{"99", "4", false},
	{"99", "0", false},
	{"100", "107", false},
	{"100", "101", false},
	{"100", "102", false},
	{"100", "7", false},
	{"100", "105", false},
	{"100", "15", false},
	{"101", "100", false},
	{"101", "31", true},
	{"102", "100", false},
	{"102", "103", false},
	{"102", "104", false},
	{"102", "16", false},
	{"103", "13", false},
	{"103", "102", false},
	{"104", "102", false},
	{"104", "105", false},
	{"105", "104", false},
	{"105", "106", false},
	{"105", "16", false},
	{"105", "108", false},
	{"105", "100", false},
	{"107", "100", false},
	{"107", "115", false},
	{"107", "108", false},
	{"107", "109", false},
	{"107", "111", false},
	{"108", "107", false},
	{"108", "105", false},
	{"109", "110", false},
	{"109", "111", false},
	{"109", "107", false},
	{"111", "113", false},
	{"111", "109", false},
	{"111", "31", false},
	{"111", "107", false},
	{"112", "115", false},
	{"112", "35", false},
	{"113", "111", false},
	{"113", "32", false},
	{"113", "112", true},
	{"113", "114", false},
	{"115", "107", false},
	{"115", "33", false},
	{"115", "112", false},
	{"115", "116", false},
}

// Crossings that cost something. One of these takes the place of the
// plain entry above for the same pair of maps.
var gates = []struct {
	from, to string
	need     Req
	oneWay   bool
}{
	{"2", "2A", has(RustyKey), false},
	{"98", "5", can(CapFire), false},
	{"14", "6", has(ForestDew), false},
	{"7", "41", Req{}, false},
	{"6", "7", Req{}, false},
	{"16", "20", Req{}, false},
	{"20", "16A", Req{}, false},
	{"17", "24", Req{}, false},
	{"19", "16B", has(IconOfIce), false},
	{"16B", "105", Req{}, false},
	{"106", "108", Req{}, false},
	{"108", "107", Req{}, false},
	{"107", "109", Req{}, false},
	{"109", "110", Req{}, false},
	{"110", "111", Req{}, false},
	{"111", "31", Req{}, false},
	{"26", "22", has(WiddasBracelet, WiddasNecklace, WiddasRing), false},
	{"6A", "42", has(BlackEye, BlackEye, BlackEye, BlackEye), false},
	{"6", "30", has(RingOfWisdom), true},
	{"30", "6A", Req{}, true},
	{"35", "37", has(IconOfWater), false},
	{"37", "36", has(JailKey), false},
	{"6A", "6", has(KingsKey), false},
	{"6", "6B", has(RingOfWisdom), false},
	{"12", "15", has(BindingCrown), true},
	{"15", "12", Req{}, true},
	{"37", "38", has(IdolOfSorrow), false},
	// 102A is only reached by warping with the Green Wand, which works from
	// any teleport pad; 006 stands in for wherever the wand is picked up in vanilla
	{"6", "102A", has(GreenWand), true},
	{"102A", "16", Req{}, false},
	{"102A", "104", Req{}, false},
	{"104", "102", Req{}, true},

	// One forge in three states. The Fire Genie in 24 lights it, the icon
	// cools the forge so there's less lava.
	{"24", "18", Req{}, true},
	{"18", "19", has(IconOfIce), true},
	{"18", "16", Req{}, true},
	{"18", "20", Req{}, true},
	{"18", "24", Req{}, true},
	{"19", "20", Req{}, true},
	{"19", "24", Req{}, true},

	// The central tower's parts only meet through other maps. The bridge and
	// the doors from the throne and level 2 open from one side only.
	{"100A", "15", Req{}, false},
	{"100A", "15A", Req{}, false},
	{"100A", "15B", Req{}, false},
	{"100A", "15C", Req{}, false},
	{"100A", "101A", Req{}, false},
	{"100A", "101B", Req{}, false},
	{"100A", "100B", has(Keystone), true},
	{"100B", "102", Req{}, false},
	{"102", "100C", has(PriestsKey), false},
	{"7", "100B", Req{}, true},

	// 105 loops through a corner of the tower (100D) to a ledge (105A) whose
	// lever opens back onto 105. Keystone shortcuts are left out, they only
	// lead back to places already reached.
	{"105", "100D", Req{}, false},
	{"100D", "105A", Req{}, false},
	{"105A", "105", Req{}, false},
	{"105A", "106", Req{}, false},

	// The end of the Egg Mines opens onto 113 with a Strange Egg.
	{"32", "113", has(StrangeEgg), false},

	// Level 2 tower: 107 from 108, 107B from 111.
	{"111", "107B", Req{}, false},
	{"107B", "111A", Req{}, false},
	{"109", "111", Req{}, true},
	{"113", "111", Req{}, true},
	{"115", "107", Req{}, true},
}

// Checks that also need another place reached, keyed by region rather than
// map file.
var errands = map[Where]string{
	// The Seabed Cave trade wants a vial of poison water from the Icon of
	// Healing.
	{Map: "28", Slot: Ground, Index: 0}: "97",
}

// The gold shops: where each shopkeeper sells, what he needs, and his vanilla
// stock without ammo, which every shop always sells.
var shops = []struct {
	region string
	need   Req
	also   string
	lock   string
	items  []uint16
}{
	{"96", Req{}, "", "", []uint16{1, 6, 9, 12, 156, 200, 224, 254, 277, 333}},
	// gives the Bow once the room is cleared, then sells
	{"15C", Req{}, "", "", []uint16{6, 7, 9, 10, 12, 158, 161, 178, 198, 201, 202, 258, 310}},
	// moves from 98 to the 7 <-> 100B shortcut once given a Rock of Life
	// TODO: check that connection in game
	{"7", has(RockOfLife), "98", "98", []uint16{6, 7, 9, 10, 12, 166, 173, 204, 230, 260, 303, 338}},
	{"109", Req{}, "", "", []uint16{6, 7, 9, 10, 11, 12, 13, 14, 21, 22, 34, 142, 206, 211, 255, 262, 279, 281, 288, 306, 337, 340}},
}

// A new game starts here.
const startRegion = "0"

// Items granting each capability.
var (
	fireItems       = []uint16{71, 77}
	projectileItems = []uint16{207, 208, 209, 210, 211, 212}
	breakItems      = []uint16{152, 154, 157, 158, 159, 160, 161, 162, 163, 164, 165, 166, 167, 168, 169, 170, 171, 172, 173, 174, 175, 176, 177, 178, 179, 180, 181, 182, 184, 185, 186, 187, 188, 189, 190, 191, 193, 194, 195, 196, 197, 198, 199, 200, 201, 202, 203, 204, 205, 206, 207, 208, 209, 210, 211, 212}
)

// Map pairs the files link but nothing can actually cross.
var blocked = [][2]string{
	{"95", "96"},
	{"101", "31"},
	{"104", "105"},
	{"110", "111"},
}
