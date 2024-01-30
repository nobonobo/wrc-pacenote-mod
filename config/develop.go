//go:build !release

package config

import (
	"log"
	"os"
)

func getRootDir() string {
	here, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	return here
}
