// Package kfconst holds file format offsets/sizes for all prm files and other useful stuff in binary/ISO.
package kfconst

// .map file layout: 128-byte header (11 uint32 chunk offsets, 13 uint16
// chunk counts), everything below derived from it.

const MapHeaderSize = 0x80

// Fixed-size chunks between the header and the first variable-length chunk.
const (
	DrawParamEntrySize  = 32
	DrawParamCount      = 32
	LightParamEntrySize = 128
	LightParamCount     = 32
	ScrollParamSize     = 0x340
	StructAFixedSize    = 512
)

// Per-entry sizes for variable-length chunks preceding the item array.
const (
	NodePieceEntrySize  = 128
	DispGroupEntrySize  = 64
	NodeObjectEntrySize = 128
	NodeEntityEntrySize = 96 // StructCreature
)

// ItemStructSize: one MAPNodeItem entry (48 bytes pos/rot/scale, classID
// u16 @0x30, omdID u16 @0x32, 12 unidentified trailing bytes).
const ItemStructSize = 64

const (
	ItemClassIDOffset   = 0x30
	ItemOmdIDOffset     = 0x32
	ItemPositionXOffset = 0x00
	ItemPositionZOffset = 0x08

	// ContainerItemPairTolerance: how close in X/Z a container and a ground
	// item must be to count as the same spot. Barrels are ~130 units across.
	ContainerItemPairTolerance = 40.0

	// ItemRespawnCapOffset: item hidden once pickup count >= this. 0 = no cap.
	ItemRespawnCapOffset = 0x36

	// ItemRespawnChanceOffset: 0-99 reroll chance on each room load/warp-in.
	// 100 always passes.
	ItemRespawnChanceOffset = 0x37

	// ItemEventIDOffset: map-script event that reveals this slot, 0 for none.
	ItemEventIDOffset = 0x3F

	// Values itemrespawn writes to force a one-time, no-reroll pickup.
	ItemRespawnCapOnce      = 1
	ItemRespawnChanceAlways = 100
)

// NodeObject ClassParams+4 (absolute offset 0x44), uint16 LE: contained
// item classID, 0 = empty.
const (
	NodeObjectClassIDOffset     = 0x30
	NodeObjectItemClassIDOffset = 0x44
	NodeObjectPositionYOffset   = 0x04
	NodeObjectPositionXOffset   = 0x00
	NodeObjectPositionZOffset   = 0x08

	// Byte +0x3A is the object's event ID: the game indexes its live
	// object-by-event table with it, so scripts can address one object.
	NodeObjectEventIDOffset = 0x3A

	// Nonzero means a map script drives the object (switches, doors, puzzle
	// chests). The exact meaning is unknown.
	NodeObjectScriptRefOffset = 0x40

	// Receptacles (obj.prm dispatch type 13) start holding their key when
	// this is 1. Three Binding Crown pedestals use it.
	NodeObjectPrefilledOffset = 0x52
)

// param/obj.prm is a flat table of per-classID object parameters, loaded at 0x38DA30 as KFParamObject.
// Byte 0 of each record selects the object's behavior handler from the game's dispatch table.
const ObjParamRecordSize = 128

// ContainerDispatchTypes are the obj.prm handler types whose ClassParams+4 holds a container-item classID.
var ContainerDispatchTypes = map[byte]bool{
	3:  true,
	4:  true,
	21: true,
}

// ContainerRemovalYOffset moves an empty container far outside map
// geometry so it's culled and never collided with. Y is inverted in this
// game (negative is up), so this lifts it well above the level.
const ContainerRemovalYOffset float32 = -100000.0

// NodeEntity (StructCreature) offset 0x38, uint16 LE: classID of the item
// dropped on kill. NodeEntityNoDropSentinel = no drop.
const (
	NodeEntityDropClassIDOffset = 0x38
	NodeEntityNoDropSentinel    = 0xFFFF
)

// Item-model chunk: length-prefixed model/texture blob pairs, one per
// whitelist entry.
const (
	MapHeaderItemChunkOffset = 0x20
	ItemWhitelistMaxEntries  = 32
)

// item_nam.prm: one flat table of 0x40-byte records. A map item's classID
// is a 1-based row index. Laid out as consecutive fixed-size category pages.
type ItemCategory struct {
	Name       string
	FirstClass int // inclusive, 1-based classID
	LastClass  int // inclusive, 1-based classID
}

type itemCategoryPage struct {
	name string
	size int
}

// ItemCategories: item_nam.prm's pages in on-disk order (380 rows total).
var ItemCategories = buildItemCategories([]itemCategoryPage{
	{"Items", 150},
	{"Weapons", 70},
	{"Shields", 30},
	{"Helmets", 25},
	{"Cuirass", 25},
	{"Gauntlets", 25},
	{"Greaves", 25},
	{"Rings", 10},
	{"Bracelets", 10},
	{"Amulets", 10},
})

func buildItemCategories(pages []itemCategoryPage) []ItemCategory {
	cats := make([]ItemCategory, 0, len(pages))
	row := 1 // classID is 1-based
	for _, p := range pages {
		cats = append(cats, ItemCategory{Name: p.name, FirstClass: row, LastClass: row + p.size - 1})
		row += p.size
	}
	return cats
}

// IsKnownClassID reports whether classID falls within an ItemCategories page.
func IsKnownClassID(classID uint16) bool {
	for _, cat := range ItemCategories {
		if int(classID) >= cat.FirstClass && int(classID) <= cat.LastClass {
			return true
		}
	}
	return false
}

// ExcludedFromRandomizationClassIDs: classIDs kept out of every shuffle pool.
// 196 (Moonlight Sword) is a quest reward tied to a dedicated one-item map.
var ExcludedFromRandomizationClassIDs = map[uint16]bool{
	196: true,
}

func IsExcludedFromRandomization(classID uint16) bool {
	return ExcludedFromRandomizationClassIDs[classID]
}

// KF4.DAT: 64-byte header (length, file count) + one 64-byte file-table
// entry per member. Members pad to a 32KB boundary.
const (
	DatHeaderSize    = 64
	DatFileEntrySize = 64
	DatNameFieldSize = 52
)

const IsoSectorSize = 2048

// Every map has a 64-entry whitelist gating item render-resource registration
// on room load. Rather than rebuild all 64 tables, the check function is
// patched to jump straight to its "match found" path.
const (
	SlusWhitelistCheckPatchVAddr = 0x1CA4B4
	SlusWhitelistCheckJumpTarget = 0x1CA4DC

	// .text segment vaddr 0x100000 backs file offset 0x1000.
	SlusElfTextVAddr   = 0x100000
	SlusElfTextFileOff = 0x1000
)

func SlusWhitelistCheckJumpInstruction() uint32 {
	const jOpcode = 0x2
	return (jOpcode << 26) | ((SlusWhitelistCheckJumpTarget >> 2) & 0x3FFFFFF)
}

// Retail picks the floor piece with a strict <, so on emulators two pieces at
// the same height turn some doorways black (105 into 100). PCSX2's own fix
// only applies to an unmodified SLUS, so every build needs this one.
const (
	SlusBlackVoidFixVAddr = 0x1BE42C

	// c.ole.s $f0, $f0, $f21, the vanilla c.olt.s with <= instead of <.
	SlusBlackVoidFixInstruction = 0x46150036
)

// The wand case in KFItemUse only warps when standing on a teleport pad.
// Branching straight to the warp call skips that check.
const (
	SlusWandPadCheckVAddr = 0x1BC598

	// b 0x1BC5B0, jal to KFWandWarpToStone
	SlusWandPadCheckSkip = 0x10000005
)

func SlusVAddrToFileOffset(vaddr int) int {
	return vaddr - SlusElfTextVAddr + SlusElfTextFileOff
}

// Pickup name popup, written into unused space in the SLUS.
const (
	SlusPickupPopupCodeVAddr = 0x431C90

	// Collision buffer base pointers start here, so nothing may be written
	// at or past it.
	SlusPickupPopupCaveEndVAddr = 0x432208

	// A jal inside the pickup finalizer at 0x1CA0E0, redirected here.
	SlusPickupPopupHookVAddr = 0x1CA124
)

func SlusPickupPopupHookJumpInstruction() uint32 {
	const jalOpcode = 0x3
	return (jalOpcode << 26) | ((SlusPickupPopupCodeVAddr >> 2) & 0x3FFFFFF)
}

// Debug HUD: a ~400-byte loader in the popup's space reads the rest from the
// disc.
const (
	// Past both popup variants (464 bytes, the larger one).
	SlusDebugHUDBootstrapVAddr = SlusPickupPopupCodeVAddr + 464

	// The scene render's `jal 0x1CF4D0`. Map streaming calls the render too, so
	// anything counting frames uses the game's own counter.
	SlusDebugHUDHookVAddr = 0x100CA0

	// Exact assembled length of DebugHUDBootstrap, including its trailing
	// filename string.
	SlusDebugHUDBootstrapSize = 404

	SlusDebugHUDPayloadMaxSize = 5 * 2048

	// Bootstrap's own scratch words: DONE (payload ready), PTR (allocated
	// block base), FRAMECOUNTER (consecutive stable-Ready frames).
	SlusDebugHUDDoneVAddr         = SlusDebugHUDBootstrapVAddr + SlusDebugHUDBootstrapSize
	SlusDebugHUDPtrVAddr          = SlusDebugHUDDoneVAddr + 4
	SlusDebugHUDFrameCounterVAddr = SlusDebugHUDPtrVAddr + 4

	// The menu's Levitate row flips this word and the stub reads it, so the
	// payload has the same address baked in.
	SlusLevitateFlagVAddr = SlusDebugHUDFrameCounterVAddr + 4
	SlusLevitateStubVAddr = SlusLevitateFlagVAddr + 4

	// One past the `sd ra` in the floor resolve's prologue, so the jal can
	// clobber ra.
	SlusLevitateHookVAddr = 0x1BDDCC

	// DAT member whose padding slack holds DebugHUDPayload. game loads a # of bytes, so the rest of the file is
	// free to use
	DebugHUDPayloadHostDatMember = "chr/c0150.chr"

	// One CD sector into that member's slot.
	DebugHUDPayloadSlackByteOffset = 2048
)

func SlusDebugHUDHookJumpInstruction() uint32 {
	const jalOpcode = 0x3
	return (jalOpcode << 26) | ((SlusDebugHUDBootstrapVAddr >> 2) & 0x3FFFFFF)
}

func SlusLevitateHookJumpInstruction() uint32 {
	const jalOpcode = 0x3
	return (jalOpcode << 26) | ((SlusLevitateStubVAddr >> 2) & 0x3FFFFFF)
}

// ISO9660 paths of the two files a build patches.

const IsoExecutableName = "SLUS_203.18"

var IsoDatPath = []string{"DATA", "KF4.DAT"}

// KnownGoodISOMD5: MD5 of an original "King's Field: The Ancient City"
// (NTSC-US) ISO.
const KnownGoodISOMD5 = "e4c7bc4021db07893580f70e66cff3f6"

// The 5 DUMMY*.OUT files are ~512MB of zeros each that nothing reads, in two
// blocks: A (four back to back) and B (one alone).
var DummyRegionAFiles = []string{"DUMMY_01.OUT", "DUMMY_02.OUT", "DUMMY_03.OUT", "DUMMY_04.OUT"}

const DummyRegionBFile = "DUMMY.OUT"

// item_wpn.prm and item_pct.prm are plain record arrays, record 0 unused. A
// stat block is 9 int16s (Slash, Hit, Stab, Fire, Earth, Wind, Water, Light,
// Dark), one per weapon upgrade level and one for armor.
const (
	ItemWeaponRecordSize  = 144
	ItemWeaponRecordCount = 70
	ItemArmorRecordSize   = 64
	ItemArmorRecordCount  = 160

	ItemWeaponLevel1Offset = 0x00
	ItemWeaponLevel2Offset = 0x12
	ItemWeaponLevel3Offset = 0x24

	ItemArmorDefenseOffset = 0x00
)

// Shop related values
const (
	ShopCount     = 5
	ShopGoldCount = 4 // shop id 4 trades in bones, scales and claws
	ShopItemCount = 380
	// Each shop has 384 bytes, 1 for each item (for ex, byte 6 = Herb quantity sold, because herb is ID6).
	// last 4 bytes are padding to reach 384 (idk why)
	ShopStockSize = 384
	// Each shop has a 5F0 table for prices; 4 bytes per item (2 bytes = buy price, 2 bytes = sell price)
	ShopPriceSize = 0x5F0
	// starting address for item quantity tables
	SlusShopStockVAddr = 0x43D9B0
	// starting address for price tables
	SlusShopPriceVAddr = 0x426650
)
