package main

import (
	"log"

	"github.com/hajimehoshi/ebiten/v2"
	"gunmayhem-go/internal/game"
)

func main() {
	ebiten.SetWindowSize(game.ScreenWidth, game.ScreenHeight)
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	ebiten.SetWindowTitle("Gun Mayhem RE - Go port")
	ebiten.SetTPS(30)

	if err := ebiten.RunGame(game.New()); err != nil {
		log.Fatal(err)
	}
}
