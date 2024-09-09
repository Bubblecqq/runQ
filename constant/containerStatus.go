package constant

const (
	RUNNING       = "running"
	STOP          = "stopped"
	EXIT          = "exited"
	InfoLoc       = "/var/lib/runQ/containers/"
	InfoLocFormat = InfoLoc + "%s/"
	ConfigName    = "config.json"
	IDLength      = 10
	LogFile       = "%s-json.log"
)
