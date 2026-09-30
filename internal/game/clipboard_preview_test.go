//go:build clipboardpreview

package game

import (
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

// Static production-font layout fixtures. No live session or OS clipboard is
// accessed, and the displayed invite is deliberately synthetic.
type clipboardPreviewGame struct {
	game   *Game
	output string
	frame  int
	err    error
}

func (p *clipboardPreviewGame) Layout(int, int) (int, int) { return 900, 600 }
func (p *clipboardPreviewGame) Update() error {
	if p.frame >= 2 || p.err != nil {
		return ebiten.Termination
	}
	return nil
}
func (p *clipboardPreviewGame) Draw(screen *ebiten.Image) {
	if p.frame >= 2 {
		return
	}
	name := "join-browser.png"
	if p.frame == 0 {
		p.game.multiplayerMessage = "CLICK THE FIELD, THEN CTRL+V ON WINDOWS"
		p.game.drawCustomNetworkPanel(screen)
	} else {
		name = "copy-invite.png"
		p.game.multiplayerMessage = "INVITE COPIED"
		p.game.drawCustomV2OnlineShell(screen)
		p.game.drawCustomV2Players(screen)
	}
	pixels := make([]byte, 900*600*4)
	screen.ReadPixels(pixels)
	f, err := os.Create(filepath.Join(p.output, name))
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
func TestClipboardPreviewScreenshots(t *testing.T) {
	t.Setenv("GUNMAYHEM_EXPERIMENTAL_INTERNET", "1")
	output := os.Getenv("GUNMAYHEM_PREVIEW_DIR")
	if output == "" {
		output = t.TempDir()
	}
	if err := os.MkdirAll(output, 0755); err != nil {
		t.Fatal(err)
	}
	g := New()
	g.customOnlineMode = true
	g.customPage = customPagePlayers
	g.netplay = &netplaySession{mode: netplayHost}
	g.onlineSessionScope = onlineSessionScopeInternet
	g.onlineSessionVisibility = onlineSessionInviteOnly
	g.onlineSessionKey = strings.Repeat("A", 26)
	g.cloudflareTunnel = &cloudflareQuickTunnel{wsURL: "wss://" + strings.Repeat("a", 63) + cloudflareTunnelHostSuffix + "/ws"}
	g.cloudflareTunnel.running.Store(true)
	g.multiplayerMessage = "INVITE COPIED"
	for i := range g.customPlayers {
		g.customCardY[i] = 600
		g.customPlayers[i].Type = 0
	}
	p := &clipboardPreviewGame{game: g, output: output}
	ebiten.SetWindowSize(900, 600)
	ebiten.SetRunnableOnUnfocused(true)
	if err := ebiten.RunGame(p); err != nil {
		t.Fatal(err)
	}
	if p.err != nil {
		t.Fatal(p.err)
	}
	if p.frame != 2 {
		t.Fatal("missing preview screen")
	}
}
