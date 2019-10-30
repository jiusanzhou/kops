package mv

import (
	"go.zoe.im/x/cli"

	"go.zoe.im/kops/cmd"

	"go.zoe.im/kops/pkg/cmd/mv"
)

var (
	mvcmd *cli.Command
)

// Register create a sub command
func Register(cmds ...*cli.Command) error {
	return mvcmd.Register(cmds...)
}

// Run execute command
func Run(opts ...cli.Option) error {
	return mvcmd.Run(opts...)
}

func init() {

	m := mv.NewManager()

	mvcmd = cli.New(
		cli.Name("mv [command]", "move"),
		cli.Short("Move resource from source to distination."),
		cli.Description(`Move resource from source to distination.

	* Just try it.

`),
		cli.Config(m.Config),
		cli.Run(func(c *cli.Command, args ...string) {
			c.Help()
		}),
	)


	cmd.Register(mvcmd)
}