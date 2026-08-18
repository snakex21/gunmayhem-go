package game

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

func TestSourceRenderQualityMapsFlashLevels(t *testing.T) {
	for _, tc := range []struct {
		quality        int
		filter         ebiten.Filter
		disableMipmaps bool
	}{
		{1, ebiten.FilterNearest, false},
		{2, ebiten.FilterLinear, true},
		{3, ebiten.FilterLinear, false},
	} {
		setSourceRenderQuality(tc.quality)
		op := &ebiten.DrawImageOptions{}
		applySourceRenderQuality(op)
		if op.Filter != tc.filter || op.DisableMipmaps != tc.disableMipmaps {
			t.Fatalf("quality %d => filter=%v disableMipmaps=%v; want %v/%v", tc.quality, op.Filter, op.DisableMipmaps, tc.filter, tc.disableMipmaps)
		}
	}
	setSourceRenderQuality(2)
}
