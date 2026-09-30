//go:build publicroompreview

package game

import (
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"gunmayhem-go/internal/masterapi"
)

// Synthetic room metadata, production fonts/assets, no sockets or clipboard.
type publicRoomPreviewGame struct {
	game   *Game
	output string
	frame  int
	err    error
}

func (p *publicRoomPreviewGame) Layout(int, int) (int, int) { return 900, 600 }
func (p *publicRoomPreviewGame) Update() error {
	if p.frame >= 3 || p.err != nil {
		return ebiten.Termination
	}
	return nil
}
func (p *publicRoomPreviewGame) Draw(screen *ebiten.Image) {
	if p.frame >= 3 {
		return
	}
	names := []string{"create-public-room.png", "public-room-browser.png", "close-public-room.png"}
	g := p.game
	switch p.frame {
	case 0:
		g.drawCustomHostSetupPanel(screen)
	case 1:
		g.multiplayerMessage = "FOUND OPEN LAN / INTERNET ROOMS"
		g.drawCustomNetworkPanel(screen)
	case 2:
		g.netplay = &netplaySession{mode: netplayHost, publicRoomID: strings.Repeat("a", 32)}
		g.publicDirectory = &publicRoomAdvertiser{status: "PUBLIC ROOM: LISTED"}
		g.customPage = customPagePlayers
		g.drawCustomV2OnlineShell(screen)
		g.drawCustomV2Players(screen)
	}
	pixels := make([]byte, 900*600*4)
	screen.ReadPixels(pixels)
	f, err := os.Create(filepath.Join(p.output, names[p.frame]))
	if err != nil {
		p.err = err
		return
	}
	p.err = png.Encode(f, &image.RGBA{Pix: pixels, Stride: 3600, Rect: image.Rect(0, 0, 900, 600)})
	if err = f.Close(); p.err == nil {
		p.err = err
	}
	p.frame++
}
func TestPublicRoomPreviewScreenshots(t *testing.T) {
	t.Setenv("GUNMAYHEM_EXPERIMENTAL_INTERNET", "1")
	t.Setenv("GUNMAYHEM_MASTER_URL", "https://directory.example.invalid")
	output := os.Getenv("GUNMAYHEM_PREVIEW_DIR")
	if output == "" {
		output = t.TempDir()
	}
	if err := os.MkdirAll(output, 0755); err != nil {
		t.Fatal(err)
	}
	g := New()
	g.customOnlineMode = true
	g.customHostScope = onlineSessionScopeInternet
	g.customHostAccess = onlineSessionOpen
	g.onlineSessionScope = onlineSessionScopeInternet
	g.onlineSessionVisibility = onlineSessionOpen
	g.multiplayerMessage = "OPEN INTERNET ROOM: ANY PLAYER CAN JOIN; CREATE PUBLISHES THE ROOM"
	id := strings.Repeat("a", 32)
	endpoint := "wss://synthetic-room.trycloudflare.com" + masterapi.PublicRoomWebSocketPrefix + id
	g.multiplayerServers = []multiplayerServerInfo{{ID: id, Name: "Weekend Test Room", Sources: "INTERNET", Transport: "public-wss", Address: endpoint, TunnelAddress: endpoint, Players: 2, MaxPlayers: 4, Protocol: netplayProtocolVersion, GameMode: SourceGameModeNormal, Lives: 5, Rounds: 2, Maps: [20]int{1, 2}, PlaylistConfigured: true, RuleSummary: "PERKS ON"}}
	g.multiplayerSelectedServer = 0
	g.multiplayerDiscoveryDone = true
	for i := range g.customPlayers {
		g.customCardY[i] = 600
		g.customPlayers[i].Type = 0
	}
	p := &publicRoomPreviewGame{game: g, output: output}
	ebiten.SetWindowSize(900, 600)
	ebiten.SetRunnableOnUnfocused(true)
	if err := ebiten.RunGame(p); err != nil {
		t.Fatal(err)
	}
	if p.err != nil {
		t.Fatal(p.err)
	}
	if p.frame != 3 {
		t.Fatal("missing preview")
	}
}
