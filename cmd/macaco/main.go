package main

import (
	"fmt"
	"path/filepath"

	"gopkgs.com/command.v1"

	"macaco"
)

var (
	mc *macaco.Macaco
)

type globalOptions struct {
	Token   string `name:"t" help:"Access token for the macaco.io service"`
	Bare    bool   `help:"Use a bare macaco runtime"`
	Verbose bool   `name:"v" help:"Verbose output"`
}

func newMacaco(bare bool) *macaco.Macaco {
	var m *macaco.Macaco
	var err error
	if bare {
		m, err = macaco.NewBare()
	} else {
		m, err = macaco.New()
	}
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
			mc = newMacaco(opts.Bare)
			mc.Token = opts.Token
			mc.Verbose = opts.Verbose
		},
	}
	command.Exit(command.RunOpts(nil, opts, commands))
}
