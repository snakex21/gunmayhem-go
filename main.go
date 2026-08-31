package main

import (
	"bytes"
	_ "embed"
	"flag"
	"fmt"
	"image"
	"image/png"
	"log"
	"os"

	"github.com/hajimehoshi/ebiten/v2"
	"gunmayhem-go/internal/game"
	"gunmayhem-go/internal/networksetup"
	"gunmayhem-go/internal/updater"
)

//go:embed assets/icon/gunmayhem.png
var windowIconPNG []byte

func main() {
	if handled, err := networksetup.HandleSpecialMode(os.Args[1:]); handled {
		if err != nil {
			log.Printf("configure LAN networking: %v", err)
			os.Exit(1)
		}
		return
	}
	if handled, err := updater.MaybeRunInternalApply(os.Args[1:]); handled {
		if err != nil {
			log.Printf("apply update: %v", err)
			os.Exit(1)
		}
		return
	}

	host := flag.Bool("host", false, "host a network game for up to four players")
	port := flag.Int("port", 7777, "TCP port used by --host")
	join := flag.String("join", "", "join a host, for example 192.168.1.20:7777")
	flag.Parse()
	if !*host && *join == "" {
		networksetup.EnsureConfigured()
	}
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
	// A local match must freeze when the player switches to another window.
	// Ebitengine runs unfocused by default, which previously let timers, AI and
	// combat continue invisibly in the background.
	ebiten.SetRunnableOnUnfocused(false)
	if icon, err := png.Decode(bytes.NewReader(windowIconPNG)); err != nil {
		log.Printf("window icon: %v", err)
	} else {
		ebiten.SetWindowIcon([]image.Image{icon})
	}
	title := "Gun Mayhem RE - Go port"
	if updater.IsReleaseBuild() {
		title += " " + updater.CurrentVersion
	}
	ebiten.SetWindowTitle(title)
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
