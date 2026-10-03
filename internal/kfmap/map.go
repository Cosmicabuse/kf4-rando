// Package kfmap reads and writes the item, container and monster arrays of
// .map files.
package kfmap

import (
	"encoding/binary"
	"fmt"
	"math"

	"kf4randomizer/internal/kfconst"
)

// Map is a parsed .map file, with its raw buffer kept alongside so patched
// fields can be written straight back into it.
type Map struct {
	Name         string // "007.map", also its DAT key
	Buf          []byte
	ItemOffset   int
	EntityOffset int
	Items        []Item
	Containers   []Container
	Creatures    []Creature
}

// Entity returns one NodeEntity's raw record as a slice of Buf, for the rare
// edit that needs fields this package does not parse. Call RefreshCreatures
// afterwards if the drop changed.
func (m *Map) Entity(index int) []byte {
	o := m.EntityOffset + index*kfconst.NodeEntityEntrySize
	return m.Buf[o : o+kfconst.NodeEntityEntrySize]
}

// RefreshCreatures re-reads the entity array. Creatures only lists entities
// that carry a drop, so editing one can add or remove an entry.
func (m *Map) RefreshCreatures() {
	m.Creatures = m.Creatures[:0]
	for i := 0; ; i++ {
		o := m.EntityOffset + i*kfconst.NodeEntityEntrySize
		if o+kfconst.NodeEntityEntrySize > m.ItemOffset {
			return
		}
		drop := binary.LittleEndian.Uint16(m.Buf[o+kfconst.NodeEntityDropClassIDOffset:])
		if drop == 0 || drop == kfconst.NodeEntityNoDropSentinel {
			continue
		}
		m.Creatures = append(m.Creatures, Creature{Index: i, Offset: o, DropClassID: drop})
	}
}

type Item struct {
	Index   int // slot index within this map, 0-based
	Offset  int // byte offset of this item's 64-byte struct within Buf
	ClassID uint16

	// Which of the map's model/texture pairs this slot renders as. Nothing
	// writes it yet; it is what per-item models would have to move.
	OmdID uint16

	// RespawnChance is the percent roll on each room load. 0 means the
	// slot never appears on its own and is placed by an event instead
	// (boss/NPC death, quest trigger). See internal/features/itemrespawn.
	RespawnChance byte

	// Map-script event that reveals this slot, 0 for none.
	EventID byte
}

// Unspawnable slots never appear: no respawn roll and no event. Three exist.
func (i Item) Unspawnable() bool {
	return i.ClassID != 0 && i.RespawnChance == 0 && i.EventID == 0
}

// Containers and Creatures only list some objects, so a slice position isn't
// the Index. The setters take the struct for that reason.
type Container struct {
	Index       int // NodeObject index within this map, counting every object
	Offset      int // byte offset of this NodeObject's 128-byte struct within Buf
	ClassID     uint16
	ItemClassID uint16 // ClassParams+4; 0 = empty

	// ItemClassID before any shuffle.
	VanillaItemClassID uint16
	EventID            byte   // +0x3A, how a map script addresses this object
	ScriptRef          uint16 // 0 = no map script drives this object
}

// Creature is one NodeEntity slot that has a guaranteed item drop
// (see kfconst.NodeEntityDropClassIDOffset).
type Creature struct {
	Index       int // NodeEntity index within this map, counting every entity
	Offset      int // byte offset of this NodeEntity's 96-byte struct within Buf
	DropClassID uint16
}

// ObjectParams is a raw param/obj.prm buffer, indexed by object classID.
type ObjectParams []byte

// IsContainer reports whether classID's handler type is a container kind.
// A nil ObjectParams reports false for everything.
func (p ObjectParams) IsContainer(classID uint16) bool {
	off := int(classID) * kfconst.ObjParamRecordSize
	if off >= len(p) {
		return false
	}
	return kfconst.ContainerDispatchTypes[p[off]]
}

// The 128-byte header is 11 uint32 chunk offsets followed by 13 uint16 chunk
// counts, in this order: NumPieceA, NumPieceCollision, NumObjectA, NumObjectB,
// NumObjectCollision, NumAnimation, NumUnused, NumItem, NumNodePiece,
// NumDispGroup, NumNodeObject, NumNodeEntity, NumNodeItem. Only the counts
// that place the item array are read.
const (
	numsBase         = 11 * 4
	offNumNodePiece  = numsBase + 8*2
	offNumDispGroup  = numsBase + 9*2
	offNumNodeObject = numsBase + 10*2
	offNumNodeEntity = numsBase + 11*2
	offNumNodeItem   = numsBase + 12*2
)

type header struct {
	NumNodePiece  uint16
	NumDispGroup  uint16
	NumNodeObject uint16
	NumNodeEntity uint16
	NumNodeItem   uint16
}

func parseHeader(buf []byte) header {
	u16 := func(off int) uint16 { return binary.LittleEndian.Uint16(buf[off:]) }
	return header{
		NumNodePiece:  u16(offNumNodePiece),
		NumDispGroup:  u16(offNumDispGroup),
		NumNodeObject: u16(offNumNodeObject),
		NumNodeEntity: u16(offNumNodeEntity),
		NumNodeItem:   u16(offNumNodeItem),
	}
}

// nodeObjectOffset is where the NodeObject array starts, counted from the
// start of the map buffer.
func nodeObjectOffset(h header) int {
	off := kfconst.MapHeaderSize
	off += kfconst.DrawParamCount * kfconst.DrawParamEntrySize
	off += kfconst.LightParamCount * kfconst.LightParamEntrySize
	off += kfconst.ScrollParamSize
	off += int(h.NumNodePiece) * kfconst.NodePieceEntrySize
	off += int(h.NumDispGroup) * kfconst.DispGroupEntrySize
	off += kfconst.StructAFixedSize
	return off
}

// The NodeEntity array follows the NodeObject array, and the item array
// follows that.
func entityOffset(h header) int {
	return nodeObjectOffset(h) + int(h.NumNodeObject)*kfconst.NodeObjectEntrySize
}

func itemArrayOffset(h header) int {
	return entityOffset(h) + int(h.NumNodeEntity)*kfconst.NodeEntityEntrySize
}

// Load parses a .map file already read into memory. params comes from
// param/obj.prm and decides which objects are containers; pass nil to skip
// container parsing entirely.
func Load(name string, buf []byte, params ObjectParams) (*Map, error) {
	if len(buf) < kfconst.MapHeaderSize {
		return nil, fmt.Errorf("%s: too short to be a valid map file (%d bytes)", name, len(buf))
	}
	h := parseHeader(buf)

	nodeObjOff := nodeObjectOffset(h)
	var containers []Container
	for i := 0; i < int(h.NumNodeObject); i++ {
		o := nodeObjOff + i*kfconst.NodeObjectEntrySize
		if o+kfconst.NodeObjectEntrySize > len(buf) {
			return nil, fmt.Errorf("%s: NodeObject array overruns buffer (object %d at offset %d, buffer is %d bytes)", name, i, o, len(buf))
		}
		classID := binary.LittleEndian.Uint16(buf[o+kfconst.NodeObjectClassIDOffset:])
		if !params.IsContainer(classID) {
			continue
		}
		item := binary.LittleEndian.Uint16(buf[o+kfconst.NodeObjectItemClassIDOffset:])
		containers = append(containers, Container{
			Index:              i,
			Offset:             o,
			ClassID:            classID,
			ItemClassID:        item,
			VanillaItemClassID: item,
			EventID:            buf[o+kfconst.NodeObjectEventIDOffset],
			ScriptRef:          binary.LittleEndian.Uint16(buf[o+kfconst.NodeObjectScriptRefOffset:]),
		})
	}

	entityOff := entityOffset(h)
	var creatures []Creature
	for i := 0; i < int(h.NumNodeEntity); i++ {
		o := entityOff + i*kfconst.NodeEntityEntrySize
		if o+kfconst.NodeEntityEntrySize > len(buf) {
			return nil, fmt.Errorf("%s: NodeEntity array overruns buffer (entity %d at offset %d, buffer is %d bytes)", name, i, o, len(buf))
		}
		dropClassID := binary.LittleEndian.Uint16(buf[o+kfconst.NodeEntityDropClassIDOffset:])
		if dropClassID == 0 || dropClassID == kfconst.NodeEntityNoDropSentinel {
			continue
		}
		creatures = append(creatures, Creature{
			Index:       i,
			Offset:      o,
			DropClassID: dropClassID,
		})
	}

	itemOff := itemArrayOffset(h)
	items := make([]Item, h.NumNodeItem)
	for i := range items {
		o := itemOff + i*kfconst.ItemStructSize
		if o+kfconst.ItemStructSize > len(buf) {
			return nil, fmt.Errorf("%s: item array overruns buffer (item %d at offset %d, buffer is %d bytes)", name, i, o, len(buf))
		}
		items[i] = Item{
			Index:         i,
			Offset:        o,
			ClassID:       binary.LittleEndian.Uint16(buf[o+kfconst.ItemClassIDOffset:]),
			OmdID:         binary.LittleEndian.Uint16(buf[o+kfconst.ItemOmdIDOffset:]),
			RespawnChance: buf[o+kfconst.ItemRespawnChanceOffset],
			EventID:       buf[o+kfconst.ItemEventIDOffset],
		}
	}

	return &Map{
		Name: name, Buf: buf,
		ItemOffset: itemOff, EntityOffset: entityOff,
		Items: items, Containers: containers, Creatures: creatures,
	}, nil
}

func (m *Map) SetItemClassID(itemIndex int, classID uint16) {
	o := m.Items[itemIndex].Offset
	binary.LittleEndian.PutUint16(m.Buf[o+kfconst.ItemClassIDOffset:], classID)
	m.Items[itemIndex].ClassID = classID
}

// SetContainerItemClassID writes c's contained-item classID into Buf. c is a
// copy, so c.ItemClassID still holds the old value afterwards; read the new
// one with ContainerItemClassID.
func (m *Map) SetContainerItemClassID(c Container, classID uint16) {
	binary.LittleEndian.PutUint16(m.Buf[c.Offset+kfconst.NodeObjectItemClassIDOffset:], classID)
}

func (m *Map) ContainerItemClassID(c Container) uint16 {
	return binary.LittleEndian.Uint16(m.Buf[c.Offset+kfconst.NodeObjectItemClassIDOffset:])
}

// SetCreatureDropClassID writes c's drop-item classID into Buf. c is a copy,
// so c.DropClassID still holds the old value afterwards; read the new one
// with CreatureDropClassID.
func (m *Map) SetCreatureDropClassID(c Creature, classID uint16) {
	binary.LittleEndian.PutUint16(m.Buf[c.Offset+kfconst.NodeEntityDropClassIDOffset:], classID)
}

func (m *Map) CreatureDropClassID(c Creature) uint16 {
	return binary.LittleEndian.Uint16(m.Buf[c.Offset+kfconst.NodeEntityDropClassIDOffset:])
}

// SetItemRespawnOnce overwrites one item slot's respawn-cap and respawn-chance
// bytes, making it a one-time pickup. See internal/features/itemrespawn.
func (m *Map) SetItemRespawnOnce(itemIndex int) {
	o := m.Items[itemIndex].Offset
	m.Buf[o+kfconst.ItemRespawnCapOffset] = kfconst.ItemRespawnCapOnce
	m.Buf[o+kfconst.ItemRespawnChanceOffset] = kfconst.ItemRespawnChanceAlways
}

// RelocateContainerOutOfBounds shifts c's Y by kfconst.ContainerRemovalYOffset.
func (m *Map) RelocateContainerOutOfBounds(c Container) {
	o := c.Offset + kfconst.NodeObjectPositionYOffset
	y := math.Float32frombits(binary.LittleEndian.Uint32(m.Buf[o:]))
	y += kfconst.ContainerRemovalYOffset
	binary.LittleEndian.PutUint32(m.Buf[o:], math.Float32bits(y))
}

// PairedWithItem reports whether a ground item shares c's X/Z footprint.
// Those containers deliver that item (it sits parked far off in Y until
// the container is opened), so they are not really empty.
func (m *Map) PairedWithItem(c Container) bool {
	cx := m.float32At(c.Offset + kfconst.NodeObjectPositionXOffset)
	cz := m.float32At(c.Offset + kfconst.NodeObjectPositionZOffset)
	for _, it := range m.Items {
		if it.ClassID == 0 {
			continue
		}
		dx := m.float32At(it.Offset+kfconst.ItemPositionXOffset) - cx
		dz := m.float32At(it.Offset+kfconst.ItemPositionZOffset) - cz
		if abs32(dx) < kfconst.ContainerItemPairTolerance && abs32(dz) < kfconst.ContainerItemPairTolerance {
			return true
		}
	}
	return false
}

// StackedOnFilledContainer reports whether a filled container sits on c.
// Moving the empty one of such a pair made Myu's Necklace unobtainable.
func (m *Map) StackedOnFilledContainer(c Container) bool {
	cx := m.float32At(c.Offset + kfconst.NodeObjectPositionXOffset)
	cy := m.float32At(c.Offset + kfconst.NodeObjectPositionYOffset)
	cz := m.float32At(c.Offset + kfconst.NodeObjectPositionZOffset)
	for _, o := range m.Containers {
		if o.Index == c.Index || o.ItemClassID == 0 {
			continue
		}
		dx := m.float32At(o.Offset+kfconst.NodeObjectPositionXOffset) - cx
		dy := m.float32At(o.Offset+kfconst.NodeObjectPositionYOffset) - cy
		dz := m.float32At(o.Offset+kfconst.NodeObjectPositionZOffset) - cz
		if abs32(dx) < kfconst.ContainerItemPairTolerance &&
			abs32(dy) < kfconst.ContainerItemPairTolerance &&
			abs32(dz) < kfconst.ContainerItemPairTolerance {
			return true
		}
	}
	return false
}

func (m *Map) float32At(off int) float32 {
	return math.Float32frombits(binary.LittleEndian.Uint32(m.Buf[off:]))
}

func abs32(v float32) float32 {
	if v < 0 {
		return -v
	}
	return v
}
