package game

import (
	"crypto/rand"
	"encoding/binary"
	"encoding/json"
	"errors"
	"net"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"golang.org/x/net/ipv4"
)

const (
	lanDiscoveryPort    = 7778
	lanDiscoveryTimeout = 650 * time.Millisecond
	lanDiscoveryMagic   = "GUNMAYHEM-LAN-17"
)

var lanDiscoveryGroup = &net.UDPAddr{IP: net.IPv4(239, 255, 77, 77), Port: lanDiscoveryPort}

type lanDiscoveryQuery struct {
	Magic    string `json:"magic"`
	Protocol int    `json:"protocol"`
	Nonce    uint64 `json:"nonce"`
	Code     string `json:"code,omitempty"`
}

type lanDiscoveryReply struct {
	Magic    string       `json:"magic"`
	Protocol int          `json:"protocol"`
	Nonce    uint64       `json:"nonce"`
	Port     int          `json:"port"`
	Probe    netProbeInfo `json:"probe"`
}

type lanDiscoveryResponder struct {
	closed  atomic.Bool
	conn    net.PacketConn
	packet  *ipv4.PacketConn
	tcpPort int
	session *netplaySession
	name    string
	done    chan struct{}
}

func startLANDiscoveryResponder(tcpPort int, session *netplaySession) (*lanDiscoveryResponder, error) {
	if tcpPort <= 0 || tcpPort > 65535 || session == nil {
		return nil, errors.New("invalid LAN discovery host")
	}
	conn, err := net.ListenPacket("udp4", ":7778")
	if err != nil {
		return nil, err
	}
	packet := ipv4.NewPacketConn(conn)
	joined := 0
	ifaces, _ := net.Interfaces()
	for i := range ifaces {
		iface := &ifaces[i]
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagMulticast == 0 {
			continue
		}
		if err := packet.JoinGroup(iface, lanDiscoveryGroup); err == nil {
			joined++
		}
	}
	// Multicast is optional. VPNs and some Windows network profiles reject
	// group membership, but the same socket can still answer loopback,
	// directed-broadcast, and explicit unicast discovery packets.
	_ = joined
	name := strings.TrimSpace(session.publicSessionName)
	if name == "" {
		name = "Gun Mayhem Game"
	}
	r := &lanDiscoveryResponder{
		conn:    conn,
		packet:  packet,
		tcpPort: tcpPort,
		session: session,
		name:    name,
		done:    make(chan struct{}),
	}
	go r.loop()
	return r, nil
}

func isLocalDiscoverySource(addr net.Addr) bool {
	udp, ok := addr.(*net.UDPAddr)
	if !ok || udp.IP == nil {
		return false
	}
	ip := udp.IP
	return ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast()
}

func (r *lanDiscoveryResponder) loop() {
	defer close(r.done)
	buf := make([]byte, 1200)
	for !r.closed.Load() {
		_ = r.conn.SetReadDeadline(time.Now().Add(time.Second))
		n, addr, err := r.conn.ReadFrom(buf)
		if err != nil {
			if r.closed.Load() {
				return
			}
			if ne, ok := err.(net.Error); ok && ne.Timeout() {
				continue
			}
			return
		}
		if n <= 0 || n > 1200 || !isLocalDiscoverySource(addr) {
			continue
		}
		var q lanDiscoveryQuery
		if json.Unmarshal(buf[:n], &q) != nil || q.Magic != lanDiscoveryMagic || q.Protocol != netplayProtocolVersion || q.Nonce == 0 {
			continue
		}
		queryCode := strings.ToUpper(strings.TrimSpace(q.Code))
		expectedCode := strings.ToUpper(strings.TrimSpace(r.session.shortJoinCode))
		if r.session.inviteOnly {
			if queryCode == "" || queryCode != expectedCode {
				continue
			}
		} else if queryCode != "" && queryCode != expectedCode {
			continue
		}
		probe := r.session.publicProbeInfo()
		if strings.TrimSpace(probe.Name) == "" {
			probe.Name = r.name
		}
		reply, _ := json.Marshal(lanDiscoveryReply{
			Magic: lanDiscoveryMagic, Protocol: netplayProtocolVersion, Nonce: q.Nonce,
			Port: r.tcpPort, Probe: probe,
		})
		if len(reply) > 0 && len(reply) <= 1200 {
			_, _ = r.conn.WriteTo(reply, addr)
		}
	}
}

func (r *lanDiscoveryResponder) Close() error {
	if r == nil || r.closed.Swap(true) {
		return nil
	}
	var err error
	if r.packet != nil {
		err = r.packet.Close()
	} else if r.conn != nil {
		err = r.conn.Close()
	}
	// Wait until the receive loop has actually returned and released UDP 7778.
	// Without this, immediately leaving/recreating a session (and sequential
	// tests doing the same) can race the old socket and silently lose LAN/code
	// discovery in the new host.
	select {
	case <-r.done:
	case <-time.After(500 * time.Millisecond):
	}
	return err
}

func randomLANDiscoveryNonce() uint64 {
	var raw [8]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return uint64(time.Now().UnixNano()) | 1
	}
	n := binary.BigEndian.Uint64(raw[:])
	if n == 0 {
		n = 1
	}
	return n
}

func lanInterfaceBroadcastTargets(iface *net.Interface) []*net.UDPAddr {
	if iface == nil || iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
		return nil
	}
	addrs, err := iface.Addrs()
	if err != nil {
		return nil
	}
	seen := map[string]bool{}
	var out []*net.UDPAddr
	for _, addr := range addrs {
		var ipnet *net.IPNet
		switch value := addr.(type) {
		case *net.IPNet:
			ipnet = value
		case *net.IPAddr:
			ipnet = &net.IPNet{IP: value.IP, Mask: net.CIDRMask(32, 32)}
		}
		if ipnet == nil {
			continue
		}
		ip := ipnet.IP.To4()
		ones, bits := ipnet.Mask.Size()
		if ip == nil || bits != 32 || ones < 0 || ones >= 31 || (!ip.IsPrivate() && !ip.IsLinkLocalUnicast()) {
			continue
		}
		broadcast := make(net.IP, net.IPv4len)
		for i := 0; i < net.IPv4len; i++ {
			broadcast[i] = ip[i] | ^ipnet.Mask[i]
		}
		key := broadcast.String()
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, &net.UDPAddr{IP: broadcast, Port: lanDiscoveryPort})
	}
	return out
}

func ipv4SubnetDiscoveryTargets(ipnet *net.IPNet) []*net.UDPAddr {
	if ipnet == nil {
		return nil
	}
	ip := ipnet.IP.To4()
	ones, bits := ipnet.Mask.Size()
	// Probe only normal small LANs. A /24 produces 254 tiny UDP datagrams on a
	// manual refresh; refusing broader ranges avoids turning discovery into a
	// subnet scanner on corporate/VPN networks.
	if ip == nil || bits != 32 || ones < 24 || ones >= 31 || (!ip.IsPrivate() && !ip.IsLinkLocalUnicast()) {
		return nil
	}
	ipValue := binary.BigEndian.Uint32(ip)
	maskValue := binary.BigEndian.Uint32(ipnet.Mask)
	network := ipValue & maskValue
	broadcast := network | ^maskValue
	out := make([]*net.UDPAddr, 0, int(broadcast-network-1))
	var raw [4]byte
	for value := network + 1; value < broadcast; value++ {
		binary.BigEndian.PutUint32(raw[:], value)
		targetIP := append(net.IP(nil), raw[:]...)
		out = append(out, &net.UDPAddr{IP: targetIP, Port: lanDiscoveryPort})
	}
	return out
}

func lanInterfaceUnicastTargets(iface *net.Interface) []*net.UDPAddr {
	if iface == nil || iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
		return nil
	}
	addrs, err := iface.Addrs()
	if err != nil {
		return nil
	}
	var out []*net.UDPAddr
	for _, addr := range addrs {
		ipnet, ok := addr.(*net.IPNet)
		if !ok {
			continue
		}
		out = append(out, ipv4SubnetDiscoveryTargets(ipnet)...)
	}
	return out
}

func discoverLANServers(timeout time.Duration) ([]multiplayerServerInfo, error) {
	return discoverLANServersWithCode(timeout, "")
}

func discoverLANServersByCode(code string, timeout time.Duration) ([]multiplayerServerInfo, error) {
	code = strings.ToUpper(strings.TrimSpace(code))
	if !validDecentralizedOpenCode(code) {
		return nil, errors.New("invalid LAN join code")
	}
	return discoverLANServersWithCode(timeout, code)
}

func discoverLANServersWithCode(timeout time.Duration, code string) ([]multiplayerServerInfo, error) {
	if timeout <= 0 {
		timeout = lanDiscoveryTimeout
	}
	conn, err := net.ListenPacket("udp4", ":0")
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	packet := ipv4.NewPacketConn(conn)
	defer packet.Close()

	nonce := randomLANDiscoveryNonce()
	query, _ := json.Marshal(lanDiscoveryQuery{Magic: lanDiscoveryMagic, Protocol: netplayProtocolVersion, Nonce: nonce, Code: code})
	ifaces, _ := net.Interfaces()
	sent := 0
	sentTargets := map[string]bool{}
	// Always probe the local machine. This makes two game processes discover
	// each other even when every physical/VPN interface blocks multicast.
	loopback := &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: lanDiscoveryPort}
	if _, err := conn.WriteTo(query, loopback); err == nil {
		sent++
		sentTargets[loopback.String()] = true
	}
	for i := range ifaces {
		iface := &ifaces[i]
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		if iface.Flags&net.FlagMulticast != 0 {
			if err := packet.SetMulticastInterface(iface); err == nil {
				if _, err := packet.WriteTo(query, nil, lanDiscoveryGroup); err == nil {
					sent++
				}
			}
		}
		// Commercial VPNs commonly suppress multicast or make their tunnel the
		// preferred interface. A directed broadcast follows the explicit local
		// subnet route (for example 192.168.0.255) and still reaches the host on
		// the physical Ethernet/Wi-Fi adapter.
		for _, target := range lanInterfaceBroadcastTargets(iface) {
			key := target.String()
			if sentTargets[key] {
				continue
			}
			sentTargets[key] = true
			if _, err := conn.WriteTo(query, target); err == nil {
				sent++
			}
		}
		// Some VPN clients and Windows firewall profiles suppress both multicast
		// and broadcast. Targeting each host of a small physical /24 subnet uses
		// its explicit on-link route and therefore remains independent of the VPN
		// default route. Replies are still nonce/protocol validated below.
		for _, target := range lanInterfaceUnicastTargets(iface) {
			key := target.String()
			if sentTargets[key] {
				continue
			}
			sentTargets[key] = true
			if _, err := conn.WriteTo(query, target); err == nil {
				sent++
			}
		}
	}
	if sent == 0 {
		_, _ = packet.WriteTo(query, nil, lanDiscoveryGroup)
	}

	_ = conn.SetReadDeadline(time.Now().Add(timeout))
	buf := make([]byte, 1200)
	seen := map[string]bool{}
	var out []multiplayerServerInfo
	for {
		n, addr, err := conn.ReadFrom(buf)
		if err != nil {
			if ne, ok := err.(net.Error); ok && ne.Timeout() {
				break
			}
			return out, err
		}
		if n <= 0 || n > 1200 || !isLocalDiscoverySource(addr) {
			continue
		}
		var reply lanDiscoveryReply
		if json.Unmarshal(buf[:n], &reply) != nil || reply.Magic != lanDiscoveryMagic || reply.Protocol != netplayProtocolVersion || reply.Nonce != nonce {
			continue
		}
		if reply.Port <= 0 || reply.Port > 65535 || validateNetProbeInfo(reply.Probe) != nil {
			continue
		}
		udp, ok := addr.(*net.UDPAddr)
		if !ok || udp.IP == nil {
			continue
		}
		address := net.JoinHostPort(udp.IP.String(), strconv.Itoa(reply.Port))
		if seen[address] {
			continue
		}
		seen[address] = true
		server := multiplayerServerInfo{Address: address, Transport: "lan", Sources: "LAN", Protocol: reply.Protocol}
		server = applyProbeInfoToServer(server, reply.Probe)
		if strings.TrimSpace(server.Name) == "" {
			server.Name = "Gun Mayhem Host"
		}
		if len([]rune(server.Name)) > 48 {
			server.Name = string([]rune(server.Name)[:48])
		}
		out = append(out, server)
	}
	return out, nil
}
