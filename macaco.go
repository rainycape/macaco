package macaco

import (
	"archive/zip"
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
)

const (
	api = "http://macaco.io/api/v1"
)

type Options struct {
	// When bare is true, no JS runtime is loaded and the Runtime
	// field is ignored.
	Bare bool
	// The runtime to load. If empty defaults to "macaco/runtime"
	Runtime string
	// Token is the macaco API token used with
	Token   string
	Verbose bool
}

type Macaco struct {
	token   string
	verbose bool
	ctx     *Context
}

func New(opts *Options) (*Macaco, error) {
	ctx, err := NewContext()
	if err != nil {
		return nil, err
	}
	mc := &Macaco{ctx: ctx}
	runtime := "macaco/runtime"
	bare := false
	if opts != nil {
		bare = opts.Bare
		if opts.Runtime != "" {
			runtime = opts.Runtime
		}
		mc.token = opts.Token
		mc.ctx.token = opts.Token
		mc.verbose = opts.Verbose
		mc.ctx.verbose = opts.Verbose
	}
	if !bare {
		if err := mc.Load(runtime); err != nil {
			return nil, err
		}
	}
	return mc, nil
}

func (m *Macaco) Context() *Context {
	return m.ctx.Copy()
}

func (m *Macaco) Verbose() bool {
	return m.verbose
}

func (m *Macaco) loadFiles(prog string) error {
	files, err := ListProgramFiles(prog)
	if err != nil {
		return err
	}
	if len(files) == 0 {
		return fmt.Errorf("no valid files found at %s", prog)
	}
	for _, v := range files {
		data, err := ioutil.ReadFile(v)
		if err != nil {
			return err
		}
		if m.verbose {
			fmt.Println("compiling", v)
		}
		script, err := m.ctx.vm.Compile(v, data)
		if err != nil {
			return err
		}
		if _, err := m.ctx.vm.Run(script); err != nil {
			return err
		}
	}
	return nil
}

func (m *Macaco) Load(prog string) error {
	if _, err := os.Stat(prog); err == nil {
		return m.loadFiles(prog)
	}
	return m.ctx.Load(prog)
}

func (m *Macaco) Upload(name string, src interface{}) error {
	if !ProgramNameIsValid(name) {
		return fmt.Errorf("program name %q is not valid", name)
	}
	if m.token == "" {
		return errors.New("can't upload without access_token")
	}
	var data []byte
	switch x := src.(type) {
	case string:
		var buf bytes.Buffer
		w := zip.NewWriter(&buf)
		files, err := ListProgramFiles(x)
		if err != nil {
			return err
		}
		for _, v := range files {
			err := func() error {
				f, err := os.Open(v)
				if err != nil {
					return err
				}
				defer f.Close()
				fw, err := w.Create(filepath.ToSlash(v))
				if err != nil {
					return err
				}
				if _, err := io.Copy(fw, f); err != nil {
					return err
				}
				return nil
			}()
			if err != nil {
				return err
			}
		}
		if err := w.Close(); err != nil {
			return err
		}
		data = buf.Bytes()
	case *os.File:
		return m.Upload(name, x.Name())
	case io.Reader:
		var err error
		data, err = ioutil.ReadAll(x)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("invalid source type %T", src)
	}
	if err := ValidateProgramZipData(data); err != nil {
		return err
	}
	values := make(url.Values)
	values.Set("name", name)
	values.Set("access_token", m.token)
	p := api + "/upload?" + values.Encode()
	resp, err := http.Post(p, "application/zip", bytes.NewReader(data))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if err := validateHTTPResponse(resp); err != nil {
		return fmt.Errorf("error uploading program %s: %s", name, err)
	}
	return nil
}
