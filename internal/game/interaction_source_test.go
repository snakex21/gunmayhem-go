package game

import "testing"

func TestInteractionSourceBounds(t *testing.T) {
	for _, name := range []string{"Symbol 95", "Symbol 965", "Symbol 1040", "Symbol 1045", "Symbol 1265", "Symbol 800", "Symbol 806", "Symbol 811", "Symbol 813", "Symbol 817", "Symbol 831", "Symbol 833", "Symbol 856", "Symbol 902", "Symbol 913", "Symbol 1061", "Symbol 630", "Symbol 792", "Symbol 947", "Symbol 633", "Symbol 625", "Symbol 654", "Symbol 1487", "Symbol 331", "Symbol 361", "Symbol 365"} {
		r, err := sourceFrameVisualBounds(name, 0)
		if err != nil {
			t.Logf("%s bounds unavailable: %v", name, err)
			continue
		}
		t.Logf("%s=%+v", name, r)
	}
}
