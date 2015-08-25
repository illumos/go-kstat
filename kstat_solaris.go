//
// Package kstat provides access to the Solaris/OmniOS kstat(s) system
// for user-level access to kernel statistics. For more documentation on
// this, see kstat(1) and kstat(3kstat).
//
// At the moment this can only retrieve named counters/kstats, although
// you can see the names and types of other kstats.
//
// This package may leak memory, especially since the Solaris kstat
// manpage is not clear on the requirements here. However I believe
// it's reasonably memory safe. It is possible to totally corrupt
// memory with use-after-free errors if you do operations on kstats
// after calling Token.Close().
//
// This is a cgo-based package. Cross compilation is up to you.
// Goroutine safety is in no way guaranteed because the underlying
// C kstat library is probably not thread or goroutine safe.
//
// General usage: call Open() to obtain a Token, then call GetNamed()
// on it to obtain Named(s) for specific kstats. If you want a number
// of kstats for a module:inst:name trio, it is more efficient to
// call .Lookup() to obtain a KStats and then call .GetNamed() on
// it.
//
// TODO: support kstat_io (KSTAT_TYPE_IO) stats.
// (There are also some KSTAT_TYPE_RAW stats, but not even kstat(1)
// prints them.)
//
// Author: Chris Siebenmann
// https://github.com/siebenmann/go-kstat
//
// Copyright: standard Go copyright.
package kstat

// #cgo LDFLAGS: -lkstat
//
// #include <sys/types.h>
// #include <stdlib.h>
// #include <strings.h>
// #include <kstat.h>
//
// /* We have to reach through unions, which cgo doesn't support.
//    So we have our own cheesy little routines for it. These assume
//    they are always being called on validly-typed named kstats.
//  */
//
// char *get_named_char(kstat_named_t *knp) {
//	return knp->value.str.addr.ptr;
// }
//
// uint64_t get_named_uint(kstat_named_t *knp) {
//	if (knp->data_type == KSTAT_DATA_UINT32)
//		return knp->value.ui32;
//	else
//		return knp->value.ui64;
// }
//
// int64_t get_named_int(kstat_named_t *knp) {
//	if (knp->data_type == KSTAT_DATA_INT32)
//		return knp->value.i32;
//	else
//		return knp->value.i64;
// }
//
import "C"

import (
	"errors"
	"fmt"
	"unsafe"
)

// Token is an access token for obtaining kstats.
type Token struct {
	kc *C.struct_kstat_ctl
}

// Open returns a kstat Token that is used to obtain kstats. It corresponds
// to kstat_open(). You should call .Close() when you're done and then not
// use any KStats or Nameds obtained through this token.
//
// (Failing to call .Close() will cause memory leaks.)
func Open() (*Token, error) {
	r, err := C.kstat_open()
	if r == nil {
		return nil, err
	}
	t := Token{r}
	return &t, nil
}

// Close a kstat access token. A closed token cannot be used for
// anything and cannot be reopened.
//
// After a Token has been closed it remains safe to look at fields
// on KStats and Named objects obtained through the Token, but it is
// not safe to call methods on them other than String(); doing so
// may cause memory corruption, although we try to avoid that.
//
// This corresponds to kstat_close().
func (t *Token) Close() error {
	if t.kc == nil {
		return nil
	}
	res, err := C.kstat_close(t.kc)
	t.kc = nil
	if res != 0 {
		return err
	}
	return nil
}

// All returns an array of all available KStats.
func (t *Token) All() []*KStats {
	n := []*KStats{}
	if t.kc == nil {
		return n
	}

	r := t.kc.kc_chain
	for {
		if r == nil {
			break
		}
		n = append(n, newKStats(t, r))
		r = r.ks_next
	}
	return n
}

//
// allocate a C string for a non-blank string
func maybeCString(src string) *C.char {
	if src == "" {
		return nil
	}
	return C.CString(src)
}

// free a non-nil C string
func maybeFree(cs *C.char) {
	if cs != nil {
		C.free(unsafe.Pointer(cs))
	}
}

// Lookup looks up a particular kstat. module and name may be "" and
// instance may be -1 to mean 'the first one that kstats can find'.
//
// Lookup() corresponds to kstat_lookup().
//
// Right now you cannot do anything useful with non-named kstats
// (as we don't provide any method to retrieve their data).
func (t *Token) Lookup(module string, instance int, name string) (*KStats, error) {
	if t == nil || t.kc == nil {
		return nil, errors.New("Token not valid or closed")
	}

	ms := maybeCString(module)
	ns := maybeCString(name)
	r, err := C.kstat_lookup(t.kc, ms, C.int(instance), ns)
	maybeFree(ms)
	maybeFree(ns)

	if r == nil {
		return nil, err
	}

	k := newKStats(t, r)
	// People rarely look up kstats to not use them, so we immediately
	// attempt to kstat_read() the data. If this fails, we don't return
	// the kstat.
	err = k.Refresh()
	if err != nil {
		return nil, err
	}
	return k, nil
}

// GetNamed obtains the Named representing a particular (named) kstat
// module:instance:name:statistic statistic.
//
// It is functionally equivalent to .Lookup() then KStats.GetNamed().
func (t *Token) GetNamed(module string, instance int, name, stat string) (*Named, error) {
	stats, err := t.Lookup(module, instance, name)
	if err != nil {
		return nil, err
	}
	return stats.GetNamed(stat)
}

// -----

// KStats is the access handle for the collection of statistics for a
// particular module:instance:name kstat.
//
type KStats struct {
	Module   string
	Instance int
	Name     string

	// Class is eg 'net' or 'disk'. In kstat(1) it shows up as a
	// ':class' statistic.
	Class string
	// Type is the type of kstat. Named kstats are the only type
	// actively supported.
	Type int

	// Creation time of a kstat in nanoseconds since sometime.
	// See gethrtime(3) and kstat(3kstat).
	Crtime int64
	// Snaptime is what kstat(1) reports as 'snaptime', the time
	// that this data was obtained. As with Crtime, it is in
	// nanoseconds since some arbitrary point in time.
	// Snaptime may not be valid until .Refresh() has been called.
	Snaptime int64

	ksp *C.struct_kstat
	// We need access to the token to refresh the data
	tok *Token
}

// internal constructor.
func newKStats(tok *Token, ks *C.struct_kstat) *KStats {
	kst := KStats{}
	kst.ksp = ks
	kst.tok = tok

	kst.Instance = int(ks.ks_instance)
	kst.Module = C.GoString((*C.char)(unsafe.Pointer(&ks.ks_module)))
	kst.Name = C.GoString((*C.char)(unsafe.Pointer(&ks.ks_name)))
	kst.Class = C.GoString((*C.char)(unsafe.Pointer(&ks.ks_class)))
	kst.Type = int(ks.ks_type)
	kst.Crtime = int64(ks.ks_crtime)
	kst.Snaptime = int64(ks.ks_snaptime)

	return &kst
}

// invalid is a desperate attempt to keep usage errors from causing
// memory corruption. Don't count on it.
func (k *KStats) invalid() bool {
	return k == nil || k.ksp == nil || k.tok == nil || k.tok.kc == nil
}

func (k *KStats) String() string {
	return fmt.Sprintf("%s:%d:%s (%s)", k.Module, k.Instance, k.Name, k.Class)
}

// Refresh the statistics data for a KStats.
//
// Under the hood this does a kstat_read(). You don't need to call it
// explicitly before using a KStats.
func (k *KStats) Refresh() error {
	if k.invalid() {
		return errors.New("invalid KStats or closed token")
	}

	res, err := C.kstat_read(k.tok.kc, k.ksp, unsafe.Pointer(nil))
	if res == -1 {
		return err
	}
	k.Snaptime = int64(k.ksp.ks_snaptime)
	return nil
}

// GetNamed obtains a particular named statistic from a kstat.
//
// It corresponds to kstat_data_lookup().
func (k *KStats) GetNamed(name string) (*Named, error) {
	if k.invalid() {
		return nil, errors.New("invalid KStats or closed token")
	}

	if k.ksp.ks_type != C.KSTAT_TYPE_NAMED {
		return nil, fmt.Errorf("kstat %s (type %d) is not a named kstat", k, k.ksp.ks_type)
	}

	// Do the initial load of the data if necessary.
	if k.ksp.ks_data == nil {
		err := k.Refresh()
		if err != nil {
			return nil, err
		}
	}

	ns := C.CString(name)
	r, err := C.kstat_data_lookup(k.ksp, ns)
	C.free(unsafe.Pointer(ns))
	if r == nil || err != nil {
		return nil, err
	}
	return newNamed(k, (*C.struct_kstat_named)(r)), err
}

// Named represents a particular kstat named statistic, ie the full
//	module:instance:name:statistic
// and its current value.
//
// Name and Type are always valid, but only one of StringVal, IntVal,
// or UintVal is valid for any particular statistic; which one is
// valid is determined by its Type. Generally you'll already know what
// type a given kstat element is.
type Named struct {
	Name string
	Type NamedTypes

	// Only one of the following values is valid; the others are zero
	// values.
	//
	// StringVal holds the value for both CharData and String Type(s).
	StringVal string
	IntVal    int64
	UintVal   uint64

	// Pointer to the parent KStats, for access to the full name.
	KStats *KStats
}

func (ks *Named) String() string {
	return fmt.Sprintf("%s:%d:%s:%s", ks.KStats.Module, ks.KStats.Instance, ks.KStats.Name, ks.Name)
}

// NamedTypes represents the various types of named kstat elements.
type NamedTypes int

// Various NamedTypes
const (
	CharData = C.KSTAT_DATA_CHAR
	Int32    = C.KSTAT_DATA_INT32
	Uint32   = C.KSTAT_DATA_UINT32
	Int64    = C.KSTAT_DATA_INT64
	Uint64   = C.KSTAT_DATA_UINT64
	String   = C.KSTAT_DATA_STRING

	// CharData is found in StringVal. At the moment we assume that
	// it is a real string, because this matches how it seems to be
	// used for short strings in the Solaris kernel. Someday we may
	// find something that uses it as just a data dump for 16 bytes.

	// Solaris sys/kstat.h also has _FLOAT (5) and _DOUBLE (6) types,
	// but labels them as obsolete.
)

func (tp NamedTypes) String() string {
	switch tp {
	case CharData:
		return "char"
	case Int32:
		return "int32"
	case Uint32:
		return "uint32"
	case Int64:
		return "int64"
	case Uint64:
		return "uint64"
	case String:
		return "string"
	default:
		return fmt.Sprintf("type-%d", tp)
	}
}

// Create a new Stat from the kstat_named_t
// We set the appropriate *Value field.
func newNamed(k *KStats, knp *C.struct_kstat_named) *Named {
	st := Named{}
	st.KStats = k
	st.Name = C.GoString((*C.char)(unsafe.Pointer(&knp.name)))
	st.Type = NamedTypes(knp.data_type)

	switch st.Type {
	case String:
		// The comments in sys/kstat.h explicitly guarantee
		// that these strings are null-terminated, although
		// knp.value.str.len also holds the length.
		st.StringVal = C.GoString(C.get_named_char(knp))
	case CharData:
		// Solaris/etc appears to use CharData for short strings
		// so that they can be embedded directly into
		// knp.value.c[16] instead of requiring an out of line
		// allocation. In theory we may find someone who is
		// using it as 128-bit ints or the like.
		// However I scanned the Illumos kernel source and
		// everyone using it appears to really be using it for
		// strings. We'll still bound the length.
		// (GoStringN does 'up to ...', fortunately.)
		st.StringVal = C.GoStringN((*C.char)(unsafe.Pointer(&knp.value)), 16)
	case Int32, Int64:
		st.IntVal = int64(C.get_named_int(knp))
	case Uint32, Uint64:
		st.UintVal = uint64(C.get_named_uint(knp))
	default:
		// TODO: should do better.
		panic(fmt.Sprintf("unknown stat type: %d", st.Type))
	}
	return &st
}
