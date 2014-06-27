package macaco

import (
	"fmt"

	"github.com/robertkrimen/otto"
)

func (c *Context) loadFmt(obj *otto.Object) {
	fmtObj := c.newMacacoObject("fmt")
	fmtObj.Set("sprintf", fmt.Sprintf)
}
