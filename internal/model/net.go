package model

import "net"

// AddrIP returns net.IP address from given net.Addr
func AddrIP(addr net.Addr) (ip net.IP) {
	if addr != nil {
		switch addr := addr.(type) {
		// case *net.UnixAddr:
		case *net.TCPAddr:
			ip = addr.IP
		case *net.UDPAddr:
			ip = addr.IP
		case *net.IPAddr:
			ip = addr.IP
			// case *net.IPNet:
			// default:
			// ip = addr.String()
		}
	}
	return // ip | nil
}
