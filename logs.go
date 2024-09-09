package main

import (
	"fmt"
	"github.com/urfave/cli"
	"runQ/container"
)

var logCommand = cli.Command{
	Name:  "logs",
	Usage: "print logs of a container",
	Action: func(ctx *cli.Context) error {
		if len(ctx.Args()) < 1 {
			return fmt.Errorf("please input your container name")
		}
		containerName := ctx.Args().Get(0)
		container.LogContainer(containerName)
		return nil
	},
}
