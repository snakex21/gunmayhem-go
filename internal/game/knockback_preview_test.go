//go:build perkpreview

package game

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"golang.org/x/image/font/opentype"
)

// Optional visual fixture: run alone under a display with -tags perkpreview.
// It renders static UI screens only; it never hosts a session or launches a tunnel.
type perkPreviewGame struct {
	game   *Game
	output string
	frame  int
	err    error
}

func (p *perkPreviewGame) Layout(int, int) (int, int) { return 900, 600 }
func (p *perkPreviewGame) Update() error {
	if p.frame >= 3 || p.err != nil {
		return ebiten.Termination
	}
	return nil
}
func (p *perkPreviewGame) Draw(screen *ebiten.Image) {
	if p.frame >= 3 {
		return
	}
	screen.Fill(color.NRGBA{R: 24, G: 28, B: 34, A: 255})
	names := []string{"custom-perk-picker.png", "online-perk-picker.png", "perk-blocklist.png"}
	switch p.frame {
	case 0:
		p.game.drawCustomV2Players(screen)
	case 1:
		p.game.onlineVisualPicker = onlineVisualCharacterPerk
		p.game.drawOnlineVisualPicker(screen)
	case 2:
		p.game.drawOnlinePerkBlocklist(screen)
	}
	pixels := make([]byte, 900*600*4)
	screen.ReadPixels(pixels)
	file, err := os.Create(filepath.Join(p.output, names[p.frame]))
	if err != nil {
		p.err = err
		return
	}
	p.err = png.Encode(file, &image.RGBA{Pix: pixels, Stride: 900 * 4, Rect: image.Rect(0, 0, 900, 600)})
	if err := file.Close(); p.err == nil {
		p.err = err
	}
	p.frame++
}
func TestKnockbackPreviewScreenshots(t *testing.T) {
	output := os.Getenv("GUNMAYHEM_PREVIEW_DIR")
	if output == "" {
		output = t.TempDir()
	}
	if err := os.MkdirAll(output, 0755); err != nil {
		t.Fatal(err)
	}
	// A fixture-only font override permits layout inspection when bundled
	// exported fonts or proprietary system fallbacks cannot be parsed. It does
	// not alter the game's production font loader or redistribute font assets.
	if path := os.Getenv("GUNMAYHEM_PREVIEW_FONT"); path != "" {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		font, err := opentype.Parse(data)
		if err != nil {
			t.Fatal(err)
		}
		for _, cache := range menuFontCaches {
			cache.once.Do(func() { cache.font = font })
		}
	}
	for _, kind := range []sourceMenuFontKind{menuFontCondensed, menuFontCondensedExtraBold, menuFontTwCen} {
		if _, err := sourceMenuFace(kind, 14); err != nil {
			t.Fatalf("preview font: %v", err)
		}
	}
	g := New()
	for i := range g.customPlayers {
		g.customPlayers[i].Type = 1
		g.customPlayers[i].Perk = perkKnockbackResistance
		g.customEditor[i] = customEditPerk
		g.customCardY[i] = -300
	}
	preview := &perkPreviewGame{game: g, output: output}
	ebiten.SetWindowSize(900, 600)
	ebiten.SetRunnableOnUnfocused(true)
	if err := ebiten.RunGame(preview); err != nil {
		t.Fatal(err)
	}
	if preview.err != nil {
		t.Fatal(preview.err)
	}
	if preview.frame != 3 {
		t.Fatal("preview did not render all screens")
	}
}
