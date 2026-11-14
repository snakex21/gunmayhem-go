package game

import (
	"strconv"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

const (
	menuHitModeCrates      = 8001
	menuHitModePickups     = 8002
	menuHitModeLives       = 8003
	menuHitModeRoundsMinus = 8004
	menuHitModeRoundsPlus  = 8005
	menuHitModeLivesMinus  = 8006
	menuHitModeLivesPlus   = 8007
)

func modeHasLives(mode int) bool {
	return mode == SourceGameModeNormal || mode == SourceGameModeInstagib || mode == SourceGameModeTeams
}

func modeHasCrateToggle(mode int) bool {
	return mode == SourceGameModeNormal || mode == SourceGameModeInstagib || mode == SourceGameModeTeams
}

func modeHasPickupToggle(mode int) bool {
	return mode == SourceGameModeNormal || mode == SourceGameModeTeams
}

func (g *Game) activateCustomModeOption(hit int) bool {
	if g.activateCustomModifierHit(hit) {
		return true
	}
	switch hit {
	case menuHitModeCrates:
		if modeHasCrateToggle(g.customMode) {
			g.CrateON = !g.CrateON
		}
		return true
	case menuHitModePickups:
		if modeHasPickupToggle(g.customMode) {
			g.PowerON = !g.PowerON
		}
		return true
	case menuHitModeLives:
		if modeHasLives(g.customMode) {
			g.customLivesFocus = true
		}
		return true
	case menuHitModeLivesMinus:
		if modeHasLives(g.customMode) && g.customLives > 1 {
			g.customLives--
		}
		g.customLivesFocus = false
		return true
	case menuHitModeLivesPlus:
		if modeHasLives(g.customMode) && g.customLives < 99999 {
			g.customLives++
		}
		g.customLivesFocus = false
		return true
	case menuHitModeRoundsMinus:
		g.customSeriesRounds = clampCustomSeriesRounds(g.customSeriesRounds - 1)
		g.customRuleEditRound = clampCustomRuleEditRound(g.customRuleEditRound, g.customSeriesRounds)
		return true
	case menuHitModeRoundsPlus:
		g.customSeriesRounds = clampCustomSeriesRounds(g.customSeriesRounds + 1)
		g.customRuleEditRound = clampCustomRuleEditRound(g.customRuleEditRound, g.customSeriesRounds)
		return true
	}
	return false
}

func (g *Game) updateCustomLivesInput() {
	if !g.customLivesFocus || g.customPage != customPageGame || !modeHasLives(g.customMode) {
		return
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) || inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		g.customLivesFocus = false
		return
	}
	value := strconv.Itoa(g.customLives)
	if g.customLives == 0 {
		value = ""
	}
	if textEraseRepeated() && len(value) > 0 {
		value = value[:len(value)-1]
	}
	for _, ch := range ebiten.AppendInputChars(nil) {
		if ch >= '0' && ch <= '9' && len(value) < 5 {
			value += string(ch)
		}
	}
	if value == "" {
		g.customLives = 0
	} else if n, err := strconv.Atoi(value); err == nil {
		g.customLives = n
	}
}
