// Package fsutil provides small filesystem helpers shared across ghsw's
// config-file packages.
package fsutil

import "os"

// ReadOrEmpty reads path, returning "" instead of an error if it does not exist.
func ReadOrEmpty(path string) (string, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return string(data), nil
}
