package container

import (
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"io"
	"math/rand"
	"os"
	"path"
	"runQ/constant"
	"strconv"
	"strings"
	"syscall"
	"text/tabwriter"
	"time"
)

type ContainerInfo struct {
	Pid         string   `json:"pid"` // 容器的init进程在宿主机上的 PID
	Id          string   `json:"id"`
	Name        string   `json:"name"`
	Command     string   `json:"command"`
	CreateTime  string   `json:"create_time"`
	Status      string   `json:"status"`
	Volume      string   `json:"volume"`
	PortMapping []string `json:"portmapping"`
	NetworkName string   `json:"networkName"`
}

func randStringBytes(n int) string {
	letterBytes := "1234567890"
	rand.Seed(time.Now().UnixNano())
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}

func RecordContainerInfo(containerPID int, commandArray []string, containerName, containerId, volume, networkName string, portMapping []string) (*ContainerInfo, error) {
	if containerName == "" {
		containerName = containerId
	}
	command := strings.Join(commandArray, "")

	containerInfo := &ContainerInfo{
		Id:          containerId,
		Pid:         strconv.Itoa(containerPID),
		Command:     command,
		CreateTime:  time.Now().Format(time.RFC3339),
		Status:      constant.RUNNING,
		Name:        containerName,
		Volume:      volume,
		NetworkName: networkName,
		PortMapping: portMapping,
	}
	JsonBytes, err := json.Marshal(containerInfo)
	if err != nil {
		return containerInfo, errors.WithMessage(err, "container info marshal failed")
	}
	jsonStr := string(JsonBytes)
	dirPath := fmt.Sprintf(constant.InfoLocFormat, containerInfo.Id)
	if err := os.MkdirAll(dirPath, constant.Perm0622); err != nil {
		return containerInfo, errors.WithMessagef(err, "mkdir %s failed", dirPath)
	}
	fileName := path.Join(dirPath, constant.ConfigName)
	file, err := os.Create(fileName)
	if err != nil {
		return containerInfo, errors.WithMessagef(err, "create file %S failed", fileName)
	}
	defer file.Close()

	if _, err = file.WriteString(jsonStr); err != nil {
		return containerInfo, errors.WithMessagef(err, "write container info to file %s failed", err)
	}
	return containerInfo, nil
}

func GenerateContainerID() string {
	return randStringBytes(constant.IDLength)
}

func DeleteContainerInfo(containerId string) error {
	dirPath := fmt.Sprintf(constant.InfoLocFormat, containerId)
	if err := os.RemoveAll(dirPath); err != nil {
		log.Errorf("Remove dir %s error %v", dirPath, err)
		return err
	}
	return nil
}

func PrintListContainers() {
	containers := ListContainers()
	var err error
	w := tabwriter.NewWriter(os.Stdout, 12, 1, 3, ' ', 0)
	_, err = fmt.Fprintf(w, "ID\tName\tPID\tSTATUS\tCOMMAND\tCREATED\n")
	if err != nil {
		log.Errorf("Fprint error %v", err)
	}
	for _, item := range containers {
		_, err = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
			item.Id,
			item.Name,
			item.Pid,
			item.Status,
			item.Command,
			item.CreateTime)
		if err != nil {
			log.Errorf("Fprint error %v", err)
		}
	}
	if err = w.Flush(); err != nil {
		log.Errorf("Flush error %v", err)
	}
}

func ListContainers() []*ContainerInfo {
	containerFiles, err := os.ReadDir(constant.InfoLoc)
	if err != nil {
		log.Errorf("read container dir %s error %v", constant.InfoLoc, err)
		return nil
	}

	containers := make([]*ContainerInfo, 0, len(containerFiles))

	for _, container := range containerFiles {
		tmpContainer, err := getContainerInfo(container)
		if err != nil {
			log.Errorf("get container info error %v", err)
			continue
		}
		containers = append(containers, tmpContainer)
	}
	return containers
}

func getContainerInfo(containerFile os.DirEntry) (*ContainerInfo, error) {
	configFileDir := fmt.Sprintf(constant.InfoLocFormat, containerFile.Name())
	configFileDir = path.Join(configFileDir, constant.ConfigName)
	content, err := os.ReadFile(configFileDir)
	if err != nil {
		log.Errorf("read container config file error %V", configFileDir, err)
		return nil, err
	}
	containerInfo := new(ContainerInfo)
	if err := json.Unmarshal(content, containerInfo); err != nil {
		log.Errorf("json unmarshal error %V", err)
		return nil, err
	}
	return containerInfo, nil
}

func LogContainer(containerId string) {
	logFileLocation := fmt.Sprintf(constant.InfoLocFormat, containerId) + GetLogfile(containerId)
	file, err := os.Open(logFileLocation)
	fmt.Println("logFileLocation", logFileLocation)
	defer file.Close()
	if err != nil {
		log.Errorf("Log container open file %s error %v", logFileLocation, err)
		return
	}
	content, err := io.ReadAll(file)
	if err != nil {
		log.Errorf("Log container read file %s error %v", logFileLocation, err)
		return
	}
	_, err = fmt.Fprintf(os.Stdout, string(content))
	if err != nil {
		log.Errorf("Log container Fprint error %v", err)
		return
	}
}

func StopContainer(containerName string) {
	containerId := GetContainerIdByName(containerName)

	containerInfo, err := getContainerInfoByContainerId(containerId)
	if err != nil {
		log.Errorf("Get container %s info error %v", containerId, err)
		return
	}
	containerPidInt, err := strconv.Atoi(containerInfo.Pid)
	fmt.Println("containerPidInt>", containerPidInt)
	if err != nil {
		log.Errorf("Conver pid from string to int error %v", err)
		return
	}
	// 2.发送SIGTERM信号
	if err = syscall.Kill(containerPidInt, syscall.SIGTERM); err != nil {
		log.Errorf("Stop container %s error %v", containerId, err)
		return
	}
	containerInfo.Status = constant.STOP
	containerInfo.Pid = " "
	newContentBytes, err := json.Marshal(containerInfo)
	fmt.Println("newContentBytes>", newContentBytes)
	if err != nil {
		log.Errorf("Json maeshal %s error %v", containerId, err)
		return
	}
	// 重新写回
	dirPath := fmt.Sprintf(constant.InfoLocFormat, containerId)
	configFilePath := path.Join(dirPath, constant.ConfigName)
	if err := os.WriteFile(configFilePath, newContentBytes, constant.Perm0622); err != nil {
		log.Errorf("Write file %s error: %v", configFilePath, err)
	}
}

func getContainerInfoByContainerId(containerId string) (*ContainerInfo, error) {
	dirPath := fmt.Sprintf(constant.InfoLocFormat, containerId)
	configFilePath := path.Join(dirPath, constant.ConfigName)
	// 读取配置文件
	contentBytes, err := os.ReadFile(configFilePath)
	if err != nil {
		return nil, errors.Wrapf(err, "read file %s", configFilePath)
	}
	var containerInfo ContainerInfo
	//将配置文件映射到containerInfo对象
	if err = json.Unmarshal(contentBytes, &containerInfo); err != nil {
		return nil, err
	}
	return &containerInfo, nil
}

func RemoveContainer(containerName string, force bool) {

	containerId := GetContainerIdByName(containerName)

	containerInfo, err := getContainerInfoByContainerId(containerId)
	if err != nil {
		log.Errorf("Get container %s containerInfo error %v", containerId, err)
		return
	}
	switch containerInfo.Status {
	case constant.STOP:
		// 先删除配置目录，再删除rootfs 目录
		if err = DeleteContainerInfo(containerId); err != nil {
			log.Errorf("Remove container [%s]'s config failed,detail: %v", containerId, err)
			return
		}
		fmt.Println("containerInfo.Volume>", containerInfo.Volume)
		DeleteWorkSpace(containerId, containerInfo.Volume)
	case constant.RUNNING:
		if !force {
			log.Errorf("Couldn't remove running container[%s], stop the container beforce attempting removal or"+
				"force remove", containerId)
			return
		}
		StopContainer(containerId)
		RemoveContainer(containerId, force)
	default:
		log.Errorf("Couldn't remove container, invalid status %s", containerInfo.Status)
		return
	}
}

func GetLogfile(containerId string) string {
	return fmt.Sprintf(constant.LogFile, containerId)
}

func GetContainerIdByName(containerName string) string {
	for _, container := range ListContainers() {
		if container.Name == containerName {
			return container.Id
		}
	}
	return ""
}
