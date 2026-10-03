package isopatch

import (
	"encoding/binary"
	"fmt"
	"os"

	"kf4randomizer/internal/kfconst"
	"kf4randomizer/internal/resources"
)

// PatchWhitelistBypass makes all items displayable, drawn with the map's first indexed model.
// Basically, items are displayed by checking a 64 slot array for their ID. if ID isn't found, the model doesn't appear so it's not collectable.
// since rando makes all items appear anywhere, it's needed.
// Tried making the correct models load but game kept crashing, not sure why exactly, and it took days to test without success.
func PatchWhitelistBypass(f *os.File, slusAbsOffset int64) error {
	fileOff := kfconst.SlusVAddrToFileOffset(kfconst.SlusWhitelistCheckPatchVAddr)
	absOff := slusAbsOffset + int64(fileOff)

	// The jump plus a nop in its delay slot.
	patch := make([]byte, 8)
	binary.LittleEndian.PutUint32(patch[0:4], kfconst.SlusWhitelistCheckJumpInstruction())

	if _, err := f.WriteAt(patch, absOff); err != nil {
		return fmt.Errorf("writing whitelist-bypass patch: %w", err)
	}
	return nil
}

// PatchPickupPopup makes item pickups show the item's real name (and,
// with showItemIndex, its slot index, like "Antidote (#4)").
func PatchPickupPopup(f *os.File, slusAbsOffset int64, showItemIndex bool) error {
	code := resources.PickupPopupCode
	if showItemIndex {
		code = resources.PickupPopupCodeWithIndex
	}

	codeFileOff := kfconst.SlusVAddrToFileOffset(kfconst.SlusPickupPopupCodeVAddr)
	codeAbsOff := slusAbsOffset + int64(codeFileOff)
	if _, err := f.WriteAt(code, codeAbsOff); err != nil {
		return fmt.Errorf("writing pickup-popup code: %w", err)
	}

	hookFileOff := kfconst.SlusVAddrToFileOffset(kfconst.SlusPickupPopupHookVAddr)
	hookAbsOff := slusAbsOffset + int64(hookFileOff)
	instr := make([]byte, 4)
	binary.LittleEndian.PutUint32(instr, kfconst.SlusPickupPopupHookJumpInstruction())
	if _, err := f.WriteAt(instr, hookAbsOff); err != nil {
		return fmt.Errorf("writing pickup-popup hook: %w", err)
	}

	return nil
}

// PatchDebugHUD adds the L3+R3 live HUD: bootstrap into SLUS_203.18,
// per-frame tick hook, and payload into KF4.DAT's padding section.
func PatchDebugHUD(f *os.File, slusAbsOffset int64) error {
	if len(resources.DebugHUDBootstrap) != kfconst.SlusDebugHUDBootstrapSize {
		return fmt.Errorf("debug-HUD bootstrap is %d bytes, kfconst.SlusDebugHUDBootstrapSize says %d",
			len(resources.DebugHUDBootstrap), kfconst.SlusDebugHUDBootstrapSize)
	}

	bootstrapFileOff := kfconst.SlusVAddrToFileOffset(kfconst.SlusDebugHUDBootstrapVAddr)
	bootstrapAbsOff := slusAbsOffset + int64(bootstrapFileOff)
	if _, err := f.WriteAt(resources.DebugHUDBootstrap, bootstrapAbsOff); err != nil {
		return fmt.Errorf("writing debug-HUD bootstrap: %w", err)
	}

	zeroScratch := make([]byte, 16)
	scratchFileOff := kfconst.SlusVAddrToFileOffset(kfconst.SlusDebugHUDDoneVAddr)
	scratchAbsOff := slusAbsOffset + int64(scratchFileOff)
	if _, err := f.WriteAt(zeroScratch, scratchAbsOff); err != nil {
		return fmt.Errorf("zeroing debug-HUD scratch words: %w", err)
	}

	datDirEntry, err := FindPath(f, kfconst.IsoDatPath)
	if err != nil {
		return fmt.Errorf("locating KF4.DAT for debug-HUD payload: %w", err)
	}
	datTable, err := ReadDatFileTable(f, datDirEntry.ByteOffset())
	if err != nil {
		return fmt.Errorf("reading KF4.DAT file table for debug-HUD payload: %w", err)
	}
	hostEntry, ok := datTable[kfconst.DebugHUDPayloadHostDatMember]
	if !ok {
		return fmt.Errorf("debug-HUD payload host DAT member %q not found in KF4.DAT", kfconst.DebugHUDPayloadHostDatMember)
	}
	if len(resources.DebugHUDPayload) > kfconst.SlusDebugHUDPayloadMaxSize {
		return fmt.Errorf("debug-HUD payload is %d bytes, the bootstrap only reads %d",
			len(resources.DebugHUDPayload), kfconst.SlusDebugHUDPayloadMaxSize)
	}
	if err := WriteDatMemberSlack(f, hostEntry, kfconst.DebugHUDPayloadSlackByteOffset, resources.DebugHUDPayload); err != nil {
		return fmt.Errorf("writing debug-HUD payload into DAT slack: %w", err)
	}

	hookFileOff := kfconst.SlusVAddrToFileOffset(kfconst.SlusDebugHUDHookVAddr)
	hookAbsOff := slusAbsOffset + int64(hookFileOff)
	instr := make([]byte, 4)
	binary.LittleEndian.PutUint32(instr, kfconst.SlusDebugHUDHookJumpInstruction())
	if _, err := f.WriteAt(instr, hookAbsOff); err != nil {
		return fmt.Errorf("writing debug-HUD hook: %w", err)
	}

	return patchLevitate(f, slusAbsOffset)
}

// patchLevitate writes the stub after the bootstrap and hooks it into the
// floor check.
func patchLevitate(f *os.File, slusAbsOffset int64) error {
	end := kfconst.SlusLevitateStubVAddr + len(resources.LevitateStub)
	if end > kfconst.SlusPickupPopupCaveEndVAddr {
		return fmt.Errorf("levitate stub ends at 0x%X, past the cave at 0x%X",
			end, kfconst.SlusPickupPopupCaveEndVAddr)
	}

	stubOff := kfconst.SlusVAddrToFileOffset(kfconst.SlusLevitateStubVAddr)
	if _, err := f.WriteAt(resources.LevitateStub, slusAbsOffset+int64(stubOff)); err != nil {
		return fmt.Errorf("writing levitate stub: %w", err)
	}

	instr := make([]byte, 4)
	binary.LittleEndian.PutUint32(instr, kfconst.SlusLevitateHookJumpInstruction())
	hookOff := kfconst.SlusVAddrToFileOffset(kfconst.SlusLevitateHookVAddr)
	if _, err := f.WriteAt(instr, slusAbsOffset+int64(hookOff)); err != nil {
		return fmt.Errorf("writing levitate hook: %w", err)
	}
	return nil
}

// PatchBlackVoidFix repairs the retail floor-resolve comparison that leaves
// some doorways as black voids. See kfconst.SlusBlackVoidFixVAddr for why every
// build needs it, not just modified ones.
func PatchBlackVoidFix(f *os.File, slusAbsOffset int64) error {
	if err := WriteSlusWord(f, slusAbsOffset,
		kfconst.SlusBlackVoidFixVAddr, kfconst.SlusBlackVoidFixInstruction); err != nil {
		return fmt.Errorf("writing black-void fix: %w", err)
	}
	return nil
}

// WriteSlusWord writes a little-endian 32-bit word into SLUS_203.18 at
// vaddr, for single-word patches like the mods.
func WriteSlusWord(f *os.File, slusAbsOffset int64, vaddr uint32, value uint32) error {
	fileOff := kfconst.SlusVAddrToFileOffset(int(vaddr))
	absOff := slusAbsOffset + int64(fileOff)

	buf := make([]byte, 4)
	binary.LittleEndian.PutUint32(buf, value)
	if _, err := f.WriteAt(buf, absOff); err != nil {
		return fmt.Errorf("writing word at vaddr 0x%08X: %w", vaddr, err)
	}
	return nil
}

// ReadSlusBytes reads n bytes of SLUS_203.18 starting at vaddr.
func ReadSlusBytes(f *os.File, slusAbsOffset int64, vaddr, n int) ([]byte, error) {
	buf := make([]byte, n)
	off := slusAbsOffset + int64(kfconst.SlusVAddrToFileOffset(vaddr))
	if _, err := f.ReadAt(buf, off); err != nil {
		return nil, fmt.Errorf("reading %d bytes at vaddr 0x%08X: %w", n, vaddr, err)
	}
	return buf, nil
}

// WriteSlusBytes writes data into SLUS_203.18 starting at vaddr.
func WriteSlusBytes(f *os.File, slusAbsOffset int64, vaddr int, data []byte) error {
	off := slusAbsOffset + int64(kfconst.SlusVAddrToFileOffset(vaddr))
	if _, err := f.WriteAt(data, off); err != nil {
		return fmt.Errorf("writing %d bytes at vaddr 0x%08X: %w", len(data), vaddr, err)
	}
	return nil
}
