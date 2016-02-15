package main

import (
	"net"
	"strings"
)

func getIPAddrDebugString() string {
	// Get IP address for logging purposes; TODO: make this suck less
	unique := make(map[string]net.IP)
	var IPs []string
	ifaces, err := net.Interfaces() // net.InterfaceByName("en0") //
	if err != nil {
		panic(err)
	}
	for _, i := range ifaces {
		if strings.HasPrefix(i.Name, "en") {
			addrs, err := i.Addrs()
			if err != nil {
				continue
			}
			for _, addr := range addrs {
				var ip net.IP
				switch v := addr.(type) {
				case *net.IPNet:
					ip = v.IP
				case *net.IPAddr:
					ip = v.IP
				}
				// process IP address
				ip = ip.To4()
				if _, ok := unique[ip.String()]; ok || ip == nil {
					continue
				}
				unique[ip.String()] = ip
				IPs = append(IPs, i.Name+" "+i.HardwareAddr.String()+" "+ip.String())
			}
		}
	}
	return strings.Join(IPs, ",")
}
