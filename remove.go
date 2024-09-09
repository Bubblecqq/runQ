package main

import (
	"github.com/urfave/cli"
	"runQ/container"
)

var removeCommand = cli.Command{
	Name:  "rm",
	Usage: "remove unused containers,e.g. runQ rm 1234567890",
	Flags: []cli.Flag{
		cli.BoolFlag{
			Name:  "f",
			Usage: "force delete running container",
		},
		cli.StringFlag{
			Name:  "name",
			Usage: "Specify a container to remove",
		},
	},
	Action: func(ctx *cli.Context) error {
		force := ctx.Bool("f")
		containerName := ctx.String("name")
		container.RemoveContainer(containerName, force)
		return nil
	},
}
