package fs

import (
	"bufio"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"os"
	"path"
	"runQ/constant"
	"strings"
)

const mountPointIndex = 4

func getCgroupPath(subsystem, cgroupPath string, autoCreate bool) (string, error) {
	cgroupRoot := findCgroupMountpoint(subsystem)
	absPath := path.Join(cgroupRoot, cgroupPath)
	if !autoCreate {
		return absPath, nil
	}

	_, err := os.Stat(absPath)
	if err != nil && os.IsNotExist(err) {
		err = os.Mkdir(absPath, constant.Perm0755)
		//fmt.Println("11")
		return absPath, err
	}
	return absPath, errors.Wrap(err, "create cgroup")
}

func findCgroupMountpoint(subsystem string) string {
	// /proc/self/mountinfo 为当前进程的 mountinfo 信息
	// 可以直接通过 cat /proc/self/mountinfo 命令查看
	f, err := os.Open(constant.MOUNTINFO)
	if err != nil {
		return ""
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)

	for scanner.Scan() {
		// txt 大概是这样的：104 85 0:20 / /sys/fs/cgroup/memory rw,nosuid,nodev,noexec,relatime - cgroup cgroup rw,memory
		txt := scanner.Text()
		// 然后按照空格分割
		fields := strings.Split(txt, " ")
		// 对最后一个元素按逗号进行分割，这里的最后一个元素就是 rw,memory
		// 其中的的 memory 就表示这是一个 memory subsystem
		subsystems := strings.Split(fields[len(fields)-1], ",")

		for _, opt := range subsystems {
			if opt == subsystem {
				// 如果等于指定的 subsystem，那么就返回这个挂载点跟目录，就是第四个元素，
				// 这里就是`/sys/fs/cgroup/memory`,即我们要找的根目录
				return fields[mountPointIndex]
			}
		}
	}
	if err = scanner.Err(); err != nil {
		log.Error("read err: ", err)
		return ""
	}
	return ""
}
