package game

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/huin/goupnp/dcps/internetgateway2"
)

func TestLiveUPnPMappingDiagnostic(t *testing.T) {
	if os.Getenv("GUNMAYHEM_NAT_LIVE_TEST") != "1" {
		t.Skip("set GUNMAYHEM_NAT_LIVE_TEST=1 for router mapping diagnostic")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	clients, _, err := internetgateway2.NewWANIPConnection1ClientsCtx(ctx)
	if err != nil {
		t.Logf("UPnP discovery error: %v", err)
	}
	if len(clients) == 0 {
		t.Fatal("router advertised IGD but exposed no WANIPConnection1 client")
	}
	const testPort = uint16(45777)
	for index, client := range clients {
		if client == nil || client.LocalAddr() == nil {
			continue
		}
		externalIP, externalErr := client.GetExternalIPAddressCtx(ctx)
		internalIP := client.LocalAddr().String()
		t.Logf("client=%d local=%s external=%s externalErr=%v", index, internalIP, externalIP, externalErr)
		err := client.AddPortMappingCtx(ctx, "", testPort, "TCP", testPort, internalIP, true, "Gun Mayhem diagnostic", 60)
		t.Logf("AddPortMapping TCP %d -> %s:%d: %v", testPort, internalIP, testPort, err)
		if err == nil {
			udpErr := client.AddPortMappingCtx(ctx, "", testPort, "UDP", testPort, internalIP, true, "Gun Mayhem P2P diagnostic", 60)
			t.Logf("AddPortMapping UDP %d -> %s:%d: %v", testPort, internalIP, testPort, udpErr)
			deleteCtx, deleteCancel := context.WithTimeout(context.Background(), 2*time.Second)
			_ = client.DeletePortMappingCtx(deleteCtx, "", testPort, "TCP")
			if udpErr == nil {
				_ = client.DeletePortMappingCtx(deleteCtx, "", testPort, "UDP")
			}
			deleteCancel()
			if udpErr == nil {
				return
			}
		}
	}
	t.Fatal("router rejected every UPnP mapping attempt")
}
