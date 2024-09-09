package container

import (
	log "github.com/sirupsen/logrus"
	"os"
)

const (
	EnvExecPid       = "runQ_pid"
	EnvExecCmd       = "runQ_cmd"
	EnvExecContainer = "runQ_containerName"
)

func setContainerENV() {
	_ = os.Setenv("PATH", "/bin:"+os.Getenv("PATH"))
	log.Infof("Environment PATH: %s", os.Getenv("PATH"))
}

func SetExecENV(pid string, cmdStr string) {
	_ = os.Setenv(EnvExecPid, pid)
	_ = os.Setenv(EnvExecCmd, cmdStr)
	log.Infof("Exec Env setup Successfully with Pid: %s command: %s", pid, cmdStr)
}
