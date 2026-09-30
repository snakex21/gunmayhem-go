package game

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/font/sfnt"
	"golang.org/x/image/math/fixed"
)

var legacySourceFontFixtures = []struct {
	file, normalizedSHA256, ascii, label string
	glyphs                               int
	advance                              fixed.Int26_6
}{
	{"1006_Century Gothic.ttf", "7bc1377ddae8f2928a92233f0232af873c6e64a22aff1af75888f63eb87b23c5",
		" %-.0123456789", "0123456789", 17, 110*64 + 60},
	{"10_Tw Cen MT.ttf", "d8547d35c67b8b744afc7584eee33762f70153ba3e1cff0e3f66dce2ee815d5a",
		" !'()-124ABCDEFGHIKLMNOPRSTUVWYabcdefghiklmnopqrstuvwxy", "LEVEL UP", 58, 69*64 + 17},
	{"18_Showcard Gothic.ttf", "2f9ab2b86e044e2143247e72d9401a106e781cc4c987c41992cedd5fc91e8691",
		" '(.0123456789:ABCDEFGHIJKLMNOPQRSTUVWXYZadeilmostuv", "GAME OVER", 55, 109*64 + 20},
	{"20_Tw Cen MT Condensed Extra Bold.ttf", "c6c48de5528579ca556510ab968dea1ab99fad80ea6eae20f45530b59e99f261",
		" !\"#$%&'()*+,-./0123456789:;<=>?@ABCDEFGHIJKLMNOPQRSTUVWXYZ[\\]^_`abcdefghijklmnopqrstuvwxyz{|}~", "KNOCKBACK RESISTANCE", 118, 188*64 + 11},
	{"49_Tw Cen MT Condensed.ttf", "1d694250debc4fe2e94f1b4e661c620780dbe8c328be1048b78365bdfcd620d5",
		" !(),-./0123456789:?ABCDEFGHIJKLMNOPQRSTUVWXYabcdefghijklmnopqrstuvwxyz", "KNOCKBACK RESISTANCE", 74, 143*64 + 32},
}

func readLegacySourceFontFixture(t *testing.T, file string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", "assets", "fonts", file))
	if err != nil {
		t.Fatal(err)
	}
	return data
}

// Independent specification check: read padded words through binary.Read,
// rather than calling the production checksum helper.
func specFontChecksum(data []byte) uint32 {
	padded := append(bytes.Clone(data), make([]byte, (4-len(data)%4)%4)...)
	reader := bytes.NewReader(padded)
	var sum, word uint32
	for reader.Len() != 0 {
		if err := binary.Read(reader, binary.BigEndian, &word); err != nil {
			panic(err)
		}
		sum += word
	}
	return sum
}

func TestLegacySourceFontsNormalizeActualAssets(t *testing.T) {
	for _, fixture := range legacySourceFontFixtures {
		t.Run(fixture.file, func(t *testing.T) {
			data := readLegacySourceFontFixture(t, fixture.file)
			original := bytes.Clone(data)
			if _, err := opentype.Parse(data); err == nil {
				t.Fatal("fixture no longer demonstrates the legacy parse failure")
			}
			normalized, ok := normalizeLegacySourceFont(data)
			if !ok || len(normalized) != len(data) || &normalized[0] == &data[0] {
				t.Fatal("expected separate normalized copy of identical length")
			}
			// Goldens generated independently with Python struct.unpack('>I')
			// word sums after the two length edits and standard checksum steps.
			if got := fmt.Sprintf("%x", sha256.Sum256(normalized)); got != fixture.normalizedSHA256 {
				t.Fatalf("normalized digest %s, want %s", got, fixture.normalizedSHA256)
			}
			tables, ok := legacySourceFontTables(data)
			if !ok {
				t.Fatal("actual legacy directory rejected")
			}
			allowedChanges := make([]bool, len(data))
			for _, table := range tables {
				for i := table.record + 4; i < table.record+8; i++ {
					allowedChanges[i] = true // directory checksum metadata
				}
				if table.tag == "head" || table.tag == "hmtx" {
					for i := table.record + 12; i < table.record+16; i++ {
						allowedChanges[i] = true // the two known lengths
					}
				}
				if table.tag == "head" {
					for i := table.offset + 8; i < table.offset+12; i++ {
						allowedChanges[i] = true // checksumAdjustment
					}
				}
				length := int(binary.BigEndian.Uint32(normalized[table.record+12:]))
				payload := bytes.Clone(normalized[table.offset : table.offset+length])
				if table.tag == "head" {
					clear(payload[8:12])
				}
				stored := binary.BigEndian.Uint32(normalized[table.record+4:])
				if checksum := specFontChecksum(payload); checksum != stored {
					t.Fatalf("%s checksum %08x, stored %08x", table.tag, checksum, stored)
				}
			}
			for i := range data {
				if !allowedChanges[i] && normalized[i] != data[i] {
					t.Fatalf("changed payload or unrelated metadata at byte %d", i)
				}
			}
			if got := specFontChecksum(normalized); got != 0xb1b0afba {
				t.Fatalf("whole-font checksum %08x", got)
			}
			parsed, err := opentype.Parse(normalized)
			if err != nil {
				t.Fatalf("ordinary parser rejected normalized actual font: %v", err)
			}
			if parsed.NumGlyphs() != fixture.glyphs {
				t.Fatalf("glyph count %d, want %d", parsed.NumGlyphs(), fixture.glyphs)
			}
			if _, err := parseSourceFont(data); err != nil {
				t.Fatalf("runtime path: %v", err)
			}
			// A second pass is already a normal font, not another normalization.
			if _, ok := normalizeLegacySourceFont(normalized); ok {
				t.Fatal("normalized copy unexpectedly re-entered compatibility path")
			}
			if _, err := parseSourceFont(normalized); err != nil {
				t.Fatalf("ordinary parse on second pass: %v", err)
			}
			if !bytes.Equal(data, original) || !bytes.Equal(readLegacySourceFontFixture(t, fixture.file), original) {
				t.Fatal("caller data or shipped file changed")
			}
		})
	}
}

func TestLegacySourceFontsGlyphCoverageAndRendering(t *testing.T) {
	for _, fixture := range legacySourceFontFixtures {
		t.Run(fixture.file, func(t *testing.T) {
			parsed, err := parseSourceFont(readLegacySourceFontFixture(t, fixture.file))
			if err != nil {
				t.Fatal(err)
			}
			var covered strings.Builder
			for r := rune(32); r < 127; r++ {
				glyph, err := parsed.GlyphIndex(nil, r)
				if err != nil {
					t.Fatal(err)
				}
				if glyph != 0 {
					covered.WriteRune(r)
				}
			}
			if got := covered.String(); got != fixture.ascii {
				t.Fatalf("ASCII coverage %q, want %q", got, fixture.ascii)
			}
			for glyph := 0; glyph < parsed.NumGlyphs(); glyph++ {
				if _, err := parsed.LoadGlyph(nil, sfnt.GlyphIndex(glyph), fixed.I(20), nil); err != nil {
					t.Fatalf("load actual glyph %d: %v", glyph, err)
				}
			}
			face, err := opentype.NewFace(parsed, &opentype.FaceOptions{Size: 20, DPI: 72, Hinting: font.HintingNone})
			if err != nil {
				t.Fatal(err)
			}
			defer face.Close()
			for _, r := range fixture.label {
				glyph, err := parsed.GlyphIndex(nil, r)
				if err != nil || glyph == 0 {
					t.Fatalf("missing label glyph %q: %v", r, err)
				}
			}
			bounds, advance := font.BoundString(face, fixture.label)
			if advance != fixture.advance {
				t.Fatalf("label advance %s, want %s", advance, fixture.advance)
			}
			canvas := image.NewAlpha(image.Rect(0, 0, advance.Ceil()+16, 48))
			drawer := font.Drawer{Dst: canvas, Src: image.NewUniform(color.White), Face: face, Dot: fixed.P(8, 28)}
			translated := bounds.Add(drawer.Dot)
			if translated.Min.X < fixed.I(1) || translated.Min.Y < fixed.I(1) ||
				translated.Max.X >= fixed.I(canvas.Rect.Max.X-1) || translated.Max.Y >= fixed.I(canvas.Rect.Max.Y-1) {
				t.Fatalf("label would clip: %v in %v", translated, canvas.Rect)
			}
			drawer.DrawString(fixture.label)
			painted := 0
			for _, alpha := range canvas.Pix {
				if alpha != 0 {
					painted++
				}
			}
			if painted == 0 {
				t.Fatal("actual normalized font rendered no pixels")
			}
		})
	}
}

func TestLegacySourceFontDirectoryFailsClosed(t *testing.T) {
	original := readLegacySourceFontFixture(t, legacySourceFontFixtures[0].file)
	tables, ok := legacySourceFontTables(original)
	if !ok {
		t.Fatal("fixture rejected")
	}
	head, hhea, hmtx, maxp := tables[4], tables[5], tables[6], tables[8]
	put32 := func(data []byte, at int, value uint32) { binary.BigEndian.PutUint32(data[at:], value) }
	put16 := func(data []byte, at int, value uint16) { binary.BigEndian.PutUint16(data[at:], value) }
	cases := []struct {
		name   string
		mutate func([]byte) []byte
	}{
		{"empty", func(b []byte) []byte { return b[:0] }},
		{"truncated header", func(b []byte) []byte { return b[:11] }},
		{"truncated directory", func(b []byte) []byte { return b[:187] }},
		{"truncated final table", func(b []byte) []byte { return b[:len(b)-1] }},
		{"wrong sfnt magic", func(b []byte) []byte { b[0] = 1; return b }},
		{"overflow directory count", func(b []byte) []byte { put16(b, 4, 65535); return b }},
		{"wrong search header", func(b []byte) []byte { b[7]++; return b }},
		{"duplicate table", func(b []byte) []byte { copy(b[tables[5].record:], []byte("head")); return b }},
		{"missing head", func(b []byte) []byte { copy(b[head.record:], []byte("xxxx")); return b }},
		{"head length 54 already", func(b []byte) []byte { put32(b, head.record+12, 54); return b }},
		{"wrong head length", func(b []byte) []byte { put32(b, head.record+12, 55); return b }},
		{"head magic", func(b []byte) []byte { b[head.offset+12]++; return b }},
		{"head version", func(b []byte) []byte { b[head.offset+1]++; return b }},
		{"head nonzero padding", func(b []byte) []byte { b[head.offset+55] = 1; return b }},
		{"hhea truncation", func(b []byte) []byte { put32(b, hhea.record+12, 35); return b }},
		{"maxp truncation", func(b []byte) []byte { put32(b, maxp.record+12, 31); return b }},
		{"hhea version", func(b []byte) []byte { b[hhea.offset+1]++; return b }},
		{"maxp version", func(b []byte) []byte { b[maxp.offset+1]++; return b }},
		{"hmtx wrong length", func(b []byte) []byte { put32(b, hmtx.record+12, uint32(hmtx.size-2)); return b }},
		{"hmtx nonzero tail", func(b []byte) []byte { b[hmtx.offset+hmtx.size-1] = 1; return b }},
		{"zero metrics", func(b []byte) []byte { put16(b, hhea.offset+34, 0); return b }},
		{"metrics exceed glyphs", func(b []byte) []byte { put16(b, hhea.offset+34, 18); return b }},
		{"zero glyphs", func(b []byte) []byte { put16(b, maxp.offset+4, 0); return b }},
		{"count length mismatch", func(b []byte) []byte { put16(b, hhea.offset+34, 16); return b }},
		{"maximum metric counts", func(b []byte) []byte { put16(b, hhea.offset+34, 65535); put16(b, maxp.offset+4, 65535); return b }},
		{"offset addition overflow", func(b []byte) []byte { put32(b, hmtx.record+8, 0xfffffffc); return b }},
		{"length addition overflow", func(b []byte) []byte { put32(b, hmtx.record+12, 0xffffffff); return b }},
		{"offset and length overflow", func(b []byte) []byte { put32(b, hmtx.record+8, 0xfffffffc); put32(b, hmtx.record+12, 8); return b }},
		{"offset inside directory", func(b []byte) []byte { put32(b, hmtx.record+8, 12); return b }},
		{"unaligned offset", func(b []byte) []byte { put32(b, hmtx.record+8, uint32(hmtx.offset+1)); return b }},
		{"overlapping tables", func(b []byte) []byte { put32(b, hmtx.record+8, uint32(head.offset)); return b }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			data := tc.mutate(bytes.Clone(original))
			before := bytes.Clone(data)
			if _, ok := legacySourceFontTables(data); ok {
				t.Fatal("bad metadata accepted by independent bounds gate")
			}
			if _, ok := normalizeLegacySourceFont(data); ok {
				t.Fatal("unknown or malformed font normalized")
			}
			if _, err := parseSourceFont(data); err == nil {
				t.Fatal("malformed input parsed")
			}
			if !bytes.Equal(data, before) {
				t.Fatal("invalid input mutated")
			}
		})
	}
}

func TestSourceFontUnknownInputsGetOrdinaryParser(t *testing.T) {
	valid := bytes.Clone(goregular.TTF)
	before := bytes.Clone(valid)
	if _, err := parseSourceFont(valid); err != nil {
		t.Fatalf("normal valid font: %v", err)
	}
	if _, ok := normalizeLegacySourceFont(valid); ok || !bytes.Equal(valid, before) {
		t.Fatal("ordinary valid font was normalized or mutated")
	}
	// A structurally identical export with any different bytes is not known.
	legacy := readLegacySourceFontFixture(t, legacySourceFontFixtures[0].file)
	tables, _ := legacySourceFontTables(legacy)
	for _, table := range tables {
		t.Run("unknown checksum "+table.tag, func(t *testing.T) {
			data := bytes.Clone(legacy)
			data[table.record+4] ^= 1
			if _, ok := legacySourceFontTables(data); !ok {
				t.Fatal("test should retain the recognized structure")
			}
			if _, ok := normalizeLegacySourceFont(data); ok {
				t.Fatal("unrecognized exact bytes normalized")
			}
			if _, err := parseSourceFont(data); err == nil {
				t.Fatal("unknown legacy font repaired")
			}
		})
	}
	// Invalid cmap on an otherwise normally parsed font must still fail. This
	// also proves no checksum regeneration is offered to unrelated corruption.
	bad, _ := normalizeLegacySourceFont(legacy)
	cmap := tables[1]
	binary.BigEndian.PutUint16(bad[cmap.offset+2:], 0) // no encoding subtables
	if _, err := opentype.Parse(bad); err == nil {
		t.Fatal("bad cmap fixture unexpectedly valid")
	}
	if _, err := parseSourceFont(bad); err == nil {
		t.Fatal("unrelated cmap defect accepted")
	}
}

func TestSourceFontChecksumSpecification(t *testing.T) {
	for _, tc := range []struct {
		data []byte
		want uint32
	}{
		{nil, 0}, {[]byte{1}, 0x01000000}, {[]byte{1, 2}, 0x01020000},
		{[]byte{1, 2, 3}, 0x01020300}, {[]byte{1, 2, 3, 4}, 0x01020304},
		{[]byte{0xff, 0xff, 0xff, 0xff, 0, 0, 0, 1}, 0},
	} {
		if got := sourceFontChecksum(tc.data); got != tc.want {
			t.Fatalf("checksum(%x)=%08x, want %08x", tc.data, got, tc.want)
		}
	}
}

func TestSourceFontMenuAndHUDLoadBundledFonts(t *testing.T) {
	for _, kind := range []sourceMenuFontKind{menuFontCondensed, menuFontCondensedExtraBold, menuFontTwCen, menuFontCenturyGothicBold, menuFontShowcardGothic} {
		face, err := sourceMenuFace(kind, 20)
		if err != nil || face == nil {
			t.Fatalf("menu font %d: face=%v err=%v", kind, face, err)
		}
	}
	face, err := sourceHUDFace(20)
	if err != nil || face == nil {
		t.Fatalf("HUD face=%v err=%v", face, err)
	}
	// Native fonts may win on Windows; the selector tests below isolate the
	// no-native case and prove the bundled Extra Bold font is used there.
}

func TestSourceFontCandidatesPreserveOrdinaryPriority(t *testing.T) {
	legacy := readLegacySourceFontFixture(t, "20_Tw Cen MT Condensed Extra Bold.ttf")
	dir := t.TempDir()
	legacyPath := filepath.Join(dir, "legacy.ttf")
	ordinaryPath := filepath.Join(dir, "ordinary.ttf")
	if err := os.WriteFile(legacyPath, legacy, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(ordinaryPath, goregular.TTF, 0600); err != nil {
		t.Fatal(err)
	}
	ordinary, err := opentype.Parse(goregular.TTF)
	if err != nil {
		t.Fatal(err)
	}
	for _, paths := range [][]string{
		{legacyPath, ordinaryPath}, {ordinaryPath, legacyPath},
		{legacyPath, filepath.Join(dir, "missing.ttf"), ordinaryPath},
	} {
		got, err := loadSourceFontCandidates(paths)
		if err != nil {
			t.Fatal(err)
		}
		if got.NumGlyphs() != ordinary.NumGlyphs() {
			t.Fatalf("normalized subset shadowed ordinary candidate: glyphs=%d want=%d", got.NumGlyphs(), ordinary.NumGlyphs())
		}
	}
	got, err := loadSourceFontCandidates([]string{legacyPath, filepath.Join(dir, "missing.ttf")})
	if err != nil {
		t.Fatal(err)
	}
	if got.NumGlyphs() != 118 {
		t.Fatal("missing native font did not select normalized bundled fallback")
	}
	for _, tc := range []struct {
		path   string
		before []byte
	}{{legacyPath, legacy}, {ordinaryPath, goregular.TTF}} {
		after, err := os.ReadFile(tc.path)
		if err != nil || !bytes.Equal(after, tc.before) {
			t.Fatalf("selector changed file %s: %v", tc.path, err)
		}
	}
	if _, err := loadSourceFontCandidates(nil); err == nil {
		t.Fatal("empty candidate list returned no error")
	}
}

func TestSourceFontCandidatesUseFirstKnownFallback(t *testing.T) {
	paths := []string{
		filepath.Join("..", "..", "assets", "fonts", "20_Tw Cen MT Condensed Extra Bold.ttf"),
		filepath.Join("..", "..", "assets", "fonts", "49_Tw Cen MT Condensed.ttf"),
	}
	got, err := loadSourceFontCandidates(paths)
	if err != nil || got.NumGlyphs() != 118 {
		t.Fatalf("first known fallback not selected: font=%v err=%v", got, err)
	}
}
