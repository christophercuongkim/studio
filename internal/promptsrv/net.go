package promptsrv

import "net"

// netInterfaceAddrs returns the non-loopback IPv4 addresses of the host, used to
// print reachable URLs for second devices (iPad over Tailscale, plan §10.2).
func netInterfaceAddrs() ([]string, error) {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return nil, err
	}
	var out []string
	for _, a := range addrs {
		ipnet, ok := a.(*net.IPNet)
		if !ok || ipnet.IP.IsLoopback() {
			continue
		}
		ip4 := ipnet.IP.To4()
		if ip4 == nil {
			continue
		}
		out = append(out, ip4.String())
	}
	return out, nil
}
