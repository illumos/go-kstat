/*
 * CDDL HEADER START
 *
 * The contents of this file are subject to the terms of the
 * Common Development and Distribution License (the "License").
 * You may not use this file except in compliance with the License.
 *
 * You can obtain a copy of the license at usr/src/OPENSOLARIS.LICENSE
 * or http://www.opensolaris.org/os/licensing.
 * See the License for the specific language governing permissions
 * and limitations under the License.
 *
 * When distributing Covered Code, include this CDDL HEADER in each
 * file and include the License file at usr/src/OPENSOLARIS.LICENSE.
 * If applicable, add the following below this CDDL HEADER, with the
 * fields enclosed by brackets "[]" replaced with your own identifying
 * information: Portions Copyright [yyyy] [name of copyright owner]
 *
 * CDDL HEADER END
 */
/*
 * Copyright (c) 1986, 2010, Oracle and/or its affiliates. All rights reserved.
 */

/*	Copyright (c) 1983, 1984, 1985, 1986, 1987, 1988, 1989 AT&T	*/
/*	  All Rights Reserved  	*/

/*
 * cks note: cgo can't handle the standard struct mntinfo_kstat because
 * it has an embedded anonymous struct type, which run into
 * https://github.com/golang/go/issues/5253
 *
 * Our crude solution is to create a variant version with the struct
 * type made non-anonymous. We can use cgo to convert, and then in
 * this case I reversed the out-of-lining of the struct type and
 * verified that the re-inlined version (in types_solaris_amd64.go)
 * is the same size as the C version and presumably the same alignment.
 */

#include <sys/utsname.h>
#include <sys/kstat.h>
#include <sys/time.h>
#include <vm/page.h>
#include <sys/thread.h>
#include <nfs/rnode.h>
#include <sys/list.h>
#include <sys/condvar_impl.h>
#include <sys/zone.h>

#define	NFS_CALLTYPES	3	/* Lookups, Reads, Writes */

/*
 * Pull this inlined structure out of line so that cgo can process it.
 */
struct mnti_timer {
	uint32_t srtt;
	uint32_t deviate;
	uint32_t rtxcur;
};

/*
 * Read-only mntinfo statistics
 */
struct mntinfo_kstat_cgo {
	char		mik_proto[KNC_STRSIZE];
	uint32_t	mik_vers;
	uint_t		mik_flags;
	uint_t		mik_secmod;
	uint32_t	mik_curread;
	uint32_t	mik_curwrite;
	int		mik_timeo;
	int		mik_retrans;
	uint_t		mik_acregmin;
	uint_t		mik_acregmax;
	uint_t		mik_acdirmin;
	uint_t		mik_acdirmax;
	struct mnti_timer mik_timers[NFS_CALLTYPES+1];
	uint32_t	mik_noresponse;
	uint32_t	mik_failover;
	uint32_t	mik_remap;
	char		mik_curserver[SYS_NMLN];
};
