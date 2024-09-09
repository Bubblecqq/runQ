package main

import (
	"github.com/urfave/cli"
	"runQ/container"
)

var listCommand = cli.Command{
	Name:  "ps",
	Usage: "list all the containers",
	Action: func(ctx *cli.Context) error {
		container.PrintListContainers()
		return nil
	},
}
