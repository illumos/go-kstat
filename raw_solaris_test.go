//
// Test raw access to KStats.

package kstat_test

import (
	"testing"
	"unsafe"

	"github.com/siebenmann/go-kstat"
)

// we reuse functions and constants from kstat_solaris_test.go
func TestRaw(t *testing.T) {
	tok := start(t)
	ks := lookup(t, tok, "sd", "sd0")
	osnap := ks.Snaptime
	r, err := ks.Raw()
	if err != nil {
		t.Fatalf("%s Raw error: %s", ks, r)
	}
	// Raw should not have changed Snaptime:
	if osnap != r.Snaptime {
		t.Fatalf("%s Raw snaptime changed: %d vs %d", ks, osnap, r.Snaptime)
	}
	// This is an IoStat; it should be the same size as the IO struct.
	si := int(unsafe.Sizeof(kstat.IO{}))
	if len(r.Data) != si {
		t.Fatalf("%s Raw size wrong: sizeof IO %d, actual %d", ks, si, len(r.Data))
	}
	// Should have only one element:
	if r.Ndata != 1 {
		t.Fatalf("%s Ndata != 1: %d", ks, r.Ndata)
	}

	// Named kstats have Ndata > 1.
	// As a side effect this tests doing .Raw() of them.
	ks = lookup(t, tok, "cpu", "sys")
	r, err = ks.Raw()
	if err != nil {
		t.Fatalf("%s Raw get failed: %s", ks, err)
	}
	// We're not going to hardcode an Ndata value for cpu:0:sys,
	// because who knows, it could change.
	if r.Ndata <= 1 {
		t.Fatalf("%s Raw Ndata not right: %d", ks, r.Ndata)
	}

	// Test getting an actual RawStat via Raw
	ks = lookup(t, tok, "unix", "sysinfo")
	r, err = ks.Raw()
	if err != nil {
		t.Fatalf("%s Raw get failed: %s", ks, err)
	}
	si = int(unsafe.Sizeof(kstat.Sysinfo{}))
	if len(r.Data) != si {
		t.Fatalf("%s Raw size wrong: sizeof Sysinfo %d, actual %d", ks, si, len(r.Data))
	}
	// KSTAT_TYPE_RAW has Ndata == sizeof the actual data chunk. SIGH.
	// I suppose that when they saw 'raw' they mean 'raw' but come on,
	// ks_data_size is a valid value too.
	if r.Ndata != uint64(si) {
		t.Fatalf("%s Ndata is not sizeof: sizeof %d vs %d", ks, si, r.Ndata)
	}

	stop(t, tok)

	// While we're here, test that .Raw() fails after a Close
	_, err = ks.Raw()
	if err == nil {
		t.Fatalf("%s Raw() call did not fail after Close()", ks)
	}
}

func ksshouldbe(t *testing.T, ks *kstat.KStat, module string, instance int, name string) {
	if ks.Module != module || ks.Instance != instance || ks.Name != name {
		t.Fatalf("%s expected to be %s:%d:%s", ks, module, instance, name)
	}
}

// Test the unix:0:* extractors. We're basically testing that these
// things can be called successfully; it's hard to audit the returned
// results for sanity.
func TestUnixStats(t *testing.T) {
	tok := start(t)
	ks, si, err := tok.Sysinfo()
	if err != nil {
		t.Fatalf("Sysinfo error: %s", err)
	}
	ksshouldbe(t, ks, "unix", 0, "sysinfo")
	if si.Updates == 0 {
		t.Fatalf("Sysinfo.Updates == 0: %+v", si)
	}
	ks, vmi, err := tok.Vminfo()
	if err != nil {
		t.Fatalf("Vminfo error: %s", err)
	}
	ksshouldbe(t, ks, "unix", 0, "vminfo")
	if vmi.Updates == 0 {
		t.Fatalf("Vminfo.Updates == 0: %+v", vmi)
	}
	// TODO: find some Var field that is never going to be 0
	ks, _, err = tok.Var()
	if err != nil {
		t.Fatalf("Var error: %s", err)
	}
	ksshouldbe(t, ks, "unix", 0, "var")

	stop(t, tok)
}

// Test CopyTo by fetching unix:0:var directly with it and comparing
// against the result from calling Var().
// We also test getting a struct variant of Sysinfo and the traditional
// 'errors out after Close'.
func TestCopyTo(t *testing.T) {
	tok := start(t)
	ks, or, err := tok.Var()
	if err != nil {
		t.Fatalf("Var() error: %s", err)
	}
	r := kstat.Var{}
	err = ks.CopyTo(&r)
	if err != nil {
		t.Fatalf("%s CopyTo failed: %s", ks, err)
	}
	if r != *or {
		t.Fatalf("Var structure difference: Var: %+v CopyTo: %+v", or, r)
	}

	// Fetch an alternate version of the Sysinfo struct with CopyTo
	// and verify it against the original.
	// (We can't just use struct == because they're not the same
	// types.)
	ks, si, err := tok.Sysinfo()
	if err != nil {
		t.Fatalf(".Sysinfo() error: %s", err)
	}
	f := Sysinfo_alt{}
	err = ks.CopyTo(&f)
	if err != nil {
		t.Fatalf("%s CopyTo failed: %s", ks, err)
	}
	if si.Updates != f.updates || si.Runque != f.Runs.Runque || si.Runocc != f.Runs.runocc || si.Swpque != f.Swps[0] || si.Swpocc != f.Swps[1] || si.Waiting != f.Waiting {
		t.Fatalf("%s struct different values: %+v vs %+v", ks, si, f)
	}

	stop(t, tok)

	err = ks.CopyTo(&f)
	if err == nil {
		t.Fatalf("%s CopyTo succeeded after Close", ks)
	}
}

// This has 6 uint32s, just like Sysinfo, but they are in a mixture of
// unexported fields, embedded structs, and arrays. It doesn't cover
// all combinations but it does cover a number of them.
//
// We pick updates as our unexported field because it's highly likely
// to be non-zero, just in case. (I am probably worrying too much, and
// the Swps often seem to be zero, so.)
type Sysinfo_alt struct {
	updates uint32
	Runs    struct {
		Runque uint32
		runocc uint32
	}
	Swps    [2]uint32
	Waiting uint32
}

// Testing GetMntinfo() is complicated by the fact that servers may
// not have any of them.
func TestMntinfo(t *testing.T) {
	tok := start(t)
	ks, err := tok.Lookup("nfs", -1, "mntinfo")
	if err != nil {
		stop(t, tok)
		t.Skip("skipping test due to lack of nfs:*:mntinfo kstat")
	}
	osnap := ks.Snaptime
	mi, err := ks.GetMntinfo()
	if err != nil {
		t.Fatalf("%s GetMntinfo error: %s", ks, err)
	}
	if mi.Proto() == "" || mi.Curserver() == "" {
		t.Fatalf("%s empty proto and/or curserv: %#v", ks, mi)
	}

	// BUG: this may be a mistake to insist on narrow values here.
	if mi.Vers < 3 || mi.Vers > 4 {
		t.Fatalf("%s Vers is not 3 or 4: %#v", ks, mi)
	}
	if osnap != ks.Snaptime {
		t.Fatalf("%s GetMntinfo changed Snaptime: %d vs %d", ks, osnap, ks.Snaptime)
	}
	stop(t, tok)

	// Test that getting what we know is supposed to be a valid kstat
	// fails after close.
	mi, err = ks.GetMntinfo()
	if err == nil {
		t.Fatalf("%s GetMntinfo succeeds after Close: %#v", ks, mi)
	}
}

// GetMntinfo should fail on things that are not mntinfos.
func TestMntinfoErrors(t *testing.T) {
	tok := start(t)
	ks := lookup(t, tok, "unix", "sysinfo")
	mi, err := ks.GetMntinfo()
	if err == nil {
		t.Fatalf("%s GetMntinfo succeeds: %#v", ks, mi)
	}
	ks = lookup(t, tok, "sd", "sd0")
	mi, err = ks.GetMntinfo()
	if err == nil {
		t.Fatalf("%s GetMntinfo succeeds: %#v", ks, mi)
	}
	stop(t, tok)
}
