//go:build !windows

package core

import "os"

func isHiddenPlatform(d os.DirEntry) (bool, error) {
	_ = d
	return false, nil
}
