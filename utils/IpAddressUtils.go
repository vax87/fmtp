package utils

import (
	"net"
)

// GetLocalIpv4List предоставляет список лакальных адресов IpV4
func GetLocalIpv4List() []string {
	var retValue []string
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return retValue
	}
	for _, address := range addrs {
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				retValue = append(retValue, ipnet.IP.String())
			}
		}
	}
	return retValue
}
