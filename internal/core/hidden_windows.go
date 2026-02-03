//go:build windows

package core

import (
	"os"
	"syscall"
)

func isHiddenPlatform(d os.DirEntry) (bool, error) {
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
