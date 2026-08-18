package game

import (
	"encoding/xml"
	"errors"
	"os"
	"path/filepath"
	"strconv"
)

type SourceVisualFrame struct {
	Matrix xflMatrix
	Alpha  float64
	Valid  bool
}

// loadChildVisualTimeline expands one direct XFL child instance into frame-by-
// frame transform + color alpha state. It is used where drawing the child raster
// alone is insufficient (e.g. Symbol 696 shield tween alphaMultiplier).
func loadChildVisualTimeline(parentLibrary, childLibrary string) ([]SourceVisualFrame, error) {
	libraryDir, err := findOriginalPath("fla", "LIBRARY")
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(filepath.Join(libraryDir, parentLibrary+".xml"))
	if err != nil {
		return nil, err
	}
	type matrixXML struct {
		A  string `xml:"a,attr"`
		B  string `xml:"b,attr"`
		C  string `xml:"c,attr"`
		D  string `xml:"d,attr"`
		TX string `xml:"tx,attr"`
		TY string `xml:"ty,attr"`
	}
	type colorXML struct {
		Alpha string `xml:"alphaMultiplier,attr"`
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
	maxFrame := 0
	for _, layer := range doc.Layers {
		for _, f := range layer.Frames {
			d := f.Duration
			if d < 1 {
				d = 1
			}
			if f.Index+d > maxFrame {
				maxFrame = f.Index + d
			}
		}
	}
	frames := make([]SourceVisualFrame, maxFrame)
	found := false
	for _, layer := range doc.Layers {
		for _, f := range layer.Frames {
			d := f.Duration
			if d < 1 {
				d = 1
			}
			for _, inst := range f.Instances {
				if inst.Library != childLibrary {
					continue
				}
				found = true
				v := SourceVisualFrame{
					Matrix: xflMatrix{
						A:  sourceFloatDefault(inst.Matrix.A, 1),
						B:  sourceFloatDefault(inst.Matrix.B, 0),
						C:  sourceFloatDefault(inst.Matrix.C, 0),
						D:  sourceFloatDefault(inst.Matrix.D, 1),
						TX: sourceFloatDefault(inst.Matrix.TX, 0),
						TY: sourceFloatDefault(inst.Matrix.TY, 0),
					},
					Alpha: sourceFloatDefault(inst.Color.Alpha, 1),
					Valid: true,
				}
				for i := f.Index; i < f.Index+d && i < len(frames); i++ {
					frames[i] = v
				}
				break
			}
		}
	}
	if !found {
		return nil, errors.New("XFL: no visual child " + childLibrary + " in " + parentLibrary)
	}
	return frames, nil
}

func sourceFloatDefault(s string, fallback float64) float64 {
	if s == "" {
		return fallback
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return fallback
	}
	return v
}
