package game

import (
	"encoding/xml"
	"errors"
	"image"
	"io"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
)

// SourceRaster keeps the original Flash symbol registration point.
// Bounds are expressed in the symbol's local XFL coordinate system, while
// Image is the untouched FFDec raster for a concrete timeline frame.
// Drawing at world (0,0) therefore means placing the image top-left at
// Bounds.X/Bounds.Y, not centering or trimming it.
type SourceRaster struct {
	Image  *ebiten.Image
	Bounds Rect
}

var symbolBoundsCache = map[string]Rect{}

// Flash's root `_quality` is global. Keep one render-quality state for the
// single-threaded game renderer so every source raster follows Options.
var sourceRenderQuality = 2

func setSourceRenderQuality(quality int) {
	if quality < 1 || quality > 3 {
		quality = 2
	}
	sourceRenderQuality = quality
}

func applySourceRenderQuality(op *ebiten.DrawImageOptions) {
	if op == nil {
		return
	}
	switch sourceRenderQuality {
	case 1:
		op.Filter = ebiten.FilterNearest
	case 2:
		op.Filter = ebiten.FilterLinear
		op.DisableMipmaps = true
	case 3:
		op.Filter = ebiten.FilterLinear
		op.DisableMipmaps = false
	}
}

func loadSourceRaster(libraryName string, frame int, parts ...string) *SourceRaster {
	img := decodeOriginalPNG(parts...)
	if img == nil {
		return nil
	}

	// Recover the raster canvas origin from two source facts:
	// 1) the visible bounds of this exact XFL frame in symbol coordinates,
	// 2) the non-transparent bbox of FFDec's untouched raster of that frame.
	// Their difference is where symbol coordinate (0,0) lies in the PNG.
	if visible, err := sourceFrameVisualBounds(libraryName, frame); err == nil && rectFinite(visible) {
		if alpha, ok := alphaBounds(img); ok {
			originX1 := visible.X - float64(alpha.Min.X)
			originY1 := visible.Y - float64(alpha.Min.Y)
			originX2 := visible.X + visible.W - float64(alpha.Max.X)
			originY2 := visible.Y + visible.H - float64(alpha.Max.Y)
			originX := (originX1 + originX2) / 2
			originY := (originY1 + originY2) / 2
			b := img.Bounds()
			return &SourceRaster{
				Image:  ebiten.NewImageFromImage(img),
				Bounds: Rect{X: originX, Y: originY, W: float64(b.Dx()), H: float64(b.Dy())},
			}
		}
	}

	// Fallback for a symbol whose visual frame cannot be reduced from XFL.
	bounds, err := sourceSymbolCanvasBounds(libraryName)
	if err != nil {
		return nil
	}
	return &SourceRaster{Image: ebiten.NewImageFromImage(img), Bounds: bounds}
}

func alphaBounds(src image.Image) (image.Rectangle, bool) {
	b := src.Bounds()
	minX, minY := b.Max.X, b.Max.Y
	maxX, maxY := b.Min.X-1, b.Min.Y-1
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			_, _, _, a := src.At(x, y).RGBA()
			if a == 0 {
				continue
			}
			if x < minX {
				minX = x
			}
			if y < minY {
				minY = y
			}
			if x > maxX {
				maxX = x
			}
			if y > maxY {
				maxY = y
			}
		}
	}
	if maxX < minX || maxY < minY {
		return image.Rectangle{}, false
	}
	return image.Rect(minX, minY, maxX+1, maxY+1), true
}

func sourceSymbolCanvasBounds(libraryName string) (Rect, error) {
	if r, ok := symbolBoundsCache[libraryName]; ok {
		return r, nil
	}
	libraryDir, err := findOriginalPath("fla", "LIBRARY")
	if err != nil {
		return Rect{}, err
	}
	visiting := map[string]bool{}
	r, err := sourceSymbolBoundsRecursive(libraryDir, libraryName, visiting)
	if err != nil {
		return Rect{}, err
	}
	// Flash symbols always have a meaningful local origin even when all visible
	// artwork lies to one side of it. FFDec keeps that registration point in the
	// exported canvas, so the canvas bounds must include (0,0).
	r = unionRectPoint(r, 0, 0)
	r.X = math.Floor(r.X)
	r.Y = math.Floor(r.Y)
	maxX := math.Ceil(r.X + r.W)
	maxY := math.Ceil(r.Y + r.H)
	r.W = maxX - r.X
	r.H = maxY - r.Y
	symbolBoundsCache[libraryName] = r
	return r, nil
}

func sourceSymbolBoundsRecursive(libraryDir, libraryName string, visiting map[string]bool) (Rect, error) {
	if r, ok := symbolBoundsCache[libraryName]; ok {
		return r, nil
	}
	if visiting[libraryName] {
		return Rect{}, errors.New("XFL symbol cycle at " + libraryName)
	}
	visiting[libraryName] = true
	defer delete(visiting, libraryName)

	path := filepath.Join(libraryDir, libraryName+".xml")
	f, err := os.Open(path)
	if err != nil {
		return Rect{}, err
	}
	defer f.Close()

	dec := xml.NewDecoder(f)
	var result Rect
	haveResult := false

	layerVisible := true
	layerDepth := 0

	type instState struct {
		library string
		matrix  xflMatrix
		alpha   float64
		depth   int
	}
	var inst *instState

	type shapeState struct {
		matrix    xflMatrix
		edges     []string
		maxStroke float64
		depth     int
	}
	var shape *shapeState

	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return Rect{}, err
		}

		switch t := tok.(type) {
		case xml.StartElement:
			switch t.Name.Local {
			case "DOMLayer":
				layerDepth++
				if layerDepth == 1 {
					layerVisible = stringAttr(t.Attr, "visible", "true") != "false"
				}
			case "DOMSymbolInstance":
				if layerVisible && inst == nil && shape == nil {
					inst = &instState{
						library: stringAttr(t.Attr, "libraryItemName", ""),
						matrix:  xflMatrix{A: 1, D: 1},
						alpha:   1,
						depth:   1,
					}
				} else if inst != nil {
					inst.depth++
				}
			case "DOMShape":
				if layerVisible && inst == nil && shape == nil {
					shape = &shapeState{matrix: xflMatrix{A: 1, D: 1}, depth: 1}
				} else if shape != nil {
					shape.depth++
				}
			case "Matrix":
				m := xflMatrix{
					A:  floatAttr(t.Attr, "a", 1),
					B:  floatAttr(t.Attr, "b", 0),
					C:  floatAttr(t.Attr, "c", 0),
					D:  floatAttr(t.Attr, "d", 1),
					TX: floatAttr(t.Attr, "tx", 0),
					TY: floatAttr(t.Attr, "ty", 0),
				}
				if shape != nil {
					shape.matrix = m
				} else if inst != nil {
					inst.matrix = m
				}
			case "Color":
				if inst != nil {
					inst.alpha = floatAttr(t.Attr, "alphaMultiplier", inst.alpha)
				}
			case "Edge":
				if shape != nil {
					shape.edges = append(shape.edges, stringAttr(t.Attr, "edges", ""))
				}
			case "SolidStroke":
				if shape != nil {
					w := floatAttr(t.Attr, "weight", 0)
					if w > shape.maxStroke {
						shape.maxStroke = w
					}
				}
			}

		case xml.EndElement:
			switch t.Name.Local {
			case "DOMSymbolInstance":
				if inst != nil {
					inst.depth--
					if inst.depth == 0 {
						if inst.alpha > 0.0001 && inst.library != "" {
							child, childErr := sourceSymbolBoundsRecursive(libraryDir, inst.library, visiting)
							if childErr == nil {
								child = transformRect(child, inst.matrix)
								result, haveResult = unionOptionalRect(result, haveResult, child)
							}
						}
						inst = nil
					}
				}
			case "DOMShape":
				if shape != nil {
					shape.depth--
					if shape.depth == 0 {
						if r, ok := boundsFromEdgeStrings(shape.edges); ok {
							if shape.maxStroke > 0 {
								pad := shape.maxStroke / 2
								r.X -= pad
								r.Y -= pad
								r.W += pad * 2
								r.H += pad * 2
							}
							r = transformRect(r, shape.matrix)
							result, haveResult = unionOptionalRect(result, haveResult, r)
						}
						shape = nil
					}
				}
			case "DOMLayer":
				if layerDepth > 0 {
					layerDepth--
					if layerDepth == 0 {
						layerVisible = true
					}
				}
			}
		}
	}

	if !haveResult {
		return Rect{X: 0, Y: 0, W: 0, H: 0}, nil
	}
	result = unionRectPoint(result, 0, 0)
	symbolBoundsCache[libraryName] = result
	return result, nil
}

func boundsFromEdgeStrings(edges []string) (Rect, bool) {
	minX, minY := math.Inf(1), math.Inf(1)
	maxX, maxY := math.Inf(-1), math.Inf(-1)
	found := false
	for _, edge := range edges {
		nums := numberRE.FindAllString(edge, -1)
		for i := 0; i+1 < len(nums); i += 2 {
			xi, errX := strconv.Atoi(nums[i])
			yi, errY := strconv.Atoi(nums[i+1])
			if errX != nil || errY != nil {
				continue
			}
			x := float64(xi) / 20
			y := float64(yi) / 20
			minX = math.Min(minX, x)
			maxX = math.Max(maxX, x)
			minY = math.Min(minY, y)
			maxY = math.Max(maxY, y)
			found = true
		}
	}
	if !found {
		return Rect{}, false
	}
	return Rect{X: minX, Y: minY, W: maxX - minX, H: maxY - minY}, true
}

func unionOptionalRect(base Rect, have bool, add Rect) (Rect, bool) {
	if !have {
		return add, true
	}
	minX := math.Min(base.X, add.X)
	minY := math.Min(base.Y, add.Y)
	maxX := math.Max(base.X+base.W, add.X+add.W)
	maxY := math.Max(base.Y+base.H, add.Y+add.H)
	return Rect{X: minX, Y: minY, W: maxX - minX, H: maxY - minY}, true
}

func unionRectPoint(r Rect, x, y float64) Rect {
	minX := math.Min(r.X, x)
	minY := math.Min(r.Y, y)
	maxX := math.Max(r.X+r.W, x)
	maxY := math.Max(r.Y+r.H, y)
	return Rect{X: minX, Y: minY, W: maxX - minX, H: maxY - minY}
}

func sourceRasterImageSize(r *SourceRaster) image.Point {
	if r == nil || r.Image == nil {
		return image.Point{}
	}
	b := r.Image.Bounds()
	return image.Pt(b.Dx(), b.Dy())
}

func drawSourceRaster(dst *ebiten.Image, r *SourceRaster, x, y, scaleX, scaleY, alpha float64) {
	if r == nil || r.Image == nil {
		return
	}
	op := &ebiten.DrawImageOptions{}
	applySourceRenderQuality(op)
	op.GeoM.Translate(r.Bounds.X, r.Bounds.Y)
	op.GeoM.Scale(scaleX, scaleY)
	op.GeoM.Translate(x, y)
	if alpha < 1 {
		op.ColorScale.ScaleAlpha(float32(alpha))
	}
	dst.DrawImage(r.Image, op)
}

func drawSourceRasterRot(dst *ebiten.Image, r *SourceRaster, x, y, scaleX, scaleY, degrees, alpha float64) {
	if r == nil || r.Image == nil {
		return
	}
	op := &ebiten.DrawImageOptions{}
	applySourceRenderQuality(op)
	op.GeoM.Translate(r.Bounds.X, r.Bounds.Y)
	op.GeoM.Scale(scaleX, scaleY)
	op.GeoM.Rotate(degrees * math.Pi / 180)
	op.GeoM.Translate(x, y)
	if alpha < 1 {
		op.ColorScale.ScaleAlpha(float32(alpha))
	}
	dst.DrawImage(r.Image, op)
}

func sourceLibraryNameFromSpriteDir(spriteDir string) string {
	base := filepath.Base(spriteDir)
	if !strings.HasPrefix(base, "DefineSprite_") {
		return ""
	}
	rest := strings.TrimPrefix(base, "DefineSprite_")
	if i := strings.IndexByte(rest, '_'); i >= 0 {
		rest = rest[:i]
	}
	if _, err := strconv.Atoi(rest); err != nil {
		return ""
	}
	return "Symbol " + rest
}
