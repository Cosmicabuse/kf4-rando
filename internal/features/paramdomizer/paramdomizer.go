// Package paramdomizer rerolls weapon and armor stats: a new total within a
// percentage of the vanilla one, spread over a random subset of the 9 stat
// slots each record carries.
package paramdomizer

import (
	"encoding/binary"
	"fmt"
	"math"
	"math/rand/v2"
	"sort"

	"kf4randomizer/internal/kfconst"
)

// NewRand salts the seed so each pass gets its own stream.
func NewRand(seed uint64) *rand.Rand {
	return rand.New(rand.NewPCG(seed, seed^0xbf58476d1ce4e5b9))
}

// StatBlock is one weapon level or one armor set of stats :
// Slash/PhysEdge, Hit/PhysBlunt, Stab/PhysPierce, Fire, Earth, Wind, Water, Light, Dark.
type StatBlock [9]int16

// Total is an int so the sum can't overflow int16.
func (s StatBlock) Total() int {
	t := 0
	for _, v := range s {
		t += int(v)
	}
	return t
}

func readStatBlock(buf []byte, off int) StatBlock {
	var s StatBlock
	for i := range s {
		s[i] = int16(binary.LittleEndian.Uint16(buf[off+i*2:]))
	}
	return s
}

func writeStatBlock(buf []byte, off int, s StatBlock) {
	for i, v := range s {
		binary.LittleEndian.PutUint16(buf[off+i*2:], uint16(v))
	}
}

type Options struct {
	// PercentRange: max range from a stat block's original total, percent
	// each direction.
	PercentRange int

	// AllowNegativeWeaponStats lets individual weapon stats roll negative.
	// Armor never does, formula for defense seems to be dividing by (defense - damage), so negative defense is risky (needs testing/more decomp)
	AllowNegativeWeaponStats bool

	// AllowWorseWeaponLevels lets level 2/3 roll lower than the level before.
	// Otherwise each level is floored at the previous one.
	AllowWorseWeaponLevels bool
}

type Result struct {
	WeaponRecordsRandomized int
	ArmorRecordsRandomized  int
}

func (r Result) Summary() string {
	return fmt.Sprintf("Paramdomizer: randomized %d weapon record(s) and %d armor record(s).\n",
		r.WeaponRecordsRandomized, r.ArmorRecordsRandomized)
}

// Run rerolls weapon levels and armor defense. Record 0 is left alone.
func Run(weaponBuf, armorBuf []byte, rng *rand.Rand, opts Options) (Result, error) {
	wantWeaponLen := kfconst.ItemWeaponRecordSize * kfconst.ItemWeaponRecordCount
	if len(weaponBuf) != wantWeaponLen {
		return Result{}, fmt.Errorf("item_wpn.prm: expected %d bytes, got %d", wantWeaponLen, len(weaponBuf))
	}
	wantArmorLen := kfconst.ItemArmorRecordSize * kfconst.ItemArmorRecordCount
	if len(armorBuf) != wantArmorLen {
		return Result{}, fmt.Errorf("item_pct.prm: expected %d bytes, got %d", wantArmorLen, len(armorBuf))
	}

	levelOffsets := [3]int{kfconst.ItemWeaponLevel1Offset, kfconst.ItemWeaponLevel2Offset, kfconst.ItemWeaponLevel3Offset}
	weaponsDone := 0
	for i := 1; i < kfconst.ItemWeaponRecordCount; i++ {
		recOff := i * kfconst.ItemWeaponRecordSize
		prevLevelTotal := 0
		for li, lo := range levelOffsets {
			off := recOff + lo
			original := readStatBlock(weaponBuf, off)

			var floor *int
			if li > 0 && !opts.AllowWorseWeaponLevels {
				floor = &prevLevelTotal
			}
			newBlock := randomizeBlock(rng, original, opts.PercentRange, opts.AllowNegativeWeaponStats, floor)
			writeStatBlock(weaponBuf, off, newBlock)
			prevLevelTotal = newBlock.Total()
		}
		weaponsDone++
	}

	armorDone := 0
	for i := 1; i < kfconst.ItemArmorRecordCount; i++ {
		off := i*kfconst.ItemArmorRecordSize + kfconst.ItemArmorDefenseOffset
		original := readStatBlock(armorBuf, off)
		newBlock := randomizeBlock(rng, original, opts.PercentRange, false, nil)
		writeStatBlock(armorBuf, off, newBlock)
		armorDone++
	}

	return Result{WeaponRecordsRandomized: weaponsDone, ArmorRecordsRandomized: armorDone}, nil
}

// randomizeBlock rolls a new total within +/-percent of original's total,
// then redistributes it across a random subset of the 9 stats.
func randomizeBlock(rng *rand.Rand, original StatBlock, percent int, allowNegative bool, floor *int) StatBlock {
	base := original.Total()
	swing := (rng.Float64()*2 - 1) * float64(percent) / 100
	newTotal := int(math.Round(float64(base) * (1 + swing)))
	if floor != nil && newTotal < *floor {
		newTotal = *floor
	}

	n := 1 + rng.IntN(9) // how many of the 9 stats get touched
	touched := rng.Perm(9)[:n]
	touchedSet := make(map[int]bool, n)
	for _, i := range touched {
		touchedSet[i] = true
	}

	untouchedSum := 0
	for i, v := range original {
		if !touchedSet[i] {
			untouchedSum += int(v)
		}
	}
	touchedTarget := newTotal - untouchedSum
	if !allowNegative && touchedTarget < 0 {
		touchedTarget = 0
	}

	parts := distribute(rng, touchedTarget, n, allowNegative)

	result := original
	for pi, statIdx := range touched {
		result[statIdx] = clampInt16(parts[pi])
	}
	return result
}

// distribute splits total across n parts summing exactly to total, in a
// random order.
func distribute(rng *rand.Rand, total, n int, allowNegative bool) []int {
	if n == 1 {
		return []int{total}
	}

	parts := make([]int, n)
	if !allowNegative {
		if total < 0 {
			total = 0
		}
		// Stick-breaking: n-1 sorted cut points in [0, total], gaps sum to total.
		cuts := make([]int, n-1)
		for i := range cuts {
			cuts[i] = rng.IntN(total + 1)
		}
		sort.Ints(cuts)
		prev := 0
		for i, c := range cuts {
			parts[i] = c - prev
			prev = c
		}
		parts[n-1] = total - prev
	} else {
		// n-1 free values within a spread of the target, last value is
		// the exact remainder.
		spread := total
		if spread < 0 {
			spread = -spread
		}
		if spread == 0 {
			spread = 20
		}
		sum := 0
		for i := 0; i < n-1; i++ {
			v := rng.IntN(2*spread+1) - spread
			parts[i] = v
			sum += v
		}
		parts[n-1] = total - sum
	}

	rng.Shuffle(n, func(i, j int) { parts[i], parts[j] = parts[j], parts[i] })
	return parts
}

func clampInt16(v int) int16 {
	if v > math.MaxInt16 {
		return math.MaxInt16
	}
	if v < math.MinInt16 {
		return math.MinInt16
	}
	return int16(v)
}
