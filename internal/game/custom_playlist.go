package game

import (
	"fmt"
	"math/rand"
)

func (g *Game) ensureCustomPlaylistState() {
	if g == nil {
		return
	}
	g.customSeriesRounds = clampCustomSeriesRounds(g.customSeriesRounds)
	if g.customPlaylistSelectedRound < 1 {
		g.customPlaylistSelectedRound = 1
	}
	if g.customPlaylistSelectedRound > g.customSeriesRounds {
		g.customPlaylistSelectedRound = g.customSeriesRounds
	}
	maxScroll := g.customSeriesRounds - customV2PlaylistVisibleRows
	if maxScroll < 0 {
		maxScroll = 0
	}
	if g.customPlaylistScroll < 0 {
		g.customPlaylistScroll = 0
	}
	if g.customPlaylistScroll > maxScroll {
		g.customPlaylistScroll = maxScroll
	}
}

func (g *Game) enableCustomSeriesPlaylist() {
	if g == nil || g.customSeriesPlaylistConfigured {
		return
	}
	for i := range g.customSeriesMaps {
		g.customSeriesMaps[i] = g.customMap
	}
	g.customSeriesPlaylistConfigured = true
	g.ensureCustomPlaylistState()
}

func (g *Game) setCustomSeriesMap(round, mapValue int) {
	if g == nil {
		return
	}
	g.enableCustomSeriesPlaylist()
	round = clampCustomRuleEditRound(round, g.customSeriesRounds)
	g.customSeriesMaps[round-1] = mapValue
	g.customPlaylistSelectedRound = round
	// Keep the legacy/current selector aligned with the actively edited round so
	// previews, lobby summaries and older compatibility paths remain meaningful.
	g.customMap = mapValue
	g.customMapSetGM2 = isGM2MapID(mapValue) || mapValue == customGM2RandomNew || mapValue == customGM2RandomAll
}

func (g *Game) setCustomSeriesMapForAll(mapValue int) {
	if g == nil {
		return
	}
	g.enableCustomSeriesPlaylist()
	for i := 0; i < clampCustomSeriesRounds(g.customSeriesRounds); i++ {
		g.customSeriesMaps[i] = mapValue
	}
	g.customMap = mapValue
	g.customMapSetGM2 = isGM2MapID(mapValue) || mapValue == customGM2RandomNew || mapValue == customGM2RandomAll
}

func (g *Game) customSeriesMapForRound(round int) int {
	if g == nil {
		return 1
	}
	if !g.customSeriesPlaylistConfigured {
		return g.customMap
	}
	round = clampCustomRuleEditRound(round, g.customSeriesRounds)
	return g.customSeriesMaps[round-1]
}

func customSeriesMapLabel(mapValue int) string {
	switch mapValue {
	case 0:
		return "RANDOM GM1"
	case customGM2RandomNew:
		return "RANDOM GM2"
	case customGM2RandomAll:
		return "RANDOM GM2 ALL"
	}
	if isGM2MapID(mapValue) {
		if name := gm2MapDisplayNames[gm2SourceMapNumber(mapValue)]; name != "" {
			return "GM2 - " + name
		}
		return fmt.Sprintf("GM2 - MAP %d", gm2SourceMapNumber(mapValue))
	}
	if name := sourceMapDisplayNames[mapValue]; name != "" {
		return fmt.Sprintf("GM1 - %s", name)
	}
	return fmt.Sprintf("MAP %d", mapValue)
}

func resolveCustomSeriesMapValue(mapValue int) int {
	switch mapValue {
	case 0:
		return rand.Intn(12) + 1
	case customGM2RandomNew:
		return gm2MapID(rand.Intn(9) + 1)
	case customGM2RandomAll:
		return gm2MapID(rand.Intn(21) + 1)
	default:
		return mapValue
	}
}
