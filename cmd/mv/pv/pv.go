package pv

import (
	"go.zoe.im/x/cli"

	mvcmd "go.zoe.im/kops/cmd/mv"

	"go.zoe.im/kops/pkg/cmd/mv/pv"

)

func init() {

	m := pv.NewManager()

	mvcmd.Register(
		cli.New(
			cli.Name("pv [pod]", "pv"),
			cli.Short("Move local persistent volume of a pod."),
			cli.Description(`Move persistent volume of a pod to another node.

	* if pod is't provided, we will take all pods on current node to proccess;
	* if distination node is't provided, we will use current node;

Warning! If have any errors occur, we can't roll back. 
`),
			cli.Config(m.Config),
			cli.Run(func(c *cli.Command, args ...string) {
				m.Start(args...)
			}),
		),
	)

}