package game

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
)

type KillFeedEntry struct {
	Victim string
	Killer string
	Mod    int // source hud.killupdate mod: 1 normal,2 cheapshot,3 grenade,4 greedy
	Time   int
	Alpha  float64
	Y      float64
	Slot   int
	Dead   bool
}

func (g *Game) playerNameByID(id int) string {
	for _, p := range g.players {
		if p.ID == id {
			return p.Name
		}
	}
	return "none"
}

func (g *Game) appendSourceKillFeed(victim *Player) {
	if victim == nil || g.GameMode == SourceGameModeSurvival || (g.campaignMode && g.campaignLevel == 1) {
		return
	}
	killer := "none"
	if victim.LastDeathBy > 0 {
		killer = g.playerNameByID(victim.LastDeathBy)
	}
	entry := KillFeedEntry{
		Victim: victim.Name,
		Killer: killer,
		Mod: victim.LastDeathMod,
		Alpha: 0,
		Slot: len(g.killfeeds) + 1,
		Y: float64(len(g.killfeeds)) * 30,
	}
	g.killfeeds = append(g.killfeeds, entry)
	// Source killupdate cancels a running non-campaign win countdown.
	if g.soloWinFrame != 0 && !g.GameWin {
		g.soloWinFrame = 0
		g.cameraX, g.cameraY = 0, 0
	}
}

func updateKillFeeds(entries []KillFeedEntry) {
	removed := false
	for i := range entries {
		e := &entries[i]
		if e.Dead {
			continue
		}
		e.Time++
		targetY := float64(e.Slot-1) * 30
		e.Y += (targetY - e.Y) / 3
		if e.Time <= 10 && e.Alpha < 1 {
			e.Alpha += 0.1
			if e.Time == 10 || e.Alpha > 1 {
				e.Alpha = 1
			}
		}
		if e.Time > 110 {
			e.Alpha -= 0.1
			if e.Alpha <= 0.01 {
				e.Dead = true
				removed = true
			}
		}
	}
	if removed {
		slot := 1
		for i := range entries {
			if entries[i].Dead {
				continue
			}
			entries[i].Slot = slot
			slot++
		}
	}
}

func compactKillFeeds(entries []KillFeedEntry) []KillFeedEntry {
	out := entries[:0]
	for _, e := range entries {
		if !e.Dead {
			out = append(out, e)
		}
	}
	return out
}

func (g *Game) drawKillFeeds(screen *ebiten.Image) {
	for _, e := range g.killfeeds {
		if e.Dead || e.Alpha <= 0 {
			continue
		}
		x := 12.0
		y := 12.0 + e.Y
		white := color.NRGBA{R: 255, G: 255, B: 255, A: uint8(e.Alpha * 255)}
		if e.Killer == "none" {
			// Source frame2 shows the Wingdings death icon and moves victim text
			// to x=61. Reproduce the text placement; the icon itself is static UI.
			drawSourceMenuText(screen, e.Victim, menuFontCondensedExtraBold, 24, white, 61, y)
			continue
		}
		drawSourceMenuText(screen, e.Killer, menuFontCondensedExtraBold, 24, white, x, y)
		verbX := float64(len(e.Killer))*11 + 23
		verb := "KILLED"
		verbOffset := 80.0
		verbColor := color.NRGBA{R: 0x99, A: uint8(e.Alpha * 255)}
		switch e.Mod {
		case 2:
			verb = "CHEAPSHOT'd"
			verbOffset = 140
			verbColor = color.NRGBA{R: 0xff, G: 0xcc, A: uint8(e.Alpha * 255)}
		case 3:
			verb = "EXPLODED"
			verbOffset = 110
			verbColor = color.NRGBA{R: 0xff, G: 0x66, A: uint8(e.Alpha * 255)}
		case 4:
			verb = "GREEDYKILLED"
			verbOffset = 150
			verbColor = color.NRGBA{G: 0x66, A: uint8(e.Alpha * 255)}
		}
		drawSourceMenuText(screen, verb, menuFontCondensedExtraBold, 24, verbColor, verbX, y)
		drawSourceMenuText(screen, e.Victim, menuFontCondensedExtraBold, 24, white, verbX+verbOffset, y)
	}
}
