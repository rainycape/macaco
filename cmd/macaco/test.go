package main

import (
	"fmt"
	"regexp"

	"gopkgs.com/command.v1"
)

var (
	testCmd = &command.Cmd{
		Name:    "test",
		Help:    "Run tests in the given program",
		Usage:   "<program-path>",
		Func:    testCommand,
		Options: &testOptions{},
	}
)

type testOptions struct {
	Run string `help:"Only run tests with names matching the given pattern"`
}

func testCommand(args []string, opts *testOptions) error {
	if _, err := loadMacacoProgram(args); err != nil {
		return err
	}
	var re *regexp.Regexp
	var err error
	if opts.Run != "" {
		re, err = regexp.Compile(opts.Run)
		if err != nil {
			return fmt.Errorf("invalid pattern %q: %s", opts.Run, err)
		}
	}
	results, err := mc.Context().RunTests(re)
	if err != nil {
		return fmt.Errorf("error running tests: %v", err)
	}
	passed := 0
	failed := 0
	for _, v := range results {
		if v.Passed() {
			passed++
		} else {
			failed++
		}
	}
	fmt.Printf("%d tests passed, %d tests failed", passed, failed)
	if !mc.Verbose {
		fmt.Print(" - run with -v for more details")
	}
	fmt.Print("\n")
	return nil
}
