//
// All of these tests depend on being able to know what kstats exist
// on any Illumos / Solaris machine. I believe I've selected kstats
// that will always be there, but I could turn out to be wrong.
//
// We do our testing from outside the kstat package, so we see only
// its public API. For our purposes this is good enough.

package kstat_test

import (
	"testing"
	"time"

	"github.com/siebenmann/go-kstat"
)

// Utility functions

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

	// Do it directly:
	n1, err := tok.GetNamed("cpu", -1, "sys", "syscall")
	if err != nil {
		t.Fatalf("2nd getting cpu:-1:sys:syscall' failure: %s", err)
	}

	// we cannot look at UintVal because it may or may not change (!?)
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
//
// kstats apparently may change even without a second kstat_read()
// call (?! as they say), so this is not a definitive test that
// KStat.Refresh() actually does anything. But we do what we can.
func TestRefresh(t *testing.T) {
	tok := start(t)
	ks := lookup(t, tok, "unix", "system_misc")
	n, err := ks.GetNamed("clk_intr")
	if err != nil {
		t.Fatalf("1st getting %s 'clk_intr' failed: %s", ks, err)
	}
	if n.Type != kstat.Uint32 {
		t.Fatalf("%s has wrong type: %s", n, n.Type)
	}

	time.Sleep(time.Second * 1)

	ks.Refresh()
	n2, err := ks.GetNamed("clk_intr")
	if err != nil {
		t.Fatalf("2nd getting %s 'clk_intr' failed: %s", ks, err)
	}
	if n2.UintVal == n.UintVal {
		t.Fatalf("clk_intr count did not change: still %d\n", n2.UintVal)
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
// We use unix:0:vminfo as our test kstat for this.
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
func TestNamedTypes(t *testing.T) {
	tok := start(t)

	n := getnamed(t, tok, "cpu_info", "cpu_info0", "state")
	if n.Type != kstat.CharData || n.StringVal == "" || n.UintVal != 0 || n.IntVal != 0 {
		t.Fatalf("bad type or value for %s %s: %#v", n, n.Type, n)
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
