package macaco

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"

	"github.com/robertkrimen/otto"
	_ "github.com/robertkrimen/otto/underscore"
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
}

func NewContext() (*Context, error) {
	vm := otto.New()
	ctx := &Context{
		Stdout: os.Stdout,
		Stderr: os.Stderr,
		vm:     vm,
	}
	if err := ctx.loadRuntime(); err != nil {
		return nil, err
	}
	return ctx, nil
}

func (c *Context) ctx() *Context {
	// Return the real context in the current vm, since
	// Copying the vm will cause the closures to point
	// to the old context.
	return c
}

func (c *Context) loadRuntime() error {
	obj, _ := c.vm.Object("M = macaco = {}")
	c.loadLogging(obj)
	c.loadHTTP(obj)
	c.loadJSON()
	obj.Set("load", c.Load)
	return c.Load("macaco/runtime")
}

func (c *Context) newObject() *otto.Object {
	obj, err := c.vm.Object("new Object()")
	if err != nil {
		panic(err)
	}
	return obj
}

func (c *Context) errObject(err error) *Value {
	return c.mustCallValue("new M.Error", nil, &Error{Message: err.Error()})
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
	req, err := http.NewRequest("GET", p, nil)
	if err != nil {
		return err
	}
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
	_, err = c.Run(data)
	return err
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

func (c *Context) mustCallValue(src string, this interface{}, args ...interface{}) *Value {
	val, err := c.Call(src, this, args...)
	if err != nil {
		panic(err)
	}
	return val
}
