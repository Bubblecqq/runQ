package main

import (
	"github.com/urfave/cli"
	"runQ/container"
)

var stopCommand = cli.Command{
	Name:  "stop",
	Usage: "stop a container, e.g. runQ stop 1234567890",
	Flags: []cli.Flag{
		cli.StringFlag{
			Name:  "name",
			Usage: `Specify a container to stop`,
		},
	},
	Action: func(ctx *cli.Context) error {
		//containerName := ctx.Args().Get(0)
		containerName := ctx.String("name")
		container.StopContainer(containerName)
		return nil
	},
}
