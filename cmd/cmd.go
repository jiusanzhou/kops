package cmd

import (
	"go.zoe.im/x/cli"
)

var (
	// global cmd contains all sub command
	cmd = cli.New(
		cli.Name("kops"),
		cli.Short("Kops is a toolbox for kubernetes ops."),
		cli.Run(func(c *cli.Command, args ...string) {
			c.Help()
		}),
	)
)

// Register create a sub command
func Register(cmds ...*cli.Command) error {
	return cmd.Register(cmds...)
}

// Run execute command
func Run(opts ...cli.Option) error {
	return cmd.Run(opts...)
}
