package core

import (
	"os"
	"path/filepath"
	"strings"
)

func isHiddenPath(path string, d os.DirEntry, root string) (bool, error) {
	if filepath.Clean(path) == filepath.Clean(root) {
		return false, nil
	}
	if strings.HasPrefix(d.Name(), ".") {
		return true, nil
	}
	return isHiddenPlatform(d)
}
