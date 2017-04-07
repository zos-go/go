// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package runtime

import (
	"runtime/internal/sys"
	"unsafe"
)

type sigctxt struct {
	info *siginfo
	ctxt unsafe.Pointer
}

func (c *sigctxt) regs() *sigcontext {
	return (*sigcontext)(unsafe.Pointer(&(*ucontext)(c.ctxt).uc_mcontext))
}
func (c *sigctxt) r0() uint64      { return c.regs().gregs[0] }
func (c *sigctxt) r1() uint64      { return c.regs().gregs[1] }
func (c *sigctxt) r2() uint64      { return c.regs().gregs[2] }
func (c *sigctxt) r3() uint64      { return c.regs().gregs[3] }
func (c *sigctxt) r4() uint64      { return c.regs().gregs[4] }
func (c *sigctxt) r5() uint64      { return c.regs().gregs[5] }
func (c *sigctxt) r6() uint64      { return c.regs().gregs[6] }
func (c *sigctxt) r7() uint64      { return c.regs().gregs[7] }
func (c *sigctxt) r8() uint64      { return c.regs().gregs[8] }
func (c *sigctxt) r9() uint64      { return c.regs().gregs[9] }
func (c *sigctxt) r10() uint64     { return c.regs().gregs[10] }
func (c *sigctxt) r11() uint64     { return c.regs().gregs[11] }
func (c *sigctxt) r12() uint64     { return c.regs().gregs[12] }
func (c *sigctxt) r13() uint64     { return c.regs().gregs[13] }
func (c *sigctxt) r14() uint64     { return c.regs().gregs[14] }
func (c *sigctxt) r15() uint64     { return c.regs().gregs[15] }
func (c *sigctxt) link() uint64    { return c.regs().gregs[14] }
func (c *sigctxt) sp() uint64      { return c.regs().gregs[15] }
func (c *sigctxt) pc() uint64      { return c.regs().psw_addr }
func (c *sigctxt) sigcode() uint32 { return uint32(c.info.si_code) }
func (c *sigctxt) sigaddr() uint64 { return c.info.si_addr }

func (c *sigctxt) set_r0(x uint64)      { c.regs().gregs[0] = x }
func (c *sigctxt) set_r13(x uint64)     { c.regs().gregs[13] = x }
func (c *sigctxt) set_link(x uint64)    { c.regs().gregs[14] = x }
func (c *sigctxt) set_sp(x uint64)      { c.regs().gregs[15] = x }
func (c *sigctxt) set_pc(x uint64)      { c.regs().psw_addr = x }
func (c *sigctxt) set_sigcode(x uint32) { c.info.si_code = int32(x) }
func (c *sigctxt) set_sigaddr(x uint64) {
	*(*uintptr)(add(unsafe.Pointer(c.info), 2*sys.PtrSize)) = uintptr(x)
}
