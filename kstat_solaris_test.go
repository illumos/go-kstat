//
// All of these tests depend on being able to know what kstats exist
// on any Illumos / Solaris machine. I believe I've selected kstats
// that will always be there, but I could turn out to be wrong.
//
// We do our testing from outside the kstat package, so we see only
// its public API. For our purposes this is good enough.

package kstat_test

import (
	"runtime"
	"testing"
	"time"

	"github.com/siebenmann/go-kstat"
)

// Utility functions
// all of these are used for operations that are not supposed to fail,
// because the error message is generic.

// kstat.Open(); fail on error
func start(t *testing.T) *kstat.Token {
	tok, err := kstat.Open()
	if err != nil {
		t.Fatalf("Open failure: %s", err)
	}
	return tok
}

// Token.Close(); fail on error
func stop(t *testing.T, tok *kstat.Token) {
	err := tok.Close()
	if err != nil {
		t.Fatalf("Close failure: %s", err)
	}
}

// Token.Lookup(); fail on error
func lookup(t *testing.T, tok *kstat.Token, module, name string) *kstat.KStat {
	ks, err := tok.Lookup(module, -1, name)
	if err != nil {
		t.Fatalf("lookup failure on %s:-1:%s: %s", module, name, err)
	}
	return ks
}

// KStat.GetNamed(); fail on error
func kgetnamed(t *testing.T, ks *kstat.KStat, stat string) *kstat.Named {
	n, err := ks.GetNamed(stat)
	if err != nil {
		t.Fatalf("getting '%s' from %s: %s", stat, ks, err)
	}
	return n
}

// Silliest little test possible.
func TestOpenClose(t *testing.T) {
	tok := start(t)
	stop(t, tok)
}

// Test Token.Lookup() and KStat.GetNamed() (because we can't test
// the latter without the former)
//
// cpu:0:sys:syscall is a Uint64 and should always be non-zero.
func TestLookupNamed(t *testing.T) {
	tok := start(t)
	ks := lookup(t, tok, "cpu", "sys")
	// Verify that the kstat has the correct label. This doesn't just
	// check for us getting the right kstat, it also checks that no
	// trailing \x00s or other random garbage has wound up in the
	// fields (since they are actually taken from the kernel data,
	// not copied from the arguments we gave).
	if ks.Module != "cpu" || ks.Name != "sys" || ks.Instance != 0 || ks.Class != "misc" {
		t.Fatalf("cpu:0:sys kstat has bad module/instance/name/class value(s): %#v", ks)
	}

	n, err := ks.GetNamed("syscall")
	if err != nil {
		t.Fatalf("1st getting cpu:-1:sys:syscall failure: %s", err)
	}
	if n.Type != kstat.Uint64 {
		t.Fatalf("%s has wrong type: %s", n, n.Type)
	}
	// Non-set fields should be zero valued, set field should
	// have a non-zero value.
	if n.UintVal == 0 || n.StringVal != "" || n.IntVal != 0 {
		t.Fatalf("%s value fields are not right: %#v", n, n)
	}
	if n.Name != "syscall" {
		t.Fatalf("%s kstat name mismatch: should be \"syscall\", is %q.", ks, n.Name)
	}

	// Do it directly:
	n1, err := tok.GetNamed("cpu", -1, "sys", "syscall")
	if err != nil {
		t.Fatalf("2nd getting cpu:-1:sys:syscall' failure: %s", err)
	}

	// we cannot look at UintVal because it may or may not change (!?)
	// answer: tok.GetNamed() does tok.Lookup() which does ks.Refresh()
	// behind the scenes, which updates the kstat data.
	if n1.Type != n.Type || n1.StringVal != n.StringVal || n1.IntVal != n.IntVal {
		t.Fatalf("inconsistent values between original and tok.GetNamed: %#v vs %#v", n, n1)
	}

	stop(t, tok)
}

// Test that KStat.Refresh() seems to do something. This is a bit
// chancy, because we have to find a kstat that's guaranteed to
// change. We pick unix:*:system_misc:clk_intr and sleep for a
// second, because that seems like a pretty good bet.
//
// Note that picking eg cpu:0:sys:syscalls is risky, because it's a
// per-CPU statistic and in a multi-cpu system that CPU may not see
// syscalls ... or it may.
func TestRefresh(t *testing.T) {
	tok := start(t)
	ks := lookup(t, tok, "unix", "system_misc")
	n := kgetnamed(t, ks, "clk_intr")
	if n.Type != kstat.Uint32 {
		t.Fatalf("%s has wrong type: %s", n, n.Type)
	}
	osnap := ks.Snaptime

	time.Sleep(time.Second * 1)

	err := ks.Refresh()
	if err != nil {
		t.Fatalf("%s refresh error: %s", ks, err)
	}
	if osnap == ks.Snaptime {
		t.Fatalf("%s snaptime did not change: %d", ks, osnap)
	}

	n2 := kgetnamed(t, ks, "clk_intr")
	if n2.UintVal == n.UintVal {
		t.Fatalf("clk_intr count did not change: still %d\n", n2.UintVal)
	}
	stop(t, tok)
}

// Explicitly test that a .Lookup() calls .Refresh() aka kstat_read()
// and so sets ks.Snaptime
func TestLookupSnaptime(t *testing.T) {
	tok := start(t)
	ks := lookup(t, tok, "cpu", "sys")
	if ks.Snaptime <= 0 {
		t.Fatalf("%s Snaptime is zero (unset) after .Lookup()", ks)
	}
	stop(t, tok)
}

// General testing of Snaptime(s) not updating and then updating.
func TestSnaptimes(t *testing.T) {
	tok := start(t)
	ks := lookup(t, tok, "cpu", "sys")
	osnap := ks.Snaptime
	n := kgetnamed(t, ks, "syscall")
	if osnap != ks.Snaptime {
		t.Fatalf("getting %s changed KStat snaptime: %d vs %d", n, osnap, ks.Snaptime)
	}
	if n.Snaptime != ks.Snaptime {
		t.Fatalf("%s snaptime not KStat snaptime: %d vs %d", n, n.Snaptime, ks.Snaptime)
	}
	n2 := kgetnamed(t, ks, "sysread")
	if n2.Snaptime != n.Snaptime {
		t.Fatalf("%s snaptime not %s snaptime: %d vs %d", n2, n, n2.Snaptime, n.Snaptime)
	}

	err := ks.Refresh()
	if err != nil {
		t.Fatalf("%s Refresh() error: %s", ks, err)
	}
	if osnap == ks.Snaptime {
		t.Fatalf("%s Snaptime not updated after Refresh", ks)
	}
	stop(t, tok)
}

// Test that we do nothing after Token.Close().
func TestPostClose(t *testing.T) {
	tok := start(t)
	ks := lookup(t, tok, "cpu", "sys")
	stop(t, tok)

	_, err := tok.Lookup("cpu", -1, "sys")
	if err == nil {
		t.Fatalf("Lookup succeeds after Close")
	}

	err = ks.Refresh()
	if err == nil {
		t.Fatalf("KStat Refresh succeeds after Close")
	}

	res := tok.All()
	if len(res) > 0 {
		t.Fatalf("tok.All succeeds after Close")
	}

	_, err = ks.GetNamed("trap")
	if err == nil {
		t.Fatalf("ks.GetNamed succeeds after Close")
	}

	_, err = ks.AllNamed()
	if err == nil {
		t.Fatalf("ks.AllNamed succeeds after Close")
	}
}

// Do a very simple test of Token.All() and KStat.AllNamed()
func TestAlls(t *testing.T) {
	tok := start(t)
	lst := tok.All()
	if len(lst) == 0 {
		t.Fatalf("tok.All gave us a zero-length list")
	}
	ks := lookup(t, tok, "cpu", "sys")
	l2, err := ks.AllNamed()
	if err != nil {
		t.Fatalf("%s AllNamed failed: %s", ks, err)
	}
	if len(l2) == 0 {
		t.Fatalf("%s AllNamed was empty", ks)
	}
	stop(t, tok)
}

// Test that we cannot do GetNamed() or AllNamed() on things that are
// not named stats.
// We use unix:0:vminfo as our test kstat for this, and also try to
// test against sd:0:sd0 (which should be an IoStat).
func TestNotNamed(t *testing.T) {
	tok := start(t)
	ks := lookup(t, tok, "unix", "vminfo")
	if ks.Type != 0 {
		t.Fatalf("unix:-1:vminfo has crazy type: %#v", ks)
	}
	// kstat -p says that this is a real field, but of course it
	// doesn't actually exist as a named field; the name is a
	// structure element and is mocked up by kstat(1).
	r, err := ks.GetNamed("swap_alloc")
	if err == nil {
		t.Fatalf("getting %s as a named succeeded", r)
	}
	_, err = ks.AllNamed()
	if err == nil {
		t.Fatalf("ks.AllNamed() on %s succeeded", ks)
	}

	// Checking for success on lookup is paranoid, but ehh.
	ks, err = tok.Lookup("sd", 0, "sd0")
	if err != nil && ks.Type == kstat.IoStat {
		_, err = ks.AllNamed()
		if err == nil {
			t.Fatalf("ks.AllNamed on %s succeeded", ks)
		}
	}
	stop(t, tok)
}

// Test that getting invalid kstats or invalid named fields in
// valid kstats fails.
func TestNoSuch(t *testing.T) {
	tok := start(t)
	ks := lookup(t, tok, "cpu", "sys")
	res, err := ks.GetNamed("nosuch")
	if err == nil {
		t.Fatalf("ks.GetNamed succeeded: %#v", res)
	}
	res2, err := tok.Lookup("nosuch", -1, "nosuch")
	if err == nil {
		t.Fatalf("tok.Lookup succeeded: %#v", res2)
	}
	stop(t, tok)
}

// Test empty-string lookups in both module and name portions.
// These should map to NULLs in the underlying kstat_lookup()
// call and then give us (predictable) wildcard results.
func TestWildcard(t *testing.T) {
	tok := start(t)

	// this should be cpu:0:sys, I believe
	res, err := tok.Lookup("", -1, "sys")
	if err != nil {
		t.Fatalf("*:-1:sys lookup failed: %s", err)
	}
	if res.Instance != 0 || res.Module != "cpu" {
		t.Fatalf("unexpected lookup return 1: %#v", res)
	}

	// this should be acpi:0:acpi.
	res, err = tok.Lookup("acpi", -1, "")
	if err != nil {
		t.Fatalf("acpi:-1:* lookup failed: %s", err)
	}
	if res.Instance != 0 || res.Name != "acpi" {
		t.Fatalf("unexpected lookup return 2: %#v", res)
	}
	stop(t, tok)
}

// Test that a given kstat has a constant KStat structure, even
// when looked up through multiple paths.
func TestSameKStat(t *testing.T) {
	tok := start(t)
	ks := lookup(t, tok, "cpu", "sys")
	k2 := lookup(t, tok, "cpu", "sys")
	if ks != k2 {
		t.Fatalf("two lookups returned different KStat: %p %p", ks, k2)
	}
	n, err := tok.GetNamed("cpu", -1, "sys", "syscall")
	if err != nil {
		t.Fatalf("getnamed cpu:-1:sys:syscall failed: %s", err)
	}
	if ks != n.KStat {
		t.Fatalf("lookup returned a different KStat than via GetNamed: %p %p", ks, n.KStat)
	}
	n2, err := tok.GetNamed("cpu", -1, "sys", "syscall")
	if err != nil {
		t.Fatalf("2nd getnammed cpu:-1:sys:syscall failed: %s", err)

	}
	if ks != n2.KStat {
		t.Fatalf("second GetNamed returned a different KStat: %p %p", ks, n2.KStat)
	}
	stop(t, tok)
}

// Token.GetNamed; fail on error
func getnamed(t *testing.T, tok *kstat.Token, module, name, stat string) *kstat.Named {
	n, err := tok.GetNamed(module, -1, name, stat)
	if err != nil {
		t.Fatalf("getnamed %s:-1:%s:%s failed: %s", module, name, stat, err)
	}
	return n
}

// Test named kstat stats other than Uint*
//
// We assume there will always be a cpu_info:*:cpu_info0 kstat, although
// maybe there are machines with CPUs but no 0. I wave my hands.
//
// We also assume that CPU 0 is online at one point.
func TestNamedTypes(t *testing.T) {
	tok := start(t)

	n := getnamed(t, tok, "cpu_info", "cpu_info0", "state")
	if n.Type != kstat.CharData || n.StringVal == "" || n.UintVal != 0 || n.IntVal != 0 {
		t.Fatalf("bad type or value for %s %s: %#v", n, n.Type, n)
	}

	// This test may create false negatives if cpu0 is offline,
	// but I want a test to verify that the CharData copying code
	// does not add junk on the end and does copy the whole
	// string, since I had just such a bug at one point.
	if n.StringVal != "on-line" {
		t.Fatalf("bad value for %s: %#v", n, n)
	}

	n = getnamed(t, tok, "cpu_info", "cpu_info0", "brand")
	if n.Type != kstat.String || n.StringVal == "" || n.UintVal != 0 || n.IntVal != 0 {
		t.Fatalf("bad type or value for %s %s: %#v", n, n.Type, n)
	}

	n = getnamed(t, tok, "cpu_info", "cpu_info0", "family")
	if n.Type != kstat.Int32 || n.StringVal != "" || n.UintVal != 0 || n.IntVal == 0 {
		t.Fatalf("bad type or value for %s %s: %#v", n, n.Type, n)
	}

	n = getnamed(t, tok, "cpu_info", "cpu_info0", "clock_MHz")
	if n.Type != kstat.Int64 || n.StringVal != "" || n.UintVal != 0 || n.IntVal == 0 {
		t.Fatalf("bad type or value for %s %s: %#v", n, n.Type, n)
	}
	stop(t, tok)
}

//
// Find and retrieve an IO kstat.
// We try to look for stats having some non-zero value.
//
// Since apparently sd:0:sd0 is guaranteed to be the boot disk, we
// also specifically retrieve it and try to verify that as many fields
// as possible are good.
func TestDiskStat(t *testing.T) {
	tok := start(t)
	tlst := tok.All()
	foundone := false
	nonzero := false
	for _, ks := range tlst {
		if ks.Type != kstat.IoStat {
			continue
		}
		foundone = true
		io, err := ks.GetIO()
		if err != nil {
			t.Fatalf("%s GetIO failed: %s", ks, err)
		}
		if ks.Snaptime == 0 {
			t.Fatalf("%s Snaptime is still 0", ks)
		}
		if io.Reads > 0 || io.Writes > 0 {
			nonzero = true
			break
		}
	}
	if !foundone {
		t.Fatalf("failed to find a single IO KStat, should be impossible?")
	}
	if !nonzero {
		t.Fatalf("could not find a single IO KStat with read or write activity, should be impossible?")
	}

	// Okay, try sd:0:sd0 specifically.
	ks := lookup(t, tok, "sd", "sd0")
	io, err := ks.GetIO()
	if err != nil {
		t.Fatalf("%s GetIO error: %s", io, err)
	}
	// We assume the boot disk has to have some IO, right?
	if io.Nread == 0 || io.Nwritten == 0 || io.Rlastupdate == 0 || io.Rlentime == 0 || io.Wtime == 0 {
		t.Fatalf("%s IO values are odd: %+v", ks, io)
	}

	stop(t, tok)
}

// GetIO on a KStat for a closed token should fail, as other things do.
func TestDiskAfterClose(t *testing.T) {
	tok := start(t)
	ks := lookup(t, tok, "sd", "sd0")
	if ks.Type != kstat.IoStat {
		t.Fatalf("%s not an IoStat, is a %s", ks, ks.Type)
	}
	stop(t, tok)
	_, err := ks.GetIO()
	if err == nil {
		t.Fatalf("%s GetIO succeeded after Close", ks)
	}
}

// Calling GetIO should update the KStat Snaptime, because it should
// refresh the data. We check since we explicitly guarantee this in
// the documentation.
func TestDiskSnaptime(t *testing.T) {
	tok := start(t)
	ks := lookup(t, tok, "sd", "sd0")
	osnap := ks.Snaptime
	_, err := ks.GetIO()
	if err != nil {
		t.Fatalf("%s GetIO failed: %s", ks, err)
	}
	if ks.Snaptime == osnap {
		t.Fatalf("%s Snaptime did not change after GetIO", ks)
	}
	stop(t, tok)
}

// This tries to test that that our usage of runtime.SetFinalizer()
// at least doesn't crash when we try to call the finalizer.
func TestTokenFinalizer(t *testing.T) {
	tok := start(t)
	tok = nil
	runtime.GC()
	_ = tok
}
