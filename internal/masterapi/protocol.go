package masterapi

const (
	ServersPath        = "/v1/servers"
	HeartbeatPath      = "/v1/servers/heartbeat"
	RelayWebSocketPath = "/v1/relay"
	HealthPath         = "/healthz"
	ServerSecretHeader = "X-GunMayhem-Server-Secret"
)

type HeartbeatRequest struct {
	ID                   string  `json:"id,omitempty"`
	Secret               string  `json:"secret,omitempty"`
	SessionID            string  `json:"session_id,omitempty"`
	JoinCode             string  `json:"join_code,omitempty"`
	Private              bool    `json:"private,omitempty"`
	Name                 string  `json:"name"`
	Port                 int     `json:"port"`
	Players              int     `json:"players"`
	MaxPlayers           int     `json:"max_players"`
	Protocol             int     `json:"protocol"`
	GameMode             int     `json:"game_mode,omitempty"`
	Lives                int     `json:"lives,omitempty"`
	Rounds               int     `json:"rounds,omitempty"`
	Maps                 [20]int `json:"maps,omitempty"`
	PlaylistConfigured   bool    `json:"playlist_configured,omitempty"`
	Crates               bool    `json:"crates,omitempty"`
	Pickups              bool    `json:"pickups,omitempty"`
	RuleSummary          string  `json:"rule_summary,omitempty"`
	ChaosEnabled         bool    `json:"chaos_enabled,omitempty"`
	ChaosEvents          int     `json:"chaos_events,omitempty"`
	ChaosIntervalSeconds int     `json:"chaos_interval_seconds,omitempty"`
}

type HeartbeatResponse struct {
	ID           string `json:"id"`
	Secret       string `json:"secret,omitempty"`
	Address      string `json:"address"`
	RelayAddress string `json:"relay_address,omitempty"`
}

type Server struct {
	ID                   string  `json:"id"`
	SessionID            string  `json:"session_id,omitempty"`
	Name                 string  `json:"name"`
	Address              string  `json:"address"`
	RelayAddress         string  `json:"relay_address,omitempty"`
	Players              int     `json:"players"`
	MaxPlayers           int     `json:"max_players"`
	Protocol             int     `json:"protocol"`
	GameMode             int     `json:"game_mode,omitempty"`
	Lives                int     `json:"lives,omitempty"`
	Rounds               int     `json:"rounds,omitempty"`
	Maps                 [20]int `json:"maps,omitempty"`
	PlaylistConfigured   bool    `json:"playlist_configured,omitempty"`
	Crates               bool    `json:"crates,omitempty"`
	Pickups              bool    `json:"pickups,omitempty"`
	RuleSummary          string  `json:"rule_summary,omitempty"`
	ChaosEnabled         bool    `json:"chaos_enabled,omitempty"`
	ChaosEvents          int     `json:"chaos_events,omitempty"`
	ChaosIntervalSeconds int     `json:"chaos_interval_seconds,omitempty"`
}

type ServerListResponse struct {
	Servers []Server `json:"servers"`
}
