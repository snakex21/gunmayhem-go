package game

import (
	"fmt"
	"net"
	"net/netip"
	"strings"
	"time"
)

// hostInternetReachability keeps Internet reachability separate from LAN
// discovery. LAN never depends on any of these checks.
type hostInternetReachability struct {
	DirectAddress string
	DirectMethod  string
	MappingDone   bool
	P2PReady      bool
	P2PEndpoint   string
	P2PErr        error
}

func (g *Game) currentHostInternetReachability() hostInternetReachability {
	var status hostInternetReachability
	if g == nil || g.netplay == nil || g.netplay.mode != netplayHost || g.onlineSessionScope != onlineSessionScopeInternet {
		return status
	}

	port := 0
	if tcpAddr, ok := g.netplay.listener.Addr().(*net.TCPAddr); ok && tcpAddr != nil {
		port = tcpAddr.Port
	}
	if port <= 0 || port > 65535 {
		return status
	}

	// Prefer native public IPv6. It needs no NAT traversal and is therefore the
	// cleanest direct path when both peers have IPv6 connectivity.
	if address := publicInterfaceAddress(port, true); address != "" {
		status.DirectAddress = address
		status.DirectMethod = "IPv6"
	}

	if g.natPortMapper != nil {
		done, mapped, method, externalIP, externalPort, _ := g.natPortMapper.Status()
		status.MappingDone = done
		if status.DirectAddress == "" && mapped && externalPort > 0 {
			if ip, err := netip.ParseAddr(strings.TrimSpace(externalIP)); err == nil && isInternetRoutableAddr(ip.Unmap()) {
				status.DirectAddress = net.JoinHostPort(ip.Unmap().String(), fmt.Sprint(externalPort))
				status.DirectMethod = strings.TrimSpace(method)
				if status.DirectMethod == "" {
					status.DirectMethod = "port mapping"
				}
			}
		}
	} else {
		status.MappingDone = true
	}

	// A host directly assigned a public IPv4 address does not need UPnP.
	if status.DirectAddress == "" {
		if address := publicInterfaceAddress(port, false); address != "" {
			status.DirectAddress = address
			status.DirectMethod = "public IPv4"
		}
	}

	if g.p2pHost != nil {
		status.P2PEndpoint, status.P2PReady, status.P2PErr = g.p2pHost.Status()
	}
	return status
}

func publicInterfaceAddress(port int, ipv6 bool) string {
	if port <= 0 || port > 65535 {
		return ""
	}
	for _, candidate := range localInviteCandidates(port) {
		addr := candidate.Addr.Unmap()
		if !isInternetRoutableAddr(addr) {
			continue
		}
		if ipv6 != addr.Is6() {
			continue
		}
		return net.JoinHostPort(addr.String(), fmt.Sprint(candidate.Port))
	}
	return ""
}

func (s hostInternetReachability) directReadyLabel() string {
	if strings.TrimSpace(s.DirectAddress) == "" {
		return ""
	}
	method := strings.TrimSpace(s.DirectMethod)
	if method == "" {
		method = "DIRECT"
	}
	return "INTERNET READY - DIRECT " + strings.ToUpper(method)
}

func (s hostInternetReachability) openSessionLabel() string {
	if direct := s.directReadyLabel(); direct != "" {
		return direct + " - OPEN SESSION IN SERVER BROWSER"
	}
	if s.P2PReady {
		return "P2P READY - DIRECT HOSTING UNAVAILABLE (NAT / CGNAT) - VISIBLE IN SERVER BROWSER"
	}
	if !s.MappingDone && s.P2PErr == nil {
		return "CHECKING INTERNET REACHABILITY..."
	}
	if s.P2PErr != nil {
		return "DIRECT HOSTING UNAVAILABLE (NAT / CGNAT) - P2P RETRYING..."
	}
	return "P2P STARTING - DIRECT HOSTING UNAVAILABLE - PUBLISHING RENDEZVOUS..."
}

func directServerAddresses(server multiplayerServerInfo) []string {
	seen := make(map[string]bool)
	out := make([]string, 0, 2)
	for _, address := range []string{server.DirectAddress, server.Address} {
		address = strings.TrimSpace(address)
		if address == "" || seen[address] {
			continue
		}
		seen[address] = true
		out = append(out, address)
	}
	return out
}

func tryDirectClientSession(address, sessionKey string, timeout time.Duration) (*netplaySession, error) {
	address = strings.TrimSpace(address)
	if address == "" {
		return nil, fmt.Errorf("empty direct address")
	}
	s, err := newClientNetplayWithKey(address, sessionKey, timeout)
	if err != nil {
		return nil, err
	}
	if err := waitForClientHandshake(s, 2*time.Second); err != nil {
		_ = s.Close()
		return nil, err
	}
	return s, nil
}
