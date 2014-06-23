package macaco

import (
	"fmt"

	"github.com/robertkrimen/otto"
)

func (c *Context) Debug(args ...interface{}) {
	if c.EnableDebug {
		fmt.Fprintln(c.Stdout, args...)
	}
}

func (c *Context) Debugf(format string, args ...interface{}) {
	if c.EnableDebug {
		fmt.Fprintf(c.Stdout, format, args...)
	}
}

func (c *Context) Log(args ...interface{}) {
	fmt.Fprintln(c.Stdout, args...)
}

func (c *Context) Logf(format string, args ...interface{}) {
	fmt.Fprintf(c.Stdout, format, args...)
}

func (c *Context) Error(args ...interface{}) {
	fmt.Fprintln(c.Stderr, args...)
}

func (c *Context) Errorf(format string, args ...interface{}) {
	fmt.Fprintf(c.Stderr, format, args...)
}

func (c *Context) loadLogging(obj *otto.Object) {
	obj.Set("debug", c.Debug)
	obj.Set("debugf", c.Debugf)
	obj.Set("log", c.Log)
	obj.Set("logf", c.Logf)
	obj.Set("error", c.Error)
	obj.Set("errorf", c.Errorf)

	console := c.newObject()
	console.Set("debug", c.Debug)
	console.Set("debugf", c.Debugf)
	console.Set("log", c.Log)
	console.Set("logf", c.Logf)
	console.Set("error", c.Error)
	console.Set("errorf", c.Errorf)
	c.vm.Set("console", console)

}
