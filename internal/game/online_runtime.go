package game

func (g *Game) onlineLocalPlayerID() int {
	if g == nil || g.netplay == nil {
		return 0
	}
	if g.netplay.mode == netplayHost {
		return 1
	}
	if g.netplay.mode == netplayClient {
		return g.netplay.assignedPlayer()
	}
	return 0
}

func (g *Game) onlineConnectedPlayerCount() int {
	if g == nil || g.netplay == nil {
		return 0
	}
	if g.netplay.mode == netplayHost {
		return 1 + g.netplay.connectedCount()
	}
	if g.onlinePlayerCount > 0 {
		return g.onlinePlayerCount
	}
	if g.netplay.connected() {
		return 2
	}
	return 0
}

func (g *Game) syncOnlineHostState() {
	if g == nil || g.netplay == nil || g.netplay.mode != netplayHost {
		return
	}

	g.updateNetplayPublicProbeInfo()
	g.customPlayers[0].Type = 1
	var allowed [5]bool
	for id := 2; id <= maxOnlinePlayers; id++ {
		allowed[id] = g.customPlayers[id-1].Type == 1
	}
	g.netplay.setAllowedRemotePlayers(allowed)

	g.onlinePlayerSlots = [4]bool{true, false, false, false}
	for _, id := range g.netplay.connectedPlayerIDs() {
		if id < 2 || id > maxOnlinePlayers {
			continue
		}
		// A peer can remain in the connection map for a few milliseconds after
		// the host closes its slot. Do not resurrect a kicked player or overwrite
		// an AI bot with that stale peer config.
		if g.customPlayers[id-1].Type != 1 {
			continue
		}
		g.onlinePlayerSlots[id-1] = true
		if cfg, ok := g.netplay.latestConfigFor(id); ok {
			cfg.Type = 1
			g.customPlayers[id-1] = cfg
		}
	}
	g.onlinePlayerCount = 1 + g.netplay.connectedCount()

	if g.screen == screenGameplay && g.onlineMatchSetup {
		g.syncOnlineGameplayPlayers()
	}
}

func (g *Game) updateNetplayPublicProbeInfo() {
	if g == nil || g.netplay == nil || g.netplay.mode != netplayHost {
		return
	}
	rounds := clampCustomSeriesRounds(g.customSeriesRounds)
	var maps [customSeriesMaxRounds]int
	for round := 1; round <= rounds; round++ {
		maps[round-1] = g.customSeriesMapForRound(round)
	}
	reachability := g.currentHostInternetReachability()
	g.netplay.setPublicProbeDetails(netProbeInfo{
		GameMode:             g.customMode,
		Lives:                g.customLives,
		Rounds:               rounds,
		Maps:                 maps,
		PlaylistConfigured:   g.customSeriesPlaylistConfigured,
		Crates:               g.CrateON,
		Pickups:              g.PowerON,
		RuleSummary:          g.developedModifierSummary(),
		ChaosEnabled:         g.customChaosEnabled,
		ChaosEvents:          g.chaosSelectedCount(),
		ChaosIntervalSeconds: g.customChaosInterval,
		DirectAddress:        reachability.DirectAddress,
	})
}

func (g *Game) onlineActiveParticipantCount() int {
	if g == nil {
		return 0
	}
	count := 1 // host
	for id := 2; id <= maxOnlinePlayers; id++ {
		cfg := g.customPlayers[id-1]
		switch cfg.Type {
		case 2:
			count++
		case 1:
			if g.netplay != nil && g.netplay.mode == netplayHost && g.onlinePlayerSlots[id-1] {
				count++
			}
		}
	}
	return count
}

func (g *Game) sendOnlineLocalConfigIfChanged() {
	if g == nil || g.netplay == nil || g.netplay.mode != netplayClient {
		return
	}
	id := g.netplay.assignedPlayer()
	if id < 2 || id > maxOnlinePlayers {
		return
	}
	cfg := g.customPlayers[id-1]
	cfg.Type = 1
	if g.onlineHasSentConfig && cfg == g.onlineLastSentConfig {
		return
	}
	g.netplay.queuePlayerConfig(cfg)
	g.onlineLastSentConfig = cfg
	g.onlineHasSentConfig = true
}

func (g *Game) syncOnlineGameplayPlayers() {
	if g == nil || g.netplay == nil || g.netplay.mode != netplayHost || !g.onlineMatchSetup {
		return
	}
	connected := [5]bool{false, true, false, false, false}
	for _, id := range g.netplay.connectedPlayerIDs() {
		if id >= 2 && id <= maxOnlinePlayers {
			connected[id] = true
		}
	}
	desired := [5]bool{false, true, false, false, false}
	for id := 2; id <= maxOnlinePlayers; id++ {
		cfg := g.customPlayers[id-1]
		desired[id] = cfg.Type == 2 || (cfg.Type == 1 && connected[id])
	}

	kept := g.players[:0]
	present := [5]bool{}
	for _, p := range g.players {
		if p == nil {
			continue
		}
		if p.ID >= 2 && p.ID <= maxOnlinePlayers && !desired[p.ID] && !p.IsDouble {
			continue
		}
		if p.ID >= 1 && p.ID <= maxOnlinePlayers && !p.IsDouble {
			present[p.ID] = true
		}
		kept = append(kept, p)
	}
	g.players = kept

	for id := 1; id <= maxOnlinePlayers; id++ {
		if !desired[id] {
			continue
		}
		if present[id] {
			for _, p := range g.players {
				if p != nil && p.ID == id && !p.IsDouble {
					cfg := g.customPlayers[id-1]
					g.applyOnlineConfigToPlayer(p, cfg)
					p.AI = cfg.Type == 2
					p.RemoteControlled = cfg.Type == 1 && id != 1
					break
				}
			}
			continue
		}
		g.players = append(g.players, g.newOnlinePlayer(id))
	}
}

func (g *Game) applyOnlineConfigToPlayer(p *Player, cfg CustomPlayerConfig) {
	if p == nil {
		return
	}
	p.Name = cfg.Name
	p.PlayerColor = cfg.Color
	p.ShirtNumber = cfg.Shirt
	p.HatNumber = cfg.Hat
	p.EyeNumber = cfg.Eyes
	p.PerkNumber = g.onlineAllowedPerk(cfg.Perk)
	p.ThrowableType = normalizedThrowableType(cfg.Throwable)
	if g.GameMode == SourceGameModeTeams {
		p.Team = cfg.Team
	}
}

func (g *Game) newOnlinePlayer(id int) *Player {
	cfg := g.customPlayers[id-1]
	p := NewPlayer(id, g.arena)
	p.Controls = g.controlConfigs[id-1]
	p.Name = cfg.Name
	p.PlayerColor = cfg.Color
	p.ShirtNumber = cfg.Shirt
	p.HatNumber = cfg.Hat
	p.EyeNumber = cfg.Eyes
	p.DefaultWeapon = g.onlineStartingWeapon(cfg.Gun)
	p.PerkNumber = g.onlineAllowedPerk(cfg.Perk)
	p.ThrowableType = normalizedThrowableType(cfg.Throwable)
	p.AI = cfg.Type == 2
	p.RemoteControlled = cfg.Type == 1 && id != 1
	if g.GameMode == SourceGameModeTeams {
		p.Team = cfg.Team
	} else {
		p.Team = id
	}
	if g.GameMode == SourceGameModeSurvival {
		p.Team = 0
	}
	p.configureSourceGameMode(g.GameMode, g.TotalLives, g.arena)
	return p
}

func (g *Game) onlineAllowedPerk(selected int) int {
	if !g.onlinePerksEnabled || selected <= 0 || selected >= len(g.onlineBlockedPerks) || g.onlineBlockedPerks[selected] {
		return 0
	}
	return selected
}

func (g *Game) onlineStartingWeapon(selected int) int {
	if g.onlineWeaponMode == onlineWeaponsOnlyOne && g.onlineOnlyWeapon > 0 {
		return g.onlineOnlyWeapon
	}
	if g.onlineWeaponMode != onlineWeaponsBlocklist {
		if selected > 0 {
			return selected
		}
		return 10
	}
	if selected > 0 && selected < len(g.onlineBlockedWeapons) && !g.onlineBlockedWeapons[selected] {
		return selected
	}
	for _, n := range onlineAllWeaponNumbers() {
		if n > 0 && n < len(g.onlineBlockedWeapons) && !g.onlineBlockedWeapons[n] {
			return n
		}
	}
	return 10
}

func (g *Game) onlineCrateWeaponPool() []int {
	if g.onlineWeaponMode == onlineWeaponsOnlyOne && g.onlineOnlyWeapon > 0 {
		return []int{g.onlineOnlyWeapon}
	}
	all := developedCrateWeapons()
	if g.onlineWeaponMode != onlineWeaponsBlocklist {
		return append([]int(nil), all...)
	}
	out := make([]int, 0, len(all))
	for _, n := range all {
		if n <= 0 || n >= len(g.onlineBlockedWeapons) || g.onlineBlockedWeapons[n] {
			continue
		}
		out = append(out, n)
	}
	if len(out) == 0 {
		return []int{10}
	}
	return out
}
