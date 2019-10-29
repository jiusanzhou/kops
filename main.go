package main

import (
	"fmt"
	"os"

	"go.zoe.im/x/cli"

	"go.zoe.im/kops/cmd"

	// import all sub commands
	_ "go.zoe.im/kops/cmd/mv"
	_ "go.zoe.im/kops/cmd/mv/pv"
)

func main() {
	if err := cmd.Run(
		cli.Version(ReleaseTag),
	); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
