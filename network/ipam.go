package network

import (
	"encoding/json"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"net"
	"os"
	"path"
	"runQ/constant"
	"strings"
)

const ipamDefaultAllocatorPath = "/var/lib/runQ/network/ipam/subnet.json"

type IPAM struct {
	SubnetAllocatorPath string
	Subnets             *map[string]string
}

// 初始化一个IPAM的对象，默认使用/var/lib/mydocker/network/ipam/subnet.json作为分配信息存储位置
var ipAllocator = &IPAM{
	SubnetAllocatorPath: ipamDefaultAllocatorPath,
}

func (ipam *IPAM) load() error {
	// 检查存储文件状态，如果不存在，则说明之前没有分配，则不需要加载
	if _, err := os.Stat(ipam.SubnetAllocatorPath); err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		return nil
	}
	subnetConfigFile, err := os.Open(ipam.SubnetAllocatorPath)
	if err != nil {
		return err
	}
	defer subnetConfigFile.Close()

	subnetJson := make([]byte, 2000)
	n, err := subnetConfigFile.Read(subnetJson)
	if err != nil {
		return errors.Wrap(err, "read subnet config file error")
	}
	err = json.Unmarshal(subnetJson[:n], ipam.Subnets)

	return errors.Wrap(err, "err dump allocation info")
}

// dump 存储网段地址分配信息
func (ipam *IPAM) dump() error {
	ipamConfigFileDir, _ := path.Split(ipam.SubnetAllocatorPath)
	if _, err := os.Stat(ipamConfigFileDir); err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		if err = os.MkdirAll(ipamConfigFileDir, constant.Perm0644); err != nil {
			return err
		}
	}
	// 打开存储文件 O_TRUNC 表示如果存在则消空， os O_CREATE 表示如果不存在则创建
	subnetConfigFile, err := os.OpenFile(ipam.SubnetAllocatorPath, os.O_TRUNC|os.O_WRONLY|os.O_CREATE, constant.Perm0644)
	if err != nil {
		return err
	}
	defer subnetConfigFile.Close()
	ipamConfigJson, err := json.Marshal(ipam.Subnets)
	if err != nil {
		return err
	}
	_, err = subnetConfigFile.Write(ipamConfigJson)
	return err
}

func (ipam *IPAM) Allocate(subnet *net.IPNet) (ip net.IP, err error) {

	ipam.Subnets = &map[string]string{}
	err = ipam.load()
	if err != nil {
		return nil, errors.Wrap(err, "load subnet allocation info error")
	}

	_, subnet, _ = net.ParseCIDR(subnet.String())
	one, size := subnet.Mask.Size()
	if _, exist := (*ipam.Subnets)[subnet.String()]; !exist {
		(*ipam.Subnets)[subnet.String()] = strings.Repeat("0", 1<<uint8(size-one))
	}
	for c := range (*ipam.Subnets)[subnet.String()] {
		if (*ipam.Subnets)[subnet.String()][c] == '0' {
			// 设置这个为“0”的序号值为“1” 即标记这个IP已经分配过了
			// Go 的字符串，创建之后就不能修改 所以通过转换成 byte 数组，修改后再转换成字符串赋值
			ipAlloc := []byte((*ipam.Subnets)[subnet.String()])
			ipAlloc[c] = '1'
			(*ipam.Subnets)[subnet.String()] = string(ipAlloc)
			// 这里的 subnet.IP只是初始IP，比如对于网段192 168.0.0/16 ，这里就是192.168.0.0
			ip = subnet.IP
			/*
				还需要通过网段的IP与上面的偏移相加计算出分配的IP地址，由于IP地址是uint的一个数组，
				需要通过数组中的每一项加所需要的值，比如网段是172.16.0.0/12，数组序号是65555,
				那么在[172,16,0,0] 上依次加[uint8(65555 >> 24)、uint8(65555 >> 16)、
				uint8(65555 >> 8)、uint8(65555 >> 0)]， 即[0, 1, 0, 19]， 那么获得的IP就
				是172.17.0.19.
				偏移量为[0 1 0 19] 与原数组[172 16 0 0]相加得-> 172.17.0.19
			*/
			for t := uint(4); t > 0; t -= 1 {
				[]byte(ip)[4-t] += uint8(c >> ((t - 1) * 8))
			}
			ip[3] += 1
			break
		}
	}
	err = ipam.dump()
	if err != nil {
		log.Error("Allocate: dump ipam error", err)
	}
	return
}

func (ipam *IPAM) Release(subnet *net.IPNet, ipaddr *net.IP) error {
	ipam.Subnets = &map[string]string{}
	_, subnet, _ = net.ParseCIDR(subnet.String())
	err := ipam.load()
	if err != nil {
		return errors.Wrap(err, "load subnet allocation info error")
	}

	c := 0
	releaseIP := ipaddr.To4()
	releaseIP[3] -= 1
	for t := uint(4); t > 0; t -= 1 {
		c += int(releaseIP[t-1]-subnet.IP[t-1]) << ((4 - t) * 8)
	}
	ipAlloc := []byte((*ipam.Subnets)[subnet.String()])
	ipAlloc[c] = '0'
	(*ipam.Subnets)[subnet.String()] = string(ipAlloc)

	err = ipam.dump()
	if err != nil {
		log.Error("Allocate: dump ipam error", err)
	}
	return nil
}
