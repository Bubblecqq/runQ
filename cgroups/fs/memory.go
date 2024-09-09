package fs

import (
	"fmt"
	"github.com/pkg/errors"
	"os"
	"path"
	"runQ/cgroups/resource"
	"runQ/constant"
	"strconv"
)

type MemorySubsystem struct {
}

func (s *MemorySubsystem) Name() string {
	return "memory"
}

func (s *MemorySubsystem) Set(cgroupPath string, res *resource.ResourceConfig) error {

	if res.MemoryLimit == "" {
		return nil
	}

	subsystemCgroupPath, err := getCgroupPath(s.Name(), cgroupPath, true)
	if err != nil {
		return err
	}
	if err := os.WriteFile(path.Join(subsystemCgroupPath, "memory.limit_in_bytes"), []byte(res.MemoryLimit), constant.Perm0644); err != nil {
		return fmt.Errorf("set cgroup memory fail %v", err)
	}
	return nil
}

func (s *MemorySubsystem) Apply(cgroupPath string, pid int, config *resource.ResourceConfig) error {

	subsystemCgroupPath, err := getCgroupPath(s.Name(), cgroupPath, true)

	if err != nil {
		return errors.Wrapf(err, "get cgroup %s", cgroupPath)
	}
	//fmt.Println("Pid>", strconv.Itoa(pid))
	if err := os.WriteFile(path.Join(subsystemCgroupPath, "tasks"), []byte(strconv.Itoa(pid)), constant.Perm0644); err != nil {
		return fmt.Errorf("set cgroup proc fail %v", err)
	}
	return nil
}

func (s *MemorySubsystem) Remove(cgroupPath string) error {
	subsystemCgroupPath, err := getCgroupPath(s.Name(), cgroupPath, false)
	if err != nil {
		return err
	}
	return os.RemoveAll(subsystemCgroupPath)
}
