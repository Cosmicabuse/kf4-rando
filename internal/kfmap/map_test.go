package kfmap

import (
	"encoding/binary"
	"testing"

	"kf4randomizer/internal/kfconst"
)

func buildHeaderBuf(numNodePiece, numDispGroup, numNodeObject, numNodeEntity, numNodeItem uint16) []byte {
	buf := make([]byte, kfconst.MapHeaderSize)
	binary.LittleEndian.PutUint16(buf[offNumNodePiece:], numNodePiece)
	binary.LittleEndian.PutUint16(buf[offNumDispGroup:], numDispGroup)
	binary.LittleEndian.PutUint16(buf[offNumNodeObject:], numNodeObject)
	binary.LittleEndian.PutUint16(buf[offNumNodeEntity:], numNodeEntity)
	binary.LittleEndian.PutUint16(buf[offNumNodeItem:], numNodeItem)
	return buf
}

// buildMinimalMap has one container, one monster drop and two items.
func buildMinimalMap(t *testing.T) []byte {
	t.Helper()
	h := buildHeaderBuf(0, 0, 1, 1, 2)

	nodeObjOff := nodeObjectOffset(header{NumNodePiece: 0, NumDispGroup: 0})
	entityOff := nodeObjOff + kfconst.NodeObjectEntrySize
	itemOff := entityOff + kfconst.NodeEntityEntrySize
	totalSize := itemOff + 2*kfconst.ItemStructSize

	buf := make([]byte, totalSize)
	copy(buf, h)

	// NodeObject 0: a container class (see testParams), holding classID 42.
	binary.LittleEndian.PutUint16(buf[nodeObjOff+kfconst.NodeObjectClassIDOffset:], 2)
	binary.LittleEndian.PutUint16(buf[nodeObjOff+kfconst.NodeObjectItemClassIDOffset:], 42)

	// NodeEntity 0: drops classID 99.
	binary.LittleEndian.PutUint16(buf[entityOff+kfconst.NodeEntityDropClassIDOffset:], 99)

	// Item 0: classID 7, omdID 3. Item 1: empty slot (classID 0).
	binary.LittleEndian.PutUint16(buf[itemOff+kfconst.ItemClassIDOffset:], 7)
	binary.LittleEndian.PutUint16(buf[itemOff+kfconst.ItemOmdIDOffset:], 3)

	return buf
}

// testParams marks classID 2 as a container and nothing else.
func testParams() ObjectParams {
	p := make(ObjectParams, 640*kfconst.ObjParamRecordSize)
	p[2*kfconst.ObjParamRecordSize] = 4
	return p
}

func TestLoad(t *testing.T) {
	buf := buildMinimalMap(t)

	m, err := Load("000.map", buf, testParams())
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if len(m.Items) != 2 {
		t.Fatalf("got %d items, want 2", len(m.Items))
	}
	if m.Items[0].ClassID != 7 || m.Items[0].OmdID != 3 {
		t.Errorf("item 0: got ClassID=%d OmdID=%d, want 7/3", m.Items[0].ClassID, m.Items[0].OmdID)
	}
	if m.Items[1].ClassID != 0 {
		t.Errorf("item 1: got ClassID=%d, want 0 (empty slot)", m.Items[1].ClassID)
	}

	if len(m.Containers) != 1 {
		t.Fatalf("got %d containers, want 1", len(m.Containers))
	}
	if m.Containers[0].ItemClassID != 42 {
		t.Errorf("container 0: got ItemClassID=%d, want 42", m.Containers[0].ItemClassID)
	}

	if len(m.Creatures) != 1 {
		t.Fatalf("got %d creatures, want 1", len(m.Creatures))
	}
	if m.Creatures[0].DropClassID != 99 {
		t.Errorf("creature 0: got DropClassID=%d, want 99", m.Creatures[0].DropClassID)
	}
}

func TestLoadTooShort(t *testing.T) {
	_, err := Load("000.map", make([]byte, 4), testParams())
	if err == nil {
		t.Fatal("expected an error for a too-short buffer, got nil")
	}
}
