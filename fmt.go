package macaco

import (
	"fmt"

	"github.com/robertkrimen/otto"
)

func (c *Context) loadFmt(obj *otto.Object) {
	fmtObj := c.newObject()
	fmtObj.Set("sprintf", fmt.Sprintf)
	obj.Set("fmt", fmtObj)
}
