package game

import "testing"

func BenchmarkLoadAssetsStartup(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = LoadAssets()
	}
}

func BenchmarkLoadMapOneStartup(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = LoadOriginalMap(1)
	}
}

func BenchmarkFirstSceneStartup(b *testing.B) {
	for i := 0; i < b.N; i++ {
		a := LoadAssets()
		a.EnsureScene(1)
	}
}
