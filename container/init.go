package container

import (
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runQ/constant"
	"strings"
	"syscall"
)

const fdIndex = 3

// RunContainerInitProcess 启动容器的init进程
/*
这里的init函数是在容器内部执行的，也就是说，代码执行到这里后，容器所在的进程其实就已经创建出来了，
这是本容器执行的第一个进程。
使用mount先去挂载proc文件系统，以便后面通过ps等系统命令去查看当前进程资源的情况。
*/
func RunContainerInitProcess() error {
	// 按位或运算符，用于将多个标志组合在一起
	//defaultMountFlags := syscall.MS_NOEXEC | syscall.MS_NOSUID | syscall.MS_NODEV
	//_ = syscall.Mount("", constant.ROOTDIR, "", syscall.MS_PRIVATE|syscall.MS_REC, "")
	//_ = syscall.Mount("proc", "/proc", "proc", uintptr(defaultMountFlags), "")
	// argv := []string{command}
	cmdArray := readUserCommand()
	// 去除空格，提取所有命令
	if len(cmdArray) == 0 {
		return errors.New("run container get user command error,cmdArray is nil")
	}

	setupMount()

	path, err := exec.LookPath(cmdArray[0])
	if err != nil {
		log.Errorf("Exec loop path error %v", err)
		return err
	}

	log.Infof("Find path %s", path)
	// command-> /bin/sh
	// argv ->  [/bin/sh]
	if err := syscall.Exec(path, cmdArray[0:], os.Environ()); err != nil {
		// syscall.Exec(command, argv, os.Environ()) 相当于调用系统执行命令，
		// 并将当前进程替换为一个新的进程，同时将当前进程的环境变量传递给新的进程。
		log.Errorf("RunContainerInitProcess exec : " + err.Error())
	}
	return nil
}

func readUserCommand() []string {
	pipe := os.NewFile(uintptr(fdIndex), "pipe")
	defer pipe.Close()
	msg, err := io.ReadAll(pipe)
	// msg [47 98 105 110 47 115 104] -> 转string，就是/bin/sh
	if err != nil {
		log.Errorf("init read pipe error %v", err)
		return nil
	}
	msgStr := string(msg)
	return strings.Split(msgStr, " ")
}

/*
*
Init 挂载点
*/

func setupMount() {
	pwd, err := os.Getwd()
	if err != nil {
		log.Errorf("Get current location error %v", err)
		return
	}

	log.Infof("Current location is %s", pwd)
	// systemd 加入linux之后, mount namespace 就变成 shared by default, 所以你必须显示
	// 声明你要这个新的mount namespace独立。
	// 如果不先做 private mount，会导致挂载事件外泄，后续执行 pivotRoot 会出现 invalid argument 错误
	err = syscall.Mount("", constant.ROOTDIR, "", syscall.MS_PRIVATE|syscall.MS_REC, "")

	err = pivotRoot(pwd)

	if err != nil {
		log.Errorf("pivotRoot failed,detail: %v", err)
		return
	}
	// mount /proc
	defaultMountFlags := syscall.MS_NOEXEC | syscall.MS_NOSUID | syscall.MS_NODEV
	_ = syscall.Mount("proc", "/proc", "proc", uintptr(defaultMountFlags), "")

	// tmpfs 是基于 件系 使用 RAM、swap 分区来存储。
	// 不挂载 /dev，会导致容器内部无法访问和使用许多设备，这可能导致系统无法正常工作
	_ = syscall.Mount("tmpfs", "/dev", "tmpfs", syscall.MS_NOSUID|syscall.MS_STRICTATIME, "mode=755")
	// Set PATH environment variable
	// ++++NEW++++
	// 需要指定好环境变量，否则进入到容器后无法使用命令
	setContainerENV()
}

func pivotRoot(root string) error {
	if err := syscall.Mount(root, root, "bind", syscall.MS_BIND|syscall.MS_REC, ""); err != nil {
		return errors.Wrap(err, "mount rootfs to itself")
	}
	// 创建 rootfs/.pivot_root 目录用于存储 old_root
	pivotDir := filepath.Join(root, constant.PIVOT_ROOT)
	if err := os.Mkdir(pivotDir, constant.Perm0777); err != nil {
		return err
	}
	// 执行pivot_root调用,将系统rootfs切换到新的rootfs,
	// PivotRoot调用会把 old_root挂载到pivotDir,也就是rootfs/.pivot_root,挂载点现在依然可以在mount命令中看到
	if err := syscall.PivotRoot(root, pivotDir); err != nil {
		return errors.WithMessagef(err, "pivotRoot failed,new_root:%v old_put:%v", root, pivotDir)
	}
	// 修改当前的工作目录到根目录
	if err := syscall.Chdir(constant.ROOTDIR); err != nil {
		return errors.WithMessage(err, "chdir to / failed")
	}

	// 最后再把old_root umount了，即 umount rootfs/.pivot_root
	// 由于当前已经是在 rootfs 下了，就不能再用上面的rootfs/.pivot_root这个路径了,现在直接用/.pivot_root这个路径即可
	pivotDir = filepath.Join(constant.ROOTDIR, constant.PIVOT_ROOT)
	if err := syscall.Unmount(pivotDir, syscall.MNT_DETACH); err != nil {
		return errors.WithMessage(err, "unmount pivot_root dir")
	}
	// 删除临时文件夹
	return os.Remove(pivotDir)
}
