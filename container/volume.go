package container

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"os"
	"os/exec"
	"path"
	"runQ/constant"
	"strings"
)

func mountVolume(mntPath string, hostPath string, containerPath string) {
	// /etc/conf: /etc/conf
	// 如果主机的目录不存在则自动创建
	if err := os.Mkdir(hostPath, constant.Perm0777); err != nil {
		log.Infof("mkdir parent dir %s error. %v", hostPath, err)
	}
	// /root/merged/
	// [root@container merged]# ls
	// bin  dev  etc  hello.txt  home  lib  lib64  proc  root  sys  tmp  usr  var
	// 在merged中的目录里创建数据卷目录
	// /root/merged/etc/conf/
	containerPathInHost := path.Join(mntPath, containerPath)
	if err := os.Mkdir(containerPathInHost, constant.Perm0777); err != nil {
		log.Infof("mkdir container %s error. %v", containerPathInHost, err)
	}
	// 执行mount bind挂载
	// mount -o bind /hostPath /containerPath
	cmd := exec.Command("mount", "-o", "bind", hostPath, containerPathInHost)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		log.Errorf("mount volume failed. %v", err)
	}
}

func umountVolume(mntPath, containerPath string) {
	containerPathInHost := path.Join(mntPath, containerPath)
	cmd := exec.Command("umount", containerPathInHost)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		log.Errorf("Umount volume failed. %v", err)
	}
}

// volumeExtract 通过冒号分割解析volume目录，比如 -v /tmp:/tmp
func volumeExtract(volume string) (sourcePath, destinationPath string, err error) {
	parts := strings.Split(volume, ":")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid volume [%s], must split by `:`", volume)
	}
	sourcePath, destinationPath = parts[0], parts[1]
	if sourcePath == "" || destinationPath == "" {
		return "", "", fmt.Errorf("invalid volume [%s], path can't be empty", volume)
	}
	return sourcePath, destinationPath, nil
}
