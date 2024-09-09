package main

import (
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli"
	"os"
	_ "runQ/nsenter"
)

const usage = `runQ is a simple container runtime implementation.`

func main() {
	app := cli.NewApp()
	app.Name = "runQ"
	app.Usage = usage

	app.Commands = []cli.Command{
		initCommand,
		runCommand,
		exportCommand,
		listCommand,
		logCommand,
		execCommand,
		stopCommand,
		removeCommand,
		networkCommand,
	}

	app.Before = func(context *cli.Context) error {
		log.SetFormatter(&log.JSONFormatter{})
		log.SetOutput(os.Stdout)
		return nil
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
