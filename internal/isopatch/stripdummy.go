package isopatch

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"sort"

	"kf4randomizer/internal/kfconst"
)

// rawRecord is a directory record with what's needed to remove or rewrite it.
type rawRecord struct {
	Offset int // byte offset of this record within its sector buffer
	Length int // on-disk length of this record (its own length byte)
	LBA    uint32
	Size   uint32
	Name   string
}

// parseRawRecords parses every record in a directory's sector buffer,
// skipping zero-length padding at sector boundaries.
func parseRawRecords(data []byte) ([]rawRecord, error) {
	var out []rawRecord
	off := 0
	for off < len(data) {
		e, length, err := parseDirRecord(data, off)
		if err != nil {
			return nil, err
		}
		if e == nil {
			off = ((off / kfconst.IsoSectorSize) + 1) * kfconst.IsoSectorSize
			continue
		}
		out = append(out, rawRecord{Offset: off, Length: length, LBA: e.LBA, Size: e.Size, Name: e.Name})
		off += length
	}
	return out, nil
}

// patchRecordField overwrites a directory record's LE+BE field pair (every
// ISO9660 numeric field is stored both ways).
func patchRecordField(buf []byte, recOffset, fieldOffset int, value uint32) {
	binary.LittleEndian.PutUint32(buf[recOffset+fieldOffset:], value)
	binary.BigEndian.PutUint32(buf[recOffset+fieldOffset+4:], value)
}

func patchRecordLBA(buf []byte, recOffset int, newLBA uint32) {
	patchRecordField(buf, recOffset, 2, newLBA)
}

func patchRecordSize(buf []byte, recOffset int, newSize uint32) {
	patchRecordField(buf, recOffset, 10, newSize)
}

func sectorsFor(size uint32) uint32 {
	return (size + kfconst.IsoSectorSize - 1) / kfconst.IsoSectorSize
}

// isDotEntry reports whether a record is the "." or ".." entry, skipped by
// every LBA-shifting pass below.
func isDotEntry(name string) bool {
	return name == "." || name == ".."
}

// copyRange copies length bytes from srcOffset in src to dst's current
// write position.
func copyRange(src *os.File, dst *os.File, srcOffset int64, length int64) error {
	if _, err := src.Seek(srcOffset, io.SeekStart); err != nil {
		return err
	}
	_, err := io.CopyN(dst, src, length)
	return err
}

// StripDummyFiles copies srcPath to dstPath without the DUMMY*.OUT padding
// files, after checking they sit where expected.
func StripDummyFiles(srcPath, dstPath string) error {
	if same, err := SameFile(srcPath, dstPath); err != nil {
		return err
	} else if same {
		return fmt.Errorf("input and output are the same file (%s)", dstPath)
	}

	src, err := os.Open(srcPath)
	if err != nil {
		return fmt.Errorf("opening %s: %w", srcPath, err)
	}
	defer src.Close()

	totalSize, err := src.Seek(0, io.SeekEnd)
	if err != nil {
		return err
	}
	if totalSize%kfconst.IsoSectorSize != 0 {
		return fmt.Errorf("%s: size %d isn't a whole number of sectors", srcPath, totalSize)
	}
	totalSectors := uint32(totalSize / kfconst.IsoSectorSize)

	pvd, err := readSectors(src, 16, 1)
	if err != nil {
		return fmt.Errorf("reading PVD: %w", err)
	}
	if string(pvd[1:6]) != "CD001" {
		return fmt.Errorf("sector 16 is not a valid Primary Volume Descriptor")
	}
	const rootRecOffset = 156
	volSpaceSize := binary.LittleEndian.Uint32(pvd[80:])
	if volSpaceSize != totalSectors {
		return fmt.Errorf("PVD volume space size (%d) doesn't match the disc's actual size (%d sectors)", volSpaceSize, totalSectors)
	}
	rootLBA := binary.LittleEndian.Uint32(pvd[rootRecOffset+2:])
	rootSize := binary.LittleEndian.Uint32(pvd[rootRecOffset+10:])

	rootBuf, err := readSectors(src, rootLBA, sectorsFor(rootSize))
	if err != nil {
		return fmt.Errorf("reading root directory: %w", err)
	}
	rootRecords, err := parseRawRecords(rootBuf)
	if err != nil {
		return fmt.Errorf("parsing root directory: %w", err)
	}

	byName := make(map[string]rawRecord, len(rootRecords))
	for _, r := range rootRecords {
		byName[r.Name] = r
	}

	var regionA []rawRecord
	for _, name := range kfconst.DummyRegionAFiles {
		r, ok := byName[name]
		if !ok {
			return fmt.Errorf("expected dummy file %q not found at disc root", name)
		}
		regionA = append(regionA, r)
	}
	regionB, ok := byName[kfconst.DummyRegionBFile]
	if !ok {
		return fmt.Errorf("expected dummy file %q not found at disc root", kfconst.DummyRegionBFile)
	}

	sort.Slice(regionA, func(i, j int) bool { return regionA[i].LBA < regionA[j].LBA })
	for i := 1; i < len(regionA); i++ {
		prevEnd := regionA[i-1].LBA + sectorsFor(regionA[i-1].Size)
		if regionA[i].LBA != prevEnd {
			return fmt.Errorf("dummy files aren't contiguous as expected (%s ends at LBA %d, but %s starts at LBA %d)",
				regionA[i-1].Name, prevEnd-1, regionA[i].Name, regionA[i].LBA)
		}
	}
	regionAStart := regionA[0].LBA
	var regionASectors uint32
	for _, r := range regionA {
		regionASectors += sectorsFor(r.Size)
	}
	regionBStart := regionB.LBA

	dataEntry, ok := byName["DATA"]
	if !ok {
		return fmt.Errorf("DATA directory not found at disc root")
	}
	iopEntry, ok := byName["IOP"]
	if !ok {
		return fmt.Errorf("IOP directory not found at disc root")
	}

	dataBuf, err := readSectors(src, dataEntry.LBA, sectorsFor(dataEntry.Size))
	if err != nil {
		return fmt.Errorf("reading DATA directory: %w", err)
	}
	dataRecords, err := parseRawRecords(dataBuf)
	if err != nil {
		return fmt.Errorf("parsing DATA directory: %w", err)
	}

	iopBuf, err := readSectors(src, iopEntry.LBA, sectorsFor(iopEntry.Size))
	if err != nil {
		return fmt.Errorf("reading IOP directory: %w", err)
	}
	iopRecords, err := parseRawRecords(iopBuf)
	if err != nil {
		return fmt.Errorf("parsing IOP directory: %w", err)
	}

	// DATA's last file must be immediately followed by region B.
	var lastDataEnd uint32
	var lastDataName string
	var firstAfterRegionA uint32 = ^uint32(0)
	for _, r := range dataRecords {
		if isDotEntry(r.Name) {
			continue
		}
		end := r.LBA + sectorsFor(r.Size)
		if end > lastDataEnd {
			lastDataEnd, lastDataName = end, r.Name
		}
		if r.LBA >= regionAStart && r.LBA < firstAfterRegionA {
			firstAfterRegionA = r.LBA
		}
	}
	if lastDataEnd != regionBStart {
		return fmt.Errorf("expected %s (DATA's last file, ending at LBA %d) to be immediately followed by %s at LBA %d, but %s starts at LBA %d instead",
			lastDataName, lastDataEnd-1, kfconst.DummyRegionBFile, lastDataEnd, kfconst.DummyRegionBFile, regionBStart)
	}
	if firstAfterRegionA != regionAStart+regionASectors {
		return fmt.Errorf("expected a DATA file to start immediately after region A (LBA %d), but the next one starts at LBA %d",
			regionAStart+regionASectors, firstAfterRegionA)
	}

	// Shift every DATA/IOP entry at or after region A back by regionASectors.
	for _, r := range dataRecords {
		if isDotEntry(r.Name) {
			continue
		}
		if r.LBA >= regionAStart {
			patchRecordLBA(dataBuf, r.Offset, r.LBA-regionASectors)
		}
	}
	for _, r := range iopRecords {
		if isDotEntry(r.Name) {
			continue
		}
		if r.LBA >= regionAStart {
			// Not expected, IOP sits before region A, but handle it anyway.
			patchRecordLBA(iopBuf, r.Offset, r.LBA-regionASectors)
		}
	}

	// Drop the 5 dummy entries from root and repack what is left.
	dummyNames := make(map[string]bool, len(kfconst.DummyRegionAFiles)+1)
	for _, n := range kfconst.DummyRegionAFiles {
		dummyNames[n] = true
	}
	dummyNames[kfconst.DummyRegionBFile] = true

	newRoot := make([]byte, len(rootBuf))
	writeOff := 0
	for _, r := range rootRecords {
		if dummyNames[r.Name] {
			continue
		}
		copy(newRoot[writeOff:], rootBuf[r.Offset:r.Offset+r.Length])
		writeOff += r.Length
	}
	newRootSize := uint32(writeOff)

	newRootRecords, err := parseRawRecords(newRoot[:writeOff])
	if err != nil {
		return fmt.Errorf("parsing repacked root directory: %w", err)
	}
	for _, r := range newRootRecords {
		if isDotEntry(r.Name) {
			patchRecordSize(newRoot, r.Offset, newRootSize)
		}
	}

	// IOP's and DATA's own '..' entries describe root, whose size just changed.
	for _, r := range dataRecords {
		if r.Name == ".." {
			patchRecordSize(dataBuf, r.Offset, newRootSize)
		}
	}
	for _, r := range iopRecords {
		if r.Name == ".." {
			patchRecordSize(iopBuf, r.Offset, newRootSize)
		}
	}

	// The PVD itself.
	newTotalSectors := regionBStart - regionASectors
	binary.LittleEndian.PutUint32(pvd[80:], newTotalSectors)
	binary.BigEndian.PutUint32(pvd[84:], newTotalSectors)
	patchRecordSize(pvd, rootRecOffset, newRootSize)

	// Assemble the output.
	dst, err := os.Create(dstPath)
	if err != nil {
		return fmt.Errorf("creating %s: %w", dstPath, err)
	}
	defer dst.Close()

	if err := copyRange(src, dst, 0, int64(regionAStart)*kfconst.IsoSectorSize); err != nil {
		return fmt.Errorf("copying initial chunk: %w", err)
	}

	if _, err := dst.WriteAt(pvd, 16*kfconst.IsoSectorSize); err != nil {
		return fmt.Errorf("writing patched PVD: %w", err)
	}
	if _, err := dst.WriteAt(newRoot, int64(rootLBA)*kfconst.IsoSectorSize); err != nil {
		return fmt.Errorf("writing patched root directory: %w", err)
	}
	if _, err := dst.WriteAt(dataBuf, int64(dataEntry.LBA)*kfconst.IsoSectorSize); err != nil {
		return fmt.Errorf("writing patched DATA directory: %w", err)
	}
	if _, err := dst.WriteAt(iopBuf, int64(iopEntry.LBA)*kfconst.IsoSectorSize); err != nil {
		return fmt.Errorf("writing patched IOP directory: %w", err)
	}

	middleStartLBA := regionAStart + regionASectors
	middleStartByte := int64(middleStartLBA) * kfconst.IsoSectorSize
	middleLen := int64(regionBStart-middleStartLBA) * kfconst.IsoSectorSize
	if err := copyRange(src, dst, middleStartByte, middleLen); err != nil {
		return fmt.Errorf("copying middle chunk: %w", err)
	}

	info, err := dst.Stat()
	if err != nil {
		return err
	}
	if uint32(info.Size()/kfconst.IsoSectorSize) != newTotalSectors || info.Size()%kfconst.IsoSectorSize != 0 {
		return fmt.Errorf("internal error: wrote %d bytes, expected exactly %d sectors (%d bytes)", info.Size(), newTotalSectors, int64(newTotalSectors)*kfconst.IsoSectorSize)
	}

	return dst.Close()
}
