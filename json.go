package macaco

import (
	"encoding/json"

	"github.com/robertkrimen/otto"
)

func (c *Context) jsonParse(call otto.FunctionCall) otto.Value {
	if len(call.ArgumentList) > 0 {
		s := call.ArgumentList[0].String()
		var m interface{}
		if err := json.Unmarshal([]byte(s), &m); err != nil {
			c.Errorf("error in JSON.parse: %s %s\n", s, err)
		} else {
			val, _ := call.Otto.ToValue(m)
			return val
		}
	}
	return otto.UndefinedValue()
}

func (c *Context) jsonStringify(call otto.FunctionCall) otto.Value {
	if len(call.ArgumentList) > 0 {
		obj, _ := call.ArgumentList[0].Export()
		data, err := json.Marshal(obj)
		if err == nil {
			val, _ := call.Otto.ToValue(string(data))
			return val
		}
		c.Errorf("error in JSON.stringify: %s\n", err)
	}
	return otto.UndefinedValue()
}

func (c *Context) loadJSON() {
	obj, _ := c.vm.Object("JSON = {}")
	obj.Set("parse", c.jsonParse)
	obj.Set("stringify", c.jsonStringify)
}
