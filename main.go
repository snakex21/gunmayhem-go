package main

import (
	"log"
	"os"

	"github.com/hajimehoshi/ebiten/v2"
	"gunmayhem-go/internal/game"
)

func main() {
	developerTools := false
	for _, arg := range os.Args[1:] {
		if arg == "--dev" || arg == "--debug" {
			developerTools = true
		}
	}

	ebiten.SetWindowSize(game.ScreenWidth, game.ScreenHeight)
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	ebiten.SetWindowTitle("Gun Mayhem RE - Go port")
	ebiten.SetTPS(30)

	if err := ebiten.RunGame(game.NewWithDeveloperTools(developerTools)); err != nil {
		log.Fatal(err)
	}
}
