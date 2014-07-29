package macaco

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path"
	"regexp"
	"strings"
	"time"

	"github.com/rainycape/otto"
	_ "github.com/rainycape/otto/underscore"
)

type Error struct {
	Message string
	Code    int
}

type Context struct {
	Verbose bool
	Token   string
	Stdout  io.Writer
	Stderr  io.Writer
	vm      *otto.Otto
	cache   *cache
}

func NewContext() (*Context, error) {
	return newContext(newCache(), true)
}

func newContext(c *cache, remote bool) (*Context, error) {
	vm := otto.New()
	ctx := &Context{
		Stdout: os.Stdout,
		Stderr: os.Stderr,
		vm:     vm,
	}
	if c == nil {
		c = newCache()
	}
	ctx.cache = c
	if err := ctx.loadRuntime(remote); err != nil {
		return nil, err
	}
	return ctx, nil
}

func (c *Context) loadRuntime(remote bool) error {
	obj, err := c.vm.Object("M = macaco = (this.macaco || {})")
	if err != nil {
		return err
	}
	c.loadLogging(obj)
	c.loadHTTP(obj)
	c.loadHTML(obj)
	c.loadJSON()
	c.loadFmt(obj)
	c.loadImage(obj)
	obj.Set("load", c.Load)
	if !remote {
		return nil
	}
	return c.Load("macaco/runtime")
}

func (c *Context) newMacacoObject(name string) *otto.Object {
	obj, err := c.vm.Object(fmt.Sprintf("macaco.%s = macaco.%s || new Object()", name, name))
	if err != nil {
		panic(err)
	}
	return obj
}

func (c *Context) errObject(err error) *Value {
	return c.mustCallValue("new M.Error", nil, &Error{Message: err.Error()})
}

func (c *Context) Copy() *Context {
	cpy := *c
	cpy.vm = cpy.vm.Copy()
	// Reload the runtime so the closures and method values
	// point to the right *Context. Don't reload the js runtime,
	// since that part does not have closures.
	// This error should never happen, but just in case...
	if err := cpy.loadRuntime(false); err != nil {
		panic(err)
	}
	return &cpy
}

func (c *Context) Run(src interface{}) (*Value, error) {
	v, err := c.vm.Run(src)
	if err != nil {
		return nil, err
	}
	return &Value{v, c.vm}, nil
}

func (c *Context) Call(src string, this interface{}, args ...interface{}) (*Value, error) {
	thisVal, err := c.vm.ToValue(this)
	if err != nil {
		return nil, err
	}
	argValues := make([]interface{}, len(args))
	for ii, v := range args {
		argVal, err := c.vm.ToValue(v)
		if err != nil {
			return nil, err
		}
		argValues[ii] = argVal
	}
	v, err := c.vm.Call(src, thisVal, argValues...)
	if err != nil {
		return nil, err
	}
	return &Value{v, c.vm}, nil
}

func (c *Context) Load(prog string) error {
	c.Debugf("loading %s\n", prog)
	var p string
	if looksLikeURL(prog) {
		p = prog
	} else {
		values := make(url.Values)
		values.Set("program", prog)
		if c.Token != "" {
			values.Set("access_token", c.Token)
		}
		p = api + "/load?" + values.Encode()
	}
	entry, script := c.cache.getCachedScript(p)
	if script != nil {
		if _, err := c.vm.Run(script); err == nil {
			return nil
		}
	}
	if entry != nil && len(entry.Data) > 0 {
		if c.loadData(p, entry.Data, entry.Headers, entry) == nil {
			return nil
		}
	}
	req, err := http.NewRequest("GET", p, nil)
	if err != nil {
		return err
	}
	c.Debugf("GET %s\n", p)
	resp, err := c.httpClient().Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if err := validateHTTPResponse(resp); err != nil {
		return fmt.Errorf("error loading program %s: %s", prog, err)
	}
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	return c.loadData(p, data, resp.Header, nil)
}

func (c *Context) loadData(url string, data []byte, headers http.Header, entry *diskEntry) error {
	script, err := c.vm.Compile(path.Base(url), data)
	if err != nil {
		return err
	}
	if _, err := c.vm.Run(script); err != nil {
		return err
	}
	var expires time.Time
	if entry != nil {
		expires = entry.Expires
	}
	if err := c.cache.cacheScript(url, data, headers, script, expires); err != nil {
		c.Debugf("error caching script %s: %s\n", url, err)
	}
	return nil
}

func (c *Context) Globals() []string {
	val, err := c.vm.Call("(function() { return Object.keys(this); })", nil)
	if err != nil {
		panic(err)
	}
	v, _ := val.Export()
	names := v.([]interface{})
	globals := make([]string, len(names))
	for ii, n := range names {
		globals[ii] = n.(string)
	}
	return globals
}

func (c *Context) Get(name string) (*Value, error) {
	v, err := c.vm.Get(name)
	if err != nil {
		return nil, err
	}
	return &Value{v, c.vm}, nil
}

func (c *Context) RunTests(re *regexp.Regexp) ([]*Test, error) {
	const testPrefix = "__test"
	defer func(stdout, stderr io.Writer) {
		c.Stdout = stdout
		c.Stderr = stderr
	}(c.Stdout, c.Stderr)
	stdout, stderr := c.Stdout, c.Stderr
	var tests []*Test
	var testStdout bytes.Buffer
	for _, name := range c.Globals() {
		if !strings.HasPrefix(name, testPrefix) {
			continue
		}
		tname := strings.TrimPrefix(name, testPrefix)
		if re != nil && !re.MatchString(tname) {
			continue
		}
		val, err := c.Get(name)
		if err != nil {
			return nil, err
		}
		if val.IsFunction() {
			testStdout.Reset()
			t := new(Test)
			t.Name = tname
			if c.Verbose {
				fmt.Fprintln(stdout, "TEST:", t.Name)
			}
			w := t.stderrWriter()
			c.Stdout = &testStdout
			c.Stderr = w
			if c.Verbose {
				c.Stdout = io.MultiWriter(c.Stdout, os.Stdout)
				c.Stderr = io.MultiWriter(c.Stderr, os.Stderr)
			}
			t.Started = time.Now()
			_, err := val.Call(nil)
			if err != nil {
				oe, ok := err.(*otto.Error)
				if !ok {
					return nil, err
				}
				t.Errors = append(t.Errors, &TestError{Message: oe.String(), Timestamp: time.Now()})
			}
			t.Finished = time.Now()
			t.Stdout = testStdout.String()
			t.Stderr = w.String()
			tests = append(tests, t)
			if t.Passed() {
				if c.Verbose {
					fmt.Fprintf(stdout, "PASS: %s (%s)\n", t.Name, t.Elapsed())
				}
			} else {
				fmt.Fprintf(stderr, "FAIL: %s (%s)\n", t.Name, t.Elapsed())
				for _, v := range t.Errors {
					fmt.Fprintf(stderr, "\terror: %s (at %s)\n", v.Message, v.Timestamp.Sub(t.Started))
				}
			}
		}
	}
	return tests, nil
}

func (c *Context) mustCallValue(src string, this interface{}, args ...interface{}) *Value {
	val, err := c.Call(src, this, args...)
	if err != nil {
		panic(err)
	}
	return val
}
