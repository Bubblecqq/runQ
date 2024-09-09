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

type CpusetSubsystem struct {
}

func (s *CpusetSubsystem) Name() string {
	return "cpu"
}

func (s *CpusetSubsystem) Set(cgroupPath string, res *resource.ResourceConfig) error {

	if res.CpuSet == "" {
		return nil
	}
	subsysCgroupPath, err := getCgroupPath(s.Name(), cgroupPath, true)
	if err != nil {
		return err
	}
	if err := os.WriteFile(path.Join(subsysCgroupPath, "cpuset.cpus"), []byte(res.CpuSet), constant.Perm0644); err != nil {
		return fmt.Errorf("set cgroup cpuset fail %v", err)
	}
	return nil
}

func (s *CpusetSubsystem) Apply(cgroupPath string, pid int, res *resource.ResourceConfig) error {
	if res.CpuSet == "" {
		return nil
	}
	subsysCgroupPath, err := getCgroupPath(s.Name(), cgroupPath, false)
	if err != nil {
		return errors.Wrapf(err, "get cgroup %s", cgroupPath)
	}
	if err = os.WriteFile(path.Join(subsysCgroupPath, "tasks"), []byte(strconv.Itoa(pid)), constant.Perm0644); err != nil {
		return fmt.Errorf("set cgroup proc fail %v", err)
	}
	return nil
}

func (s *CpusetSubsystem) Remove(cgroupPath string) error {
	subsysCgroupPath, err := getCgroupPath(s.Name(), cgroupPath, false)
	if err != nil {
		return err
	}
	return os.RemoveAll(subsysCgroupPath)
}
