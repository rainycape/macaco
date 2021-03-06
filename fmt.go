package macaco

import (
	"fmt"

	"github.com/rainycape/otto"
)

func (c *Context) loadFmt(obj *otto.Object) {
	fmtObj := c.newMacacoObject("fmt")
	fmtObj.Set("sprint", fmt.Sprint)
	fmtObj.Set("sprintf", fmt.Sprintf)
	fmtObj.Set("sprintln", fmt.Sprintln)
}
