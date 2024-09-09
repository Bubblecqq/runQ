package network

import (
	"fmt"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/vishvananda/netlink"
	"net"
	"os/exec"
	"strings"
	"time"
)

type BridgeNetworkDriver struct {
}

func (d *BridgeNetworkDriver) Name() string {
	return "bridge"
}
func (d *BridgeNetworkDriver) Create(subnet string, name string) (*Network, error) {
	ip, ipRange, _ := net.ParseCIDR(subnet)
	ipRange.IP = ip
	n := &Network{Name: name, IPRange: ipRange, Driver: d.Name()}
	err := d.initBridge(n)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to create bridge network")
	}
	return n, err
}
func (d *BridgeNetworkDriver) Delete(name string) error {
	br, err := netlink.LinkByName(name)
	if err != nil {
		return err
	}
	return netlink.LinkDel(br)
}
func (d *BridgeNetworkDriver) Connect(networkName string, endpoint *Endpoint) error {

	bridgeName := networkName
	br, err := netlink.LinkByName(bridgeName)
	if err != nil {
		return err
	}
	// 创建 Veth 接口的配置
	la := netlink.NewLinkAttrs()
	// linux 接口名的限制，取endpoint前5
	la.Name = endpoint.ID[:5]
	// 通过设置 Veth 接口 master 属性，设置这个Veth的一端挂载到网络对应的 Linux Bridge
	la.MasterIndex = br.Attrs().Index
	// 创建 Veth 对象，通过 PeerNarne 配置 Veth 另外 端的接口名
	// 配置 Veth 另外 端的名字 cif {endpoint ID 的前 位｝
	endpoint.Device = netlink.Veth{
		LinkAttrs: la,
		PeerName:  "cif-" + endpoint.ID[:5],
	}
	// 调用netlink的LinkAdd方法创建出这个Veth接口
	// 因为上面指定了link的MasterIndex是网络对应的Linux Bridge
	// 所以Veth的一端就已经挂载到了网络对应的LinuxBridge.上
	if err = netlink.LinkAdd(&endpoint.Device); err != nil {
		return fmt.Errorf("error Add Endpoint Device: %v", err)
	}
	// ip link set xxx up
	if err = netlink.LinkSetUp(&endpoint.Device); err != nil {
		return fmt.Errorf("error Add Endpoint Device: %v", err)
	}

	return nil
}

func (d *BridgeNetworkDriver) DisConnect(network *Network, endpoint *Endpoint) error {
	vethName := endpoint.ID[:5]
	veth, err := netlink.LinkByName(vethName)
	if err != nil {
		return err
	}
	// 从网桥解绑
	err = netlink.LinkSetNoMaster(veth)
	if err != nil {
		return errors.WithMessagef(err, "find veth [%s] failed,", vethName)
	}
	// 删除 veth-pair
	// 一端为 xxx,另一端为 cif-xxx
	err = netlink.LinkDel(veth)
	if err != nil {
		return errors.WithMessagef(err, "delete veth [%s] failed,", vethName)
	}
	veth2Name := "cif-" + vethName
	veth2, err := netlink.LinkByName(veth2Name)
	if err != nil {
		return errors.WithMessagef(err, "find veth [%s] failed,", veth2Name)
	}
	err = netlink.LinkDel(veth2)
	if err != nil {
		return errors.WithMessagef(err, "delete veth [%s] failed", veth2Name)
	}

	return nil
}

// initBridge 初始化Linux Bridge
/*
Linux Bridge 初始化流程如下：
* 1）创建 Bridge 虚拟设备
* 2）设置 Bridge 设备地址和路由
* 3）启动 Bridge 设备
* 4）设置 iptables SNAT 规则
*/
func (d *BridgeNetworkDriver) initBridge(n *Network) error {

	bridgeName := n.Name
	// 1) 创建 Bridge 虚拟设备
	if err := createBridgeInterface(bridgeName); err != nil {
		return errors.Wrapf(err, "Failed to create bridge %s", bridgeName)
	}

	// 2）设置 Bridge 设备地址和路由
	gatewayIP := *n.IPRange
	gatewayIP.IP = n.IPRange.IP

	if err := setInterfaceIP(bridgeName, gatewayIP.String()); err != nil {
		return errors.Wrapf(err, "Error set bridge ip: %s no bridge: %s", gatewayIP.String(), bridgeName)
	}

	// 3） 启动 Bridge 设备
	if err := setInterfaceUP(bridgeName); err != nil {
		return errors.Wrapf(err, "Failed to set %s up", bridgeName)
	}

	// 4) 设置iptables SNAT 规则
	if err := setupIPTables(bridgeName, n.IPRange); err != nil {
		return errors.Wrapf(err, "Failed to set up iptables for %s", bridgeName)
	}
	return nil
}
func (d *BridgeNetworkDriver) deleteBridge(n *Network) error {
	bridgeName := n.Name

	l, err := netlink.LinkByName(bridgeName)
	if err != nil {
		return fmt.Errorf("Getting link with name %s failed: %v", bridgeName, err)
	}
	if err := netlink.LinkDel(l); err != nil {
		return fmt.Errorf("failed to remove bridge interface %s delete: %v", bridgeName, err)
	}
	return nil
}

// 用于实现 ip link add x
func createBridgeInterface(bridgeName string) error {
	// 先检查是否已经存在了这个同名的Bridge设备
	_, err := net.InterfaceByName(bridgeName)
	// 如果已经存在或者报错则返回创建错误
	if err == nil || !strings.Contains(err.Error(), "no such network interface") {
		return err
	}

	// create *netlink.Bridge object
	la := netlink.NewLinkAttrs()
	la.Name = bridgeName
	// 实验刚创建的Link属性创建netlink Bridge对象
	br := &netlink.Bridge{LinkAttrs: la}
	// 创建Bridge 虚拟网络设备
	if err = netlink.LinkAdd(br); err != nil {
		return errors.Wrapf(err, "create bridge %s error", bridgeName)
	}
	return nil
}

// 实现ip addr add
func setInterfaceIP(name, rawIP string) error {
	retries := 2
	var iface netlink.Link
	var err error
	for i := 0; i < retries; i++ {
		iface, err = netlink.LinkByName(name)
		if err == nil {
			break
		}
		log.Debugf("error retrieving new bridge netlink link [ %s ]... retrying", name)
		time.Sleep(2 * time.Second)
	}
	if err != nil {
		return errors.Wrap(err, "abandoning retrieving the new bridge link from netlink, Run [ ip link ] to troubleshoot")
	}

	ipNet, err := netlink.ParseIPNet(rawIP)
	if err != nil {
		return err
	}
	// 通过  netlink.AddrAdd给网络接口配置地址，相当于ip addr add xxx命令
	// 同时如果配置了地址所在网段的信息，例如 192.168.0.0/24
	// 还会配置路由表 192.168.0.0/24 转发到这 testbridge 的网络接口上
	addr := &netlink.Addr{IPNet: ipNet}
	return netlink.AddrAdd(iface, addr)
}

func setInterfaceUP(interfaceName string) error {
	link, err := netlink.LinkByName(interfaceName)
	if err != nil {
		return errors.Wrapf(err, "error retrieving a link named [ %s ]:", link.Attrs().Name)
	}
	// 等价 ip link set xxx up
	if err = netlink.LinkSetUp(link); err != nil {
		return errors.Wrapf(err, "nabling interface for %s", interfaceName)
	}
	return nil
}

// iptables -t nat -A POSTROUTING -s 172.18.0.0/24 -o eth0 -j MASQUERADE
// 语法：iptables -t nat -A POSTROUTING -s {subnet} -o {deviceName} -j MASQUERADE
func setupIPTables(bridgeName string, subnet *net.IPNet) error {
	iptablesCmd := fmt.Sprintf("-t nat -A POSTROUTING -s %s ! -o %s -j MASQUERADE", subnet.String(), bridgeName)
	cmd := exec.Command("iptables", strings.Split(iptablesCmd, " ")...)
	output, err := cmd.Output()
	if err != nil {
		log.Errorf("iptables Output, %v", output)
	}
	return err
}
