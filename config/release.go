//go:build !develop

package config

import (
	"log"
	"os"
	"path/filepath"
)

func getRootDir() string {
	self, err := os.Executable()
	if err != nil {
		log.Fatal(err)
	}
	return filepath.Dir(self)
}
