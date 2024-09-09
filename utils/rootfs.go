package utils

import "fmt"

const (
	ImagePath       = "/var/lib/runQ/image/"
	RootPath        = "/var/lib/runQ/overlay2/"
	lowerDirFormat  = RootPath + "%s/lower"
	upperDirFormat  = RootPath + "%s/upper"
	workDirFormat   = RootPath + "%s/work"
	mergedDirFormat = RootPath + "%s/merged"
	overlayFSFormat = "lowerdir=%s,upperdir=%s,workdir=%s"
)

func GetRoot(containerId string) string {
	return RootPath + containerId
}

func GetImage(imageName string) string {
	return fmt.Sprintf("%s%s.tar", ImagePath, imageName)
}

func GetLower(containerId string) string {
	return fmt.Sprintf(lowerDirFormat, containerId)
}

func GetUpper(containerId string) string {
	return fmt.Sprintf(upperDirFormat, containerId)
}
func GetWorker(containerId string) string {
	return fmt.Sprintf(workDirFormat, containerId)
}
func GetMerged(containerId string) string {
	return fmt.Sprintf(mergedDirFormat, containerId)
}

func GetOverlayFsDirs(lower, upper, worker string) string {
	return fmt.Sprintf(overlayFSFormat, lower, upper, worker)
}
