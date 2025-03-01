package network

import (
	"github.com/vishvananda/netlink"
	"net"
)

type Network struct {
	Name    string
	IPRange *net.IPNet
	Driver  string
}

type Endpoint struct {
	ID          string           `json:"id"`
	Device      netlink.Veth     `json:"dev"`
	IPAddress   net.IP           `json:"ip"`
	MacAddress  net.HardwareAddr `json:"mac"`
	Network     *Network
	PortMapping []string
}

type Driver interface {
	Name() string
	Create(subnet string, name string) (*Network, error)
	Delete(name string) error
	Connect(networkName string, endpoint *Endpoint) error
	DisConnect(network *Network, endpoint *Endpoint) error
}

type IPAMer interface {
	Allocate(subnet *net.IPNet) (ip net.IP, err error) // 从指定的 subnet 网段中分配 IP 地址
	Release(subnet *net.IPNet, ipaddr *net.IP) error   //  从指定的 subnet 网段中释放掉指定的 IP 地址。
}
