package iso8583

import (
	"bytes"
	"testing"
)

func TestLLVARHelpers(t *testing.T) {
	var buf bytes.Buffer
	if err := packLLVAR(&buf, "ABCD"); err != nil {
		t.Fatalf("packLLVAR: %v", err)
	}
	if got := buf.String(); got != "04ABCD" {
		t.Fatalf("packLLVAR got %q", got)
	}
	off := 0
	v, err := unpackLLVAR([]byte(buf.String()), &off)
	if err != nil {
		t.Fatalf("unpackLLVAR: %v", err)
	}
	if v != "ABCD" || off != len(buf.String()) {
		t.Fatalf("unpackLLVAR got %q off %d", v, off)
	}
}

func TestLLLVARHelpers(t *testing.T) {
	var buf bytes.Buffer
	if err := packLLLVAR(&buf, "HELLO"); err != nil {
		t.Fatalf("packLLLVAR: %v", err)
	}
	if got := buf.String(); got != "005HELLO" {
		t.Fatalf("packLLLVAR got %q", got)
	}
	off := 0
	v, err := unpackLLLVAR([]byte(buf.String()), &off)
	if err != nil {
		t.Fatalf("unpackLLLVAR: %v", err)
	}
	if v != "HELLO" || off != len(buf.String()) {
		t.Fatalf("unpackLLLVAR got %q off %d", v, off)
	}
}

func TestMessageWithVariableFields(t *testing.T) {
	m := New("0800")
	m.Set(7, "0102030405")
	m.Set(11, "123456")
	m.Set(48, "HELLO WORLD")
	m.Set(102, "ACC1234567")

	packed, err := m.Pack()
	if err != nil {
		t.Fatalf("Pack: %v", err)
	}
	// Ensure length headers are present
	if !bytes.Contains(packed, []byte("011HELLO WORLD")) {
		t.Fatalf("packed message missing DE48")
	}
	if !bytes.Contains(packed, []byte("10ACC1234567")) {
		t.Fatalf("packed message missing DE102")
	}

	m2, err := Unpack(packed)
	if err != nil {
		t.Fatalf("Unpack: %v", err)
	}
	if v, _ := m2.Get(48); v != "HELLO WORLD" {
		t.Fatalf("DE48 roundtrip got %q", v)
	}
	if v, _ := m2.Get(102); v != "ACC1234567" {
		t.Fatalf("DE102 roundtrip got %q", v)
	}
}
