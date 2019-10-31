// Package io contains common io util.
package io

import (
	"os"
	"path"
	"path/filepath"
)

var stat = os.Stat
var mkdirAll = os.MkdirAll

// IsFileExist checks whether specified file is existed
func IsFileExist(path string) (exists bool, err error) {
	_, err = stat(path)
	exists = err == nil || os.IsExist(err)
	return
}

// EnsureDirectory make sure that the specified dir is existed or created.
func EnsureDirectory(targetDir string) (string, bool) {
	var (
		targetPath string
		err        error
	)

	if path.IsAbs(targetDir) {
		targetPath = targetDir
	} else {
		targetPath, err = filepath.Abs(targetDir)
		if err != nil {
			return targetDir, false
		}
	}

	if _, err = stat(targetPath); os.IsNotExist(err) {
		err = mkdirAll(targetPath, 0755)
		if err != nil {
			return targetDir, false
		}
	}

	return targetPath, true
}
