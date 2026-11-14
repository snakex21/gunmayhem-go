package game

import (
	"sync"
)

const (
	customGM2RandomNew = -101
	customGM2RandomAll = -102
)

var (
	gm2MapAssetsOnce sync.Once
	gm2MapAssetsOK   bool
)

func gm2MapAssetsAvailable() bool {
	gm2MapAssetsOnce.Do(func() {
		gm2MapAssetsOK = true
		checks := [][]string{
			{"fla", "LIBRARY", "Symbol 1940.xml"},
			{"sprites", "DefineSprite_1642", "1.png"},
			{"sprites", "DefineSprite_1642", "21.png"},
			{"sprites", "DefineSprite_1835", "21.png"},
		}
		for _, parts := range checks {
			if _, err := findOriginalPathIn("gm2", parts...); err != nil {
				gm2MapAssetsOK = false
				return
			}
		}
	})
	return gm2MapAssetsOK
}
