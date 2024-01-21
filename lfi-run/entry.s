// enter_sandbox(struct proc* p)
.text
.align 4
.globl enter_sandbox
.type enter_sandbox,@function
enter_sandbox:
	mov x19, x0
	bl signal_enable
	mov x0, x19
	b restore_regs
	// should not return
	brk #0

// save registers, assuming sandbox base is x21
.macro SAVE_REGS
	ldr x21,      [x21, #128] // x21 now points to struct proc
	stp x0, x1,   [x21, #8+16*0]
	stp x2, x3,   [x21, #8+16*1]
	stp x4, x5,   [x21, #8+16*2]
	stp x6, x7,   [x21, #8+16*3]
	stp x18, x19, [x21, #8+16*9]
	str x20,      [x21, #8+16*10]
	stp x22, x23, [x21, #8+16*11]
	stp x24, x25, [x21, #8+16*12]
	stp x26, x27, [x21, #8+16*13]
	stp x28, x29, [x21, #8+16*14]
	stp q8, q9,   [x21, #16*19+32*4]
	stp q10, q11, [x21, #16*19+32*5]
	stp q12, q13, [x21, #16*19+32*6]
	stp q14, q15, [x21, #16*19+32*7]
	mov x1, sp
	stp x30, x1,  [x21, #8+16*15]
	mrs x1, nzcv
	mrs x2, fpsr
	stp x1, x2, [x21, #8+8*34]
	mrs x1, tpidr_el0
	str x1,     [x21, #8+8*36]
	// reset x21 by loading it back
	ldr x21,    [x21, #8+16*10+8]
.endm

// save caller-saved registers, assuming sandbox base is x21
.macro SAVE_PARTIAL_REGS
	ldr x21,      [x21, #128] // x21 now points to struct proc
	stp x0, x1,   [x21, #8+16*0]
	stp x2, x3,   [x21, #8+16*1]
	stp x4, x5,   [x21, #8+16*2]
	stp x6, x7,   [x21, #8+16*3]
	stp x8, x9,   [x21, #8+16*4]
	stp x10, x11, [x21, #8+16*5]
	stp x12, x13, [x21, #8+16*6]
	stp x14, x15, [x21, #8+16*7]
	stp x16, x17, [x21, #8+16*8]
	str x18,      [x21, #8+16*9]
	mov x1, sp
	stp x30, x1,  [x21, #8+16*15]
	mrs x0, nzcv
	mrs x1, fpsr
	stp x0, x1, [x21, #8+8*34]
	mrs x0, tpidr_el0
	str x0,     [x21, #8+8*36]
	// reset x21 by loading it back
	ldr x21,    [x21, #8+16*10+8]
.endm

.text
.align 4
.globl syscall_entry
syscall_entry:
	SAVE_PARTIAL_REGS
	ldr x0, [x21, #128+8] // load kernel tpidr_el0
	msr tpidr_el0, x0
	ldr x0, [x21, #128]  // load struct proc*
	ldr x1, [x0]       // load stack
	mov sp, x1
	bl syscall_handler
	ldr x0, [x21, #128]
	b  restore_partial_regs
	brk #0

.globl restore_partial_regs
restore_partial_regs:
	ldp x1, x2, [x0, #8+8*34]
	msr nzcv, x1
	msr fpsr, x2
	ldr x1,       [x0, #8+8*36]
	msr tpidr_el0, x1
	ldp x2, x3,   [x0, #8+16*1]
	ldp x4, x5,   [x0, #8+16*2]
	ldp x6, x7,   [x0, #8+16*3]
	ldp x8, x9,   [x0, #8+16*4]
	ldp x10, x11, [x0, #8+16*5]
	ldp x12, x13, [x0, #8+16*6]
	ldp x14, x15, [x0, #8+16*7]
	ldp x16, x17, [x0, #8+16*8]
	ldr x18,      [x0, #8+16*9]
	ldp x30, x1,  [x0, #8+16*15]
	mov sp, x1
	ldp x0, x1,   [x0, #8+16*0]
	ret

// Yields directly to the sandbox referred to in x0, specified by pid. If no
// such sandbox exists, this will trap and the runtime can handle the fault.
.text
.align 4
.globl yield_fast
yield_fast:
	SAVE_REGS
	lsl x21, x0, #32
	// This access assumes every 4GB-aligned page is either accessible and
	// corresponds to a valid sandbox, or is inaccessible and will cause a
	// trap.
	ldr x0, [x21, #128]
	// fallthrough to restore_regs

// restore registers from the struct proc pointed to by x0
// does not return
.globl restore_regs
restore_regs:
	ldp x1, x2, [x0, #8+8*34]
	msr nzcv, x1
	msr fpsr, x2
	ldr x1,       [x0, #8+8*36]
	msr tpidr_el0, x1
	ldp x2, x3,   [x0, #8+16*1]
	ldp x4, x5,   [x0, #8+16*2]
	ldp x6, x7,   [x0, #8+16*3]
	mov x8, #0
	mov x9, #0
	mov x10, #0
	mov x11, #0
	mov x12, #0
	mov x13, #0
	mov x14, #0
	mov x15, #0
	mov x16, #0
	mov x17, #0
	ldp x18, x19, [x0, #8+16*9]
	ldp x20, x21, [x0, #8+16*10]
	ldp x22, x23, [x0, #8+16*11]
	ldp x24, x25, [x0, #8+16*12]
	ldp x26, x27, [x0, #8+16*13]
	ldp x28, x29, [x0, #8+16*14]
	ldp x30, x1,  [x0, #8+16*15]
	// clobber caller-saved registers
	movi v0.2D, #0
	movi v1.2D, #0
	movi v2.2D, #0
	movi v3.2D, #0
	movi v4.2D, #0
	movi v5.2D, #0
	movi v6.2D, #0
	movi v7.2D, #0
	ldp q8, q9,   [x0, #16*19+32*4]
	ldp q10, q11, [x0, #16*19+32*5]
	ldp q12, q13, [x0, #16*19+32*6]
	ldp q14, q15, [x0, #16*19+32*7]
	movi v16.2D, #0
	movi v17.2D, #0
	movi v18.2D, #0
	movi v19.2D, #0
	movi v20.2D, #0
	movi v21.2D, #0
	movi v22.2D, #0
	movi v23.2D, #0
	movi v24.2D, #0
	movi v25.2D, #0
	movi v26.2D, #0
	movi v27.2D, #0
	movi v28.2D, #0
	movi v29.2D, #0
	movi v30.2D, #0
	movi v31.2D, #0
	mov sp, x1
	ldp x0, x1,   [x0, #8+16*0]
	ret