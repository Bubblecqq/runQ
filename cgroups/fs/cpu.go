package fs

import (
	"fmt"
	"os"
	"path"
	"runQ/cgroups/resource"
	"runQ/constant"
	"strconv"
)

type CpuSubsystem struct {
}

const (
	PeriodDefault = 100000
	Percent       = 100
)

func (s *CpuSubsystem) Name() string {
	return "cpu"
}

func (s *CpuSubsystem) Set(cgroupPath string, res *resource.ResourceConfig) error {

	if res.CpuCfsQuota == 0 && res.CpuShare == "" {
		return nil
	}
	subsysCgroupPath, err := getCgroupPath(s.Name(), cgroupPath, true)
	if err != nil {
		return err
	}
	if res.CpuShare != "" {
		if err = os.WriteFile(path.Join(subsysCgroupPath, "cpu.shares"), []byte(res.CpuShare), constant.Perm0644); err != nil {
			return fmt.Errorf("set cgroup cpu share fail %v", err)
		}
	}

	if res.CpuCfsQuota != 0 {
		if err = os.WriteFile(path.Join(subsysCgroupPath, "cpu.cfs_period_us"), []byte(strconv.Itoa(PeriodDefault)), constant.Perm0644); err != nil {
			return fmt.Errorf("set cgroup cpu share fail %v", err)
		}

		if err = os.WriteFile(path.Join(subsysCgroupPath, "cpu.cfs_period_us"), []byte(strconv.Itoa(PeriodDefault/Percent*res.CpuCfsQuota)), constant.Perm0644); err != nil {
			return fmt.Errorf("set cgroup cpu share fail %v", err)
		}
	}
	return nil
}

func (s *CpuSubsystem) Apply(cgroupPath string, pid int, res *resource.ResourceConfig) error {
	if res.CpuCfsQuota == 0 && res.CpuShare == "" {
		return nil
	}
	subsysCgroupPath, err := getCgroupPath(s.Name(), cgroupPath, false)
	if err != nil {
		return fmt.Errorf("get cgroup %s error: %v", cgroupPath, err)
	}
	if err = os.WriteFile(path.Join(subsysCgroupPath, "tasks"), []byte(strconv.Itoa(pid)), constant.Perm0644); err != nil {
		return fmt.Errorf("set cgroup proc fail %v", err)
	}
	return nil
}

func (s *CpuSubsystem) Remove(cgroupPath string) error {
	subsysCgroupPath, err := getCgroupPath(s.Name(), cgroupPath, false)
	if err != nil {
		return err
	}
	return os.RemoveAll(subsysCgroupPath)
}
