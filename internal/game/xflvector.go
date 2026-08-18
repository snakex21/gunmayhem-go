package game

import (
	"encoding/xml"
	"errors"
	"fmt"
	"image"
	"image/color"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	evector "github.com/hajimehoshi/ebiten/v2/vector"
	xvector "golang.org/x/image/vector"
)

type xflVectorDoc struct {
	Timeline xflVectorTimeline `xml:"timeline>DOMTimeline"`
}

type xflVectorTimeline struct {
	Layers []xflVectorLayer `xml:"layers>DOMLayer"`
}

type xflVectorLayer struct {
	Visible string           `xml:"visible,attr"`
	Frames  []xflVectorFrame `xml:"frames>DOMFrame"`
}

type xflVectorFrame struct {
	Index    int               `xml:"index,attr"`
	Duration int               `xml:"duration,attr"`
	Elements xflVectorElements `xml:"elements"`
}

type xflVectorElements struct {
	Shapes    []xflVectorShape    `xml:"DOMShape"`
	Instances []xflVectorInstance `xml:"DOMSymbolInstance"`
}

type xflVectorShape struct {
	Matrix  xflVectorMatrixContainer `xml:"matrix"`
	Fills   []xflVectorFillStyle     `xml:"fills>FillStyle"`
	Strokes []xflVectorStrokeStyle   `xml:"strokes>StrokeStyle"`
	Edges   []xflVectorEdge          `xml:"edges>Edge"`
}

type xflVectorMatrixContainer struct {
	Value xflVectorMatrix `xml:"Matrix"`
}

type xflVectorMatrix struct {
	A  string `xml:"a,attr"`
	B  string `xml:"b,attr"`
	C  string `xml:"c,attr"`
	D  string `xml:"d,attr"`
	TX string `xml:"tx,attr"`
	TY string `xml:"ty,attr"`
}

type xflVectorFillStyle struct {
	Index  int                `xml:"index,attr"`
	Solid  *xflVectorColor    `xml:"SolidColor"`
	Linear *xflVectorGradient `xml:"LinearGradient"`
	Radial *xflVectorGradient `xml:"RadialGradient"`
}

type xflVectorGradient struct {
	Matrix  xflVectorMatrixContainer `xml:"matrix"`
	Entries []xflVectorGradientEntry `xml:"GradientEntry"`
}

type xflVectorGradientEntry struct {
	Color string `xml:"color,attr"`
	Alpha string `xml:"alpha,attr"`
	Ratio string `xml:"ratio,attr"`
}

type xflGradientPaint struct {
	Matrix  xflMatrix
	Entries []xflVectorGradientEntry
	Radial  bool
}

type xflVectorStrokeStyle struct {
	Index int              `xml:"index,attr"`
	Solid *xflVectorStroke `xml:"SolidStroke"`
}

type xflVectorStroke struct {
	Weight float64        `xml:"weight,attr"`
	Fill   xflVectorColor `xml:"fill>SolidColor"`
}

type xflVectorColor struct {
	Color string `xml:"color,attr"`
	Alpha string `xml:"alpha,attr"`
}

type xflVectorEdge struct {
	Fill0  int    `xml:"fillStyle0,attr"`
	Fill1  int    `xml:"fillStyle1,attr"`
	Stroke int    `xml:"strokeStyle,attr"`
	Data   string `xml:"edges,attr"`
}

type xflVectorInstance struct {
	Library    string                   `xml:"libraryItemName,attr"`
	SymbolType string                   `xml:"symbolType,attr"`
	FirstFrame int                      `xml:"firstFrame,attr"`
	Matrix     xflVectorMatrixContainer `xml:"matrix"`
	Color      xflVectorColorContainer  `xml:"color"`
}

type xflVectorColorContainer struct {
	Value struct {
		Alpha string `xml:"alphaMultiplier,attr"`
	} `xml:"Color"`
}

type vectorPoint struct{ X, Y float64 }

type vectorSegment struct {
	Start, End vectorPoint
	Control    vectorPoint
	Curve      bool
}

// renderSolidXFLFrame rasterizes a source frame directly from XFL vector data.
// It intentionally accepts only solid fills/strokes. The map foreground symbols
// 1377..1388 and the frame-13 shape in 1390 satisfy this exactly.
func renderSolidXFLFrame(libraryName string, frame int) (*SourceRaster, error) {
	bounds, err := sourceFrameVisualBounds(libraryName, frame)
	if err != nil {
		return nil, err
	}
	minX := math.Floor(bounds.X)
	minY := math.Floor(bounds.Y)
	maxX := math.Ceil(bounds.X + bounds.W)
	maxY := math.Ceil(bounds.Y + bounds.H)
	w := int(maxX - minX)
	h := int(maxY - minY)
	if w <= 0 || h <= 0 {
		return nil, errors.New("XFL vector frame has empty bounds")
	}

	img := ebiten.NewImage(w, h)
	root := xflMatrix{A: 1, D: 1, TX: -minX, TY: -minY}
	if err := drawSolidXFLFrame(img, libraryName, frame, root, 1, map[string]bool{}); err != nil {
		return nil, err
	}
	return &SourceRaster{Image: img, Bounds: Rect{X: minX, Y: minY, W: float64(w), H: float64(h)}}, nil
}

// renderSolidXFLFrameShapesOnly renders only shapes owned directly by the
// requested symbol/frame and deliberately skips nested symbol instances. This
// is useful for source overlays such as Symbol913 frame2: the white locked-gun
// mask is a direct shape, while its nested Symbol595 dropgun must not be
// flattened again (FFDec exposes helper/control-point artwork there).
func renderSolidXFLFrameShapesOnly(libraryName string, frame int) (*SourceRaster, error) {
	bounds, err := sourceFrameVisualBounds(libraryName, frame)
	if err != nil {
		return nil, err
	}
	minX := math.Floor(bounds.X)
	minY := math.Floor(bounds.Y)
	maxX := math.Ceil(bounds.X + bounds.W)
	maxY := math.Ceil(bounds.Y + bounds.H)
	w := int(maxX - minX)
	h := int(maxY - minY)
	if w <= 0 || h <= 0 {
		return nil, errors.New("XFL vector frame has empty bounds")
	}
	doc, err := loadXFLVectorDoc(libraryName)
	if err != nil {
		return nil, err
	}
	img := ebiten.NewImage(w, h)
	root := xflMatrix{A: 1, D: 1, TX: -minX, TY: -minY}
	for li := len(doc.Timeline.Layers) - 1; li >= 0; li-- {
		layer := doc.Timeline.Layers[li]
		if layer.Visible == "false" {
			continue
		}
		active, ok := activeVectorFrame(layer.Frames, frame)
		if !ok {
			continue
		}
		for _, shape := range active.Elements.Shapes {
			if err := drawSolidXFLShape(img, shape, root, 1); err != nil {
				return nil, fmt.Errorf("%s frame %d direct shape: %w", libraryName, frame, err)
			}
		}
	}
	return &SourceRaster{Image: img, Bounds: Rect{X: minX, Y: minY, W: float64(w), H: float64(h)}}, nil
}

func drawSolidXFLFrame(dst *ebiten.Image, libraryName string, frame int, parent xflMatrix, alpha float64, visiting map[string]bool) error {
	key := fmt.Sprintf("%s#%d", libraryName, frame)
	if visiting[key] {
		return errors.New("XFL vector cycle at " + key)
	}
	visiting[key] = true
	defer delete(visiting, key)

	doc, err := loadXFLVectorDoc(libraryName)
	if err != nil {
		return err
	}

	// XFL stores the visually topmost layer first. Draw back-to-front.
	for li := len(doc.Timeline.Layers) - 1; li >= 0; li-- {
		layer := doc.Timeline.Layers[li]
		if layer.Visible == "false" {
			continue
		}
		active, ok := activeVectorFrame(layer.Frames, frame)
		if !ok {
			continue
		}

		// DOM element order is back-to-front inside a layer.
		for _, shape := range active.Elements.Shapes {
			if err := drawSolidXFLShape(dst, shape, parent, alpha); err != nil {
				return fmt.Errorf("%s frame %d: %w", libraryName, frame, err)
			}
		}
		for _, inst := range active.Elements.Instances {
			if inst.Library == "" {
				continue
			}
			childFrame := 0
			if inst.SymbolType == "graphic" {
				childFrame = inst.FirstFrame
			}
			childMatrix := multiplyXFLMatrix(parent, matrixFromVector(inst.Matrix.Value))
			childAlpha := alpha * vectorAlphaMultiplier(inst.Color.Value.Alpha)
			if childAlpha <= 0.0001 {
				continue
			}
			if err := drawSolidXFLFrame(dst, inst.Library, childFrame, childMatrix, childAlpha, visiting); err != nil {
				return err
			}
		}
	}
	return nil
}

func loadXFLVectorDoc(libraryName string) (xflVectorDoc, error) {
	libraryDir, err := findOriginalPath("fla", "LIBRARY")
	if err != nil {
		return xflVectorDoc{}, err
	}
	data, err := os.ReadFile(filepath.Join(libraryDir, libraryName+".xml"))
	if err != nil {
		return xflVectorDoc{}, err
	}
	var doc xflVectorDoc
	if err := xml.Unmarshal(data, &doc); err != nil {
		return xflVectorDoc{}, err
	}
	return doc, nil
}

func activeVectorFrame(frames []xflVectorFrame, frame int) (xflVectorFrame, bool) {
	for _, f := range frames {
		d := f.Duration
		if d <= 0 {
			d = 1
		}
		if frame >= f.Index && frame < f.Index+d {
			return f, true
		}
	}
	return xflVectorFrame{}, false
}

func drawSolidXFLShape(dst *ebiten.Image, shape xflVectorShape, parent xflMatrix, alpha float64) error {
	m := multiplyXFLMatrix(parent, matrixFromVector(shape.Matrix.Value))
	solids := make(map[int]color.NRGBA, len(shape.Fills))
	gradients := make(map[int]xflGradientPaint, len(shape.Fills))
	for _, fill := range shape.Fills {
		if fill.Index == 0 {
			continue
		}
		switch {
		case fill.Solid != nil:
			solids[fill.Index] = parseVectorColor(*fill.Solid, alpha)
		case fill.Linear != nil:
			gradients[fill.Index] = xflGradientPaint{
				Matrix:  multiplyXFLMatrix(m, matrixFromVector(fill.Linear.Matrix.Value)),
				Entries: fill.Linear.Entries,
			}
		case fill.Radial != nil:
			gradients[fill.Index] = xflGradientPaint{
				Matrix:  multiplyXFLMatrix(m, matrixFromVector(fill.Radial.Matrix.Value)),
				Entries: fill.Radial.Entries,
				Radial:  true,
			}
		default:
			return fmt.Errorf("unsupported fill style %d", fill.Index)
		}
	}
	strokes := make(map[int]struct {
		Color color.NRGBA
		Width float64
	}, len(shape.Strokes))
	for _, stroke := range shape.Strokes {
		if stroke.Index == 0 || stroke.Solid == nil {
			continue
		}
		strokes[stroke.Index] = struct {
			Color color.NRGBA
			Width float64
		}{parseVectorColor(stroke.Solid.Fill, alpha), stroke.Solid.Weight}
	}

	fillSegments := map[int][]vectorSegment{}
	strokeSegments := map[int][]vectorSegment{}
	for _, edge := range shape.Edges {
		segments, err := parseXFLEdgeSegments(edge.Data)
		if err != nil {
			return err
		}
		if edge.Fill1 != 0 {
			fillSegments[edge.Fill1] = append(fillSegments[edge.Fill1], segments...)
		}
		if edge.Fill0 != 0 {
			for i := len(segments) - 1; i >= 0; i-- {
				fillSegments[edge.Fill0] = append(fillSegments[edge.Fill0], reverseVectorSegment(segments[i]))
			}
		}
		if edge.Stroke != 0 {
			strokeSegments[edge.Stroke] = append(strokeSegments[edge.Stroke], segments...)
		}
	}

	for index, segments := range fillSegments {
		if clr, ok := solids[index]; ok {
			if clr.A == 0 {
				continue
			}
			path := buildStitchedVectorPath(segments, m, true)
			op := &evector.DrawPathOptions{AntiAlias: true}
			op.ColorScale.ScaleWithColor(clr)
			evector.FillPath(dst, path, &evector.FillOptions{FillRule: evector.FillRuleNonZero}, op)
			continue
		}
		if gradient, ok := gradients[index]; ok {
			if err := drawGradientXFLFill(dst, segments, m, gradient, alpha); err != nil {
				return err
			}
		}
	}
	for index, segments := range strokeSegments {
		stroke, ok := strokes[index]
		if !ok || stroke.Color.A == 0 || stroke.Width <= 0 {
			continue
		}
		path := buildStitchedVectorPath(segments, m, false)
		op := &evector.DrawPathOptions{AntiAlias: true}
		op.ColorScale.ScaleWithColor(stroke.Color)
		evector.StrokePath(dst, path, &evector.StrokeOptions{Width: float32(stroke.Width)}, op)
	}
	return nil
}

func drawGradientXFLFill(dst *ebiten.Image, segments []vectorSegment, shapeMatrix xflMatrix, gradient xflGradientPaint, alpha float64) error {
	if len(segments) == 0 || len(gradient.Entries) == 0 {
		return nil
	}
	rect, ok := transformedSegmentsBounds(segments, shapeMatrix, dst.Bounds())
	if !ok || rect.Empty() {
		return nil
	}

	mask := image.NewAlpha(image.Rect(0, 0, rect.Dx(), rect.Dy()))
	raster := xvector.NewRasterizer(rect.Dx(), rect.Dy())
	appendSegmentsToRasterizer(raster, segments, shapeMatrix, float64(rect.Min.X), float64(rect.Min.Y), true)
	raster.Draw(mask, mask.Bounds(), image.NewUniform(color.Alpha{A: 255}), image.Point{})

	paint := image.NewNRGBA(mask.Bounds())
	for y := 0; y < rect.Dy(); y++ {
		for x := 0; x < rect.Dx(); x++ {
			ma := mask.AlphaAt(x, y).A
			if ma == 0 {
				continue
			}
			wx := float64(rect.Min.X+x) + 0.5
			wy := float64(rect.Min.Y+y) + 0.5
			clr := sampleXFLGradient(gradient, wx, wy, alpha)
			clr.A = uint8((uint16(clr.A) * uint16(ma)) / 255)
			paint.SetNRGBA(x, y, clr)
		}
	}

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(rect.Min.X), float64(rect.Min.Y))
	dst.DrawImage(ebiten.NewImageFromImage(paint), op)
	return nil
}

func transformedSegmentsBounds(segments []vectorSegment, m xflMatrix, clip image.Rectangle) (image.Rectangle, bool) {
	minX, minY := math.Inf(1), math.Inf(1)
	maxX, maxY := math.Inf(-1), math.Inf(-1)
	add := func(p vectorPoint) {
		p = transformVectorPoint(p, m)
		minX = math.Min(minX, p.X)
		minY = math.Min(minY, p.Y)
		maxX = math.Max(maxX, p.X)
		maxY = math.Max(maxY, p.Y)
	}
	for _, seg := range segments {
		add(seg.Start)
		add(seg.End)
		if seg.Curve {
			add(seg.Control)
		}
	}
	if math.IsInf(minX, 0) || math.IsInf(minY, 0) {
		return image.Rectangle{}, false
	}
	r := image.Rect(int(math.Floor(minX))-2, int(math.Floor(minY))-2, int(math.Ceil(maxX))+3, int(math.Ceil(maxY))+3)
	r = r.Intersect(clip)
	return r, !r.Empty()
}

func appendSegmentsToRasterizer(r *xvector.Rasterizer, segments []vectorSegment, m xflMatrix, offsetX, offsetY float64, closeContours bool) {
	unused := append([]vectorSegment(nil), segments...)
	for len(unused) > 0 {
		seg := unused[0]
		unused = unused[1:]
		start := seg.Start
		current := seg.End
		move := transformVectorPoint(start, m)
		r.MoveTo(float32(move.X-offsetX), float32(move.Y-offsetY))
		appendRasterSegment(r, seg, m, offsetX, offsetY)

		for len(unused) > 0 && !sameVectorPoint(current, start) {
			found := -1
			for i, candidate := range unused {
				if sameVectorPoint(candidate.Start, current) {
					found = i
					break
				}
			}
			if found < 0 {
				break
			}
			next := unused[found]
			unused = append(unused[:found], unused[found+1:]...)
			appendRasterSegment(r, next, m, offsetX, offsetY)
			current = next.End
		}
		if closeContours && sameVectorPoint(current, start) {
			r.ClosePath()
		}
	}
}

func appendRasterSegment(r *xvector.Rasterizer, seg vectorSegment, m xflMatrix, offsetX, offsetY float64) {
	end := transformVectorPoint(seg.End, m)
	if seg.Curve {
		control := transformVectorPoint(seg.Control, m)
		r.QuadTo(float32(control.X-offsetX), float32(control.Y-offsetY), float32(end.X-offsetX), float32(end.Y-offsetY))
		return
	}
	r.LineTo(float32(end.X-offsetX), float32(end.Y-offsetY))
}

func sampleXFLGradient(g xflGradientPaint, x, y, parentAlpha float64) color.NRGBA {
	gx, gy, ok := inverseTransformPoint(x, y, g.Matrix)
	if !ok {
		return color.NRGBA{}
	}
	const gradientRadius = 819.2
	var t float64
	if g.Radial {
		t = math.Hypot(gx, gy) / gradientRadius
	} else {
		t = (gx + gradientRadius) / (gradientRadius * 2)
	}
	if t < 0 {
		t = 0
	}
	if t > 1 {
		t = 1
	}

	firstRatio := sourceFloatDefault(g.Entries[0].Ratio, 0)
	if t <= firstRatio || len(g.Entries) == 1 {
		return parseVectorColor(xflVectorColor{Color: g.Entries[0].Color, Alpha: g.Entries[0].Alpha}, parentAlpha)
	}
	for i := 1; i < len(g.Entries); i++ {
		left := g.Entries[i-1]
		right := g.Entries[i]
		lr := sourceFloatDefault(left.Ratio, 0)
		rr := sourceFloatDefault(right.Ratio, 1)
		if t > rr && i != len(g.Entries)-1 {
			continue
		}
		mix := 0.0
		if rr > lr {
			mix = (t - lr) / (rr - lr)
		}
		if mix < 0 {
			mix = 0
		}
		if mix > 1 {
			mix = 1
		}
		a := parseVectorColor(xflVectorColor{Color: left.Color, Alpha: left.Alpha}, parentAlpha)
		b := parseVectorColor(xflVectorColor{Color: right.Color, Alpha: right.Alpha}, parentAlpha)
		return lerpNRGBA(a, b, mix)
	}
	last := g.Entries[len(g.Entries)-1]
	return parseVectorColor(xflVectorColor{Color: last.Color, Alpha: last.Alpha}, parentAlpha)
}

func inverseTransformPoint(x, y float64, m xflMatrix) (float64, float64, bool) {
	det := m.A*m.D - m.B*m.C
	if math.Abs(det) < 1e-12 {
		return 0, 0, false
	}
	dx := x - m.TX
	dy := y - m.TY
	return (m.D*dx - m.C*dy) / det, (-m.B*dx + m.A*dy) / det, true
}

func lerpNRGBA(a, b color.NRGBA, t float64) color.NRGBA {
	lerp := func(x, y uint8) uint8 {
		return uint8(math.Round(float64(x) + (float64(y)-float64(x))*t))
	}
	return color.NRGBA{R: lerp(a.R, b.R), G: lerp(a.G, b.G), B: lerp(a.B, b.B), A: lerp(a.A, b.A)}
}

func parseXFLEdgeSegments(data string) ([]vectorSegment, error) {
	tokens := strings.Fields(strings.NewReplacer("!", " ! ", "|", " | ", "[", " [ ").Replace(data))
	var segments []vectorSegment
	var current vectorPoint
	haveCurrent := false
	for i := 0; i < len(tokens); {
		switch tokens[i] {
		case "!":
			if i+2 >= len(tokens) {
				return nil, errors.New("truncated XFL move edge")
			}
			x, err1 := parseXFLCoordinate(tokens[i+1])
			y, err2 := parseXFLCoordinate(tokens[i+2])
			if err1 != nil || err2 != nil {
				return nil, errors.New("invalid XFL move coordinate")
			}
			current = vectorPoint{X: x / 20, Y: y / 20}
			haveCurrent = true
			i += 3
		case "|":
			if !haveCurrent || i+2 >= len(tokens) {
				return nil, errors.New("invalid XFL line edge")
			}
			x, err1 := parseXFLCoordinate(tokens[i+1])
			y, err2 := parseXFLCoordinate(tokens[i+2])
			if err1 != nil || err2 != nil {
				return nil, errors.New("invalid XFL line coordinate")
			}
			end := vectorPoint{X: x / 20, Y: y / 20}
			segments = append(segments, vectorSegment{Start: current, End: end})
			current = end
			i += 3
		case "[":
			if !haveCurrent || i+4 >= len(tokens) {
				return nil, errors.New("invalid XFL curve edge")
			}
			cx, e1 := parseXFLCoordinate(tokens[i+1])
			cy, e2 := parseXFLCoordinate(tokens[i+2])
			x, e3 := parseXFLCoordinate(tokens[i+3])
			y, e4 := parseXFLCoordinate(tokens[i+4])
			if e1 != nil || e2 != nil || e3 != nil || e4 != nil {
				return nil, errors.New("invalid XFL curve coordinate")
			}
			end := vectorPoint{X: x / 20, Y: y / 20}
			segments = append(segments, vectorSegment{
				Start: current, End: end, Curve: true,
				Control: vectorPoint{X: cx / 20, Y: cy / 20},
			})
			current = end
			i += 5
		default:
			// XFL may use compact coordinate tokens around command boundaries.
			// Anything else means this renderer would not be source-exact.
			return nil, fmt.Errorf("unsupported XFL edge token %q", tokens[i])
		}
	}
	return segments, nil
}

func parseXFLCoordinate(token string) (float64, error) {
	if !strings.HasPrefix(token, "#") {
		return strconv.ParseFloat(token, 64)
	}

	// Animate/XFL also emits coordinates as hexadecimal signed fixed-point
	// numbers, e.g. #30FB.9C or #FFFE2B.42. Removing the dot yields the
	// two's-complement 24.8 value used by Flash; divide the signed value by 256.
	hex := strings.ReplaceAll(strings.TrimPrefix(token, "#"), ".", "")
	if hex == "" || len(hex) > 8 {
		return 0, fmt.Errorf("invalid XFL hexadecimal coordinate %q", token)
	}
	u, err := strconv.ParseUint(hex, 16, 32)
	if err != nil {
		return 0, err
	}
	return float64(int32(uint32(u))) / 256, nil
}

func buildStitchedVectorPath(segments []vectorSegment, m xflMatrix, closeContours bool) *evector.Path {
	path := &evector.Path{}
	unused := append([]vectorSegment(nil), segments...)
	for len(unused) > 0 {
		seg := unused[0]
		unused = unused[1:]
		start := seg.Start
		current := seg.End
		move := transformVectorPoint(start, m)
		path.MoveTo(float32(move.X), float32(move.Y))
		appendVectorSegment(path, seg, m)

		for len(unused) > 0 && !sameVectorPoint(current, start) {
			found := -1
			for i, candidate := range unused {
				if sameVectorPoint(candidate.Start, current) {
					found = i
					break
				}
			}
			if found < 0 {
				break
			}
			next := unused[found]
			unused = append(unused[:found], unused[found+1:]...)
			appendVectorSegment(path, next, m)
			current = next.End
		}
		if closeContours && sameVectorPoint(current, start) {
			path.Close()
		}
	}
	return path
}

func appendVectorSegment(path *evector.Path, seg vectorSegment, m xflMatrix) {
	end := transformVectorPoint(seg.End, m)
	if seg.Curve {
		control := transformVectorPoint(seg.Control, m)
		path.QuadTo(float32(control.X), float32(control.Y), float32(end.X), float32(end.Y))
		return
	}
	path.LineTo(float32(end.X), float32(end.Y))
}

func reverseVectorSegment(seg vectorSegment) vectorSegment {
	seg.Start, seg.End = seg.End, seg.Start
	return seg
}

func sameVectorPoint(a, b vectorPoint) bool {
	return math.Abs(a.X-b.X) < 0.00001 && math.Abs(a.Y-b.Y) < 0.00001
}

func transformVectorPoint(p vectorPoint, m xflMatrix) vectorPoint {
	return vectorPoint{X: p.X*m.A + p.Y*m.C + m.TX, Y: p.X*m.B + p.Y*m.D + m.TY}
}

func multiplyXFLMatrix(parent, child xflMatrix) xflMatrix {
	return xflMatrix{
		A:  parent.A*child.A + parent.C*child.B,
		B:  parent.B*child.A + parent.D*child.B,
		C:  parent.A*child.C + parent.C*child.D,
		D:  parent.B*child.C + parent.D*child.D,
		TX: parent.A*child.TX + parent.C*child.TY + parent.TX,
		TY: parent.B*child.TX + parent.D*child.TY + parent.TY,
	}
}

func matrixFromVector(m xflVectorMatrix) xflMatrix {
	return xflMatrix{
		A:  sourceFloatDefault(m.A, 1),
		B:  sourceFloatDefault(m.B, 0),
		C:  sourceFloatDefault(m.C, 0),
		D:  sourceFloatDefault(m.D, 1),
		TX: sourceFloatDefault(m.TX, 0),
		TY: sourceFloatDefault(m.TY, 0),
	}
}

func vectorAlphaMultiplier(s string) float64 {
	return sourceFloatDefault(s, 1)
}

func parseVectorColor(c xflVectorColor, parentAlpha float64) color.NRGBA {
	hex := strings.TrimPrefix(c.Color, "#")
	if hex == "" {
		hex = "000000"
	}
	v, _ := strconv.ParseUint(hex, 16, 32)
	a := sourceFloatDefault(c.Alpha, 1) * parentAlpha
	if a < 0 {
		a = 0
	}
	if a > 1 {
		a = 1
	}
	return color.NRGBA{R: uint8(v >> 16), G: uint8(v >> 8), B: uint8(v), A: uint8(math.Round(a * 255))}
}
