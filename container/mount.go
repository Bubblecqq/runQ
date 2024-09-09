package container

import (
	log "github.com/sirupsen/logrus"
	"os"
	"os/exec"
	"runQ/constant"
	"runQ/utils"
)

func createLower(containerId, imageName string) {
	lowerPath := utils.GetLower(containerId)
	imagePath := utils.GetImage(imageName)
	log.Infof("lower:%s image.tar:%s", lowerPath, imagePath)
	exist, err := utils.PathExists(lowerPath)
	if err != nil {
		log.Infof("Fail to judge whether dir %s exists. %v", lowerPath, err)
	}
	if !exist {
		if err = os.MkdirAll(lowerPath, constant.Perm0777); err != nil {
			log.Errorf("Mkdir dir %s error. %v", lowerPath, err)
		}
		if _, err = exec.Command("tar", "-xvf", imagePath, "--strip-components=1", "-C", lowerPath).CombinedOutput(); err != nil {
			log.Errorf("Untar dir %s error %v", lowerPath, err)
		}
	}
}

func createDirs(containerId string) {
	dirs := []string{
		utils.GetMerged(containerId),
		utils.GetUpper(containerId),
		utils.GetWorker(containerId),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, constant.Perm0777); err != nil {
			log.Errorf("mkdir dir %s error. %v", dir, err)
		}
	}
}

func mountOverlayFS(containerId string) {
	// 拼接参数
	// e.g. lowerdir=/root/busybox,upperdir=/root/upper,workdir=/root/work
	lower := utils.GetLower(containerId)
	upper := utils.GetUpper(containerId)
	work := utils.GetWorker(containerId)
	dirs := utils.GetOverlayFsDirs(lower, upper, work)
	mergePath := utils.GetMerged(containerId)
	//完整命令：mount -t overlay overlay -o lowerdir=/root/{containerID}/lower,
	//	upperdir=/root/{containerID}/upper,
	//	workdir=/root/{containerID}/work /root/{containerID}/merged
	cmd := exec.Command("mount", "-t", "overlay", "overlay", "-o", dirs, mergePath)
	log.Infof("mount overlayfs: [%s]", cmd.String())
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		log.Errorf("%v", err)
	}
}

func deleteDirs(containerId string) {
	dirs := []string{
		utils.GetMerged(containerId),
		utils.GetUpper(containerId),
		utils.GetWorker(containerId),
		utils.GetLower(containerId),
		utils.GetRoot(containerId),
	}
	for _, dir := range dirs {
		if err := os.RemoveAll(dir); err != nil {
			log.Errorf("Remove dir %s error %v", dir, err)
		}
	}
}

func unmountOverlayFS(containerId string) {
	mntPath := utils.GetMerged(containerId)
	cmd := exec.Command("umount", mntPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	log.Infof("umountOverlayFS,cmd:%v", cmd.String())
	if err := cmd.Run(); err != nil {
		log.Errorf("%v", err)
	}
}
