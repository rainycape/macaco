package main

import (
	"fmt"
	"path/filepath"

	"gopkgs.com/command.v1"

	"macaco.io/macaco"
)

var (
	mc *macaco.Macaco
)

type globalOptions struct {
	Token   string `name:"t" help:"Access token for the macaco.io service"`
	Bare    bool   `help:"Use a bare macaco runtime"`
	Runtime string `name:"rt" help:"Macaco runtime to use"`
	Verbose bool   `name:"v" help:"Verbose output"`
}

func newMacaco(opts *macaco.Options) *macaco.Macaco {
	m, err := macaco.New(opts)
	if err != nil {
		panic(err)
	}
	return m
}

func loadMacacoProgram(args []string) (string, error) {
	var prog, name string
	if len(args) > 0 {
		prog = args[0]
		name = args[0]
	} else {
		prog = "."
	}
	if prog == "." {
		if abs, _ := filepath.Abs(prog); abs != "" {
			name = filepath.Base(abs)
		}
	}
	if err := mc.Load(prog); err != nil {
		return "", fmt.Errorf("error loading program %s: %s", prog, err)
	}
	return name, nil
}

func main() {
	commands := []*command.Cmd{
		runCmd,
		uploadCmd,
		testCmd,
	}
	opts := &command.Options{
		Options: &globalOptions{},
		Func: func(opts *globalOptions) {
			mopts := &macaco.Options{
				Bare:    opts.Bare,
				Runtime: opts.Runtime,
				Token:   opts.Token,
				Verbose: opts.Verbose,
			}
			mc = newMacaco(mopts)
		},
	}
	command.Exit(command.RunOpts(nil, opts, commands))
}
