//
// Initially created by
//	cgo -godefs ctypes_solaris.go
//
// Now contains edits for documentation. This is considered okay by me
// because these structs are not exactly likely to change any time
// soon; that would break API compatibility.
//
// This is specific to amd64. It's unlikely that Go will support
// 32-bit Solaris ('386'), but.

package kstat

// IO represents the entire collection of KStat (disk) IO statistics
// exposed by an IoStat type KStat.
//
// Because IO is an exact copy of the C kstat_io_t structure from the
// kernel, it does not have a Snaptime or KStat field. You must save
// that information separately if you need it, perhaps by embedded the
// IO struct as an anonymous struct in an additional struct of your
// own.
type IO struct {
	Nread       uint64
	Nwritten    uint64
	Reads       uint32
	Writes      uint32
	Wtime       int64
	Wlentime    int64
	Wlastupdate int64
	Rtime       int64
	Rlentime    int64
	Rlastupdate int64
	Wcnt        uint32
	Rcnt        uint32
}

// Sysinfo is the data from unix:0:sysinfo, which is a sysinfo_t.
type Sysinfo struct {
	Updates uint32
	Runque  uint32
	Runocc  uint32
	Swpque  uint32
	Swpocc  uint32
	Waiting uint32
}

// Vminfo is the data from unix:0:vminfo, which is a vminfo_t
type Vminfo struct {
	Freemem uint64
	Resv    uint64
	Alloc   uint64
	Avail   uint64
	Free    uint64
	Updates uint64
}

// Var is the data from unix:0:var, which is a 'struct var'
type Var struct {
	Buf       int32
	Call      int32
	Proc      int32
	Maxupttl  int32
	Nglobpris int32
	Maxsyspri int32
	Clist     int32
	Maxup     int32
	Hbuf      int32
	Hmask     int32
	Pbuf      int32
	Sptmap    int32
	Maxpmem   int32
	Autoup    int32
	Bufhwm    int32
}

// Mntinfo is the data from nfs:*:mntinfo, which is a 'struct
// mntinfo_kstat'.
type Mntinfo struct {
	Proto    [128]int8
	Vers     uint32
	Flags    uint32
	Secmod   uint32
	Curread  uint32
	Curwrite uint32
	Timeo    int32
	Retrans  int32
	Acregmin uint32
	Acregmax uint32
	Acdirmin uint32
	Acdirmax uint32
	Timers   [4]struct {
		Srtt    uint32
		Deviate uint32
		Rtxcur  uint32
	}
	Noresponse uint32
	Failover   uint32
	Remap      uint32
	Curserver  [257]int8
	pad0       [3]byte
}

// The Mntinfo type is not an exact conversion as produced by cgo;
// because the original struct mntinfo_kstat contains an embedded
// anonymously typed struct, it runs into
// https://github.com/golang/go/issues/5253. This version is manually
// produced from a cgo starting point and then verified to be the same
// size.
