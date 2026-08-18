package main

import (
	"log"

	"github.com/hajimehoshi/ebiten/v2"
	"gunmayhem-go/internal/game"
)

func main() {
	cfg, err := game.LoadAppConfig()
	if err != nil {
		log.Printf("config: %v", err)
	}
	save, err := game.LoadSaveData()
	if err != nil {
		log.Printf("save: %v", err)
	}

	game.ApplyWindowConfig(cfg)
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	ebiten.SetWindowTitle("Gun Mayhem RE - Go port")
	ebiten.SetTPS(30)

	g := game.NewPersistent(cfg, save)
	runErr := ebiten.RunGame(g)
	if err := g.SavePersistentState(); err != nil {
		log.Printf("save persistent state: %v", err)
	}
	if runErr != nil {
		log.Fatal(runErr)
	}
}
