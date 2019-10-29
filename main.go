package main

import (
	"fmt"
	"os"

	"go.zoe.im/x/cli"

	"go.zoe.im/kops/cmd"
)

func main() {
	if err := cmd.Run(
		cli.Version(ReleaseTag),
	); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
