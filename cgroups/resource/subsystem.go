package resource

type Subsystem interface {
	Name() string

	Set(path string, res *ResourceConfig) error

	Apply(path string, pid int, config *ResourceConfig) error

	Remove(path string) error
}
