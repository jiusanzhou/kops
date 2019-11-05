package main

import (
	"fmt"
	"os"

	"go.zoe.im/x/cli"

	"go.zoe.im/kops/cmd"

	// import allcommands
	_ "go.zoe.im/kops/cmd/mv"
	_ "go.zoe.im/kops/cmd/mv/pv"
)

func init() {
	cmd.Register(
		cli.New(
			cli.Name("version"),
			cli.Short("Show version information."),
			cli.Run(func(c *cli.Command, args ...string) {
				fmt.Println("Version:", cmd.Version)
				fmt.Println("Release-Tag:", cmd.ReleaseTag)
				fmt.Println("Commit-ID:", cmd.CommitID)
				fmt.Println()
				fmt.Println("GOROOT:", cmd.GOROOT)
				fmt.Println("GOPATH:", cmd.GOPATH)
			}),
		),
	)
}

func main() {
	if err := cmd.Run(
		cli.Version(cmd.ReleaseTag),
	); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
