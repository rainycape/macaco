package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"macaco"
)

var (
	accessToken = flag.String("t", "", "Access token")
	upload      = flag.String("upload", "", "Upload a program, might be either a file or a directory")
	name        = flag.String("name", "", "Used with upload to set the program name")
)

func main() {
	flag.Parse()
	mc := &macaco.Macaco{
		Token: *accessToken,
	}
	switch {
	case *upload != "":
		n := *name
		if n == "" {
			n = filepath.Base(*upload)
		}
		if err := mc.Upload(n, *upload); err != nil {
			fmt.Fprintf(os.Stderr, "error uploading program %s: %s\n", n, err)
		}
	}
}
