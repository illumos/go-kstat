//
// Package kstat provides a Go interface to the Solaris/OmniOS
// kstat(s) system for user-level access to a lot of kernel
// statistics. For more documentation on kstats, see kstat(1) and
// kstat(3kstat).
//
// At the moment this can only retrieve what are called 'named' kstat
// statistics, although you can see the names and types of other
// kstats. Fortunately these are the most common and usually the most
// interesting, although this limit does mean that you can't currently
// retrieve disk IO stats. (This will change at some point.)
//
// General usage: call Open() to obtain a Token, then call GetNamed()
// on it to obtain Named(s) for specific statistics. Note that this
// always gives you the very latest value for the statistic. If you
// want a number of statistics from the same module:inst:name triplet
// (eg several network counters from the same network interface) and
// you want them all to have been gathered at the same time, you need
// to call .Lookup() to obtain a KStat and then repeatedly call its
// .GetNamed() (this is also slightly more efficient).
//
// The short version: a kstat is a collection of some related
// statistics, eg disk IO stats for a disk or various network counters
// for a particular network interface. A Token is a handle for a
// collection of kstats. You go collection (Token) -> kstat (KStat) ->
// specific statistic (Named) in order to retrieve the value of a
// specific statistic.
//
// This is a cgo-based package. Cross compilation is up to you.
// Goroutine safety is in no way guaranteed because the underlying
// C kstat library is probably not thread or goroutine safe (and
// there are some all-Go concurrency races involving .Close()).
//
// This package may leak memory, especially since the Solaris kstat
// manpage is not clear on the requirements here. However I believe
// it's reasonably memory safe. It's possible to totally corrupt
// memory with use-after-free errors if you do operations on kstats
// after calling Token.Close(), although we try to avoid that.
//
// NOTE: this package is quite young. The API may well change as
// I (and other people) gain more experience with it.
//
// PERFORMANCE
//
// In general this is not going to be as lean and mean as calling
// C directly, partly because of intrinsic CGo overheads and partly
// because we do more memory allocation and deallocation than a C
// program would (partly because we prioritize not leaking memory).
//
// API LIMITATIONS AND TODOS
//
// At the moment we don't support anything except named kstats, which
// are the most common ones. There are generic disk IO stats
// (kstat_io_t, KSTAT_TYPE_IO) and kstat(1) knows about a number of
// magic specific raw stats for eg unix:*:sysinfo and unix:*:vminfo
// that are potentially interesting.
//
// (These specific raw stats are listed in cmd/stat/kstat/kstat.h
// in the ks_raw_lookup array. See cmd/stat/kstat/kstat.c for how
// they're interpreted.)
//
// Although we support refreshing specific kstats via
// KStat.Refresh(), we don't support refreshing the entire collection
// of kstats in order to pick up entirely new kstats and so on.  In
// other words, we don't support kstat_chain_update(). At the moment
// you must do this by closing your current Token and opening a new
// one.
//
// Author: Chris Siebenmann
// https://github.com/siebenmann/go-kstat
//
// Copyright: standard Go copyright.
//
// (If you're reading this documentation on a non-Solaris platform,
// you're probably not seeing the detailed API documentation for
// constants, types, and so on because of tooling limitations in godoc
// et al.)
//
package kstat

//
// This exists in large part to give non-Solaris systems something
// that makes the package name visible. Otherwise eg goimports thinks
// that this is an empty package and deletes 'import ...' statements
// for it if you process a Go file that imports the package on a
// non-Solaris platform.
//
// Since this file exists, I've put the package-level documentation
// here to increase the odds that tools running on non-Solaris systems
// will be able to show you at least some documentation.
//
// This is a hack, and annoying.
