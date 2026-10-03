package paramdomizer

import (
	"math/rand/v2"
	"testing"

	"kf4randomizer/internal/kfconst"
)

func newTestRNG(seed uint64) *rand.Rand {
	return rand.New(rand.NewPCG(seed, seed))
}

func newWeaponBuf() []byte {
	buf := make([]byte, kfconst.ItemWeaponRecordSize*kfconst.ItemWeaponRecordCount)
	// Plausible vanilla-shaped stat block per level: Slash/Hit/Stab nonzero.
	for i := 1; i < kfconst.ItemWeaponRecordCount; i++ {
		for _, lo := range [3]int{kfconst.ItemWeaponLevel1Offset, kfconst.ItemWeaponLevel2Offset, kfconst.ItemWeaponLevel3Offset} {
			off := i*kfconst.ItemWeaponRecordSize + lo
			writeStatBlock(buf, off, StatBlock{75, 40, 60, 0, 0, 0, 0, 0, 0})
		}
	}
	return buf
}

func newArmorBuf() []byte {
	buf := make([]byte, kfconst.ItemArmorRecordSize*kfconst.ItemArmorRecordCount)
	for i := 1; i < kfconst.ItemArmorRecordCount; i++ {
		off := i*kfconst.ItemArmorRecordSize + kfconst.ItemArmorDefenseOffset
		writeStatBlock(buf, off, StatBlock{10, 5, 5, 0, 0, 0, 0, 0, 0})
	}
	return buf
}

func TestRun_ValidatesBufferSizes(t *testing.T) {
	if _, err := Run(make([]byte, 10), newArmorBuf(), newTestRNG(1), Options{PercentRange: 25}); err == nil {
		t.Error("expected an error for a wrong-size weapon buffer")
	}
	if _, err := Run(newWeaponBuf(), make([]byte, 10), newTestRNG(1), Options{PercentRange: 25}); err == nil {
		t.Error("expected an error for a wrong-size armor buffer")
	}
}

func TestRun_Deterministic(t *testing.T) {
	opts := Options{PercentRange: 25, AllowNegativeWeaponStats: true}

	w1, a1 := newWeaponBuf(), newArmorBuf()
	if _, err := Run(w1, a1, newTestRNG(12345), opts); err != nil {
		t.Fatal(err)
	}

	w2, a2 := newWeaponBuf(), newArmorBuf()
	if _, err := Run(w2, a2, newTestRNG(12345), opts); err != nil {
		t.Fatal(err)
	}

	if string(w1) != string(w2) {
		t.Error("same seed produced different weapon output")
	}
	if string(a1) != string(a2) {
		t.Error("same seed produced different armor output")
	}
}

func TestRun_RecordZeroUntouched(t *testing.T) {
	weaponBuf := newWeaponBuf() // record 0 is all-zero, as in the real file
	armorBuf := newArmorBuf()

	if _, err := Run(weaponBuf, armorBuf, newTestRNG(1), Options{PercentRange: 25, AllowNegativeWeaponStats: true}); err != nil {
		t.Fatal(err)
	}

	zero := make([]byte, kfconst.ItemWeaponRecordSize)
	if string(weaponBuf[:kfconst.ItemWeaponRecordSize]) != string(zero) {
		t.Error("weapon record 0 (the unarmed placeholder) should be left untouched")
	}
	zeroArmor := make([]byte, kfconst.ItemArmorRecordSize)
	if string(armorBuf[:kfconst.ItemArmorRecordSize]) != string(zeroArmor) {
		t.Error("armor record 0 (the nothing-equipped placeholder) should be left untouched")
	}
}

func TestRun_ArmorNeverNegative_EvenWithAllowNegativeWeaponStats(t *testing.T) {
	armorBuf := newArmorBuf()
	weaponBuf := newWeaponBuf()

	if _, err := Run(weaponBuf, armorBuf, newTestRNG(7), Options{PercentRange: 90, AllowNegativeWeaponStats: true}); err != nil {
		t.Fatal(err)
	}

	for i := 1; i < kfconst.ItemArmorRecordCount; i++ {
		off := i*kfconst.ItemArmorRecordSize + kfconst.ItemArmorDefenseOffset
		block := readStatBlock(armorBuf, off)
		for _, v := range block {
			if v < 0 {
				t.Fatalf("armor record %d has a negative stat %d despite armor never allowing negatives", i, v)
			}
		}
	}
}

func TestRun_WeaponNegativesOnlyWhenAllowed(t *testing.T) {
	weaponBuf := newWeaponBuf()
	// Large percent range makes a negative roll likely if possible at all.
	if _, err := Run(weaponBuf, newArmorBuf(), newTestRNG(99), Options{PercentRange: 90, AllowNegativeWeaponStats: false}); err != nil {
		t.Fatal(err)
	}
	for i := 1; i < kfconst.ItemWeaponRecordCount; i++ {
		for _, lo := range [3]int{kfconst.ItemWeaponLevel1Offset, kfconst.ItemWeaponLevel2Offset, kfconst.ItemWeaponLevel3Offset} {
			block := readStatBlock(weaponBuf, i*kfconst.ItemWeaponRecordSize+lo)
			for _, v := range block {
				if v < 0 {
					t.Fatalf("record %d has a negative stat %d with AllowNegativeWeaponStats=false", i, v)
				}
			}
		}
	}

	sawNegative := false
	weaponBuf2 := newWeaponBuf()
	if _, err := Run(weaponBuf2, newArmorBuf(), newTestRNG(99), Options{PercentRange: 90, AllowNegativeWeaponStats: true}); err != nil {
		t.Fatal(err)
	}
	for i := 1; i < kfconst.ItemWeaponRecordCount && !sawNegative; i++ {
		for _, lo := range [3]int{kfconst.ItemWeaponLevel1Offset, kfconst.ItemWeaponLevel2Offset, kfconst.ItemWeaponLevel3Offset} {
			block := readStatBlock(weaponBuf2, i*kfconst.ItemWeaponRecordSize+lo)
			for _, v := range block {
				if v < 0 {
					sawNegative = true
					break
				}
			}
		}
	}
	if !sawNegative {
		t.Error("expected at least one negative stat somewhere with AllowNegativeWeaponStats=true and a 90% range")
	}
}

func TestRun_LevelsNeverWorse_WhenDisallowed(t *testing.T) {
	weaponBuf := newWeaponBuf()
	if _, err := Run(weaponBuf, newArmorBuf(), newTestRNG(42), Options{PercentRange: 50, AllowWorseWeaponLevels: false}); err != nil {
		t.Fatal(err)
	}

	for i := 1; i < kfconst.ItemWeaponRecordCount; i++ {
		l1 := readStatBlock(weaponBuf, i*kfconst.ItemWeaponRecordSize+kfconst.ItemWeaponLevel1Offset).Total()
		l2 := readStatBlock(weaponBuf, i*kfconst.ItemWeaponRecordSize+kfconst.ItemWeaponLevel2Offset).Total()
		l3 := readStatBlock(weaponBuf, i*kfconst.ItemWeaponRecordSize+kfconst.ItemWeaponLevel3Offset).Total()
		if l2 < l1 {
			t.Fatalf("record %d: level 2 total %d is worse than level 1 total %d", i, l2, l1)
		}
		if l3 < l2 {
			t.Fatalf("record %d: level 3 total %d is worse than level 2 total %d", i, l3, l2)
		}
	}
}

func TestRun_LevelsCanBeWorse_WhenAllowed(t *testing.T) {
	// only checks there's no hidden floor
	weaponBuf := newWeaponBuf()
	if _, err := Run(weaponBuf, newArmorBuf(), newTestRNG(42), Options{PercentRange: 90, AllowWorseWeaponLevels: true}); err != nil {
		t.Fatal(err)
	}

	sawRegression := false
	for i := 1; i < kfconst.ItemWeaponRecordCount; i++ {
		l1 := readStatBlock(weaponBuf, i*kfconst.ItemWeaponRecordSize+kfconst.ItemWeaponLevel1Offset).Total()
		l2 := readStatBlock(weaponBuf, i*kfconst.ItemWeaponRecordSize+kfconst.ItemWeaponLevel2Offset).Total()
		l3 := readStatBlock(weaponBuf, i*kfconst.ItemWeaponRecordSize+kfconst.ItemWeaponLevel3Offset).Total()
		if l2 < l1 || l3 < l2 {
			sawRegression = true
			break
		}
	}
	if !sawRegression {
		t.Error("expected at least one level-to-level regression somewhere with AllowWorseWeaponLevels=true")
	}
}

func TestRandomizeBlock_UntouchedStatsKeepOriginalValue(t *testing.T) {
	rng := newTestRNG(1)
	original := StatBlock{75, 40, 60, 10, 20, 30, 40, 50, 60}
	result := randomizeBlock(rng, original, 25, true, nil)

	touchedCount := 0
	for i := range original {
		if result[i] != original[i] {
			touchedCount++
		}
	}
	if touchedCount < 1 || touchedCount > 9 {
		t.Errorf("expected 1-9 stats touched, got %d", touchedCount)
	}
}

func TestDistribute_SumsExactly(t *testing.T) {
	rng := newTestRNG(2)
	for _, tc := range []struct {
		total         int
		n             int
		allowNegative bool
	}{
		{45, 3, false},
		{0, 5, false},
		{-10, 4, true},
		{100, 1, false},
		{7, 9, true},
	} {
		parts := distribute(rng, tc.total, tc.n, tc.allowNegative)
		if len(parts) != tc.n {
			t.Errorf("total=%d n=%d: got %d parts, want %d", tc.total, tc.n, len(parts), tc.n)
		}
		sum := 0
		for _, p := range parts {
			sum += p
			if !tc.allowNegative && p < 0 {
				t.Errorf("total=%d n=%d: negative part %d with allowNegative=false", tc.total, tc.n, p)
			}
		}
		wantSum := tc.total
		if !tc.allowNegative && wantSum < 0 {
			wantSum = 0
		}
		if sum != wantSum {
			t.Errorf("total=%d n=%d: parts sum to %d, want %d", tc.total, tc.n, sum, wantSum)
		}
	}
}
