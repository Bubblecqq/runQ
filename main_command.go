package main

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli"
	"os"
	"runQ/cgroups/resource"
	"runQ/container"
)

var runCommand = cli.Command{
	Name: "run",
	Usage: `Create a container with namespace and cgroups limit
			runQ run -it [command]`,
	Flags: []cli.Flag{
		cli.BoolFlag{Name: "it", Usage: "enable tty"},
		cli.BoolFlag{Name: "d", Usage: "detach container"},
		cli.StringFlag{Name: "mem", Usage: "memory limit,e.g.: -mem 100m"},
		cli.StringFlag{Name: "cpu", Usage: "cpu quota,e.g.: -cpu 100"},
		cli.StringFlag{Name: "cpuset", Usage: "cpuset limit,e.g.: -cpuset 2,4"},
		cli.StringFlag{Name: "v", Usage: "volume,e.g.: -v /etc/conf:/etc/conf"},
		cli.StringFlag{Name: "name,n", Usage: "container name"},
		cli.StringFlag{Name: "image,i", Usage: "container image"},
		cli.StringSliceFlag{Name: "e", Usage: "set environment, e.g. -e -name=runQ"},
		cli.StringFlag{Name: "net", Usage: "container network, e.g. -net testbr"},
		cli.StringSliceFlag{
			Name:  "p",
			Usage: "port mapping,e.g. -p 8080:80 -p 30336:3306",
		},
	},
	Action: func(ctx *cli.Context) error {
		if len(ctx.Args()) < 1 {
			return fmt.Errorf("missing container command")
		}
		//cmd := ctx.Args().Get(0)
		var cmdArray []string
		for _, arg := range ctx.Args() {
			cmdArray = append(cmdArray, arg)
		}
		resConf := &resource.ResourceConfig{
			MemoryLimit: ctx.String("mem"),
			CpuSet:      ctx.String("cpuset"),
			CpuCfsQuota: ctx.Int("cpu"),
		}
		imageNmae := ctx.String("image")
		tty := ctx.Bool("it")
		detach := ctx.Bool("d")
		containerName := ctx.String("name")
		envSlice := ctx.StringSlice("e")
		volume := ctx.String("v")
		network := ctx.String("net")
		portMapping := ctx.StringSlice("p")

		if tty && detach {
			return fmt.Errorf("it and d paramter can not both provided")
		}
		if !detach {
			tty = true
		}
		log.Infof("createTTY %v", tty)
		Run(tty, cmdArray, envSlice, resConf, volume, containerName, imageNmae, network, portMapping)
		return nil
	},
}

var initCommand = cli.Command{
	Name:  "init",
	Usage: `Init container process run user's process in container. Do not call it outside`,
	Action: func(ctx *cli.Context) error {
		log.Infof("init come on")
		err := container.RunContainerInitProcess()
		return err
	},
}

var execCommand = cli.Command{
	Name:  "exec",
	Usage: "exec a command into container",
	Flags: []cli.Flag{
		cli.StringFlag{Name: "name,n", Usage: "exec container Name"},
	},
	Action: func(ctx *cli.Context) error {
		if os.Getenv(container.EnvExecPid) != "" {
			log.Infof("pid callback pid %v", os.Getgid())
			return nil
		}
		// runQ exec {containerId} [Command]
		//if len(ctx.Args()) < 1 {
		//	return fmt.Errorf("missing container name of command")
		//}
		//fmt.Println("len Arg>", len(ctx.Args()))

		//containerName := ctx.Args().Get(0)
		containerName := ctx.String("name")
		commandArray := ctx.Args()
		ExecContainer(containerName, commandArray)
		return nil
	},
}
