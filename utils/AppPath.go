// utils содержит всякие полезнае полезности
package utils

import (
	"os"
	"path/filepath"
)

// AppPath предоставляет путь к директории приложения.
func AppPath() string {
	ex, err := os.Executable()
	if err != nil {
		panic(err)
	}
	return filepath.Dir(ex)
}
