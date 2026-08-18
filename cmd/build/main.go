package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

func main() {
	root, err := os.Getwd()
	if err != nil {
		fmt.Println("Nie można ustalić katalogu projektu:", err)
		os.Exit(1)
	}
	if _, err := os.Stat(filepath.Join(root, "go.mod")); err != nil {
		fmt.Println("Brak go.mod w katalogu projektu:", root)
		os.Exit(1)
	}
	out := filepath.Join(root, "GunMayhem.exe")

	cmd := exec.Command("go", "build", "-trimpath", "-ldflags=-s -w", "-o", out, ".")
	cmd.Dir = root
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	fmt.Println("Budowanie GunMayhem.exe...")
	if err := cmd.Run(); err != nil {
		fmt.Println("Build nieudany:", err)
		os.Exit(1)
	}
	fmt.Println("Gotowe:", out)
}
