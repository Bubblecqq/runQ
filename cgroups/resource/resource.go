package resource

type ResourceConfig struct {
	MemoryLimit string
	CpuCfsQuota int
	CpuShare    string
	CpuSet      string
}
