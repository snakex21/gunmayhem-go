package game

import (
	"net"
	"strconv"
	"strings"
	"testing"
)

func TestLANDiscoveryFindsOpenHost(t *testing.T) {
	host, err := newHostNetplay("127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer host.Close()
	tcpAddr := host.listener.Addr().(*net.TCPAddr)
	responder, err := startLANDiscoveryResponder(tcpAddr.Port, host)
	if err != nil {
		t.Skipf("LAN multicast unavailable on this machine: %v", err)
	}
	defer responder.Close()

	servers, err := discoverLANServers(lanDiscoveryTimeout)
	if err != nil {
		t.Fatal(err)
	}
	wantPort := ":" + strconv.Itoa(tcpAddr.Port)
	for _, server := range servers {
		if strings.HasSuffix(server.Address, wantPort) && server.Protocol == netplayProtocolVersion {
			return
		}
	}
	t.Fatalf("open host was not found by LAN discovery; servers=%+v", servers)
}

func TestLANDiscoveryBuildsDirectedBroadcastForPrivateSubnet(t *testing.T) {
	iface := &net.Interface{Flags: net.FlagUp}
	// net.Interface.Addrs cannot be injected, so validate the byte-level helper
	// through the current machine's physical private interface. This test is
	// skipped only on builders without any IPv4 LAN adapter.
	ifaces, err := net.Interfaces()
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for i := range ifaces {
		iface = &ifaces[i]
		for _, target := range lanInterfaceBroadcastTargets(iface) {
			if target.Port != lanDiscoveryPort || target.IP == nil || target.IP.To4() == nil {
				t.Fatalf("invalid broadcast target for %s: %+v", iface.Name, target)
			}
			found = true
		}
	}
	if !found {
		t.Skip("no private IPv4 LAN adapter on this machine")
	}
}

func TestIPv4SubnetDiscoveryTargetsStayInsideSmallPrivateLAN(t *testing.T) {
	targets := ipv4SubnetDiscoveryTargets(&net.IPNet{IP: net.IPv4(192, 168, 50, 42), Mask: net.CIDRMask(24, 32)})
	if len(targets) != 254 {
		t.Fatalf("/24 discovery targets=%d want254", len(targets))
	}
	if got := targets[0].String(); got != "192.168.50.1:"+strconv.Itoa(lanDiscoveryPort) {
		t.Fatalf("first discovery target=%s", got)
	}
	if got := targets[len(targets)-1].String(); got != "192.168.50.254:"+strconv.Itoa(lanDiscoveryPort) {
		t.Fatalf("last discovery target=%s", got)
	}
	if targets := ipv4SubnetDiscoveryTargets(&net.IPNet{IP: net.IPv4(10, 20, 30, 40), Mask: net.CIDRMask(16, 32)}); len(targets) != 0 {
		t.Fatalf("broad /16 unexpectedly produced %d probe targets", len(targets))
	}
	if targets := ipv4SubnetDiscoveryTargets(&net.IPNet{IP: net.ParseIP("8.8.8.8"), Mask: net.CIDRMask(24, 32)}); len(targets) != 0 {
		t.Fatalf("public subnet unexpectedly produced %d probe targets", len(targets))
	}
}
