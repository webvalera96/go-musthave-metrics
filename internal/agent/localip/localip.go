package localip

import (
	"net"
)

// Host возвращает предпочтительный IPv4 хоста для заголовка X-Real-IP (исходящий адрес).
func Host() string {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err == nil {
		defer conn.Close()
		if addr, ok := conn.LocalAddr().(*net.UDPAddr); ok {
			if v4 := addr.IP.To4(); v4 != nil {
				return v4.String()
			}
			return addr.IP.String()
		}
	}
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "127.0.0.1"
	}
	for _, a := range addrs {
		ipnet, ok := a.(*net.IPNet)
		if !ok || ipnet.IP.IsLoopback() {
			continue
		}
		if v4 := ipnet.IP.To4(); v4 != nil {
			return v4.String()
		}
	}
	return "127.0.0.1"
}
