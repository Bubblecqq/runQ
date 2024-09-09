package container

import (
	log "github.com/sirupsen/logrus"
	"runQ/utils"
)

//func NewWorkSpace0ld(rootPath string, mntURL, volume, imageName string) {
//	// rootPath = "/root/"   mntURL= "/root/merged/"
//	// 先挂载了coW，lower是镜像内的rootfs
//	createLower(rootPath)
//	createDirs(rootPath)
//	mountOverlayFS(rootPath, mntURL)
//
//	if volume != "" {
//		mntPath := path.Join(rootPath, "merged")
//		//  /etc/conf:/etc/conf   hostPath:containerPath
//		hostPath, containerPath, err := volumeExtract(volume)
//		if err != nil {
//			log.Errorf("extract volume failed，maybe volume parameter input is not correct，detail:%v", err)
//			return
//		}
//		mountVolume(mntPath, hostPath, containerPath)
//	}
//}

func NewWorkSpace(containerId, imageName, volume string) {
	createLower(containerId, imageName)
	createDirs(containerId)
	mountOverlayFS(containerId)

	if volume != "" {
		mntPath := utils.GetMerged(containerId)
		hostPath, containerPath, err := volumeExtract(volume)
		if err != nil {
			log.Errorf("extract volume failed，maybe volume parameter input is not correct，detail:%v", err)
			return
		}
		mountVolume(mntPath, hostPath, containerPath)
	}

}

func DeleteWorkSpace(containerId string, volume string) {
	// 如果指定了volume则需要umount volume
	// NOTE: 一定要要先 umount volume ，然后再删除目录，
	// 否则由于 bind mount 存在，删除临时目录会导致 volume 目录中的数据丢失。
	if volume != "" {
		_, containerPath, err := volumeExtract(volume)
		if err != nil {
			log.Errorf("extract volume failed, maybe volume parameter input is not correct, detail: %v", err)
			return
		}
		// umount /containerPath
		mntPath := utils.GetMerged(containerId)
		umountVolume(mntPath, containerPath)
	}

	unmountOverlayFS(containerId)
	deleteDirs(containerId)
}
