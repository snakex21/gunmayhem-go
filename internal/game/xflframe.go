package game

import (
	"encoding/xml"
	"errors"
	"math"
	"os"
	"path/filepath"
)

type xflSymbolDoc struct {
	Timeline xflTimelineDoc `xml:"timeline>DOMTimeline"`
}

type xflTimelineDoc struct {
	Layers []xflLayerDoc `xml:"layers>DOMLayer"`
}

type xflLayerDoc struct {
	Visible string        `xml:"visible,attr"`
	Frames  []xflFrameDoc `xml:"frames>DOMFrame"`
}

type xflFrameDoc struct {
	Index    int            `xml:"index,attr"`
	Duration int            `xml:"duration,attr"`
	Elements xflElementsDoc `xml:"elements"`
}

type xflElementsDoc struct {
	Shapes    []xflShapeDoc    `xml:"DOMShape"`
	Instances []xflInstanceDoc `xml:"DOMSymbolInstance"`
}

type xflShapeDoc struct {
	Matrix  xflMatrixContainer `xml:"matrix"`
	Edges   []xflEdgeDoc       `xml:"edges>Edge"`
	Strokes []xflStrokeDoc     `xml:"strokes>StrokeStyle>SolidStroke"`
}

type xflEdgeDoc struct {
	Value string `xml:"edges,attr"`
}

type xflStrokeDoc struct {
	Weight float64 `xml:"weight,attr"`
}

type xflInstanceDoc struct {
	Library    string             `xml:"libraryItemName,attr"`
	SymbolType string             `xml:"symbolType,attr"`
	FirstFrame int                `xml:"firstFrame,attr"`
	Matrix     xflMatrixContainer `xml:"matrix"`
	Color      xflColorContainer  `xml:"color"`
}

type xflMatrixContainer struct {
	Value xflMatrixDoc `xml:"Matrix"`
}

type xflMatrixDoc struct {
	A  float64 `xml:"a,attr"`
	B  float64 `xml:"b,attr"`
	C  float64 `xml:"c,attr"`
	D  float64 `xml:"d,attr"`
	TX float64 `xml:"tx,attr"`
	TY float64 `xml:"ty,attr"`
}

type xflColorContainer struct {
	Value xflColorDoc `xml:"Color"`
}

type xflColorDoc struct {
	AlphaMultiplier *float64 `xml:"alphaMultiplier,attr"`
}

var frameBoundsCache = map[string]Rect{}
var frameBoundsHave = map[string]bool{}

func sourceFrameVisualBounds(libraryName string, frame int) (Rect, error) {
	libraryDir, err := findOriginalPath("fla", "LIBRARY")
	if err != nil {
		return Rect{}, err
	}
	return sourceFrameVisualBoundsInDir(libraryDir, libraryName, frame)
}

func sourceFrameVisualBoundsInDir(libraryDir, libraryName string, frame int) (Rect, error) {
	return sourceFrameBoundsRecursive(libraryDir, libraryName, frame, map[string]bool{})
}

func sourceFrameBoundsRecursive(libraryDir, libraryName string, frame int, visiting map[string]bool) (Rect, error) {
	key := filepath.Clean(libraryDir) + "|" + libraryName + "#" + itoaFast(frame)
	if frameBoundsHave[key] {
		return frameBoundsCache[key], nil
	}
	if visiting[key] {
		return Rect{}, errors.New("XFL frame symbol cycle at " + key)
	}
	visiting[key] = true
	defer delete(visiting, key)

	path := filepath.Join(libraryDir, libraryName+".xml")
	data, err := os.ReadFile(path)
	if err != nil {
		return Rect{}, err
	}
	var doc xflSymbolDoc
	if err := xml.Unmarshal(data, &doc); err != nil {
		return Rect{}, err
	}

	var result Rect
	have := false
	for _, layer := range doc.Timeline.Layers {
		if layer.Visible == "false" {
			continue
		}
		active, ok := activeXFLFrame(layer.Frames, frame)
		if !ok {
			continue
		}

		for _, shape := range active.Elements.Shapes {
			var edges []string
			for _, edge := range shape.Edges {
				edges = append(edges, edge.Value)
			}
			r, ok := boundsFromEdgeStrings(edges)
			if !ok {
				continue
			}
			maxStroke := 0.0
			for _, stroke := range shape.Strokes {
				if stroke.Weight > maxStroke {
					maxStroke = stroke.Weight
				}
			}
			if maxStroke > 0 {
				pad := maxStroke / 2
				r.X -= pad
				r.Y -= pad
				r.W += 2 * pad
				r.H += 2 * pad
			}
			r = transformRect(r, matrixFromDoc(shape.Matrix.Value))
			result, have = unionOptionalRect(result, have, r)
		}

		for _, inst := range active.Elements.Instances {
			if inst.Library == "" || instanceAlpha(inst) <= 0.0001 {
				continue
			}
			childFrame := 0
			if inst.SymbolType == "graphic" {
				childFrame = inst.FirstFrame
			}
			child, childErr := sourceFrameBoundsRecursive(libraryDir, inst.Library, childFrame, visiting)
			if childErr != nil {
				continue
			}
			child = transformRect(child, matrixFromDoc(inst.Matrix.Value))
			result, have = unionOptionalRect(result, have, child)
		}
	}

	if !have {
		return Rect{}, errors.New("XFL: no visible bounds for " + key)
	}
	frameBoundsCache[key] = result
	frameBoundsHave[key] = true
	return result, nil
}

func activeXFLFrame(frames []xflFrameDoc, frame int) (xflFrameDoc, bool) {
	for _, f := range frames {
		d := f.Duration
		if d <= 0 {
			d = 1
		}
		if frame >= f.Index && frame < f.Index+d {
			return f, true
		}
	}
	return xflFrameDoc{}, false
}

func matrixFromDoc(m xflMatrixDoc) xflMatrix {
	a, d := m.A, m.D
	// Missing XFL matrix attributes mean identity, but encoding/xml leaves zero.
	if a == 0 && m.B == 0 && m.C == 0 && d == 0 {
		a, d = 1, 1
	}
	return xflMatrix{A: a, B: m.B, C: m.C, D: d, TX: m.TX, TY: m.TY}
}

func instanceAlpha(inst xflInstanceDoc) float64 {
	if inst.Color.Value.AlphaMultiplier == nil {
		return 1
	}
	return *inst.Color.Value.AlphaMultiplier
}

func itoaFast(v int) string {
	if v == 0 {
		return "0"
	}
	sign := ""
	if v < 0 {
		sign = "-"
		v = -v
	}
	buf := [24]byte{}
	i := len(buf)
	for v > 0 {
		i--
		buf[i] = byte('0' + v%10)
		v /= 10
	}
	return sign + string(buf[i:])
}

func rectFinite(r Rect) bool {
	return !math.IsInf(r.X, 0) && !math.IsInf(r.Y, 0) && !math.IsNaN(r.X) && !math.IsNaN(r.Y)
}
