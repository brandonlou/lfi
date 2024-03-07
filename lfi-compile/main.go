package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func fatal(err ...interface{}) {
	fmt.Fprintln(os.Stderr, err...)
	rmtemps()
	os.Exit(1)
}

func run(command string, args ...string) {
	cmd := exec.Command(command, args...)
	log.Println(cmd)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	cmd.Stdin = os.Stdin
	err := cmd.Run()
	if err != nil {
		fatal(err)
	}
}

var keep bool

var temps []string

func rmtemps() {
	if !keep {
		for _, t := range temps {
			os.Remove(t)
		}
	}
}

func temp(dir string) string {
	tmp, err := os.CreateTemp(dir, ".lfi.*.s")
	if err != nil {
		fatal(err)
	}
	tmp.Close()
	temps = append(temps, tmp.Name())
	return tmp.Name()
}

func compile(cmdargs []string) {
	compiler := cmdargs[0]

	var args, lfiargs, inputs, objs []string
	var compile, assemble, preprocess, verbose, lto bool
	var out string

	lfienv := os.Getenv("LFIFLAGS")
	if lfienv != "" {
		lfiargs = append(lfiargs, strings.Fields(lfienv)...)
	}

	for i := 1; i < len(cmdargs); i++ {
		arg := cmdargs[i]

		if strings.HasPrefix(arg, "-flfi") {
			lfiargs = append(lfiargs, arg[len("-flfi"):])
			continue
		}

		switch arg {
		case "-flto", "-flto=full", "-flto=thin":
			lto = true
			args = append(args, arg)
		case "-E":
			preprocess = true
		case "-c":
			compile = true
		case "-S":
			assemble = true
		case "-V":
			verbose = true
		case "-K":
			keep = true
		case "-o":
			if i+1 >= len(cmdargs) {
				fatal("-o needs an argument")
			}
			out = cmdargs[i+1]
			i++
		case "-MT", "-MQ":
			if i+1 >= len(cmdargs) {
				fatal("-MT needs an argument")
			}
			args = append(args, arg)
			args = append(args, cmdargs[i+1])
			i++
		default:
			switch {
			case strings.HasSuffix(arg, ".s"),
				strings.HasSuffix(arg, ".S"),
				strings.HasSuffix(arg, ".c"),
				strings.HasSuffix(arg, ".cxx"),
				strings.HasSuffix(arg, ".cc"),
				strings.HasSuffix(arg, ".cpp"),
				strings.HasSuffix(arg, ".c++"),
				strings.HasSuffix(arg, ".C"):
				inputs = append(inputs, arg)
			case strings.HasSuffix(arg, ".o"),
				strings.HasSuffix(arg, ".a"):
				objs = append(objs, arg)
			default:
				args = append(args, arg)
			}
		}
	}

	if verbose {
		log.SetOutput(os.Stdout)
		log.SetFlags(0)
		log.SetPrefix("LFI: ")
	} else {
		log.SetOutput(io.Discard)
	}

	if len(inputs) == 0 {
		var oldout string
		if preprocess {
			run(compiler, append(args, "-E")...)
			return
		}
		if lto {
			oldout = out
			out = temp(os.TempDir())
		} else if out == "" {
			out = "a.out"
		}
		flags := []string{
			"-o", out,
		}
		flags = append(flags, objs...)
		flags = append(flags, args...)
		if !lto {
			run(compiler, flags...)
			return
		}

		flags = append(flags, "-Wl,--lto-emit-asm", "-Wl,-plugin-opt=--aarch64-enable-compress-jump-tables=false")
		run(compiler, flags...)
		inputs = []string{out}
		out = oldout
	}

	if len(inputs) != 1 {
		fatal("multiple inputs not supported")
	}
	input := inputs[0]

	base := strings.TrimSuffix(input, filepath.Ext(input))
	targetasm := base + ".s"
	targeto := base + ".o"
	inputdir := filepath.Dir(input)

	asm := input
	if preprocess {
		flags := []string{
			"-E",
			input,
		}
		flags = append(flags, args...)
		run(compiler, flags...)
		return
	} else if lto && compile {
		if out == "" {
			out = targeto
		}
		flags := []string{
			"-c",
			"-o", out,
			input,
		}
		flags = append(flags, args...)
		run(compiler, flags...)
		return
	} else if filepath.Ext(input) == ".S" {
		asm = temp(inputdir)
		stage1 := []string{
			"-E",
			"-o", asm,
			input,
		}
		stage1 = append(stage1, args...)
		run(compiler, stage1...)
	} else if filepath.Ext(input) != ".s" {
		asm = temp(inputdir)
		stage1 := []string{
			"-S",
			"-o", asm,
			input,
		}
		stage1 = append(stage1, args...)
		run(compiler, stage1...)
	}

	asmmc := temp(inputdir)
	run("llvm-mc", "-arch=aarch64", "-filetype=asm", "-o", asmmc, asm)

	lfiasm := temp(inputdir)
	lfiflags := []string{asmmc, "-o", lfiasm}
	lfiflags = append(lfiflags, lfiargs...)
	run("lfi-gen", lfiflags...)

	if !assemble {
		if compile && out == "" {
			out = targeto
		} else if !compile && out == "" {
			out = "a.out"
		}

		stage2 := []string{
			"-o", out,
			lfiasm,
		}
		if compile {
			stage2 = append(stage2, "-c")
		}
		stage2 = append(stage2, args...)
		run(compiler, stage2...)
	} else {
		run("cp", lfiasm, targetasm)
	}

	rmtemps()
}

func main() {
	if len(os.Args) <= 1 {
		fatal("no command")
	}

	compile(os.Args[1:])
}
