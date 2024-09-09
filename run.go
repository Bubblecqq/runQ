package main

import (
	log "github.com/sirupsen/logrus"
	"os"
	"runQ/cgroups"
	"runQ/cgroups/resource"
	"runQ/container"
	"runQ/network"
	"strconv"
	"strings"
)

func Run(tty bool, comArray, envSlice []string, res *resource.ResourceConfig, volume, containerName, imageName string,
	net string, portMapping []string) {

	containerId := container.GenerateContainerID()
	parent, writePipe := container.NewParentProcess(tty, volume, containerId, imageName, envSlice)
	if parent == nil {
		log.Errorf("New parent process error")
		return
	}

	if err := parent.Start(); err != nil {
		log.Errorf("Run parent. Start err: %v", err)
		return
	}

	_, err := container.RecordContainerInfo(parent.Process.Pid, comArray, containerName, containerId, volume, net, portMapping)

	if err != nil {
		log.Errorf("Record container info error %v", err)
		return
	}

	cgroupManager := cgroups.NewCgroupManager("runQ-cgroup")
	defer cgroupManager.Destroy()
	_ = cgroupManager.Set(res)
	_ = cgroupManager.Apply(parent.Process.Pid, res)
	log.Infof("Current container pid is %d", parent.Process.Pid)

	if net != "" {
		containerInfo := &container.ContainerInfo{
			Id:          containerId,
			Pid:         strconv.Itoa(parent.Process.Pid),
			Name:        containerName,
			PortMapping: portMapping,
		}
		if _, err = network.Connect(net, containerInfo); err != nil {
			log.Errorf("Error Connect Network %v", err)
		}
	}

	// 在子进程创建后通过管道来发送参数
	sendInitCommand(comArray, writePipe)

	//go func() {
	//	if !tty {
	//		_, _ = parent.Process.Wait()
	//	}
	//
	//	//清理工作
	//	container.DeleteWorkSpace(containerId, volume)
	//	_ = container.DeleteContainerInfo(containerId)
	//	_ = cgroupManager.Destroy()
	//}()
	if tty {
		_ = parent.Wait()
		container.DeleteWorkSpace(containerId, volume)
		_ = container.DeleteContainerInfo(containerId)
	}

}

func sendInitCommand(comArray []string, writePipe *os.File) {
	command := strings.Join(comArray, " ")
	log.Infof("command all is %s", command)
	_, _ = writePipe.WriteString(command)
	_ = writePipe.Close()
}
