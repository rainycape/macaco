package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"gopkgs.com/command.v1"

	"macaco.io/macaco"
)

var (
	runCmd = &command.Cmd{
		Name:  "run",
		Help:  "run the specified program name",
		Usage: "<program-path> [function-name] [args...]",
		Func:  runCommand,
	}
)

func runCommand(args []string) error {
	if _, err := loadMacacoProgram(args); err != nil {
		return err
	}
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
				return fmt.Errorf("error running %s: %s", call, err)
			}
			return fmt.Errorf("error calling %s with arguments %v: %s", call, funcArgs, err)
		}
		fmt.Printf("result %v\n", val.Interface())
	}
	return nil
}
