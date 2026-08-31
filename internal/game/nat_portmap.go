package game

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/huin/goupnp/dcps/internetgateway2"
	"github.com/jackpal/gateway"
	natpmp "github.com/jackpal/go-nat-pmp"
)

const (
	natUPnPTimeout      = 4 * time.Second
	natPMPTimeout       = 2 * time.Second
	natDiscoveryTimeout = natUPnPTimeout + natPMPTimeout
	natCleanupTimeout   = 2 * time.Second
	natPMPLifetime      = 2 * 60 * 60 // seconds
)

type natPortMapper struct {
	internalPort int

	closed atomic.Bool
	done   chan struct{}

	mu           sync.RWMutex
	method       string
	externalIP   string
	externalPort int
	lastErr      error
	mapped       bool
	cleanup      func()
}

func startAutomaticPortMapping(internalPort int) *natPortMapper {
	m := &natPortMapper{internalPort: internalPort, done: make(chan struct{})}
	go m.run()
	return m
}

func (m *natPortMapper) run() {
	defer close(m.done)
	if m == nil || m.internalPort <= 0 || m.internalPort > 65535 {
		if m != nil {
			m.setResult("", "", 0, errors.New("invalid local port"), nil)
		}
		return
	}

	upnpCtx, upnpCancel := context.WithTimeout(context.Background(), natUPnPTimeout)
	method, externalIP, externalPort, cleanup, upnpErr := mapUPnP(upnpCtx, m.internalPort)
	upnpCancel()
	if upnpErr == nil {
		m.setResult(method, externalIP, externalPort, nil, cleanup)
		return
	}
	if m.closed.Load() {
		return
	}

	pmpCtx, pmpCancel := context.WithTimeout(context.Background(), natPMPTimeout)
	method, externalIP, externalPort, cleanup, pmpErr := mapNATPMP(pmpCtx, m.internalPort)
	pmpCancel()
	if pmpErr == nil {
		m.setResult(method, externalIP, externalPort, nil, cleanup)
		return
	}
	m.setResult("", "", 0, fmt.Errorf("automatic port mapping failed: UPnP: %v; NAT-PMP: %w", upnpErr, pmpErr), nil)
}

func (m *natPortMapper) setResult(method, externalIP string, externalPort int, err error, cleanup func()) {
	if m == nil {
		return
	}
	m.mu.Lock()
	m.method = method
	m.externalIP = externalIP
	m.externalPort = externalPort
	m.lastErr = err
	m.mapped = err == nil && externalPort > 0
	m.cleanup = cleanup
	m.mu.Unlock()
	networkDiagnostic("NAT mapping internal=%d mapped=%v method=%q external=%s:%d err=%v", m.internalPort, err == nil && externalPort > 0, method, externalIP, externalPort, err)
	if m.closed.Load() && cleanup != nil {
		cleanup()
	}
}

func (m *natPortMapper) Status() (done, mapped bool, method, externalIP string, externalPort int, err error) {
	if m == nil {
		return true, false, "", "", 0, nil
	}
	select {
	case <-m.done:
		done = true
	default:
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	return done, m.mapped, m.method, m.externalIP, m.externalPort, m.lastErr
}

func (m *natPortMapper) Wait(timeout time.Duration) {
	if m == nil {
		return
	}
	if timeout <= 0 {
		<-m.done
		return
	}
	select {
	case <-m.done:
	case <-time.After(timeout):
	}
}

func (m *natPortMapper) AdvertisedPort(fallback int) int {
	if m == nil {
		return fallback
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.mapped && m.externalPort > 0 {
		return m.externalPort
	}
	return fallback
}

func (m *natPortMapper) Close() error {
	if m == nil || m.closed.Swap(true) {
		return nil
	}
	m.mu.RLock()
	cleanup := m.cleanup
	m.mu.RUnlock()
	if cleanup != nil {
		cleanup()
	}
	return nil
}

func mapUPnP(ctx context.Context, internalPort int) (string, string, int, func(), error) {
	if internalPort <= 0 || internalPort > 65535 {
		return "", "", 0, nil, errors.New("invalid UPnP port")
	}
	p := uint16(internalPort)

	// Prefer IGD v2 WANIPConnection2 because it supports AddAnyPortMapping.
	clients2, _, _ := internetgateway2.NewWANIPConnection2ClientsCtx(ctx)
	for _, c := range clients2 {
		if c == nil || c.LocalAddr() == nil {
			continue
		}
		internalIP := c.LocalAddr().String()
		reserved := p
		err := c.AddPortMappingCtx(ctx, "", p, "TCP", p, internalIP, true, "Gun Mayhem", 0)
		if err != nil {
			if any, anyErr := c.AddAnyPortMappingCtx(ctx, "", p, "TCP", p, internalIP, true, "Gun Mayhem", 0); anyErr == nil {
				reserved = any
				err = nil
			}
		}
		if err != nil {
			continue
		}
		udpMapped := c.AddPortMappingCtx(ctx, "", reserved, "UDP", p, internalIP, true, "Gun Mayhem P2P", 0) == nil
		externalIP, _ := c.GetExternalIPAddressCtx(ctx)
		cleanup := func() {
			cleanupCtx, cancel := context.WithTimeout(context.Background(), natCleanupTimeout)
			defer cancel()
			_ = c.DeletePortMappingCtx(cleanupCtx, "", reserved, "TCP")
			if udpMapped {
				_ = c.DeletePortMappingCtx(cleanupCtx, "", reserved, "UDP")
			}
		}
		return "UPnP IGD2", externalIP, int(reserved), cleanup, nil
	}

	clients1, _, _ := internetgateway2.NewWANIPConnection1ClientsCtx(ctx)
	for _, c := range clients1 {
		if c == nil || c.LocalAddr() == nil {
			continue
		}
		internalIP := c.LocalAddr().String()
		if err := c.AddPortMappingCtx(ctx, "", p, "TCP", p, internalIP, true, "Gun Mayhem", 0); err != nil {
			continue
		}
		udpMapped := c.AddPortMappingCtx(ctx, "", p, "UDP", p, internalIP, true, "Gun Mayhem P2P", 0) == nil
		externalIP, _ := c.GetExternalIPAddressCtx(ctx)
		cleanup := func() {
			cleanupCtx, cancel := context.WithTimeout(context.Background(), natCleanupTimeout)
			defer cancel()
			_ = c.DeletePortMappingCtx(cleanupCtx, "", p, "TCP")
			if udpMapped {
				_ = c.DeletePortMappingCtx(cleanupCtx, "", p, "UDP")
			}
		}
		return "UPnP IGD1", externalIP, internalPort, cleanup, nil
	}

	pppClients, _, _ := internetgateway2.NewWANPPPConnection1ClientsCtx(ctx)
	for _, c := range pppClients {
		if c == nil || c.LocalAddr() == nil {
			continue
		}
		internalIP := c.LocalAddr().String()
		if err := c.AddPortMappingCtx(ctx, "", p, "TCP", p, internalIP, true, "Gun Mayhem", 0); err != nil {
			continue
		}
		udpMapped := c.AddPortMappingCtx(ctx, "", p, "UDP", p, internalIP, true, "Gun Mayhem P2P", 0) == nil
		externalIP, _ := c.GetExternalIPAddressCtx(ctx)
		cleanup := func() {
			cleanupCtx, cancel := context.WithTimeout(context.Background(), natCleanupTimeout)
			defer cancel()
			_ = c.DeletePortMappingCtx(cleanupCtx, "", p, "TCP")
			if udpMapped {
				_ = c.DeletePortMappingCtx(cleanupCtx, "", p, "UDP")
			}
		}
		return "UPnP PPP", externalIP, internalPort, cleanup, nil
	}
	return "", "", 0, nil, errors.New("UPnP port mapping unavailable")
}

func mapNATPMP(ctx context.Context, internalPort int) (string, string, int, func(), error) {
	gw, err := gateway.DiscoverGateway()
	if err != nil || gw == nil {
		return "", "", 0, nil, fmt.Errorf("NAT-PMP gateway unavailable: %w", err)
	}
	remaining := natDiscoveryTimeout
	if deadline, ok := ctx.Deadline(); ok {
		remaining = time.Until(deadline)
		if remaining <= 0 {
			return "", "", 0, nil, context.DeadlineExceeded
		}
	}
	client := natpmp.NewClientWithTimeout(gw, remaining)
	result, err := client.AddPortMapping("tcp", internalPort, internalPort, natPMPLifetime)
	if err != nil {
		return "", "", 0, nil, fmt.Errorf("NAT-PMP mapping failed: %w", err)
	}
	externalPort := int(result.MappedExternalPort)
	if externalPort <= 0 {
		return "", "", 0, nil, errors.New("NAT-PMP returned invalid external port")
	}
	externalIP := ""
	if addr, addrErr := client.GetExternalAddress(); addrErr == nil {
		externalIP = net.IP(addr.ExternalIPAddress[:]).String()
	}
	udpMapped := false
	if udpResult, udpErr := client.AddPortMapping("udp", internalPort, externalPort, natPMPLifetime); udpErr == nil && int(udpResult.MappedExternalPort) == externalPort {
		udpMapped = true
	}
	cleanup := func() {
		cleanupClient := natpmp.NewClientWithTimeout(gw, natCleanupTimeout)
		_, _ = cleanupClient.AddPortMapping("tcp", internalPort, externalPort, 0)
		if udpMapped {
			_, _ = cleanupClient.AddPortMapping("udp", internalPort, externalPort, 0)
		}
	}
	return "NAT-PMP", externalIP, externalPort, cleanup, nil
}
