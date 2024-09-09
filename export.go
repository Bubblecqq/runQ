package main

import (
	"fmt"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli"
	"os/exec"
	"runQ/utils"
)

var exportCommand = cli.Command{
	Name:  "export",
	Usage: "export container to image, e.g. runQ export 1234567890 myimage",
	Action: func(ctx *cli.Context) error {
		if len(ctx.Args()) < 2 {
			return fmt.Errorf("missing containerName and image name")
		}
		containerId := ctx.Args().Get(0)
		imageName := ctx.Args().Get(1)

		return exportContainer(containerId, imageName)
	},
}

var ErrImageAlreadyExists = errors.New("Image Already Exists")

//func exportContainerOld(containerId, imageName string) error {
//
//	mntPath := "/root/merged"
//	imageTar := "/root/" + imageName + ".tar"
//	fmt.Println("export imageTar at:", imageTar)
//	if _, err := exec.Command("tar", "-czf", imageTar, "-C", mntPath, ".").CombinedOutput(); err != nil {
//		log.Errorf("tar folder %s error %v", mntPath, err)
//	}
//}

func exportContainer(containerId, imageName string) error {

	mntPath := utils.GetMerged(containerId)
	imageTar := utils.GetImage(imageName)
	exists, err := utils.PathExists(imageTar)
	if err != nil {
		return errors.WithMessagef(err, "check is image [%s%s] exist failed", imageName, imageTar)
	}

	if exists {
		return ErrImageAlreadyExists
	}
	log.Infof("export Container imageTar:%s", imageTar)

	if _, err := exec.Command("tar", "-czf", imageTar, "-C", mntPath, ".").CombinedOutput(); err != nil {
		log.Errorf("tar folder %s error %v", mntPath, err)
	}
	return nil
}
