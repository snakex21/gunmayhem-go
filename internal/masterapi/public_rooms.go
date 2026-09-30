package masterapi

import (
	"errors"
	"net/url"
	"strings"
	"unicode"
	"unicode/utf8"
)

const (
	PublicRoomsPath           = "/v2/rooms"
	PublicRoomOwnerHeader     = "X-GunMayhem-Room-Owner"
	PublicRoomWebSocketPrefix = "/ws/room/"
	MaxPublicRooms            = 128
	MaxPublicRoomBody         = 4096
	MaxPublicRoomListBody     = 256 << 10
)

// PublicRoom contains only publicly shareable routing and preview data. The
// ID is NOT authentication. Private invites and ownership credentials have no
// field here and must never be placed in a room name or preview.
type PublicRoom struct {
	ID                   string  `json:"id"`
	Endpoint             string  `json:"endpoint"`
	Name                 string  `json:"name"`
	Protocol             int     `json:"protocol"`
	Players              int     `json:"players"`
	MaxPlayers           int     `json:"max_players"`
	GameMode             int     `json:"game_mode"`
	Lives                int     `json:"lives"`
	Rounds               int     `json:"rounds"`
	Maps                 [20]int `json:"maps"`
	PlaylistConfigured   bool    `json:"playlist_configured"`
	Crates               bool    `json:"crates"`
	Pickups              bool    `json:"pickups"`
	RuleSummary          string  `json:"rule_summary"`
	ChaosEnabled         bool    `json:"chaos_enabled"`
	ChaosEvents          int     `json:"chaos_events"`
	ChaosIntervalSeconds int     `json:"chaos_interval_seconds"`
}

type PublicRoomList struct {
	Rooms []PublicRoom `json:"rooms"`
}

func ValidHexIdentity(value string, bytes int) bool {
	if bytes <= 0 || bytes > 64 || len(value) != bytes*2 {
		return false
	}
	for _, ch := range value {
		if !(ch >= '0' && ch <= '9') && !(ch >= 'a' && ch <= 'f') {
			return false
		}
	}
	return true
}

// Deliberately narrow: no arbitrary TCP/UDP endpoint, URL credentials, redirect,
// query, fragment, custom port or alternate route can come from a directory.
func NormalizePublicRoomEndpoint(raw, roomID string) (string, error) {
	if !ValidHexIdentity(roomID, 16) || len(raw) > 200 {
		return "", errors.New("invalid public room endpoint")
	}
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "wss" || u.Opaque != "" || u.User != nil || u.Port() != "" || u.RawQuery != "" || u.ForceQuery || u.Fragment != "" || u.RawPath != "" || u.Path != PublicRoomWebSocketPrefix+roomID {
		return "", errors.New("invalid public room endpoint")
	}
	host := strings.ToLower(u.Host)
	const suffix = ".trycloudflare.com"
	if !strings.HasSuffix(host, suffix) {
		return "", errors.New("unsupported public room host")
	}
	label := strings.TrimSuffix(host, suffix)
	if len(label) < 1 || len(label) > 63 || label[0] == '-' || label[len(label)-1] == '-' {
		return "", errors.New("invalid public room host")
	}
	for _, ch := range label {
		if !(ch >= 'a' && ch <= 'z') && !(ch >= '0' && ch <= '9') && ch != '-' {
			return "", errors.New("invalid public room host")
		}
	}
	canonical := "wss://" + host + PublicRoomWebSocketPrefix + roomID
	if raw != canonical {
		return "", errors.New("noncanonical public room endpoint")
	}
	return canonical, nil
}

func validPublicText(value string, maxRunes, maxBytes int) bool {
	if len(value) > maxBytes || !utf8.ValidString(value) || utf8.RuneCountInString(value) > maxRunes || strings.TrimSpace(value) != value {
		return false
	}
	for _, ch := range value {
		if unicode.IsControl(ch) || unicode.Is(unicode.Cf, ch) || unicode.Is(unicode.Zl, ch) || unicode.Is(unicode.Zp, ch) {
			return false
		}
	}
	return true
}

func ValidatePublicRoom(room PublicRoom) error {
	endpoint, err := NormalizePublicRoomEndpoint(room.Endpoint, room.ID)
	if err != nil || endpoint != room.Endpoint {
		return errors.New("invalid public room route")
	}
	if room.Name == "" || !validPublicText(room.Name, 48, 192) || !validPublicText(room.RuleSummary, 96, 192) {
		return errors.New("invalid public room text")
	}
	if room.Protocol <= 0 || room.Protocol > 100000 || room.MaxPlayers < 1 || room.MaxPlayers > 4 || room.Players < 1 || room.Players > room.MaxPlayers || room.GameMode < 0 || room.GameMode > 16 || room.Lives < 0 || room.Lives > 99999 || room.Rounds < 0 || room.Rounds > 20 || room.ChaosEvents < 0 || room.ChaosEvents > 256 || room.ChaosIntervalSeconds < 0 || room.ChaosIntervalSeconds > 3600 {
		return errors.New("invalid public room preview")
	}
	for _, v := range room.Maps {
		if v < -100000 || v > 100000 {
			return errors.New("invalid public room map")
		}
	}
	return nil
}
