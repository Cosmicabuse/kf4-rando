package logic

// Item classIDs some requirement asks for.
const (
	AltarKey       uint16 = 105
	AncientKey     uint16 = 53
	BindingCrown   uint16 = 107
	BlackEye       uint16 = 130
	CraftsmansKey  uint16 = 57
	CrystalVial    uint16 = 14
	EarthFolksKey  uint16 = 50
	EngineersKey   uint16 = 56
	ExpeditionsKey uint16 = 48
	ForestDew      uint16 = 120
	GreenWand      uint16 = 140
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
	WiddasRing     uint16 = 353
	ZastarisKey    uint16 = 47
)

// What each requirement asks for, keyed lowercase (see need). Anything not
// listed here is free.
var requirements = map[string]Req{
	"2x binding crown":                  {Items: map[uint16]int{BindingCrown: 2}},
	"2x release rock":                   {Items: map[uint16]int{ReleaseRock: 2}},
	"2x release rock, 2x binding crown": {Items: map[uint16]int{ReleaseRock: 2, BindingCrown: 2}},
	"3x release rock":                   {Items: map[uint16]int{ReleaseRock: 3}},
	"altar key":                         {Items: map[uint16]int{AltarKey: 1}},
	"ancient key":                       {Items: map[uint16]int{AncientKey: 1}},
	"ancient key, serrak's key":         {Items: map[uint16]int{AncientKey: 1, SerraksKey: 1}},
	"binding crown":                     {Items: map[uint16]int{BindingCrown: 1}},
	"black eye x4":                      {Items: map[uint16]int{BlackEye: 4}},
	"black eye x4, lawful blade":        {Items: map[uint16]int{BlackEye: 4, LawfulBlade: 1}},
	"craftsman's key":                   {Items: map[uint16]int{CraftsmansKey: 1}},
	"crystal vial":                      {Items: map[uint16]int{CrystalVial: 1}},
	"earth folk's key":                  {Items: map[uint16]int{EarthFolksKey: 1}},
	"earth folk's key ?":                {Items: map[uint16]int{EarthFolksKey: 1}},
	"engineer's key":                    {Items: map[uint16]int{EngineersKey: 1}},
	"expedition key":                    {Items: map[uint16]int{ExpeditionsKey: 1}},
	"fire spell":                        {Caps: []Cap{CapFire}},
	"fireball used in map097":           {Caps: []Cap{CapFire}},
	"forest dew":                        {Items: map[uint16]int{ForestDew: 1}},
	"green wand":                        {Items: map[uint16]int{GreenWand: 1}},
	"icon of ice":                       {Items: map[uint16]int{IconOfIce: 1}},
	"icon of water":                     {Items: map[uint16]int{IconOfWater: 1}},
	"idol of sorrow":                    {Items: map[uint16]int{IdolOfSorrow: 1}},
	"jail key":                          {Items: map[uint16]int{JailKey: 1}},
	"keystone":                          {Items: map[uint16]int{Keystone: 1}},
	"king's key":                        {Items: map[uint16]int{KingsKey: 1}},
	"moonlight sword":                   {Items: map[uint16]int{MoonlightSword: 1}},
	"priest's key":                      {Items: map[uint16]int{PriestsKey: 1}},
	"projectile":                        {Caps: []Cap{CapProjectile}},
	"projectile weapon":                 {Caps: []Cap{CapProjectile}},
	"release rock":                      {Items: map[uint16]int{ReleaseRock: 1}},
	"release rock, zastari's key":       {Items: map[uint16]int{ReleaseRock: 1, ZastarisKey: 1}},
	"ring of wisdom":                    {Items: map[uint16]int{RingOfWisdom: 1}},
	"rock of life":                      {Items: map[uint16]int{RockOfLife: 1}},
	"rusty key":                         {Items: map[uint16]int{RustyKey: 1}},
	"serrak's key":                      {Items: map[uint16]int{SerraksKey: 1}},
	"strange egg":                       {Items: map[uint16]int{StrangeEgg: 1}},
	"weapon can break container":        {Caps: []Cap{CapBreak}},
	"weapon can break container, binding crown": {Items: map[uint16]int{BindingCrown: 1}, Caps: []Cap{CapBreak}},
	"weapon can break woodplanks":               {},
	"widda's bracelet, necklace, ring":          {Items: map[uint16]int{WiddasBracelet: 1, WiddasNecklace: 1, WiddasRing: 1}},
	"widda's key":                               {Items: map[uint16]int{WiddaKey: 1}},
	"zastari's key":                             {Items: map[uint16]int{ZastarisKey: 1}},
}

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
	req      string
	oneWay   bool
}{
	{"2", "2A", "Rusty Key", false},
	{"98", "5", "Fire spell", false},
	{"14", "6", "Forest Dew", false},
	{"7", "41", "", false},
	{"6", "7", "", false},
	{"16", "20", "", false},
	{"20", "16A", "", false},
	{"17", "24", "", false},
	{"19", "16B", "Icon of Ice", false},
	{"16B", "105", "", false},
	{"106", "108", "", false},
	{"108", "107", "", false},
	{"107", "109", "", false},
	{"109", "110", "", false},
	{"110", "111", "", false},
	{"111", "31", "", false},
	{"26", "22", "widda's bracelet, necklace, ring", false},
	{"6A", "42", "Black Eye x4", false},
	{"6", "30", "Ring of Wisdom", true},
	{"30", "6A", "", true},
	{"35", "37", "Icon of Water", false},
	{"37", "36", "Jail Key", false},
	{"6A", "6", "King's Key", false},
	{"6", "6B", "Ring of Wisdom", false},
	{"12", "15", "Binding Crown", true},
	{"15", "12", "", true},
	{"37", "38", "Idol of Sorrow", false},
	// 102A is only reached by warping with the Green Wand, which works from
	// any teleport pad; 006 stands in for wherever the wand is picked up.
	{"6", "102A", "Green Wand", true},
	{"102A", "16", "", false},
	{"102A", "104", "", false},
	{"104", "102", "", true},

	// One forge in three states. The Fire Lord lights it, the icon freezes it,
	// and each state is only left by its doors.
	{"25", "18", "", true},
	{"18", "19", "Icon of Ice", true},
	{"18", "16", "", true},
	{"18", "20", "", true},
	{"18", "24", "", true},
	{"19", "20", "", true},
	{"19", "24", "", true},

	// The central tower's parts only meet through other maps. The bridge and
	// the doors from the throne and level 2 open from one side only.
	{"100A", "15", "", false},
	{"100A", "15A", "", false},
	{"100A", "15B", "", false},
	{"100A", "15C", "", false},
	{"100A", "101A", "", false},
	{"100A", "101B", "", false},
	{"100A", "100B", "Keystone", true},
	{"100B", "102", "", false},
	{"102", "100C", "Priest's Key", false},
	{"7", "100B", "", true},

	// 105 loops through a corner of the tower (100D) to a ledge (105A) whose
	// lever opens back onto 105. The bridge from 100D lowers onto 100B.
	{"105", "100D", "", false},
	{"100D", "105A", "", false},
	{"105A", "105", "", false},
	{"100D", "100B", "Keystone", true},
	{"105A", "106", "", false},

	// Level 2 tower: 107 from 108, 107B from 111, both onto the stairs (107A)
	// with a keystone.
	{"107", "107A", "Keystone", false},
	{"107B", "107A", "Keystone", false},
	{"107A", "100B", "", true},
	{"111", "107B", "", false},
	{"107B", "111A", "", false},
	{"109", "111", "", true},
	{"113", "111", "", true},
	{"115", "107", "", true},
}

// Checks that also need another place reached, keyed by region rather than
// map file.
var errands = map[Where]string{
	// The Seabed Cave trade wants a vial of poison water from the Icon of
	// Healing.
	{Map: "28", Slot: Ground, Index: 0}: "97",
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
