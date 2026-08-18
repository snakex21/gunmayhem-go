package game

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func writeNamespaceTestSymbol(t *testing.T, dir string, edgeMax int) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	xml := fmt.Sprintf(`<?xml version="1.0" encoding="utf-8"?>
<DOMSymbolItem>
  <timeline><DOMTimeline><layers><DOMLayer><frames>
    <DOMFrame index="0" duration="1"><elements>
      <DOMShape>
        <matrix><Matrix a="1" d="1"/></matrix>
        <edges><Edge edges="!0 0|%d 0|%d %d|0 %d"/></edges>
      </DOMShape>
    </elements></DOMFrame>
  </frames></DOMLayer></layers></DOMTimeline></timeline>
</DOMSymbolItem>`, edgeMax, edgeMax, edgeMax, edgeMax)
	if err := os.WriteFile(filepath.Join(dir, "Symbol 1.xml"), []byte(xml), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestXFLBoundsCacheIsNamespacedByLibraryDirectory(t *testing.T) {
	gm1 := filepath.Join(t.TempDir(), "gm1")
	gm2 := filepath.Join(t.TempDir(), "gm2")
	writeNamespaceTestSymbol(t, gm1, 20) // 1 px in XFL coordinates
	writeNamespaceTestSymbol(t, gm2, 80) // 4 px in XFL coordinates

	one, err := sourceFrameVisualBoundsInDir(gm1, "Symbol 1", 0)
	if err != nil {
		t.Fatal(err)
	}
	four, err := sourceFrameVisualBoundsInDir(gm2, "Symbol 1", 0)
	if err != nil {
		t.Fatal(err)
	}
	if one.W == four.W || one.H == four.H {
		t.Fatalf("frame cache crossed namespaces: gm1=%+v gm2=%+v", one, four)
	}

	oneCanvas, err := sourceSymbolCanvasBoundsInDir(gm1, "Symbol 1")
	if err != nil {
		t.Fatal(err)
	}
	fourCanvas, err := sourceSymbolCanvasBoundsInDir(gm2, "Symbol 1")
	if err != nil {
		t.Fatal(err)
	}
	if oneCanvas.W == fourCanvas.W || oneCanvas.H == fourCanvas.H {
		t.Fatalf("symbol cache crossed namespaces: gm1=%+v gm2=%+v", oneCanvas, fourCanvas)
	}
}
