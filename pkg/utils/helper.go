package utils

import (
	"bufio"
	"path/filepath"
	"fmt"
	"os"
)

func Hostname() (string, error) {
	return os.Hostname()
}

func Ask(str string) string {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print(str)
	text, _ := reader.ReadString('\n')
	return text
}

func ParentPath(path string) string {
	return filepath.Dir(path)
}

func Exits(path string) bool {
	 _, err := os.Stat(path)
	 return !os.IsNotExist(err)
}