//go:build !windows

package main

import (
	"fmt"
	"runtime"
)

// No native file dialog outside Windows. The path fields stay editable.
func browseForFile(title string, _ fileFilter) (string, error) {
	return "", fmt.Errorf("no file dialog on %s, type the path instead", runtime.GOOS)
}

func browseForSavePath(title string) (string, error) {
	return "", fmt.Errorf("no file dialog on %s, type the path instead", runtime.GOOS)
}
