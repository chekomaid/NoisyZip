//go:build windows

package main

import (
	"os"
	"path/filepath"
	"strings"
	"syscall"
)

func isHiddenPath(path string, d os.DirEntry, root string) (bool, error) {
	if filepath.Clean(path) == filepath.Clean(root) {
		return false, nil
	}
	if strings.HasPrefix(d.Name(), ".") {
		return true, nil
	}
	info, err := d.Info()
	if err != nil {
		return false, err
	}
	data, ok := info.Sys().(*syscall.Win32FileAttributeData)
	if !ok {
		return false, nil
	}
	if data.FileAttributes&syscall.FILE_ATTRIBUTE_HIDDEN != 0 {
		return true, nil
	}
	return false, nil
}
