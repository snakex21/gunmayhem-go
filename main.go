package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/hajimehoshi/ebiten/v2"
	"gunmayhem-go/internal/game"
)

func main() {
	host := flag.Bool("host", false, "host a two-player network game")
	port := flag.Int("port", 7777, "TCP port used by --host")
	join := flag.String("join", "", "join a host, for example 192.168.1.20:7777")
	flag.Parse()
	if *host && *join != "" {
		log.Fatal("use either --host or --join, not both")
	}

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
	if *host {
		if err := g.StartNetHost(fmt.Sprintf(":%d", *port)); err != nil {
			log.Fatalf("host multiplayer: %v", err)
		}
		ebiten.SetWindowTitle(fmt.Sprintf("Gun Mayhem RE - Host :%d", *port))
		log.Printf("multiplayer host listening on TCP port %d", *port)
	} else if *join != "" {
		if err := g.StartNetClient(*join); err != nil {
			log.Fatalf("join multiplayer: %v", err)
		}
		ebiten.SetWindowTitle("Gun Mayhem RE - Client")
		log.Printf("multiplayer client connected to %s", *join)
	}
	defer g.CloseNetplay()

	runErr := ebiten.RunGame(g)
	if err := g.SavePersistentState(); err != nil {
		log.Printf("save persistent state: %v", err)
	}
	if runErr != nil {
		log.Fatal(runErr)
	}
}
