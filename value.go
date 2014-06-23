package macaco

import (
	"fmt"

	"github.com/robertkrimen/otto"
)

type Value struct {
	val otto.Value
	vm  *otto.Otto
}

func (v *Value) IsBoolean() bool {
	return v != nil && v.val.IsBoolean()
}

func (v *Value) IsDefined() bool {
	return v != nil && v.val.IsDefined()
}

func (v *Value) IsFunction() bool {
	return v != nil && v.val.IsFunction()
}

func (v *Value) IsNaN() bool {
	return v != nil && v.val.IsNaN()
}

func (v *Value) IsNull() bool {
	return v != nil && v.val.IsNull()
}

func (v *Value) IsNumber() bool {
	return v != nil && v.val.IsNumber()
}

func (v *Value) IsObject() bool {
	return v != nil && v.val.IsObject()
}

func (v *Value) IsPrimitive() bool {
	return v != nil && v.val.IsPrimitive()
}

func (v *Value) IsUndefined() bool {
	return v == nil || v.val.IsUndefined()
}

func (v *Value) String() string {
	if v != nil {
		return v.val.String()
	}
	return ""
}

func (v *Value) ToBoolean() (bool, error) {
	if v != nil {
		return v.val.ToBoolean()
	}
	return false, nil
}

func (v *Value) ToFloat() (float64, error) {
	if v != nil {
		return v.val.ToFloat()
	}
	return 0, nil
}

func (v *Value) ToInteger() (int64, error) {
	if v != nil {
		return v.val.ToInteger()
	}
	return 0, nil
}

func (v *Value) ToString() (string, error) {
	if v != nil {
		return v.val.ToString()
	}
	return "", nil
}

func (v *Value) Class() string {
	if v.IsObject() {
		return v.val.Object().Class()
	}
	if v != nil {
		return v.val.Class()
	}
	return ""
}

func (v *Value) Keys() []string {
	if v.IsObject() {
		return v.val.Object().Keys()
	}
	return nil
}

func (v *Value) Get(name string) (*Value, error) {
	if v.IsObject() {
		val, err := v.val.Object().Get(name)
		if err != nil {
			return nil, err
		}
		return &Value{val, v.vm}, nil
	}
	return nil, fmt.Errorf("value %v is not an object", v)
}

func (v *Value) Set(name string, value interface{}) error {
	if v.IsObject() {
		return v.val.Object().Set(name, value)
	}
	return fmt.Errorf("value %v is not an object", v)
}

func (v *Value) Call(this interface{}, args ...interface{}) (*Value, error) {
	if v.IsFunction() {
		thisVal, err := v.vm.ToValue(this)
		if err != nil {
			return nil, err
		}
		argVals := make([]interface{}, len(args))
		for ii, item := range args {
			argVal, err := v.vm.ToValue(item)
			if err != nil {
				return nil, err
			}
			argVals[ii] = argVal
		}
		val, err := v.val.Call(thisVal, argVals)
		if err != nil {
			return nil, err
		}
		return &Value{val, v.vm}, nil
	}
	return nil, fmt.Errorf("value %v is not a function", v)
}

func (v *Value) Method(name string, args ...interface{}) (*Value, error) {
	if v.IsObject() {
		argVals := make([]interface{}, len(args))
		for ii, item := range args {
			argVal, err := v.vm.ToValue(item)
			if err != nil {
				return nil, err
			}
			argVals[ii] = argVal
		}
		val, err := v.val.Object().Call(name, argVals...)
		if err != nil {
			return nil, err
		}
		return &Value{val, v.vm}, nil
	}
	return nil, fmt.Errorf("value %v is not an object", v)
}

func (v *Value) Interface() interface{} {
	if v == nil {
		return nil
	}
	iface, _ := v.val.Export()
	return iface
}
