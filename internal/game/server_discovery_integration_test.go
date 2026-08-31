package game

import (
	"net"
	"strconv"
	"testing"
	"time"
)

func TestLocalServerDiscoveryFindsRealHostedSession(t *testing.T) {
	host, err := newHostNetplay("127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer host.Close()
	tcpAddr, ok := host.listener.Addr().(*net.TCPAddr)
	if !ok {
		t.Fatalf("listener address is %T", host.listener.Addr())
	}
	advertiser, err := startLocalServerAdvertiser(tcpAddr.Port, host)
	if err != nil {
		t.Fatal(err)
	}
	defer advertiser.Close()

	servers, err := discoverLocalServers()
	if err != nil {
		t.Fatal(err)
	}
	for _, server := range servers {
		_, port, splitErr := net.SplitHostPort(server.Address)
		if splitErr == nil && port == strconv.Itoa(tcpAddr.Port) {
			return
		}
	}
	t.Fatalf("real local host on port %d was not discovered; got %+v", tcpAddr.Port, servers)
}

func TestTwoGameInstancesDiscoverJoinAndReachTwoOfTwo(t *testing.T) {
	hostGame := New()
	if err := hostGame.StartNetHost("127.0.0.1:0"); err != nil {
		t.Fatal(err)
	}
	defer hostGame.CloseNetplay()

	servers, err := discoverLocalServers()
	if err != nil {
		t.Fatal(err)
	}
	if len(servers) == 0 {
		t.Fatal("second instance found no hosted session")
	}
	var address string
	hostPort := hostGame.netplay.listener.Addr().(*net.TCPAddr).Port
	for _, server := range servers {
		_, port, splitErr := net.SplitHostPort(server.Address)
		if splitErr == nil && port == strconv.Itoa(hostPort) {
			address = server.Address
			break
		}
	}
	if address == "" {
		t.Fatalf("host port %d missing from server list: %+v", hostPort, servers)
	}

	clientGame := New()
	if err := clientGame.StartNetClient(address); err != nil {
		t.Fatal(err)
	}
	defer clientGame.CloseNetplay()
	waitFor(t, "host lobby to reach 2/2", hostGame.netplay.connected)
	if !clientGame.netplay.connected() {
		t.Fatal("client is not connected after join")
	}
}

func TestBrowserPublishesLANResultWithoutWaitingForDHT(t *testing.T) {
	hostGame := New()
	if err := hostGame.StartNetHostConfig(":0", onlineSessionScopeLAN, onlineSessionOpen); err != nil {
		t.Fatal(err)
	}
	defer hostGame.CloseNetplay()

	browser := New()
	browser.refreshMultiplayerServers()
	if browser.multiplayerDiscoveryCh == nil {
		t.Fatal("browser discovery did not start")
	}

	select {
	case result := <-browser.multiplayerDiscoveryCh:
		if result.Done {
			t.Fatal("first browser result waited for global discovery instead of publishing LAN immediately")
		}
		for _, server := range result.Servers {
			if server.ID == hostGame.netplay.publicSessionID {
				return
			}
		}
		t.Fatalf("fast browser result did not contain LAN host: %+v", result.Servers)
	case <-time.After(2 * time.Second):
		t.Fatal("LAN browser result was blocked by slow global DHT discovery")
	}
}

func TestLANBrowserEntryJoinsOpenHost(t *testing.T) {
	hostGame := New()
	if err := hostGame.StartNetHostConfig(":0", onlineSessionScopeLAN, onlineSessionOpen); err != nil {
		t.Fatal(err)
	}
	defer hostGame.CloseNetplay()

	servers, err := discoverLANServers(1200 * time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	var server multiplayerServerInfo
	for _, candidate := range servers {
		if candidate.ID == hostGame.netplay.publicSessionID {
			server = candidate
			break
		}
	}
	if server.Address == "" {
		t.Fatalf("LAN browser did not return the hosted session: %+v", servers)
	}
	clientGame := New()
	defer clientGame.CloseNetplay()
	if err := clientGame.StartNetClientServer(server); err != nil {
		t.Fatalf("joining the LAN browser entry failed: %v; server=%+v", err, server)
	}
	if clientGame.netplay == nil || clientGame.netplay.assignedPlayer() != 2 {
		t.Fatal("LAN browser join did not assign Player 2")
	}
}
