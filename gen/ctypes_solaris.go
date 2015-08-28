//
// This is an experiment in taking the tedium out of copying actual
// C structures from the kernel into Go structures that are a good
// fit for people to use. This file is to be processed with
// 'go tool cgo -godefs' to create the actual compiled file.
//
// According to a go-nuts mailing list thread, eg
//	http://grokbase.com/t/gg/golang-nuts/12cemmrhk5/go-nuts-cgo-cast-c-struct-to-go-struct
// these structures are directly compatible with the C structures and
// I may cast one to the other.
//
// +build ignore
//go:generate sh -c "go tool cgo -godefs ctypes_solaris.go >new_types_solaris.go"

package kstat

// #cgo LDFLAGS: -lkstat
//
// #include <kstat.h>
// #include <sys/kstat.h>
// #include <sys/sysinfo.h>
// #include <sys/var.h>
// #include <nfs/nfs_clnt.h>
//
// /* This is a gory hack */
// #include "mntinfo_cgo.h"
import "C"

// Disk IO in general.
type IO C.kstat_io_t

const Sizeof_IO = C.sizeof_struct_kstat_io

// unix:* stats:

// unix:0:sysinfo
type Sysinfo C.sysinfo_t

const Sizeof_SI = C.sizeof_sysinfo_t

// unix:0:vminfo
type Vminfo C.vminfo_t

const Sizeof_VI = C.sizeof_vminfo_t

// unix:0:var
type Var C.struct_var

const Sizeof_Var = C.sizeof_struct_var

// These CPU stats are apparently an obsolete form of what is now
// surfaced as named kstats in cpu:*:sys and cpu:*:vm. As such I'm
// currently not planning to support them. Inspection with kstat
// suggests that a significant number of these stats are actually
// zero, suggesting strongly that they are no longer relevant (no
// matter how attractive they look).

// cpu_stat*:*:cpu_stat*
// One copy exists for each different CPU in the system.
// Just to be irritating, their names go cpu_stat:0:cpu_stat0,
// cpu_stat:1:cpu_stat1, etc. Instance numbers are not good
// enough for you or what?
//
// (This is kind of acknowledged as a bug in Kstat.xs; the
// 'spec' is that you strip digits from the module and name
// before checking for a match.)
//
// Apparently you probably want cpu:*:sys and cpu:*:vm instead, which
// are newer? named kstats for much of the same thing.
//
//OBS:type C_sysinfo C.cpu_sysinfo_t
//OBS:type C_syswait C.cpu_syswait_t
//OBS:type C_vminfo C.cpu_vminfo_t

// CPU embeds all three of the above structures. Go team go!
// It is what cpu_stat:*:cpu_stat* actually returns.
//OBS:type CPU C.cpu_stat_t

//OBS:const (
//OBS:	CPU_IDLE   = C.CPU_IDLE
//OBS:	CPU_USER   = C.CPU_USER
//OBS:	CPU_KERNEL = C.CPU_KERNEL
//OBS:	CPU_WAIT   = C.CPU_WAIT
//OBS:
//OBS:	W_IO   = C.W_IO
//OBS:	W_SWAP = C.W_SWAP
//OBS:	W_PIO  = C.W_PIO
//OBS:)

//
// Other currently mysterious kstats of KSTAT_TYPE_RAW:
//
//	unix:0:kstat_headers
//
// a list of 'struct k_sockinfo's, used by netstat:
//	sockfs:0:sock_unix_list
//
// a list of 'struct memunit's, which are a pair of uint64s: address,size
//	unix:0:page_retire_list
//	mm:0:phys_installed
//
// kstat(1) does not print anything about any of these.

// ----

// Although kstat defines KSTAT_TYPE_TIMER, there is nothing in the
// current Illumos kernel source that actually sets up Timer kstats.
// Accordingly I decline to implement reading it.
// This is handy because Timer.Name being a [31]int8 instead of a
// string would be irritating to users.
//
// type Timer C.kstat_timer_t

// Although things in the kernel do create KSTAT_TYPE_INTR kstats,
// most of them appear to be very old drivers for very old hardware.
// The exception is the vioif driver. At the moment I don't feel
// like implementing reading this, partly because I have no way of
// testing it.
//
// type Intr C.kstat_intr_t

// ----

// Types that don't get converted (well) by cgo -godefs yet.
// Probably https://github.com/golang/go/issues/5253
//
// We can't handle this like the CPU type because its internal
// mik_timers struct is anonymous; there's no way to predeclare
// a Go struct for it, the way we do with CPU.
//
type KMntinfo C.struct_mntinfo_kstat

const Sizeof_KM = C.sizeof_struct_mntinfo_kstat

type MITimer C.struct_mnti_timer
type Mntinfo C.struct_mntinfo_kstat_cgo

const Sizeof_Mnti = C.sizeof_struct_mntinfo_kstat_cgo

// This type is explicitly marked as obsolete in its header file;
// export of it has apparently been long since replaced in practice
// with the use of named kstats.
// This is visible as unix:0:ncstats
//
// type Ncstats C.struct_ncstats
