package network

import (
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netns"
	"io/fs"
	"net"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runQ/constant"
	"runQ/container"
	"runtime"
	"strings"
	"text/tabwriter"
)

var (
	defaultNetworkPath = "/var/lib/runQ/network/network/"
	drivers            = map[string]Driver{}
)

func init() {
	var bridgeDriver = BridgeNetworkDriver{}
	drivers[bridgeDriver.Name()] = &bridgeDriver

	// 文件不存在则创建
	if _, err := os.Stat(defaultNetworkPath); err != nil {
		if !os.IsNotExist(err) {
			log.Errorf("check %s is exist failed,detail: %v", defaultNetworkPath, err)
			return
		}
		if err = os.MkdirAll(defaultNetworkPath, constant.Perm0644); err != nil {
			log.Errorf("create %s failed,detail: %v", defaultNetworkPath, err)
			return
		}
	}
}

func (net *Network) dump(dumpPath string) error {
	if _, err := os.Stat(dumpPath); err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		if err = os.MkdirAll(dumpPath, constant.Perm0644); err != nil {
			return errors.Wrapf(err, "create network dump path %s failed", dumpPath)
		}
	}

	netPath := path.Join(dumpPath, net.Name)
	// 打开保存的文件用于写入,后面打开的模式参数分别是存在内容则清空、只写入、不存在则创建
	netFile, err := os.OpenFile(netPath, os.O_TRUNC|os.O_WRONLY|os.O_CREATE, constant.Perm0644)
	if err != nil {
		return errors.Wrapf(err, "open file %s failed", dumpPath)
	}
	defer netFile.Close()

	netJson, err := json.Marshal(net)
	if err != nil {
		return errors.Wrapf(err, "Marshal %s failed", net)
	}
	_, err = netFile.Write(netJson)
	return errors.Wrapf(err, "write %s failed", netJson)
}

func (net *Network) remove(dumpPath string) error {
	fullPath := path.Join(dumpPath, net.Name)
	if _, err := os.Stat(fullPath); err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		return nil
	}
	return os.Remove(fullPath)
}

func (net *Network) load(dumpPath string) error {
	netConfigFile, err := os.Open(dumpPath)
	if err != nil {
		return err
	}
	defer netConfigFile.Close()
	netJson := make([]byte, 2000)
	n, err := netConfigFile.Read(netJson)
	if err != nil {
		return err
	}
	err = json.Unmarshal(netJson[:n], net)
	return errors.Wrapf(err, "Unmarshal %s failed", netJson[:n])
}

func loadNetwork() (map[string]*Network, error) {
	networks := map[string]*Network{}
	// 检查网络配置目录中的所有文件,并执行第二个参数中的函数指针去处理目录下的每一个文件
	err := filepath.Walk(defaultNetworkPath, func(netPath string, info fs.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}

		_, netName := path.Split(netPath)
		// 加载文件名作为网络名
		net := &Network{Name: netName}
		if err = net.load(netPath); err != nil {
			log.Errorf("error load network: %s", err)
		}
		networks[netName] = net
		return nil
	})
	return networks, err
}

// CreateNetwork 根据不同 driver 创建 Network
func CreateNetwork(driver, subnet, name string) error {
	_, cidr, _ := net.ParseCIDR(subnet)
	// 通过IPAM分配网关IP，获取到网段中的第一个IP作为网关的IP
	ip, err := ipAllocator.Allocate(cidr)
	if err != nil {
		return err
	}
	cidr.IP = ip
	// 调用指定的网络驱动创建网络，这里的 drivers 字典是各个网络驱动的实例字典 通过调用网络驱动
	// Create 方法创建网络，后面会以 Bridge 驱动为例介绍它的实现
	net, err := drivers[driver].Create(cidr.String(), name)
	if err != nil {
		return err
	}
	// 保存网络信息，将网络的信息保存在文件系统中，以便查询和在网络上连接网络端点
	return net.dump(defaultNetworkPath)
}

func ListNetwork() {
	networks, err := loadNetwork()
	if err != nil {
		log.Errorf("load network from file failed,detail: %v", err)
		return
	}
	w := tabwriter.NewWriter(os.Stdout, 12, 1, 3, ' ', 0)
	_, _ = fmt.Fprint(w, "NAME\tIpRange\tDriver\n")
	for _, network := range networks {
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\n",
			network.Name,
			network.IPRange,
			network.Driver,
		)
		if err != nil {
			return
		}
	}
	if err = w.Flush(); err != nil {
		log.Errorf("Flush error %v", err)
		return
	}
}

func DeleteNetwork(networkName string) error {
	networks, err := loadNetwork()
	if err != nil {
		return errors.WithMessage(err, "load network from file failed")
	}
	net, ok := networks[networkName]

	if !ok {
		return fmt.Errorf("no Such Network: %s", networkName)
	}

	if err = ipAllocator.Release(net.IPRange, &net.IPRange.IP); err != nil {
		return errors.Wrap(err, "remove Network gateway ip failed")
	}

	if err = drivers[net.Driver].Delete(net.Name); err != nil {
		return errors.Wrap(err, "remove Network DriverError failed")
	}

	return net.remove(defaultNetworkPath)
}

// Connect 连接容器到之前创建的网络 mydocker run -net testnet -p 8080:80 xxxx
func Connect(networkName string, info *container.ContainerInfo) (net.IP, error) {
	networks, err := loadNetwork()
	if err != nil {
		return nil, errors.WithMessage(err, "load network from file failed")
	}
	network, ok := networks[networkName]
	if !ok {
		return nil, fmt.Errorf("no Such Network: %s", networkName)
	}
	ip, err := ipAllocator.Allocate(network.IPRange)
	if err != nil {
		return ip, errors.Wrapf(err, "allocate ip")
	}

	ep := &Endpoint{
		ID:          fmt.Sprintf("%s-%s", info.Id, networkName),
		IPAddress:   ip,
		Network:     network,
		PortMapping: info.PortMapping,
	}
	if err = drivers[network.Driver].Connect(network.Name, ep); err != nil {
		return ip, err
	}
	if err = configEndpointIpAddressAndRoute(ep, info); err != nil {
		return ip, err
	}
	return ip, configPortMapping(ep)
}

func configEndpointIpAddressAndRoute(ep *Endpoint, info *container.ContainerInfo) error {
	peerLink, err := netlink.LinkByName(ep.Device.PeerName)
	if err != nil {
		return fmt.Errorf("fail config endpoint: %v", err)
	}
	// 将容器的网络端点加入到容器的网络空间中
	// 并使这个函数下面的操作都在这个网络空间中进行
	// 执行完函数后，恢复为默认的网络空间，具体实现下面再做介绍
	defer enterContainerNetNS(&peerLink, info)()
	// 获取到容器的IP地址及网段，用于配置容器内部接口地址
	// 比如容器IP是192.168.1.2， 而网络的网段是192.168.1.0/24
	// 那么这里产出的IP字符串就是192.168.1.2/24，用于容器内Veth端点配置

	interfaceIP := *ep.Network.IPRange
	interfaceIP.IP = ep.IPAddress
	// 设置容器内Veth端点的IP
	if err = setInterfaceIP(ep.Device.PeerName, interfaceIP.String()); err != nil {
		return fmt.Errorf("%v%s", ep.Network, err)
	}

	if err = setInterfaceUP(ep.Device.PeerName); err != nil {
		return err
	}
	// Net Namespace 中默认本地地址 127 的勺。”网卡是关闭状态的
	// 启动它以保证容器访问自己的请求
	if err = setInterfaceUP("lo"); err != nil {
		return err
	}
	// 设置容器内的外部请求都通过容器内的Veth端点访问
	// 0.0.0.0/0的网段，表示所有的IP地址段
	_, cidr, _ := net.ParseCIDR("0.0.0.0/0")
	defaultRoute := &netlink.Route{
		LinkIndex: peerLink.Attrs().Index,
		Gw:        ep.Network.IPRange.IP,
		Dst:       cidr,
	}
	if err = netlink.RouteAdd(defaultRoute); err != nil {
		return err
	}
	return nil
}

func configPortMapping(ep *Endpoint) error {
	var err error
	for _, pm := range ep.PortMapping {
		portMapping := strings.Split(pm, ":")
		if len(portMapping) != 2 {
			log.Errorf("port mapping format error, %v", err)
			continue
		}
		iptablesCmd := fmt.Sprintf("-t nat -A PREROUTING ! -i %s -p tcp -m tcp --dport %s -j DNAT --to-destination %s:%s",
			ep.Network.Name, portMapping[0], ep.IPAddress.String(), portMapping[1])
		cmd := exec.Command("iptables", strings.Split(iptablesCmd, " ")...)
		log.Infoln("配置端口映射 cmd：", cmd.String())
		output, err := cmd.Output()
		if err != nil {
			log.Errorf("iptables Output, %v", output)
			continue
		}
	}
	return err
}

func enterContainerNetNS(enLink *netlink.Link, info *container.ContainerInfo) func() {
	// 找到容器的Net Namespace
	// /proc/[pid]/ns/net 打开这个文件的文件描述符就可以来操作Net Namespace
	// 而ContainerInfo中的PID,即容器在宿主机上映射的进程ID
	f, err := os.OpenFile(fmt.Sprintf("/proc/%s/ns/net", info.Pid), os.O_RDONLY, 0)
	if err != nil {
		log.Errorf("error get container net namespace, %v", err)
	}

	nsFD := f.Fd()
	// 锁定当前程序所执行的线程，如果不锁定操作系统线程的话
	// Go语言的goroutine可能会被调度到别的线程上去
	// 就不能保证一直在所需要的网络空间中了
	// 所以先调用runtime.LockOSThread()锁定当前程序执行的线程
	runtime.LockOSThread()
	// 修改网络端点Veth的另外一端，将其移动到容器的Net Namespace 中
	if err = netlink.LinkSetNsFd(*enLink, int(nsFD)); err != nil {
		log.Errorf("error set link netns, %v", err)
	}

	// 获取当前网络的namespace
	origns, err := netns.Get()
	if err != nil {
		log.Errorf("error get current netns, %v", err)
	}
	// 调用 netns.Set方法，将当前进程加入容器的Net Namespace
	if err = netns.Set(netns.NsHandle(nsFD)); err != nil {
		log.Errorf("error set netns, %v", err)
	}
	// 返回之前Net Namespace的函数
	// 在容器的网络空间中执行完容器配置之后调用此函数就可以将程序恢复到原生的Net Namespace
	return func() {
		// 恢复到上面获取到的之前的 Net Namespace
		netns.Set(origns)
		origns.Close()
		// 取消对当附程序的线程锁定
		runtime.UnlockOSThread()
		f.Close()
	}
}
