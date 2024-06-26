package main

import (
	"fmt"
	"io"
	"log"
	"os"

	"github.com/spf13/pflag"
)

func showOps(w io.Writer, list *OpList) {
	op := list.Front
	for op != nil {
		// if op.Leader {
		// 	fmt.Print("leader: ")
		// }
		fmt.Fprintf(w, "%v\n", op.Value)
		op = op.Next
	}
}

var instrument = pflag.Bool("inst", false, "add instrumentation for profiling")
var opt = pflag.IntP("opt", "O", 1, "optimization level")
var noloads = pflag.Bool("no-loads", false, "do not sandbox loads")
var poc = pflag.Bool("poc", false, "enable position-oblivious code")
var gasDirect = pflag.Bool("gas", false, "enable direct gas metering")
var gasRel = pflag.Bool("gas-rel", false, "enable relative gas metering")
var hideSys = pflag.Bool("hide-sys", false, "keep the syspage outside the sandbox")
var precise = pflag.Bool("precise", false, "use precise relative gas")
var align = pflag.Bool("aligned", false, "use 16-byte aligned bundles")
var gas bool

func main() {
	out := pflag.StringP("output", "o", "", "output file")
	native := pflag.Bool("native", false, "do not include any guards")

	pflag.Parse()
	args := pflag.Args()
	if len(args) < 1 {
		log.Fatal("no input")
	}

	gas = *gasDirect || *gasRel

	if (*poc || gas) && *opt >= 2 {
		*opt = 1
	}

	f, err := os.Open(args[0])
	if err != nil {
		log.Fatal(err)
	}
	ops, err := ParseFile(f, args[0])
	if err != nil {
		log.Fatal(err)
	}
	if *opt <= 1 {
		reserved[optReg] = false
		reserved[optReg2] = false
	}
	if *hideSys {
		sysReg = Reg("x25")
	}

	// MarkLeaders(*ops)
	if *native {
		syscallPass(true, ops)
	} else {
		branchPass(ops)
		fixupReservedPass(ops)
		syscallPass(false, ops)
		if *poc {
			posObliviousPass(ops)
		}
		if *opt >= 2 {
			rangePass(ops)
		}
		memPass(ops)
		specialRegPass(ops)
		if *instrument {
			instrumentPass(ops)
		}
		if *opt >= 2 && !*noloads {
			preExtensionPass(ops)
		}
		// if gas {
		// 	alignLabelsPass(ops)
		// }
		if *gasRel && *precise {
			branchPrecisePass(ops)
		}
		if *gasDirect {
			gasDirectPass(ops)
		} else if *gasRel {
			gasRelativePass(ops)
		}
		branchFixupPass(ops)
	}

	var w io.Writer
	if *out != "" {
		f, err := os.Create(*out)
		if err != nil {
			log.Fatal(err)
		}
		w = f
		defer f.Close()
	} else {
		w = os.Stdout
	}
	showOps(w, ops)
}
