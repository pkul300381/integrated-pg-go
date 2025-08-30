package iso8583

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"strconv"
	"time"
)

// Message represents a minimal ISO8583 message used here.
// MTI: 4 ASCII bytes
// Bitmap: 8 bytes (primary)
// Supported fields in this skeleton: 7 (MMDDhhmmss), 11 (STAN, 6n), 70 (3n)
type Message struct {
	MTI    string
	Fields map[int]string // field number -> ASCII string
}

// New creates an empty ISO8583 message with given MTI.
func New(mti string) *Message {
	return &Message{MTI: mti, Fields: make(map[int]string)}
}

// Set sets a field value as ASCII string.
func (m *Message) Set(field int, value string) { m.Fields[field] = value }

// Get gets a field value (ASCII string) and presence bool.
func (m *Message) Get(field int) (string, bool) { v, ok := m.Fields[field]; return v, ok }

// Pack builds a wire message: [2B MLI][4B MTI ASCII][8B bitmap][fields...]
// For our three fields, encoding is ASCII numeric, fixed-length except DE7 (10) and DE70 (3).
func (m *Message) Pack() ([]byte, error) {
	if len(m.MTI) != 4 { return nil, fmt.Errorf("invalid MTI: %q", m.MTI) }

	// Build bitmap
	var bitmap uint64
	set := func(bit int) { bitmap |= (1 << (64 - bit)) }
	for f := range m.Fields {
		if f < 1 || f > 64 { return nil, fmt.Errorf("unsupported field %d", f) }
		set(f)
	}

	body := bytes.NewBuffer(nil)
	body.WriteString(m.MTI)
	var bm [8]byte
	binary.BigEndian.PutUint64(bm[:], bitmap)
	body.Write(bm[:])

	// Encode fields in numeric order
	for f := 1; f <= 64; f++ {
		v, ok := m.Fields[f]
		if !ok { continue }
		switch f {
		case 7: // MMDDhhmmss (10n)
			if len(v) != 10 { return nil, fmt.Errorf("DE7 must be 10 digits, got %d", len(v)) }
			body.WriteString(v)
		case 11: // STAN (6n)
			if len(v) != 6 { return nil, fmt.Errorf("DE11 must be 6 digits, got %d", len(v)) }
			body.WriteString(v)
		case 70: // Network Mgmt Code (3n)
			if len(v) != 3 { return nil, fmt.Errorf("DE70 must be 3 digits, got %d", len(v)) }
			body.WriteString(v)
		default:
			return nil, fmt.Errorf("field %d not implemented in skeleton", f)
		}
	}

	// Prepend MLI
	msg := body.Bytes()
	mli := make([]byte, 2)
	binary.BigEndian.PutUint16(mli, uint16(len(msg)))
	return append(mli, msg...), nil
}

// Unpack parses the minimal wire format from Pack().
func Unpack(b []byte) (*Message, error) {
	if len(b) < 2 { return nil, errors.New("buffer too short for MLI") }
	mli := int(binary.BigEndian.Uint16(b[:2]))
	if len(b)-2 < mli { return nil, fmt.Errorf("incomplete message: need %d, have %d", mli, len(b)-2) }
	p := b[2 : 2+mli]
	if len(p) < 12 { return nil, errors.New("too short for MTI+bitmap") }
	mti := string(p[:4])
	bitmap := binary.BigEndian.Uint64(p[4:12])
	off := 12

	m := New(mti)
	present := func(bit int) bool { return (bitmap & (1 << (64 - bit))) != 0 }

	for f := 1; f <= 64; f++ {
		if !present(f) { continue }
		switch f {
		case 7:
			if off+10 > len(p) { return nil, errors.New("truncated DE7") }
			m.Fields[7] = string(p[off : off+10])
			off += 10
		case 11:
			if off+6 > len(p) { return nil, errors.New("truncated DE11") }
			m.Fields[11] = string(p[off : off+6])
			off += 6
		case 70:
			if off+3 > len(p) { return nil, errors.New("truncated DE70") }
			m.Fields[70] = string(p[off : off+3])
			off += 3
		default:
			return nil, fmt.Errorf("field %d not implemented in skeleton", f)
		}
	}
	if off != len(p) {
		return nil, fmt.Errorf("extra bytes at end: %d", len(p)-off)
	}
	return m, nil
}

// Helpers for Echo Test messages
func NewEchoRequest(stan int) *Message {
	m := New("0800")
	m.Set(7, time.Now().UTC().Format("0102150405")) // MMDDhhmmss
	m.Set(11, fmt.Sprintf("%06d", stan%1000000))
	m.Set(70, "301") // 301 = Echo Test
	return m
}

func IsEchoResponse(m *Message) bool {
	if m.MTI != "0810" { return false }
	if v, ok := m.Get(70); !ok || v != "301" { return false }
	_, ok := m.Get(11)
	return ok
}

// MustParseSTAN parses DE11 to int (for logging/correlation).
func MustParseSTAN(m *Message) int {
	v, _ := m.Get(11)
	n, _ := strconv.Atoi(v)
	return n
}
