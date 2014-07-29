package main

import (
	"fmt"

	"gopkgs.com/command.v1"
)

var (
	uploadCmd = &command.Cmd{
		Name:    "upload",
		Help:    "upload the program to macaco.io",
		Usage:   "<program-path>",
		Func:    uploadCommand,
		Options: &uploadOptions{},
	}
)

type uploadOptions struct {
	Name string `help:"Remote program name. If empty, defaults to the local program name"`
}

func uploadCommand(args []string, opts *uploadOptions) error {
	prog, err := loadMacacoProgram(args)
	if err != nil {
		return err
	}
	name := opts.Name
	if name == "" {
		name = prog
	}
	p := "."
	if len(args) > 0 {
		p = args[0]
	}
	if err := mc.Upload(name, p); err != nil {
		return fmt.Errorf("error uploading program %s: %s", name, err)
	}
	return nil
}
