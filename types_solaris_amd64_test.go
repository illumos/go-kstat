//
package kstat_test

import (
	"testing"
	"unsafe"

	"github.com/siebenmann/go-kstat"
)

// Because we played a sleazy trick to generate the Mntinfo struct,
// we test that its size is exactly the same as the C version. While
// we're here, we test the others as well.
// These sizes come from cgo. I suppose I could make this itself a
// cgo file and directly use C.sizeof_, but no, not right now.
const sizeof_IO = 0x50
const sizeof_SI = 0x18
const sizeof_VI = 0x30
const sizeof_Var = 0x3c
const sizeof_KM = 0x1ec

func TestStructSizes(t *testing.T) {
	sz := unsafe.Sizeof(kstat.Mntinfo{})
	if sz != sizeof_KM {
		t.Fatalf("Mntinfo has the wrong size: %d vs %d", sz, sizeof_KM)
	}
	sz = unsafe.Sizeof(kstat.IO{})
	if sz != sizeof_IO {
		t.Fatalf("IO has the wrong size: %d vs %d", sz, sizeof_IO)
	}
	sz = unsafe.Sizeof(kstat.Sysinfo{})
	if sz != sizeof_SI {
		t.Fatalf("Sysinfo has the wrong size: %d vs %d", sz, sizeof_SI)
	}
	sz = unsafe.Sizeof(kstat.Vminfo{})
	if sz != sizeof_VI {
		t.Fatalf("Vminfo has the wrong size: %d vs %d", sz, sizeof_VI)
	}
	sz = unsafe.Sizeof(kstat.Var{})
	if sz != sizeof_Var {
		t.Fatalf("Var has the wrong size: %d vs %d", sz, sizeof_Var)
	}
}

func toint8(str string) *[256]int8 {
	var buf [256]int8
	for i := 0; i < len(str); i++ {
		buf[i] = int8(str[i])
	}
	// remaining buf is zeroed by implication.
	return &buf
}

// Given a string, this tests variants of buffer conversion: string
// with trailing 0's, string exactly filling the slice passed to
// CFieldString (simulating an exactly-full field), and first
// character (exactly filling the field).
func teststring(t *testing.T, s string) {
	buf := toint8(s)
	r := kstat.CFieldString(buf[:])
	if r != s {
		t.Fatalf("full buf mismatch: %q vs %q", s, r)
	}
	r = kstat.CFieldString(buf[:len(s)])
	if r != s {
		t.Fatalf("exact buf mismatch: %q vs %q", s, r)
	}
	r = kstat.CFieldString(buf[:len(s)+1])
	if r != s {
		t.Fatalf("string + one null mismatch: %q vs %q", s, r)
	}
	if len(s) > 1 {
		r = kstat.CFieldString(buf[:1])
		if r != s[:1] {
			t.Fatalf("first character mismatch: %q vs %q", s[:1], r)
		}
	}
}

// This function is sufficiently potentially tricky that I want to test
// it directly, including with some torture tests.
func TestCFieldString(t *testing.T) {
	teststring(t, "this is a test string")
	teststring(t, "")
	buf := toint8("abc\x00def")
	r := kstat.CFieldString(buf[:])
	if r != "abc" {
		t.Fatalf("embedded null not properly handled: %q", r)
	}
}
