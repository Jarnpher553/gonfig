package addr

import (
	"fmt"
	"net"
	"strings"
)

var (
	privateBlocks []*net.IPNet
)

func init() {
	for _, b := range []string{"10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16", "100.64.0.0/10"} {
		if _, block, err := net.ParseCIDR(b); err == nil {
			privateBlocks = append(privateBlocks, block)
		}
	}
}

func isPrivateIP(ipAddr string) bool {
	ip := net.ParseIP(ipAddr)
	for _, priv := range privateBlocks {
		if priv.Contains(ip) {
			return true
		}
	}
	return false
}

//ParseIP parse address
func ParseIP(addr string) (string, error) {
	addrSplit := strings.Split(addr, ":")

	if addrSplit[0] != "" && addrSplit[0] != "0.0.0.0" && addrSplit[0] != "[::]" {
		return addrSplit[0], nil
	}

	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "", fmt.Errorf("failed to get interface addresses err: %v", err)
	}

	var ipAddr []byte
	var publicIP []byte

	for _, rawAddr := range addrs {
		var ip net.IP
		switch addr := rawAddr.(type) {
		case *net.IPAddr:
			ip = addr.IP
		case *net.IPNet:
			ip = addr.IP
		default:
			continue
		}

		if ip.To4() == nil {
			continue
		}

		if !isPrivateIP(ip.String()) {
			publicIP = ip
			continue
		}

		ipAddr = ip
		break
	}

	if ipAddr != nil {
		return net.IP(ipAddr).String(), nil
	}

	if publicIP != nil {
		return net.IP(publicIP).String(), nil
	}

	return "", fmt.Errorf("no ip address found, and explicit ip not provided")
}
