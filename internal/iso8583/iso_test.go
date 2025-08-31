package iso8583

import (
	"encoding/binary"
	"strings"
	"testing"
)

func TestSetGet(t *testing.T) {
	m := New("0100")
	m.Set(3, "000000")
	if v, ok := m.Get(3); !ok || v != "000000" {
		t.Fatalf("Get returned %q %v", v, ok)
	}
}

func TestPackInvalidLength(t *testing.T) {
	m := New("0200")
	m.Set(11, "12345") // should be 6
	if _, err := m.Pack(); err == nil {
		t.Fatalf("expected error for invalid length")
	}
}

func TestPackInvalidMTI(t *testing.T) {
	m := New("123")
	m.Set(11, "123456")
	if _, err := m.Pack(); err == nil {
		t.Fatalf("expected MTI length error")
	}
}

func TestPackUnknownField(t *testing.T) {
	m := New("0200")
	m.Set(100, "abc")
	if _, err := m.Pack(); err == nil || !strings.Contains(err.Error(), "not implemented") {
		t.Fatalf("expected spec error, got %v", err)
	}
}

func TestPackLLVARTooLong(t *testing.T) {
	m := New("0200")
	m.Set(2, strings.Repeat("1", 100))
	if _, err := m.Pack(); err == nil {
		t.Fatalf("expected LLVAR length error")
	}
}

func TestPackLLLVARTooLong(t *testing.T) {
	m := New("0200")
	m.Set(55, strings.Repeat("A", 1000))
	if _, err := m.Pack(); err == nil {
		t.Fatalf("expected LLLVAR length error")
	}
}

func TestUnpackTruncated(t *testing.T) {
	m := New("0200")
	m.Set(7, "0102030405")
	m.Set(11, "123456")
	p, err := m.Pack()
	if err != nil {
		t.Fatalf("Pack: %v", err)
	}
	p = p[:len(p)-1]
	if _, err := Unpack(p); err == nil {
		t.Fatalf("expected error for truncated message")
	}
}

func TestUnpackUnknownField(t *testing.T) {
	m := New("0200")
	m.Set(70, "301") // ensures secondary bitmap
	p, err := m.Pack()
	if err != nil {
		t.Fatalf("Pack: %v", err)
	}
	// Set bit 100 in secondary bitmap
	primaryOffset := 2 + 4
	secondaryOffset := primaryOffset + 8
	sec := binary.BigEndian.Uint64(p[secondaryOffset : secondaryOffset+8])
	sec |= 1 << 28 // bit 100
	binary.BigEndian.PutUint64(p[secondaryOffset:secondaryOffset+8], sec)

	if _, err := Unpack(p); err == nil || !strings.Contains(err.Error(), "field 100 not implemented") {
		t.Fatalf("expected unknown field error, got %v", err)
	}
}

func TestUnpackExtraBytes(t *testing.T) {
	m := New("0200")
	m.Set(11, "123456")
	p, err := m.Pack()
	if err != nil {
		t.Fatalf("Pack: %v", err)
	}
	p = append(p, 'X')
	binary.BigEndian.PutUint16(p[:2], uint16(len(p)-2))
	if _, err := Unpack(p); err == nil || !strings.Contains(err.Error(), "extra bytes") {
		t.Fatalf("expected extra bytes error, got %v", err)
	}
}

func TestBadDataResponse(t *testing.T) {
	// Simulate acquirer responding with format error (39=30)
	m := New("0210")
	m.Set(39, "30")
	p, err := m.Pack()
	if err != nil {
		t.Fatalf("Pack: %v", err)
	}
	m2, err := Unpack(p)
	if err != nil {
		t.Fatalf("Unpack: %v", err)
	}
	if v, _ := m2.Get(39); v != "30" {
		t.Fatalf("expected resp code 30, got %q", v)
	}
}

func TestEchoHelpers(t *testing.T) {
	m := NewEchoRequest(123456)
	if m.MTI != "0800" {
		t.Fatalf("unexpected MTI %q", m.MTI)
	}
	if v, ok := m.Get(70); !ok || v != "301" {
		t.Fatalf("missing DE70 in echo request")
	}
	if MustParseSTAN(m) != 123456%1000000 {
		t.Fatalf("MustParseSTAN mismatch")
	}

	resp := New("0810")
	resp.Set(11, "123456")
	resp.Set(70, "301")
	if !IsEchoResponse(resp) {
		t.Fatalf("IsEchoResponse false for valid response")
	}
	resp.Set(70, "999")
	if IsEchoResponse(resp) {
		t.Fatalf("IsEchoResponse true for invalid response")
	}
}
