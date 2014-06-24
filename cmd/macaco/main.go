package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"macaco"
)

var (
	accessToken = flag.String("t", "", "Access token for macaco.io")
	upload      = flag.Bool("upload", false, "Upload the program to macaco.io")
	name        = flag.String("name", "", "Used with upload to set the program name")
	run         = flag.Bool("run", false, "Run the program. First argument is the function name, then its arguments")
	test        = flag.Bool("test", false, "Run tests in the program")
	verbose     = flag.Bool("v", false, "Verbose output")
)

func main() {
	flag.Parse()
	args := flag.Args()
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "missing program name")
		flag.Usage()
		os.Exit(1)
	}
	prog := args[0]
	mc, err := macaco.New()
	if err != nil {
		panic(err)
	}
	mc.Token = *accessToken
	mc.Verbose = *verbose
	if err := mc.Load(prog); err != nil {
		fmt.Fprintf(os.Stderr, "error loading program %s: %s\n", prog, err)
		os.Exit(1)
	}
	switch {
	case *run:
		if len(args) > 1 {
			call := args[1]
			var funcArgs []interface{}
			var val *macaco.Value
			var err error
			file := false
			if filepath.Ext(call) == ".js" {
				file = true
				f, err := os.Open(call)
				if err != nil {
					panic(err)
				}
				defer f.Close()
				val, err = mc.Context().Run(f)
			} else {
				for ii := 2; ii < len(args); ii++ {
					v := args[ii]
					if n, err := strconv.ParseFloat(v, 64); err == nil {
						funcArgs = append(funcArgs, n)
					} else {
						funcArgs = append(funcArgs, v)
					}
				}
				val, err = mc.Context().Call(call, nil, funcArgs...)
			}
			if err != nil {
				if file {
					fmt.Fprintf(os.Stderr, "error running %s: %s\n", call, err)
				} else {
					fmt.Fprintf(os.Stderr, "error calling %s with arguments %v: %s\n", call, funcArgs, err)
				}
			}
			fmt.Printf("result %v\n", val.Interface())
		}
	case *test:
		panic("unimplemented")
	case *upload:
		n := *name
		if n == "" {
			n = filepath.Base(prog)
		}
		if err := mc.Upload(n, prog); err != nil {
			fmt.Fprintf(os.Stderr, "error uploading program %s: %s\n", n, err)
		}
	default:
		flag.Usage()
	}
}
