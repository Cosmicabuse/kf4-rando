//go:build windows

package main

import (
	"fmt"
	"strings"
	"syscall"
	"unicode/utf16"
	"unsafe"
)

// Alternating description/pattern pairs, the order the dialog wants them.
var isoFilterParts = []string{"ISO files (*.iso)", "*.iso", "All files (*.*)", "*.*"}

// lpstrFilter wants the strings run together and double-NUL terminated.
func utf16MultiString(parts ...string) *uint16 {
	var buf []uint16
	for _, p := range parts {
		buf = append(buf, utf16.Encode([]rune(p))...)
		buf = append(buf, 0)
	}
	buf = append(buf, 0)
	return &buf[0]
}

// browseForFile shows the native "open file" dialog, "" if cancelled.
func browseForFile(title string) (string, error) {
	return openFileDialog(title, false)
}

// browseForSavePath shows the native "save file" dialog, "" if cancelled.
func browseForSavePath(title string) (string, error) {
	return openFileDialog(title, true)
}

// Win32's OPENFILENAMEW. comdlg32 reads it raw, so field order and types
// have to match.
type openFileNameW struct {
	lStructSize       uint32
	hwndOwner         uintptr
	hInstance         uintptr
	lpstrFilter       *uint16
	lpstrCustomFilter *uint16
	nMaxCustFilter    uint32
	nFilterIndex      uint32
	lpstrFile         *uint16
	nMaxFile          uint32
	lpstrFileTitle    *uint16
	nMaxFileTitle     uint32
	lpstrInitialDir   *uint16
	lpstrTitle        *uint16
	flags             uint32
	nFileOffset       uint16
	nFileExtension    uint16
	lpstrDefExt       *uint16
	lCustData         uintptr
	lpfnHook          uintptr
	lpTemplateName    *uint16
	pvReserved        uintptr
	dwReserved        uint32
	flagsEx           uint32
}

const (
	ofnPathMustExist   = 0x00000800
	ofnFileMustExist   = 0x00001000
	ofnOverwritePrompt = 0x00000002
	ofnNoChangeDir     = 0x00000008
	maxPathBuf         = 32768 // long paths, not the legacy 260
)

func openFileDialog(title string, save bool) (string, error) {
	comdlg32 := syscall.NewLazyDLL("comdlg32.dll")
	proc := comdlg32.NewProc("GetOpenFileNameW")
	flags := uint32(ofnPathMustExist | ofnFileMustExist | ofnNoChangeDir)
	if save {
		proc = comdlg32.NewProc("GetSaveFileNameW")
		flags = ofnNoChangeDir | ofnOverwritePrompt
	}

	filter := utf16MultiString(isoFilterParts...)
	titlePtr, err := syscall.UTF16PtrFromString(title)
	if err != nil {
		return "", fmt.Errorf("encoding title: %w", err)
	}

	fileBuf := make([]uint16, maxPathBuf)

	ofn := openFileNameW{
		lpstrFilter: filter,
		lpstrFile:   &fileBuf[0],
		nMaxFile:    uint32(len(fileBuf)),
		lpstrTitle:  titlePtr,
		flags:       flags,
	}
	ofn.lStructSize = uint32(unsafe.Sizeof(ofn))

	ret, _, _ := proc.Call(uintptr(unsafe.Pointer(&ofn)))
	if ret == 0 {
		return "", nil // cancelled or failed, both mean no path
	}

	return strings.TrimRight(syscall.UTF16ToString(fileBuf), "\x00"), nil
}
