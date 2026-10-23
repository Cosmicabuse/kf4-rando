package tracker

import (
	"testing"
	"time"

	"kf4randomizer/internal/kfconst"
	"kf4randomizer/internal/kfmap"
)

func mapWithItems(name string, slots map[int]uint16) *kfmap.Map {
	m := &kfmap.Map{Name: name}
	for i, id := range slots {
		m.Items = append(m.Items, kfmap.Item{Index: i, ClassID: id})
	}
	return m
}

func TestCountSkipsCopiesAndDupes(t *testing.T) {
	maps := []*kfmap.Map{
		// KITE SHIELD dupe group across two forge states
		mapWithItems("017.map", map[int]uint16{3: 14, 99: 14}),
		mapWithItems("018.map", map[int]uint16{2: 14}),
		// Skip and Copy slots
		mapWithItems("000.map", map[int]uint16{3: 14}),
		mapWithItems("100.map", map[int]uint16{0: 14}),
	}
	if got := Count(maps, nil, nil, nil, 14); got != 2 {
		t.Errorf("Count = %d, want 2", got)
	}
}

func TestCountChrDrop(t *testing.T) {
	chr := make([]byte, 16*0x450)
	if err := kfmap.SetChrDrop(chr, 15, 14); err != nil {
		t.Fatal(err)
	}
	if got := Count(nil, nil, chr, nil, 14); got != 1 {
		t.Errorf("Count = %d, want 1", got)
	}
	if got := Count(nil, nil, chr, nil, 107); got != 0 {
		t.Errorf("Count = %d, want 0", got)
	}
}

func TestCountShopAndIni(t *testing.T) {
	stock := make([]byte, kfconst.ShopCount*kfconst.ShopStockSize)
	stock[3*kfconst.ShopStockSize+14] = 1
	if got := Count(nil, stock, nil, []byte("item_014=2\n"), 14); got != 3 {
		t.Errorf("Count = %d, want 3", got)
	}
	stock[4*kfconst.ShopStockSize+14] = 255
	if got := Count(nil, stock, nil, nil, 14); got != 2 {
		t.Errorf("unlimited stock counts %d, want 2", got)
	}
}

func TestNewStart(t *testing.T) {
	ini := []byte("item_049=1\nitem_196=1\nitem_106=7\nmagic_fire_1=1\nmagic_fire_2=0\nmagic_fire_3=2\n")
	f := New(18446744073709551615, nil, nil, nil, ini, time.Unix(0, 0))
	want := map[string]int{
		"rusty_key": 1, "sword": 2, "keystone": 5, "magic_fire": 2,
		"green_stone": 1, "blue_stone": 1, "binding_crown": 1, "idol": 0,
	}
	for id, v := range want {
		if f.Start[id] != v {
			t.Errorf("Start[%s] = %d, want %d", id, f.Start[id], v)
		}
		if f.State[id] != v {
			t.Errorf("State[%s] = %d, want %d", id, f.State[id], v)
		}
	}
	if f.Seed != "18446744073709551615" {
		t.Errorf("Seed = %q", f.Seed)
	}
	if _, ok := f.Max["keystone"]; ok {
		t.Error("keystone has a Max override")
	}
	if _, ok := f.Max["binding_crown"]; ok {
		t.Error("binding_crown has a Max override")
	}
	if f.Cap("binding_crown") != 3 {
		t.Errorf("Cap(binding_crown) = %d, want 3", f.Cap("binding_crown"))
	}
}

func TestNewSwordOne(t *testing.T) {
	f := New(1, nil, nil, nil, []byte("item_195=1\n"), time.Time{})
	if f.Start["sword"] != 1 {
		t.Errorf("sword = %d, want 1", f.Start["sword"])
	}
}

func TestNewNoVials(t *testing.T) {
	f := New(1, nil, nil, nil, nil, time.Time{})
	if v, ok := f.Max["crystal_vial"]; !ok || v != 0 {
		t.Errorf("Max[crystal_vial] = %d, %v, want 0 present", v, ok)
	}
	if f.Cap("crystal_vial") != 1 {
		t.Errorf("Cap = %d, want 1", f.Cap("crystal_vial"))
	}
}
