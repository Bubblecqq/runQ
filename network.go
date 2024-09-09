package main

import (
	"fmt"
	"github.com/urfave/cli"
	"runQ/network"
)

var networkCommand = cli.Command{
	Name:  "network",
	Usage: "container network commands",
	Subcommands: []cli.Command{
		{
			Name:  "create",
			Usage: "create a container network",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "driver",
					Usage: "Network Driver",
				},
				cli.StringFlag{
					Name:  "subnet",
					Usage: "Network CIDR",
				},
			},
			Action: func(ctx *cli.Context) error {
				if len(ctx.Args()) < 1 {
					return fmt.Errorf("missing network name")
				}
				driver := ctx.String("driver")
				subnet := ctx.String("subnet")
				name := ctx.Args()[0]
				err := network.CreateNetwork(driver, subnet, name)
				if err != nil {
					return fmt.Errorf("create network error: %+v", err)
				}
				return nil
			},
		},
		{
			Name:  "list",
			Usage: "list container network",
			Action: func(ctx *cli.Context) error {
				network.ListNetwork()
				return nil
			},
		},
		{
			Name:  "remove",
			Usage: "remove container network",
			Action: func(ctx *cli.Context) error {
				if len(ctx.Args()) < 1 {
					return fmt.Errorf("missing network name")
				}
				err := network.DeleteNetwork(ctx.Args()[0])
				if err != nil {
					return fmt.Errorf("remove network error: %+v", err)
				}
				return nil
			},
		},
	},
}
