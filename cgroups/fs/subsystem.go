package fs

import "runQ/cgroups/resource"

var SubsystemIns = []resource.Subsystem{
	&CpusetSubsystem{},
	&MemorySubsystem{},
	&CpuSubsystem{},
}
