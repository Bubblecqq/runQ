package main

import (
	"encoding/json"
	"fmt"
	log "github.com/sirupsen/logrus"
	"os"
	"os/exec"
	"path"
	"runQ/constant"
	"runQ/container"
	_ "runQ/nsenter"
	"strings"
)

func ExecContainer(containerName string, comArray []string) {

	containerId := container.GetContainerIdByName(containerName)

	pid, err := GetPidByContainerId(containerId)
	fmt.Println("pid>", pid)
	if err != nil {
		log.Errorf("Exec container getContainerPidByName %s error %v", containerId, err)
		return
	}
	//fmt.Println("cmdArray>", comArray)
	// cmdArray> [/bin/sh]
	cmd := exec.Command(constant.EXECSELF, "exec")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	cmdStr := strings.Join(comArray, " ")
	log.Infof("container pid: %s command: %s", pid, cmdStr)
	_ = os.Setenv(container.EnvExecPid, pid)
	_ = os.Setenv(container.EnvExecCmd, cmdStr)
	// 把指定PID进程的环境变量传递给新启动的进程，实现通过exec命令也能查询到容器的环境变量
	containerEnvs := GetEnvsByPid(pid)
	cmd.Env = append(os.Environ(), containerEnvs...)
	if err = cmd.Run(); err != nil {
		log.Errorf("Exec container %s error %v", containerId, err)
	}
}

func GetPidByContainerId(containerId string) (string, error) {
	dirPath := fmt.Sprintf(constant.InfoLocFormat, containerId)

	configFilePath := path.Join(dirPath, constant.ConfigName)
	fmt.Println("configFilePath>", configFilePath)
	contentBytes, err := os.ReadFile(configFilePath)
	if err != nil {
		return "", err
	}
	var containerInfo container.ContainerInfo
	if err = json.Unmarshal(contentBytes, &containerInfo); err != nil {
		return "", err
	}
	return containerInfo.Pid, nil
}

func GetEnvsByPid(pid string) []string {
	EnvsPath := fmt.Sprintf("/proc/%s/environ", pid)
	contentBytes, err := os.ReadFile(EnvsPath)
	if err != nil {
		log.Errorf("Read file %s error %v", EnvsPath, err)
		return nil
	}
	envs := strings.Split(string(contentBytes), "\u0000")
	return envs
}
