package cgroups

import (
	log "github.com/sirupsen/logrus"
	"runQ/cgroups/fs"
	"runQ/cgroups/resource"
)

type CgroupManager struct {
	Path     string
	Resource *resource.ResourceConfig
}

func NewCgroupManager(path string) *CgroupManager {
	return &CgroupManager{Path: path}
}

func (c *CgroupManager) Apply(pid int, config *resource.ResourceConfig) error {
	for _, subSysIns := range fs.SubsystemIns {
		err := subSysIns.Apply(c.Path, pid, config)
		if err != nil {
			log.Errorf("apply subsystem: %s err: %s", subSysIns.Name(), err)
		}

	}
	return nil
}

func (c *CgroupManager) Set(config *resource.ResourceConfig) error {
	for _, subSysIns := range fs.SubsystemIns {
		err := subSysIns.Set(c.Path, config)
		if err != nil {
			log.Errorf("apply subsystem: %s err:%s", subSysIns.Name(), err)
		}
	}
	return nil
}

func (c *CgroupManager) Destroy() error {
	for _, subSysIns := range fs.SubsystemIns {
		if err := subSysIns.Remove(c.Path); err != nil {
			log.Warnf("remove cgroup fail %v", err)
		}
	}
	return nil
}
