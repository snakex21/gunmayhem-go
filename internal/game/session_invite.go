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

const inviteCodePrefix = "GM16-"
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

type decodedInviteCode struct {
	InviteOnly bool
	SessionKey string
	Candidates []string
}

// encodeShortOpenCode packs an OPEN server's IPv4 address and network port into
// ten Base32 characters. Unlike an invite code it deliberately carries no
// session secret, so it must never be used to authorize INVITE ONLY sessions.
func encodeShortOpenCode(addr netip.Addr, port uint16) (string, error) {
	addr = addr.Unmap()
	if !addr.Is4() || port == 0 {
		return "", errors.New("short server code requires IPv4 and a TCP port")
	}
	raw := make([]byte, 6)
	ip := addr.As4()
	copy(raw[:4], ip[:])
	binary.BigEndian.PutUint16(raw[4:], port)
	return shortOpenCodePrefix + inviteCodeEncoding.EncodeToString(raw), nil
}

func decodeShortOpenCode(code string) (string, error) {
	code = strings.ToUpper(strings.TrimSpace(code))
	code = strings.ReplaceAll(code, " ", "")
	if !strings.HasPrefix(code, shortOpenCodePrefix) || strings.HasPrefix(code, inviteCodePrefix) {
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

func onlineSessionKeyEqual(a, b string) bool {
	if a == "" || b == "" || len(a) != len(b) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
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

func selectHostInviteCandidates(in []inviteCandidate) []inviteCandidate {
	in = uniqueInviteCandidates(in, 8)
	selected := make([]inviteCandidate, 0, 2)
	for _, c := range in {
		if isInternetRoutableAddr(c.Addr) {
			selected = append(selected, c)
			break
		}
	}
	for _, c := range in {
		if c.Addr.IsPrivate() {
			selected = append(selected, c)
			break
		}
	}
	if len(selected) < 2 {
		for _, c := range in {
			duplicate := false
			for _, chosen := range selected {
				if chosen == c {
					duplicate = true
					break
				}
			}
			if !duplicate {
				selected = append(selected, c)
			}
			if len(selected) == 2 {
				break
			}
		}
	}
	return uniqueInviteCandidates(selected, 2)
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

func (g *Game) currentHostInviteCode() (code string, internetReady bool, ready bool) {
	if g == nil || g.netplay == nil || g.netplay.mode != netplayHost || g.onlineSessionKey == "" {
		return "", false, false
	}
	tcpAddr, ok := g.netplay.listener.Addr().(*net.TCPAddr)
	if !ok || tcpAddr.Port <= 0 {
		return "", false, false
	}

	candidates := localInviteCandidates(tcpAddr.Port)
	mapperDone := true
	if g.natPortMapper != nil {
		done, mapped, _, externalIP, externalPort, _ := g.natPortMapper.Status()
		mapperDone = done
		if mapped && externalPort > 0 {
			if parsed, err := netip.ParseAddr(strings.TrimSpace(externalIP)); err == nil {
				parsed = parsed.Unmap()
				if isInternetRoutableAddr(parsed) {
					public := inviteCandidate{Addr: parsed, Port: uint16(externalPort)}
					candidates = append([]inviteCandidate{public}, candidates...)
					internetReady = true
				}
			}
		}
	}
	for _, c := range candidates {
		if isInternetRoutableAddr(c.Addr) {
			internetReady = true
		}
	}
	// Do not put the QUIC/DHT-observed UDP endpoint into the direct candidate
	// list. Invite candidates are attempted as TCP first. Treating that UDP port
	// as TCP could displace the valid UPnP/NAT-PMP TCP mapping and made invite
	// codes fail whenever a client's VPN blocked DHT. QUIC still remains the
	// fallback below StartNetClientInvite's direct TCP attempts.
	candidates = selectHostInviteCandidates(candidates)
	if len(candidates) == 0 {
		return "", false, mapperDone
	}
	code, err := encodeInviteCode(g.onlineSessionVisibility == onlineSessionInviteOnly, g.onlineSessionKey, candidates)
	if err != nil {
		return "", internetReady, mapperDone
	}
	_ = mapperDone // TCP mapping is optional now; the code can use DHT/QUIC P2P.
	return code, internetReady, true
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
				return "INVITE CODE: TUNNEL ERROR"
			default:
				return "INVITE CODE: STARTING TUNNEL..."
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
		// through the master server by code.
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
		if configuredMasterServerURL() != "" {
			masterServers, masterErr := fetchMasterServersByCode(encoded, 2500*time.Millisecond)
			networkDiagnostic("JOIN CODE master lookup code=%s candidates=%v err=%v", encoded, serverAddressesForDiagnostic(masterServers), masterErr)
			for _, server := range masterServers {
				if err := g.startNetClientCodeServer(server, authKey); err == nil {
					return nil
				} else {
					failures = append(failures, "MASTER "+server.Address+": "+err.Error())
				}
			}
			if masterErr != nil {
				failures = append(failures, "MASTER: "+masterErr.Error())
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
		if strings.TrimSpace(server.RelayAddress) != "" {
			timeout = 750 * time.Millisecond
		}
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

	relayID := strings.TrimSpace(server.RelayID)
	if relayID == "" {
		relayID = strings.TrimSpace(server.ID)
	}
	if strings.TrimSpace(server.RelayAddress) != "" && relayID != "" {
		conn, err := dialRelayNetplay(server.RelayAddress, relayID)
		if err == nil {
			s, sessionErr := newClientNetplayFromConnWithKey(conn, "relay code "+server.RelayAddress, sessionKey)
			if sessionErr == nil {
				if handshakeErr := waitForClientHandshake(s, 4*time.Second); handshakeErr == nil {
					g.netplay = s
					return nil
				} else {
					sessionErr = handshakeErr
					_ = s.Close()
				}
			}
			_ = conn.Close()
			err = sessionErr
		}
		if directErr != nil {
			return fmt.Errorf("direct connection failed: %v; relay failed: %w", directErr, err)
		}
		return fmt.Errorf("relay failed: %w", err)
	}
	if directErr != nil {
		return directErr
	}
	return errors.New("session has no usable network route")
}

func encodeInviteCode(inviteOnly bool, sessionKey string, candidates []inviteCandidate) (string, error) {
	key, err := decodeOnlineSessionKey(sessionKey)
	if err != nil {
		return "", err
	}
	candidates = uniqueInviteCandidates(candidates, 2)
	if len(candidates) == 0 {
		return "", errors.New("invite code has no network address")
	}
	payload := make([]byte, 0, 48)
	flags := byte(0)
	if inviteOnly {
		flags |= 1
	}
	payload = append(payload, 1, flags)
	payload = append(payload, key[:]...)
	payload = append(payload, byte(len(candidates)))
	for _, c := range candidates {
		if c.Addr.Is4() {
			payload = append(payload, 4)
			v := c.Addr.As4()
			payload = append(payload, v[:]...)
		} else if c.Addr.Is6() {
			payload = append(payload, 6)
			v := c.Addr.As16()
			payload = append(payload, v[:]...)
		} else {
			return "", errors.New("unsupported invite address")
		}
		var port [2]byte
		binary.BigEndian.PutUint16(port[:], c.Port)
		payload = append(payload, port[:]...)
	}
	return inviteCodePrefix + inviteCodeEncoding.EncodeToString(payload), nil
}

func decodeInviteCode(code string) (decodedInviteCode, error) {
	code = strings.ToUpper(strings.TrimSpace(code))
	code = strings.ReplaceAll(code, " ", "")
	if !strings.HasPrefix(code, inviteCodePrefix) {
		return decodedInviteCode{}, errors.New("not a Gun Mayhem invite code")
	}
	raw, err := inviteCodeEncoding.DecodeString(strings.TrimPrefix(code, inviteCodePrefix))
	if err != nil || len(raw) < 20 {
		return decodedInviteCode{}, errors.New("invalid invite code")
	}
	if raw[0] != 1 || raw[1]&^byte(1) != 0 {
		return decodedInviteCode{}, errors.New("unsupported invite code version")
	}
	flags := raw[1]
	var key [16]byte
	copy(key[:], raw[2:18])
	count := int(raw[18])
	if count <= 0 || count > 2 {
		return decodedInviteCode{}, errors.New("invalid invite candidate count")
	}
	pos := 19
	out := decodedInviteCode{InviteOnly: flags&1 != 0, SessionKey: inviteCodeEncoding.EncodeToString(key[:])}
	for i := 0; i < count; i++ {
		if pos >= len(raw) {
			return decodedInviteCode{}, errors.New("truncated invite code")
		}
		family := raw[pos]
		pos++
		var addr netip.Addr
		switch family {
		case 4:
			if pos+4+2 > len(raw) {
				return decodedInviteCode{}, errors.New("truncated IPv4 invite")
			}
			var v [4]byte
			copy(v[:], raw[pos:pos+4])
			addr = netip.AddrFrom4(v)
			pos += 4
		case 6:
			if pos+16+2 > len(raw) {
				return decodedInviteCode{}, errors.New("truncated IPv6 invite")
			}
			var v [16]byte
			copy(v[:], raw[pos:pos+16])
			addr = netip.AddrFrom16(v)
			pos += 16
		default:
			return decodedInviteCode{}, errors.New("invalid invite address family")
		}
		port := binary.BigEndian.Uint16(raw[pos : pos+2])
		pos += 2
		if port == 0 || !addr.IsValid() || addr.IsUnspecified() || addr.IsMulticast() {
			return decodedInviteCode{}, errors.New("invalid invite address")
		}
		out.Candidates = append(out.Candidates, net.JoinHostPort(addr.String(), fmt.Sprint(port)))
	}
	if pos != len(raw) {
		return decodedInviteCode{}, errors.New("invalid trailing invite data")
	}
	return out, nil
}
