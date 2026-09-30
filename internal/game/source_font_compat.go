package game

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"os"

	"golang.org/x/image/font/opentype"
)

// loadSourceFontCandidates preserves the original candidate policy: any font
// accepted normally wins over a normalized subset, even if that subset's path
// came first. This avoids reducing glyph coverage on systems with native fonts.
func loadSourceFontCandidates(paths []string) (*opentype.Font, error) {
	var fallback *opentype.Font
	var lastErr error = os.ErrNotExist
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			lastErr = err
			continue
		}
		parsed, err := opentype.Parse(data)
		if err == nil {
			return parsed, nil
		}
		lastErr = err
		if fallback == nil {
			if normalized, ok := normalizeLegacySourceFont(data); ok {
				// Keep only a copy accepted by the ordinary parser. Continue
				// looking for an ordinary native/bundled candidate first.
				fallback, _ = opentype.Parse(normalized)
			}
		}
	}
	if fallback != nil {
		return fallback, nil
	}
	return nil, lastErr
}

// parseSourceFont leaves ordinary fonts to the normal parser. Only the exact
// five legacy assets below may take the compatibility path after parsing fails.
func parseSourceFont(data []byte) (*opentype.Font, error) {
	parsed, err := opentype.Parse(data)
	if err == nil {
		return parsed, nil
	}
	if normalized, ok := normalizeLegacySourceFont(data); ok {
		// Metadata normalization is not a substitute for normal validation.
		return opentype.Parse(normalized)
	}
	return nil, err
}

func isKnownLegacySourceFont(data []byte) bool {
	// These hashes identify checked-in asset versions, not authenticity or a
	// security guarantee. A changed/replaced font gets ordinary parser handling.
	switch fmt.Sprintf("%x", sha256.Sum256(data)) {
	case "e8e4f3354fa8b7c439a04e4a29dbbe30a8098e771d6ce00bc9c7fcdb6493c74f", // 1006_Century Gothic.ttf
		"cce02e3f77e2a305f3a440c70d10b772857263eab585d87ac00928f94a0d93e2", // 10_Tw Cen MT.ttf
		"5ab80b52c022e8067f9ec7212e80638173c2f27290056a07415adc7a92e5dd05", // 18_Showcard Gothic.ttf
		"a778fda43fee611d4a9c80ee2831056a3815d02a40c634df7270412a427bd8b9", // 20_Tw Cen MT Condensed Extra Bold.ttf
		"a5ec295577c27eaeaac4c674818beb00b5c0d873ed0ee73fefc14613e5cdff42": // 49_Tw Cen MT Condensed.ttf
		return true
	}
	return false
}

type legacySourceFontTable struct {
	tag          string
	record       int
	offset, size int
}

// legacySourceFontTables checks the narrow directory/length pattern before any
// writes, even though the caller has also checked the exact asset hash. Widened
// arithmetic precedes conversion to int and every table is checked for overlap.
func legacySourceFontTables(data []byte) ([]legacySourceFontTable, bool) {
	const directoryEnd = 12 + 11*16
	if len(data) < directoryEnd || !bytes.Equal(data[:12], []byte{
		0, 1, 0, 0, 0, 11, 0, 128, 0, 3, 0, 48,
	}) {
		return nil, false
	}
	tags := [...]string{"OS/2", "cmap", "glyf", "hdmx", "head", "hhea", "hmtx", "loca", "maxp", "name", "post"}
	tables := make([]legacySourceFontTable, len(tags))
	for i, tag := range tags {
		record := 12 + i*16
		offset := uint64(binary.BigEndian.Uint32(data[record+8:]))
		size := uint64(binary.BigEndian.Uint32(data[record+12:]))
		if string(data[record:record+4]) != tag || offset < directoryEnd || offset%4 != 0 ||
			size == 0 || offset+size > uint64(len(data)) {
			return nil, false
		}
		for _, other := range tables[:i] {
			if offset < uint64(other.offset)+uint64(other.size) && uint64(other.offset) < offset+size {
				return nil, false
			}
		}
		tables[i] = legacySourceFontTable{tag: tag, record: record, offset: int(offset), size: int(size)}
	}
	head, hhea, hmtx, maxp := tables[4], tables[5], tables[6], tables[8]
	if head.size != 56 || hhea.size != 36 || maxp.size != 32 ||
		binary.BigEndian.Uint32(data[head.offset:]) != 0x00010000 ||
		binary.BigEndian.Uint32(data[head.offset+12:]) != 0x5f0f3cf5 ||
		binary.BigEndian.Uint32(data[hhea.offset:]) != 0x00010000 ||
		binary.BigEndian.Uint32(data[maxp.offset:]) != 0x00010000 ||
		!bytes.Equal(data[head.offset+54:head.offset+56], []byte{0, 0}) {
		return nil, false
	}
	glyphs := uint64(binary.BigEndian.Uint16(data[maxp.offset+4:]))
	metrics := uint64(binary.BigEndian.Uint16(data[hhea.offset+34:]))
	if metrics == 0 || metrics > glyphs {
		return nil, false
	}
	// Both counts originate in uint16 fields; widened arithmetic cannot wrap.
	expectedMetricsSize := 4*metrics + 2*(glyphs-metrics)
	if uint64(hmtx.size) != expectedMetricsSize+4 ||
		!bytes.Equal(data[hmtx.offset+hmtx.size-4:hmtx.offset+hmtx.size], []byte{0, 0, 0, 0}) {
		return nil, false
	}
	return tables, true
}

func normalizeLegacySourceFont(data []byte) ([]byte, bool) {
	if !isKnownLegacySourceFont(data) {
		return nil, false
	}
	tables, ok := legacySourceFontTables(data)
	if !ok {
		return nil, false
	}
	// The parser retains its byte slice. Never mutate the caller's bytes or
	// write back to disk. Glyphs, cmap, names/licensing and metrics stay intact.
	normalized := bytes.Clone(data)
	tables[4].size = 54
	tables[6].size -= 4
	head := tables[4]
	binary.BigEndian.PutUint32(normalized[head.offset+8:], 0)
	for _, table := range tables {
		binary.BigEndian.PutUint32(normalized[table.record+12:], uint32(table.size))
		// The known exports have invalid checksums throughout the directory,
		// not just in head/hmtx. Recompute standard checksums for this copy.
		checksum := sourceFontChecksum(normalized[table.offset : table.offset+table.size])
		binary.BigEndian.PutUint32(normalized[table.record+4:], checksum)
	}
	// SFNT: head's directory checksum is calculated with adjustment zero;
	// only then is the whole-font adjustment written, modulo 2^32.
	// https://learn.microsoft.com/en-us/typography/opentype/spec/otff#calculating-checksums
	binary.BigEndian.PutUint32(normalized[head.offset+8:], 0xb1b0afba-sourceFontChecksum(normalized))
	return normalized, true
}

// sourceFontChecksum sums big-endian uint32 words, padding the last word with
// zeros without reading past the supplied table. uint32 wrap is required here.
func sourceFontChecksum(data []byte) uint32 {
	var sum uint32
	for len(data) >= 4 {
		sum += binary.BigEndian.Uint32(data)
		data = data[4:]
	}
	for i, value := range data {
		sum += uint32(value) << uint(24-8*i)
	}
	return sum
}
