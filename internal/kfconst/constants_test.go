package kfconst

import "testing"

func TestSlusWhitelistCheckJumpInstruction(t *testing.T) {
	// Hand-computed J encoding: (0x2<<26) | ((0x1CA4DC>>2)&0x3FFFFFF).
	const want = 0x08072937
	if got := SlusWhitelistCheckJumpInstruction(); got != want {
		t.Errorf("got 0x%08X, want 0x%08X", got, want)
	}
}

func TestSlusPickupPopupHookJumpInstruction(t *testing.T) {
	// Hand-computed JAL encoding: (0x3<<26) | ((0x431C90>>2)&0x3FFFFFF).
	const want = 0x0C10C724
	if got := SlusPickupPopupHookJumpInstruction(); got != want {
		t.Errorf("got 0x%08X, want 0x%08X", got, want)
	}
}

func TestSlusVAddrToFileOffset(t *testing.T) {
	if got := SlusVAddrToFileOffset(0x100000); got != 0x1000 {
		t.Errorf("start of .text: got 0x%X, want 0x1000", got)
	}
	if got := SlusVAddrToFileOffset(0x1CA4DC); got != 0xCB4DC {
		t.Errorf("got 0x%X, want 0xCB4DC", got)
	}
}

func TestIsKnownClassID(t *testing.T) {
	cases := []struct {
		classID uint16
		want    bool
	}{
		{0, false},   // classID is 1-based; 0 always means "empty slot"
		{1, true},    // first Items row
		{150, true},  // last Items row
		{151, true},  // first Weapons row
		{380, true},  // last Amulets row (final category)
		{381, false}, // one past the end of every category
		{65535, false},
	}
	for _, c := range cases {
		if got := IsKnownClassID(c.classID); got != c.want {
			t.Errorf("IsKnownClassID(%d) = %v, want %v", c.classID, got, c.want)
		}
	}
}

func TestIsExcludedFromRandomization(t *testing.T) {
	if !IsExcludedFromRandomization(196) {
		t.Error("classID 196 (Moonlight Sword) should be excluded from randomization")
	}
	if IsExcludedFromRandomization(195) {
		t.Error("classID 195 (Lawful Blade) should not be excluded")
	}
}
