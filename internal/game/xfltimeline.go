package game

import (
	"encoding/xml"
	"errors"
	"io"
	"os"
	"path/filepath"
)

// SourceTransformFrame is one concrete Flash timeline frame for a nested
// symbol instance. Matrix is copied directly from the XFL DOMSymbolInstance.
type SourceTransformFrame struct {
	Matrix xflMatrix
	Alpha  float64
	Valid  bool
}

// loadChildTransformTimelines returns one expanded child timeline per XFL
// layer, preserving the layer order from the source file (top layer first).
// This matters for symbols like mapfx_snow where the same child Symbol 45 is
// used independently on two layers.
func loadChildTransformTimelines(parentLibrary, childLibrary string) ([][]SourceTransformFrame, error) {
	libraryDir, err := findOriginalPath("fla", "LIBRARY")
	if err != nil {
		return nil, err
	}
	path := filepath.Join(libraryDir, parentLibrary+".xml")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	type matrixXML struct {
		A  float64 `xml:"a,attr"`
		B  float64 `xml:"b,attr"`
		C  float64 `xml:"c,attr"`
		D  float64 `xml:"d,attr"`
		TX float64 `xml:"tx,attr"`
		TY float64 `xml:"ty,attr"`
	}
	type colorXML struct {
		Alpha *float64 `xml:"alphaMultiplier,attr"`
	}
	type instanceXML struct {
		Library string    `xml:"libraryItemName,attr"`
		Matrix  matrixXML `xml:"matrix>Matrix"`
		Color   colorXML  `xml:"color>Color"`
	}
	type frameXML struct {
		Index     int           `xml:"index,attr"`
		Duration  int           `xml:"duration,attr"`
		Instances []instanceXML `xml:"elements>DOMSymbolInstance"`
	}
	type layerXML struct {
		Frames []frameXML `xml:"frames>DOMFrame"`
	}
	type docXML struct {
		Layers []layerXML `xml:"timeline>DOMTimeline>layers>DOMLayer"`
	}
	var doc docXML
	if err := xml.Unmarshal(data, &doc); err != nil {
		return nil, err
	}

	parentFrames := 0
	for _, layer := range doc.Layers {
		for _, frame := range layer.Frames {
			d := frame.Duration
			if d < 1 {
				d = 1
			}
			if end := frame.Index + d; end > parentFrames {
				parentFrames = end
			}
		}
	}
	if parentFrames == 0 {
		return nil, errors.New("XFL: empty timeline " + parentLibrary)
	}

	var timelines [][]SourceTransformFrame
	for _, layer := range doc.Layers {
		frames := make([]SourceTransformFrame, parentFrames)
		foundLayer := false
		for _, frame := range layer.Frames {
			d := frame.Duration
			if d < 1 {
				d = 1
			}
			for _, inst := range frame.Instances {
				if inst.Library != childLibrary {
					continue
				}
				foundLayer = true
				m := xflMatrix{
					A: inst.Matrix.A, B: inst.Matrix.B,
					C: inst.Matrix.C, D: inst.Matrix.D,
					TX: inst.Matrix.TX, TY: inst.Matrix.TY,
				}
				// Missing a/d attributes decode as zero; XFL defaults them to 1.
				// Distinguish an actually absent attribute by re-reading through the
				// source token parser would be overkill here: all exported child
				// matrices with omitted a/d are pure translations. Treat zero/zero
				// as the XFL identity defaults.
				if m.A == 0 && m.D == 0 && m.B == 0 && m.C == 0 {
					m.A, m.D = 1, 1
				}
				alpha := 1.0
				if inst.Color.Alpha != nil {
					alpha = *inst.Color.Alpha
				}
				for i := frame.Index; i < frame.Index+d && i < parentFrames; i++ {
					frames[i] = SourceTransformFrame{Matrix: m, Alpha: alpha, Valid: true}
				}
				break
			}
		}
		if foundLayer {
			timelines = append(timelines, frames)
		}
	}
	if len(timelines) == 0 {
		return nil, errors.New("XFL: no child timeline " + childLibrary + " in " + parentLibrary)
	}
	return timelines, nil
}

// loadChildTransformTimeline is for symbols where the child occurs on exactly
// one layer. It returns that source layer unchanged.
func loadChildTransformTimeline(parentLibrary, childLibrary string) ([]SourceTransformFrame, error) {
	timelines, err := loadChildTransformTimelines(parentLibrary, childLibrary)
	if err != nil {
		return nil, err
	}
	return timelines[0], nil
}

// legacy token parser retained below only for source constructs not expressible
// by the direct DOM layer parser above.
func loadChildTransformTimelineLegacy(parentLibrary, childLibrary string) ([]SourceTransformFrame, error) {
	libraryDir, err := findOriginalPath("fla", "LIBRARY")
	if err != nil {
		return nil, err
	}
	path := filepath.Join(libraryDir, parentLibrary+".xml")
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	dec := xml.NewDecoder(f)
	type held struct {
		index    int
		duration int
		matrix   xflMatrix
		found    bool
	}
	var current *held
	var inTarget bool
	var targetDepth int
	var keys []held
	maxFrame := 0

	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		switch t := tok.(type) {
		case xml.StartElement:
			switch t.Name.Local {
			case "DOMFrame":
				current = &held{
					index:    intAttr(t.Attr, "index", 0),
					duration: intAttr(t.Attr, "duration", 1),
					matrix:   xflMatrix{A: 1, D: 1},
				}
				if current.duration < 1 {
					current.duration = 1
				}
			case "DOMSymbolInstance":
				if current != nil && !inTarget && stringAttr(t.Attr, "libraryItemName", "") == childLibrary {
					inTarget = true
					targetDepth = 1
					current.found = true
				} else if inTarget {
					targetDepth++
				}
			case "Matrix":
				if current != nil && inTarget {
					current.matrix = xflMatrix{
						A:  floatAttr(t.Attr, "a", 1),
						B:  floatAttr(t.Attr, "b", 0),
						C:  floatAttr(t.Attr, "c", 0),
						D:  floatAttr(t.Attr, "d", 1),
						TX: floatAttr(t.Attr, "tx", 0),
						TY: floatAttr(t.Attr, "ty", 0),
					}
				}
			}
		case xml.EndElement:
			switch t.Name.Local {
			case "DOMSymbolInstance":
				if inTarget {
					targetDepth--
					if targetDepth == 0 {
						inTarget = false
					}
				}
			case "DOMFrame":
				if current != nil {
					if current.found {
						keys = append(keys, *current)
						end := current.index + current.duration
						if end > maxFrame {
							maxFrame = end
						}
					}
					current = nil
				}
			}
		}
	}

	if len(keys) == 0 || maxFrame == 0 {
		return nil, errors.New("XFL: no child timeline " + childLibrary + " in " + parentLibrary)
	}
	frames := make([]SourceTransformFrame, maxFrame)
	for _, key := range keys {
		for i := key.index; i < key.index+key.duration && i < len(frames); i++ {
			frames[i] = SourceTransformFrame{Matrix: key.matrix, Valid: true}
		}
	}
	return frames, nil
}
