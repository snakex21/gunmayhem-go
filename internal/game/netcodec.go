package game

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"hash"
	"math"
	"reflect"
	"sort"
	"strings"
	"sync"
	"unicode"

	"gunmayhem-go/internal/updater"
)

// netcodec.go is the compact, schema-driven binary encoding used for host
// snapshots (protocol 19+).
//
// Why not CBOR for snapshots: CBOR encodes every struct as a map with string
// keys, so a busy four-player frame was ~42 KB, mostly field names. This codec
// walks the Go types by reflection and writes:
//   - struct: a presence bitmap (one bit per field; bool fields ARE the bit),
//     then the values of the non-zero fields in declaration order;
//   - ints: zigzag varints, uints: varints;
//   - floats: float32 (4 bytes), which is far below one pixel of error;
//   - strings/slices: varint length + elements, arrays: elements.
//
// The format has no field names at all, so host and client must have the
// same types. That is enforced by netSchemaFingerprint(), which hashes the
// whole snapshot type tree (plus weapon tables) and is checked in the
// handshake: a build that adds a Player field simply gets a new fingerprint
// and a clear "version mismatch" message instead of garbage.
//
// New fields need no work here: every exported field is encoded
// automatically. Fields that only the host simulation needs can be dropped
// from the wire by netWireSkipField below (single place to edit), or by a
// `net:"-"` struct tag.

const (
	maxNetCodecSliceLen  = 16384
	maxNetCodecStringLen = 4096
)

type netFieldPlan struct {
	index int
	kind  reflect.Kind
}

type netStructPlan struct {
	fields      []netFieldPlan
	bitmapBytes int
}

var netStructPlans sync.Map // reflect.Type -> *netStructPlan

var (
	netPlayerType   = reflect.TypeOf(Player{})
	netSnapshotType = reflect.TypeOf(netSnapshot{})
)

// netPlayerHostOnlyField lists Player fields that only the host simulation
// reads. They are not sent: the client is a renderer and never runs input or
// AI decisions. Keep AI (HUD/ghost rendering) and AISpecial (render effects).
func netPlayerHostOnlyField(name string) bool {
	switch name {
	case "Controls", "RemoteInput", "PendingSounds":
		return true
	case "AI", "AISpecial":
		return false
	}
	// AILeft, AITargetX, AISearchTime, ... including AI fields added later.
	return len(name) > 2 && strings.HasPrefix(name, "AI") && unicode.IsUpper(rune(name[2]))
}

// netWireSkipField is THE place that decides which struct fields stay off the
// snapshot wire. Everything else is sent.
func netWireSkipField(owner reflect.Type, f reflect.StructField) bool {
	if f.PkgPath != "" { // unexported
		return true
	}
	if f.Tag.Get("net") == "-" || f.Tag.Get("cbor") == "-" {
		return true
	}
	if owner == netPlayerType && netPlayerHostOnlyField(f.Name) {
		return true
	}
	return false
}

func netPlanFor(t reflect.Type) *netStructPlan {
	if cached, ok := netStructPlans.Load(t); ok {
		return cached.(*netStructPlan)
	}
	plan := &netStructPlan{}
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if netWireSkipField(t, f) {
			continue
		}
		plan.fields = append(plan.fields, netFieldPlan{index: i, kind: f.Type.Kind()})
	}
	plan.bitmapBytes = (len(plan.fields) + 7) / 8
	actual, _ := netStructPlans.LoadOrStore(t, plan)
	return actual.(*netStructPlan)
}

// ---------------------------------------------------------------------------
// Encoder

type netEncoder struct {
	buf []byte
}

func (e *netEncoder) value(v reflect.Value) {
	switch v.Kind() {
	case reflect.Bool:
		if v.Bool() {
			e.buf = append(e.buf, 1)
		} else {
			e.buf = append(e.buf, 0)
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		e.buf = binary.AppendVarint(e.buf, v.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		e.buf = binary.AppendUvarint(e.buf, v.Uint())
	case reflect.Float32, reflect.Float64:
		e.buf = binary.LittleEndian.AppendUint32(e.buf, math.Float32bits(float32(v.Float())))
	case reflect.String:
		s := v.String()
		e.buf = binary.AppendUvarint(e.buf, uint64(len(s)))
		e.buf = append(e.buf, s...)
	case reflect.Slice:
		n := v.Len()
		e.buf = binary.AppendUvarint(e.buf, uint64(n))
		for i := 0; i < n; i++ {
			e.value(v.Index(i))
		}
	case reflect.Array:
		for i := 0; i < v.Len(); i++ {
			e.value(v.Index(i))
		}
	case reflect.Struct:
		plan := netPlanFor(v.Type())
		start := len(e.buf)
		for i := 0; i < plan.bitmapBytes; i++ {
			e.buf = append(e.buf, 0)
		}
		for i, f := range plan.fields {
			fv := v.Field(f.index)
			if fv.IsZero() {
				continue
			}
			e.buf[start+i/8] |= 1 << (i % 8)
			if f.kind == reflect.Bool {
				continue // the presence bit is the value
			}
			e.value(fv)
		}
	default:
		panic(fmt.Sprintf("netcodec: unsupported kind %s (%s)", v.Kind(), v.Type()))
	}
}

// netEncodeValue appends the compact encoding of *ptr to dst.
func netEncodeValue(dst []byte, ptr any) []byte {
	e := netEncoder{buf: dst}
	e.value(reflect.ValueOf(ptr).Elem())
	return e.buf
}

// ---------------------------------------------------------------------------
// Decoder (input is untrusted: every length is bounded)

type netDecoder struct {
	buf []byte
	pos int
}

var errNetCodecShort = errors.New("netcodec: truncated data")

func (d *netDecoder) remaining() int { return len(d.buf) - d.pos }

func (d *netDecoder) byte1() (byte, error) {
	if d.pos >= len(d.buf) {
		return 0, errNetCodecShort
	}
	b := d.buf[d.pos]
	d.pos++
	return b, nil
}

func (d *netDecoder) uvarint() (uint64, error) {
	x, n := binary.Uvarint(d.buf[d.pos:])
	if n <= 0 {
		return 0, errNetCodecShort
	}
	d.pos += n
	return x, nil
}

func (d *netDecoder) varint() (int64, error) {
	x, n := binary.Varint(d.buf[d.pos:])
	if n <= 0 {
		return 0, errNetCodecShort
	}
	d.pos += n
	return x, nil
}

func (d *netDecoder) value(v reflect.Value) error {
	switch v.Kind() {
	case reflect.Bool:
		b, err := d.byte1()
		if err != nil {
			return err
		}
		v.SetBool(b != 0)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		x, err := d.varint()
		if err != nil {
			return err
		}
		if v.OverflowInt(x) {
			return fmt.Errorf("netcodec: %s overflow", v.Type())
		}
		v.SetInt(x)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		x, err := d.uvarint()
		if err != nil {
			return err
		}
		if v.OverflowUint(x) {
			return fmt.Errorf("netcodec: %s overflow", v.Type())
		}
		v.SetUint(x)
	case reflect.Float32, reflect.Float64:
		if d.remaining() < 4 {
			return errNetCodecShort
		}
		f := float64(math.Float32frombits(binary.LittleEndian.Uint32(d.buf[d.pos:])))
		d.pos += 4
		if math.IsNaN(f) || math.IsInf(f, 0) {
			f = 0
		}
		v.SetFloat(f)
	case reflect.String:
		n, err := d.uvarint()
		if err != nil {
			return err
		}
		if n > maxNetCodecStringLen || n > uint64(d.remaining()) {
			return errors.New("netcodec: string too long")
		}
		v.SetString(string(d.buf[d.pos : d.pos+int(n)]))
		d.pos += int(n)
	case reflect.Slice:
		n, err := d.uvarint()
		if err != nil {
			return err
		}
		if n > maxNetCodecSliceLen {
			return fmt.Errorf("netcodec: %s length %d out of range", v.Type(), n)
		}
		if n == 0 {
			v.Set(reflect.Zero(v.Type()))
			return nil
		}
		if minSize := netMinEncodedSize(v.Type().Elem()); minSize > 0 && n*uint64(minSize) > uint64(d.remaining()) {
			return fmt.Errorf("netcodec: %s length %d exceeds data", v.Type(), n)
		}
		s := reflect.MakeSlice(v.Type(), int(n), int(n))
		for i := 0; i < int(n); i++ {
			if err := d.value(s.Index(i)); err != nil {
				return err
			}
		}
		v.Set(s)
	case reflect.Array:
		for i := 0; i < v.Len(); i++ {
			if err := d.value(v.Index(i)); err != nil {
				return err
			}
		}
	case reflect.Struct:
		plan := netPlanFor(v.Type())
		if d.remaining() < plan.bitmapBytes {
			return errNetCodecShort
		}
		bitmap := d.buf[d.pos : d.pos+plan.bitmapBytes]
		d.pos += plan.bitmapBytes
		for i, f := range plan.fields {
			if bitmap[i/8]&(1<<(i%8)) == 0 {
				continue
			}
			fv := v.Field(f.index)
			if f.kind == reflect.Bool {
				fv.SetBool(true)
				continue
			}
			if err := d.value(fv); err != nil {
				return err
			}
		}
	default:
		return fmt.Errorf("netcodec: unsupported kind %s", v.Kind())
	}
	return nil
}

// netMinEncodedSize is the smallest number of bytes one value of t occupies.
// It bounds slice lengths against the remaining input so a hostile length
// prefix cannot force a huge allocation.
func netMinEncodedSize(t reflect.Type) int {
	switch t.Kind() {
	case reflect.Float32, reflect.Float64:
		return 4
	case reflect.Array:
		return t.Len() * netMinEncodedSize(t.Elem())
	case reflect.Struct:
		return netPlanFor(t).bitmapBytes
	default:
		return 1
	}
}

// netDecodeValue decodes one value into *ptr and returns the bytes consumed.
func netDecodeValue(data []byte, ptr any) (int, error) {
	d := netDecoder{buf: data}
	if err := d.value(reflect.ValueOf(ptr).Elem()); err != nil {
		return d.pos, err
	}
	return d.pos, nil
}

// ---------------------------------------------------------------------------
// Schema fingerprint and version string

var (
	netSchemaOnce sync.Once
	netSchemaHash string
)

// writeNetSchema hashes the wire schema of t. It panics on anything the codec
// cannot encode (maps, pointers, interfaces, channels, recursive types). It
// runs when the first session starts and in TestNetCodecSupportsSnapshotSchema,
// so a new unsupported field is reported with its full path.
func writeNetSchema(h hash.Hash, t reflect.Type, path string, stack map[reflect.Type]bool) {
	switch t.Kind() {
	case reflect.Bool, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64, reflect.String:
		fmt.Fprintf(h, "%s;", t.Kind())
	case reflect.Slice:
		fmt.Fprint(h, "[]")
		writeNetSchema(h, t.Elem(), path+"[]", stack)
	case reflect.Array:
		fmt.Fprintf(h, "[%d]", t.Len())
		writeNetSchema(h, t.Elem(), path+"[]", stack)
	case reflect.Struct:
		if stack[t] {
			panic(fmt.Sprintf("netcodec: recursive type at %s", path))
		}
		stack[t] = true
		fmt.Fprintf(h, "struct %s{", t.Name())
		plan := netPlanFor(t)
		for _, f := range plan.fields {
			sf := t.Field(f.index)
			fmt.Fprintf(h, "%s:", sf.Name)
			writeNetSchema(h, sf.Type, path+"."+sf.Name, stack)
		}
		fmt.Fprint(h, "}")
		delete(stack, t)
	default:
		panic(fmt.Sprintf("netcodec: unsupported field kind %s at %s; snapshot fields must be plain values, slices, arrays or structs (or be skipped in netWireSkipField)", t.Kind(), path))
	}
}

// netSchemaFingerprint identifies the snapshot wire schema and the weapon
// tables the client uses to rebuild weapon definitions from their number.
func netSchemaFingerprint() string {
	netSchemaOnce.Do(func() {
		h := sha256.New()
		writeNetSchema(h, netSnapshotType, "netSnapshot", map[reflect.Type]bool{})
		writeNetSchema(h, reflect.TypeOf(netStateEnvelope{}), "netStateEnvelope", map[reflect.Type]bool{})
		writeNetSchema(h, reflect.TypeOf(Map{}), "Map", map[reflect.Type]bool{})
		numbers := make([]int, 0, len(weaponCatalog)+len(developedWeaponCatalog))
		for n := range weaponCatalog {
			numbers = append(numbers, n)
		}
		for n := range developedWeaponCatalog {
			if _, dup := weaponCatalog[n]; !dup {
				numbers = append(numbers, n)
			}
		}
		sort.Ints(numbers)
		for _, n := range numbers {
			def, _ := WeaponByNumber(n)
			fmt.Fprintf(h, "%#v;", def)
		}
		netSchemaHash = hex.EncodeToString(h.Sum(nil)[:4])
	})
	return netSchemaHash
}

// netplayGameVersion is the human readable build identity exchanged in the
// handshake, e.g. "v0.17.0 (net 1a2b3c4d)".
func netplayGameVersion() string {
	return fmt.Sprintf("%s (net %s)", updater.CurrentVersion, netSchemaFingerprint())
}
