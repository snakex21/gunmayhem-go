package game

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/fxamacker/cbor/v2"
)

const (
	// A client only ever needs to send hello, input and a small player config.
	// Keeping this tiny makes hostile allocation attempts cheap to reject.
	maxClientWireFrame = 16 * 1024

	// Host snapshots are larger. Four MiB is intentionally generous for current
	// gameplay while still preventing unbounded allocations from an untrusted host.
	maxServerWireFrame = 4 * 1024 * 1024

	netHandshakeTimeout     = 5 * time.Second
	netHeartbeatInterval    = time.Second
	netPeerIdleTimeout      = 10 * time.Second
	netInputMaxAge          = 500 * time.Millisecond
	netWriteTimeout         = 5 * time.Second
	maxClientMessagesPS     = 240
	maxPendingNetHandshakes = 16
)

var (
	netCBORDec = mustNetCBORDecMode()
	netCBOREnc = mustNetCBOREncMode()
)

func mustNetCBORDecMode() cbor.DecMode {
	mode, err := (cbor.DecOptions{
		DupMapKey:         cbor.DupMapKeyEnforcedAPF,
		MaxNestedLevels:   16,
		MaxArrayElements:  16384,
		MaxMapPairs:       256,
		IndefLength:       cbor.IndefLengthForbidden,
		TagsMd:            cbor.TagsForbidden,
		ExtraReturnErrors: cbor.ExtraDecErrorUnknownField,
		UTF8:              cbor.UTF8RejectInvalid,
		FieldNameMatching: cbor.FieldNameMatchingCaseSensitive,
		NaN:               cbor.NaNDecodeForbidden,
		Inf:               cbor.InfDecodeForbidden,
		BinaryUnmarshaler: cbor.BinaryUnmarshalerNone,
		TextUnmarshaler:   cbor.TextUnmarshalerNone,
	}).DecMode()
	if err != nil {
		panic(err)
	}
	return mode
}

func mustNetCBOREncMode() cbor.EncMode {
	mode, err := (cbor.EncOptions{}).EncMode()
	if err != nil {
		panic(err)
	}
	return mode
}

func writeNetMessage(conn net.Conn, msg netWireMessage, maxFrame int) error {
	return writeNetFrame(conn, cborNetplayCodec{}, msg, maxFrame)
}

func writeNetFrame(conn net.Conn, codec netplayCodec, msg netWireMessage, maxFrame int) error {
	if conn == nil || codec == nil {
		return errors.New("invalid netplay writer")
	}
	payload, err := codec.Encode(msg)
	if err != nil {
		return err
	}
	if len(payload) == 0 || len(payload) > maxFrame {
		return fmt.Errorf("netplay frame too large: %d bytes", len(payload))
	}
	var header [4]byte
	binary.BigEndian.PutUint32(header[:], uint32(len(payload)))
	_ = conn.SetWriteDeadline(time.Now().Add(netWriteTimeout))
	if err := writeAll(conn, header[:]); err != nil {
		return err
	}
	if err := writeAll(conn, payload); err != nil {
		return err
	}
	_ = conn.SetWriteDeadline(time.Time{})
	return nil
}

func readNetMessage(conn net.Conn, maxFrame int, msg *netWireMessage) error {
	return readNetFrame(conn, cborNetplayCodec{}, maxFrame, msg)
}

func readNetFrame(conn net.Conn, codec netplayCodec, maxFrame int, msg *netWireMessage) error {
	if conn == nil || codec == nil || msg == nil {
		return errors.New("invalid netplay read")
	}
	var header [4]byte
	if _, err := io.ReadFull(conn, header[:]); err != nil {
		return err
	}
	n := int(binary.BigEndian.Uint32(header[:]))
	if n <= 0 || n > maxFrame {
		return fmt.Errorf("invalid netplay frame size: %d", n)
	}
	payload := make([]byte, n)
	if _, err := io.ReadFull(conn, payload); err != nil {
		return err
	}
	return codec.Decode(payload, msg)
}

func writeAll(w io.Writer, data []byte) error {
	for len(data) > 0 {
		n, err := w.Write(data)
		if err != nil {
			return err
		}
		if n <= 0 {
			return io.ErrUnexpectedEOF
		}
		data = data[n:]
	}
	return nil
}

func sanitizeRemotePlayerConfig(cfg CustomPlayerConfig) (CustomPlayerConfig, error) {
	cfg.Name = sanitizeRemotePlayerName(cfg.Name)
	if cfg.Name == "" {
		cfg.Name = "Player"
	}
	if cfg.Color < 1 || cfg.Color > maxDevelopedColorValue() {
		return CustomPlayerConfig{}, errors.New("invalid player color")
	}
	if cfg.Shirt < 1 || cfg.Shirt > maxDevelopedShirtValue() {
		return CustomPlayerConfig{}, errors.New("invalid player shirt")
	}
	if cfg.Hat < 1 || cfg.Hat > maxDevelopedHatValue() {
		return CustomPlayerConfig{}, errors.New("invalid player hat")
	}
	if cfg.Eyes < 1 || cfg.Eyes > maxDevelopedEyeValue() {
		return CustomPlayerConfig{}, errors.New("invalid player eyes")
	}
	if !isStarterHandgunNumber(cfg.Gun) {
		return CustomPlayerConfig{}, errors.New("invalid starter weapon")
	}
	if cfg.Perk < 1 || cfg.Perk > 9 {
		return CustomPlayerConfig{}, errors.New("invalid player perk")
	}
	if cfg.Throwable < 0 || cfg.Throwable >= throwableTypeCount {
		return CustomPlayerConfig{}, errors.New("invalid throwable")
	}
	if cfg.Team != 1 && cfg.Team != 2 {
		return CustomPlayerConfig{}, errors.New("invalid player team")
	}
	cfg.Type = 1
	return cfg, nil
}

func sanitizeRemotePlayerName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return ""
	}
	var b strings.Builder
	b.Grow(min(len(name), 64))
	runes := 0
	for _, r := range name {
		if runes >= 24 {
			break
		}
		if r == utf8.RuneError || unicode.IsControl(r) {
			continue
		}
		b.WriteRune(r)
		runes++
	}
	return strings.TrimSpace(b.String())
}

type netRateGuard struct {
	window time.Time
	count  int
}

func (g *netRateGuard) Allow(now time.Time) bool {
	if g.window.IsZero() || now.Sub(g.window) >= time.Second {
		g.window = now
		g.count = 0
	}
	g.count++
	return g.count <= maxClientMessagesPS
}

func emptyNetProbeInfo(p netProbeInfo) bool {
	return p.ID == "" && p.Name == "" && p.Players == 0 && p.MaxPlayers == 0 && p.GameMode == 0 && p.Lives == 0 && p.Rounds == 0 && p.Maps == [customSeriesMaxRounds]int{} && !p.PlaylistConfigured && !p.Crates && !p.Pickups && p.RuleSummary == "" && !p.ChaosEnabled && p.ChaosEvents == 0 && p.ChaosIntervalSeconds == 0 && p.DirectAddress == "" && p.TunnelAddress == ""
}

func validateNetProbeInfo(p netProbeInfo) error {
	if len(p.ID) > 64 || !utf8.ValidString(p.ID) {
		return errors.New("invalid probe server id")
	}
	if len(p.Name) > 96 || !utf8.ValidString(p.Name) {
		return errors.New("invalid probe server name")
	}
	if len(p.DirectAddress) > 128 || !utf8.ValidString(p.DirectAddress) {
		return errors.New("invalid probe direct address")
	}
	if len(p.TunnelAddress) > cloudflareTunnelURLMaxLen || !utf8.ValidString(p.TunnelAddress) {
		return errors.New("invalid probe tunnel address")
	}
	if p.TunnelAddress != "" {
		if _, err := normalizeCloudflareTunnelURL(p.TunnelAddress); err != nil {
			return errors.New("invalid probe tunnel address")
		}
	}
	if p.DirectAddress != "" {
		host, port, err := net.SplitHostPort(strings.TrimSpace(p.DirectAddress))
		if err != nil || strings.TrimSpace(host) == "" || strings.TrimSpace(port) == "" {
			return errors.New("invalid probe direct address")
		}
	}
	if p.Players < 1 || p.MaxPlayers < 1 || p.MaxPlayers > maxOnlinePlayers || p.Players > p.MaxPlayers {
		return errors.New("invalid probe player count")
	}
	// Low-level/legacy test hosts may expose only identity and player counts.
	// GameMode==0 means "no graphical match preview metadata available".
	if p.GameMode == 0 {
		if p.Lives != 0 || p.Rounds != 0 || p.Maps != [customSeriesMaxRounds]int{} || p.PlaylistConfigured || p.Crates || p.Pickups || p.RuleSummary != "" || p.ChaosEnabled || p.ChaosEvents != 0 || p.ChaosIntervalSeconds != 0 || p.DirectAddress != "" || p.TunnelAddress != "" {
			return errors.New("partial probe preview metadata")
		}
		return nil
	}
	if p.GameMode < SourceGameModeNormal || p.GameMode > SourceGameModeSurvival {
		return errors.New("invalid probe game mode")
	}
	if p.Lives < 0 || p.Lives > 99999 || p.Rounds < customSeriesMinRounds || p.Rounds > customSeriesMaxRounds {
		return errors.New("invalid probe match settings")
	}
	if len(p.RuleSummary) > 128 || !utf8.ValidString(p.RuleSummary) {
		return errors.New("invalid probe rules summary")
	}
	if p.ChaosEvents < 0 || p.ChaosEvents >= chaosEffectCount || p.ChaosIntervalSeconds < 0 || p.ChaosIntervalSeconds > 90 {
		return errors.New("invalid probe chaos settings")
	}
	for i := 0; i < p.Rounds; i++ {
		m := p.Maps[i]
		if m != 0 && m != customGM2RandomNew && m != customGM2RandomAll && (m < 1 || (m > 12 && !isGM2MapID(m))) {
			return errors.New("invalid probe playlist map")
		}
	}
	return nil
}

func validateNetMessageForHost(msg netWireMessage) error {
	if msg.Protocol != netplayProtocolVersion {
		return fmt.Errorf("netplay protocol mismatch: remote=%d local=%d", msg.Protocol, netplayProtocolVersion)
	}
	switch msg.Type {
	case netMessageHello:
		if msg.Challenge != "" || len(msg.AuthProof) > 64 || !emptyNetProbeInfo(msg.Probe) {
			return errors.New("invalid client authentication response")
		}
		return nil
	case netMessageProbe:
		if msg.Challenge != "" || msg.AuthProof != "" || !emptyNetProbeInfo(msg.Probe) {
			return errors.New("invalid public server probe")
		}
		return nil
	case netMessageInput, netMessageKeepAlive:
		if msg.Challenge != "" || msg.AuthProof != "" || !emptyNetProbeInfo(msg.Probe) {
			return errors.New("authentication fields are only allowed during handshake")
		}
		return nil
	case netMessagePlayerConfig:
		if msg.Challenge != "" || msg.AuthProof != "" || !emptyNetProbeInfo(msg.Probe) {
			return errors.New("authentication fields are only allowed during handshake")
		}
		_, err := sanitizeRemotePlayerConfig(msg.Config)
		return err
	case netMessagePostGameReady:
		if msg.Challenge != "" || msg.AuthProof != "" || !emptyNetProbeInfo(msg.Probe) {
			return errors.New("authentication fields are only allowed during handshake")
		}
		return nil
	default:
		return errors.New("client sent forbidden netplay message type")
	}
}

func validateNetMessageForClient(msg netWireMessage) error {
	if msg.Protocol != netplayProtocolVersion {
		return fmt.Errorf("netplay protocol mismatch: remote=%d local=%d", msg.Protocol, netplayProtocolVersion)
	}
	switch msg.Type {
	case netMessageChallenge:
		if msg.AuthProof != "" || len(msg.Challenge) < 16 || len(msg.Challenge) > 64 || !emptyNetProbeInfo(msg.Probe) {
			return errors.New("invalid host authentication challenge")
		}
		return nil
	case netMessageHello, netMessageState, netMessageKeepAlive:
		if msg.Challenge != "" || msg.AuthProof != "" || !emptyNetProbeInfo(msg.Probe) {
			return errors.New("unexpected host authentication fields")
		}
		return nil
	case netMessageProbeReply:
		if msg.Challenge != "" || msg.AuthProof != "" {
			return errors.New("unexpected probe authentication fields")
		}
		return validateNetProbeInfo(msg.Probe)
	default:
		return errors.New("host sent forbidden netplay message type")
	}
}

func validateNetSnapshot(s netSnapshot) error {
	// These are intentionally above normal gameplay maxima. Their purpose is to
	// make a hostile server bounded, not to constrain legitimate effects.
	limits := []struct {
		name string
		got  int
		max  int
	}{
		{"players", len(s.Players), maxOnlinePlayers},
		{"map platforms", len(s.Arena.Platforms), 128},
		{"bullets", len(s.Bullets), 1024},
		{"instagib trails", len(s.InstagibTrails), 512},
		{"player trails", len(s.PlayerTrails), 512},
		{"shells", len(s.Shells), 1024},
		{"muzzle flashes", len(s.Flashes), 512},
		{"grenades", len(s.Grenades), 256},
		{"fire patches", len(s.FirePatches), 512},
		{"portals", len(s.Portals), 128},
		{"dynamite fx", len(s.DynamiteFX), 512},
		{"blast fx", len(s.BlastFX), 512},
		{"jet fx", len(s.JetThrustFX), 512},
		{"drop fx", len(s.DropPackFX), 256},
		{"explosions", len(s.Explosions), 512},
		{"killfeeds", len(s.Killfeeds), 128},
		{"crates", len(s.Crates), 256},
		{"powerups", len(s.Powerups), 256},
		{"powerup names", len(s.PowerupNameFX), 512},
		{"life fx", len(s.LifeBlingFX), 256},
		{"sfx events", len(s.SFXEvents), 512},
	}
	for _, limit := range limits {
		if limit.got > limit.max {
			return fmt.Errorf("host snapshot %s exceeds safe limit: %d > %d", limit.name, limit.got, limit.max)
		}
	}
	for _, p := range s.Players {
		if len(p.Name) > 96 || !utf8.ValidString(p.Name) {
			return errors.New("invalid player name in host snapshot")
		}
	}
	for _, ev := range s.SFXEvents {
		if len(ev.Name) > 96 || !utf8.ValidString(ev.Name) {
			return errors.New("invalid sound event in host snapshot")
		}
	}
	for _, cfg := range s.OnlineCustomPlayers {
		if len(cfg.Name) > 96 || !utf8.ValidString(cfg.Name) {
			return errors.New("invalid online player config in host snapshot")
		}
	}
	if s.CustomSeriesRounds != 0 && (s.CustomSeriesRounds < customSeriesMinRounds || s.CustomSeriesRounds > customSeriesMaxRounds) {
		return errors.New("invalid custom series length in host snapshot")
	}
	if s.CustomSeriesRound < 0 || (s.CustomSeriesRounds == 0 && s.CustomSeriesRound != 0) || (s.CustomSeriesRounds > 0 && s.CustomSeriesRound > s.CustomSeriesRounds) {
		return errors.New("invalid custom series round in host snapshot")
	}
	for _, mapValue := range s.CustomSeriesMaps {
		valid := mapValue == 0 || mapValue == customGM2RandomNew || mapValue == customGM2RandomAll || (mapValue >= 1 && mapValue <= 12) || isGM2MapID(mapValue)
		if !valid {
			return errors.New("invalid custom playlist map in host snapshot")
		}
	}
	if s.ChaosCurrentEffect < chaosEffectNone || s.ChaosCurrentEffect >= chaosEffectCount {
		return errors.New("invalid chaos effect in host snapshot")
	}
	if s.ChaosIntervalSeconds != 0 && (s.ChaosIntervalSeconds < 10 || s.ChaosIntervalSeconds > 90) {
		return errors.New("invalid chaos interval in host snapshot")
	}
	if s.ChaosFrame < 0 || (s.ChaosIntervalSeconds > 0 && s.ChaosFrame > s.ChaosIntervalSeconds*60+60) {
		return errors.New("invalid chaos timer in host snapshot")
	}
	if s.ChaosBannerFrame < 0 || s.ChaosBannerFrame > 180 {
		return errors.New("invalid chaos banner timer in host snapshot")
	}
	for _, stat := range s.CustomSeriesStats {
		if len(stat.Name) > 96 || !utf8.ValidString(stat.Name) {
			return errors.New("invalid series player name in host snapshot")
		}
		if stat.MapWins < 0 || stat.Kills < 0 || stat.Deaths < 0 || stat.ShotsFired < 0 || stat.HitsLanded < 0 || stat.RoundsPlayed < 0 {
			return errors.New("invalid negative series statistic in host snapshot")
		}
	}
	return nil
}
