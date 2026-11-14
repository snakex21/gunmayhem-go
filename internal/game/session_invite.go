package game

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base32"
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"net/netip"
	"sort"
	"strings"
	"time"
)

const (
	onlineSessionOpen = iota
	onlineSessionInviteOnly
)

// Network reach and access policy are intentionally independent. A session can
// therefore be LAN+OPEN, LAN+INVITE, INTERNET+OPEN, or INTERNET+INVITE.
const (
	onlineSessionScopeLAN = iota
	onlineSessionScopeInternet
)

const shortOpenCodePrefix = "GM-"

var inviteCodeEncoding = base32.StdEncoding.WithPadding(base32.NoPadding)

func decentralizedOpenCode(sessionKey string) string {
	sessionKey = strings.TrimSpace(sessionKey)
	if sessionKey == "" {
		return ""
	}
	sum := sha256.Sum256([]byte("GunMayhem-RE-open-code-16|" + sessionKey))
	return inviteCodeEncoding.EncodeToString(sum[:])[:6]
}

func decentralizedOpenCodeDHTHash(code string) [20]byte {
	code = strings.ToUpper(strings.TrimSpace(code))
	code = strings.TrimPrefix(code, shortOpenCodePrefix)
	return sha1.Sum([]byte("GunMayhem-RE-open-code-dht-16|" + code))
}

func decentralizedCodeSessionKey(code string) string {
	code = strings.ToUpper(strings.TrimSpace(code))
	code = strings.TrimPrefix(code, shortOpenCodePrefix)
	sum := sha256.Sum256([]byte("GunMayhem-RE-short-code-auth-16|" + code))
	return inviteCodeEncoding.EncodeToString(sum[:16])
}

func validDecentralizedOpenCode(code string) bool {
	code = strings.ToUpper(strings.TrimSpace(code))
	if len(code) != 6 {
		return false
	}
	for _, r := range code {
		if !(r >= 'A' && r <= 'Z') && !(r >= '2' && r <= '7') {
			return false
		}
	}
	return true
}

type inviteCandidate struct {
	Addr netip.Addr
	Port uint16
}

func decodeShortOpenCode(code string) (string, error) {
	code = strings.ToUpper(strings.TrimSpace(code))
	code = strings.ReplaceAll(code, " ", "")
	if !strings.HasPrefix(code, shortOpenCodePrefix) {
		return "", errors.New("not a short Gun Mayhem server code")
	}
	encoded := strings.TrimPrefix(code, shortOpenCodePrefix)
	if len(encoded) != 10 {
		return "", errors.New("invalid short server code")
	}
	raw, err := inviteCodeEncoding.DecodeString(encoded)
	if err != nil || len(raw) != 6 {
		return "", errors.New("invalid short server code")
	}
	port := binary.BigEndian.Uint16(raw[4:])
	if port == 0 {
		return "", errors.New("invalid short server port")
	}
	addr := netip.AddrFrom4([4]byte{raw[0], raw[1], raw[2], raw[3]})
	return net.JoinHostPort(addr.String(), fmt.Sprint(port)), nil
}

func newOnlineSessionKey() (string, error) {
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", err
	}
	return inviteCodeEncoding.EncodeToString(raw[:]), nil
}

func newOnlineAuthChallenge() (string, error) {
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", err
	}
	return inviteCodeEncoding.EncodeToString(raw[:]), nil
}

func onlineSessionProof(sessionKey, challenge string) (string, error) {
	key, err := decodeOnlineSessionKey(sessionKey)
	if err != nil {
		return "", err
	}
	challenge = strings.TrimSpace(challenge)
	if challenge == "" || len(challenge) > 64 {
		return "", errors.New("invalid authentication challenge")
	}
	mac := hmac.New(sha256.New, key[:])
	_, _ = mac.Write([]byte(challenge))
	sum := mac.Sum(nil)
	return inviteCodeEncoding.EncodeToString(sum[:16]), nil
}

func onlineSessionProofEqual(sessionKey, challenge, proof string) bool {
	expected, err := onlineSessionProof(sessionKey, challenge)
	if err != nil || proof == "" || len(expected) != len(proof) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(expected), []byte(proof)) == 1
}

func decodeOnlineSessionKey(key string) ([16]byte, error) {
	var raw [16]byte
	decoded, err := inviteCodeEncoding.DecodeString(strings.ToUpper(strings.TrimSpace(key)))
	if err != nil || len(decoded) != len(raw) {
		return raw, errors.New("invalid session key")
	}
	copy(raw[:], decoded)
	return raw, nil
}

func localInviteCandidates(port int) []inviteCandidate {
	if port <= 0 || port > 65535 {
		return nil
	}
	var out []inviteCandidate
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, rawAddr := range addrs {
			var ip net.IP
			switch v := rawAddr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			addr, ok := netip.AddrFromSlice(ip)
			if !ok {
				continue
			}
			addr = addr.Unmap()
			if addr.IsLoopback() || addr.IsLinkLocalUnicast() || addr.IsUnspecified() || addr.IsMulticast() {
				continue
			}
			if !addr.IsPrivate() && !isInternetRoutableAddr(addr) {
				continue
			}
			out = append(out, inviteCandidate{Addr: addr, Port: uint16(port)})
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		return inviteCandidateScore(out[i].Addr) < inviteCandidateScore(out[j].Addr)
	})
	return uniqueInviteCandidates(out, 8)
}

func isInternetRoutableAddr(addr netip.Addr) bool {
	addr = addr.Unmap()
	if !addr.IsValid() || !addr.IsGlobalUnicast() || addr.IsPrivate() || addr.IsLoopback() || addr.IsLinkLocalUnicast() {
		return false
	}
	if addr.Is4() {
		// RFC 6598 shared address space is CGNAT, not directly reachable from
		// arbitrary Internet clients even though Go classifies it as global unicast.
		if netip.MustParsePrefix("100.64.0.0/10").Contains(addr) {
			return false
		}
	}
	return true
}

func inviteCandidateScore(addr netip.Addr) int {
	switch {
	case addr.Is4() && addr.IsPrivate():
		return 0
	case addr.Is6() && addr.IsPrivate():
		return 1
	case addr.Is6() && isInternetRoutableAddr(addr):
		return 2
	case addr.Is4() && isInternetRoutableAddr(addr):
		return 3
	default:
		return 9
	}
}

func uniqueInviteCandidates(in []inviteCandidate, maxCount int) []inviteCandidate {
	seen := make(map[string]bool)
	out := make([]inviteCandidate, 0, min(len(in), maxCount))
	for _, c := range in {
		if !c.Addr.IsValid() || c.Port == 0 {
			continue
		}
		key := net.JoinHostPort(c.Addr.String(), fmt.Sprint(c.Port))
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, c)
		if len(out) >= maxCount {
			break
		}
	}
	return out
}

func (g *Game) currentHostShortOpenCode() (string, bool) {
	// Invite-only Internet sessions encode the temporary trycloudflare hostname
	// and the session secret directly in the invite code. No central lookup and
	// no DHT record is required. LAN invite sessions keep the compact GM-XXXXXX
	// rendezvous code used by local discovery.
	if g == nil || g.netplay == nil || g.netplay.mode != netplayHost || g.onlineSessionVisibility != onlineSessionInviteOnly {
		return "", false
	}
	if g.onlineSessionScope == onlineSessionScopeInternet && g.cloudflareTunnel != nil {
		ready, _, wsURL, _ := g.cloudflareTunnel.Status()
		if !ready || wsURL == "" {
			return "", false
		}
		code, err := encodeCloudflareInviteCode(wsURL, g.onlineSessionKey)
		return code, err == nil
	}
	shortCode := strings.TrimSpace(g.netplay.shortJoinCode)
	if shortCode == "" {
		shortCode = decentralizedOpenCode(g.onlineSessionKey)
	}
	if !validDecentralizedOpenCode(shortCode) {
		return "", false
	}
	if g.onlineSessionScope == onlineSessionScopeInternet {
		if g.p2pHost == nil {
			return "", false
		}
		if _, ready, _ := g.p2pHost.Status(); !ready {
			return "", false
		}
	}
	return shortOpenCodePrefix + shortCode, true
}

func (g *Game) currentHostDisplayJoinCode() (string, bool) {
	return g.currentHostShortOpenCode()
}

func (g *Game) currentHostDisplayJoinCodeLabel() string {
	if g == nil || g.netplay == nil || g.netplay.mode != netplayHost {
		return "OFFLINE"
	}
	if g.onlineSessionVisibility == onlineSessionInviteOnly {
		if code, ready := g.currentHostDisplayJoinCode(); ready && code != "" {
			return "INVITE CODE: " + code
		}
		if g.onlineSessionScope == onlineSessionScopeInternet {
			if g.cloudflareTunnel == nil {
				return "INVITE CODE: STARTING TUNNEL..."
			}
			ready, _, _, err := g.cloudflareTunnel.Status()
			switch {
			case ready:
				return "INVITE CODE: PREPARING SESSION..."
			case err != nil:
				return "INVITE CODE: TUNNEL ERROR - " + strings.ToUpper(err.Error())
			default:
				return "INVITE CODE: " + g.cloudflareTunnel.StartupStatusText()
			}
		}
		return "INVITE CODE: PREPARING LAN SESSION..."
	}

	if g.onlineSessionScope == onlineSessionScopeInternet {
		return g.currentHostInternetReachability().openSessionLabel()
	}
	return "OPEN SESSION: VISIBLE IN LAN SERVER BROWSER"
}

func (g *Game) StartNetClientShortOpenCode(code string) error {
	normalized := strings.ToUpper(strings.ReplaceAll(strings.TrimSpace(code), " ", ""))
	encoded := strings.TrimPrefix(normalized, shortOpenCodePrefix)
	if len(encoded) == 6 {
		// Six-character GM- codes are INVITE ONLY rendezvous codes. Public OPEN
		// sessions are discovered in the server browser and are never resolved
		// by code.
		if !validDecentralizedOpenCode(encoded) {
			return errors.New("invalid invite session code")
		}
		authKey := decentralizedCodeSessionKey(encoded)
		lanServers, lanErr := discoverLANServersByCode(encoded, 900*time.Millisecond)
		networkDiagnostic("JOIN CODE LAN lookup code=%s candidates=%v err=%v", encoded, serverAddressesForDiagnostic(lanServers), lanErr)
		var failures []string
		for _, server := range lanServers {
			if err := g.startNetClientCodeServer(server, authKey); err == nil {
				return nil
			} else {
				failures = append(failures, "LAN "+server.Address+": "+err.Error())
			}
		}
		peers, dhtErr := discoverDHTPeerAddresses(decentralizedOpenCodeDHTHash(encoded), 12*time.Second)
		servers := dhtPeerServerCandidates(peers, "JOIN CODE "+encoded, "JOIN CODE")
		networkDiagnostic("JOIN CODE lookup code=%s candidates=%v err=%v", encoded, serverAddressesForDiagnostic(servers), dhtErr)
		for _, server := range servers {
			if err := g.startNetClientCodeServer(server, authKey); err == nil {
				return nil
			} else {
				failures = append(failures, server.Address+": "+err.Error())
			}
		}
		if len(failures) > 0 {
			return errors.New("session code found the host but connection failed: " + strings.Join(failures, "; "))
		}
		if dhtErr != nil {
			return fmt.Errorf("session code was not found in DHT: %w", dhtErr)
		}
		return errors.New("session code is offline or has not finished publishing")
	}
	address, err := decodeShortOpenCode(code)
	if err != nil {
		return err
	}
	host, _, splitErr := net.SplitHostPort(address)
	if splitErr == nil {
		if addr, parseErr := netip.ParseAddr(host); parseErr == nil && isInternetRoutableAddr(addr) {
			return g.StartNetClientServer(multiplayerServerInfo{
				Name: "JOIN CODE", Address: address, Transport: "quic", Sources: "JOIN CODE",
				Players: -1, MaxPlayers: maxOnlinePlayers, Protocol: netplayProtocolVersion,
			})
		}
	}
	return g.StartNetClient(address)
}

func (g *Game) startNetClientCodeServer(server multiplayerServerInfo, sessionKey string) error {
	if g == nil {
		return errors.New("nil game")
	}
	_ = g.CloseNetplay()
	if strings.EqualFold(strings.TrimSpace(server.Transport), "quic") {
		conn, err := dialP2PQUIC(server.Address, sessionKey, p2pDialTimeout)
		if err != nil {
			return err
		}
		s, err := newClientNetplayFromConnWithKey(conn, "P2P CODE "+server.Address, sessionKey)
		if err != nil {
			_ = conn.Close()
			return err
		}
		if err := waitForClientHandshake(s, 4*time.Second); err != nil {
			_ = s.Close()
			return err
		}
		g.netplay = s
		return nil
	}

	var directErr error
	if strings.TrimSpace(server.Address) != "" {
		timeout := 3 * time.Second
		if s, err := newClientNetplayWithKey(server.Address, sessionKey, timeout); err == nil {
			if err := waitForClientHandshake(s, 4*time.Second); err == nil {
				g.netplay = s
				return nil
			} else {
				directErr = err
				_ = s.Close()
			}
		} else {
			directErr = err
		}
	}

	if directErr != nil {
		return directErr
	}
	return errors.New("session has no usable network route")
}
