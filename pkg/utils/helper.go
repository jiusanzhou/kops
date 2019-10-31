package utils

import (
	"os"
)

func Hostname() (string, error) {
	return os.Hostname()
}