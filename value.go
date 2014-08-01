package macaco

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/rainycape/otto"
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

func (v *Value) IsArray() bool {
	return v != nil && v.val.IsArray()
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

func (v *Value) Length() int {
	if v != nil {
		return v.val.Length()
	}
	return -1
}

func (v *Value) At(idx int) (*Value, error) {
	length := v.Length()
	if idx < 0 || idx >= length {
		return nil, fmt.Errorf("index %d out of bounds", idx)
	}
	return v.Get(strconv.Itoa(idx))
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

func (v *Value) prepareArguments(this interface{}, args []interface{}) (otto.Value, []interface{}, error) {
	thisValue, err := v.vm.ToValue(this)
	if err != nil {
		return otto.Value{}, nil, err
	}
	var argValues []interface{}
	if len(args) > 0 {
		argValues := make([]interface{}, len(args))
		for ii, item := range args {
			v, err := v.vm.ToValue(item)
			if err != nil {
				return otto.Value{}, nil, err
			}
			argValues[ii] = v
		}
	}
	return thisValue, argValues, nil
}

func (v *Value) Call(this interface{}, args ...interface{}) (*Value, error) {
	if v.IsFunction() {
		thisValue, argValues, err := v.prepareArguments(this, args)
		if err != nil {
			return nil, err
		}
		val, err := v.val.Call(thisValue, argValues...)
		if err != nil {
			return nil, err
		}
		return &Value{val, v.vm}, nil
	}
	return nil, fmt.Errorf("value %v is not a function", v)
}

func (v *Value) Method(name string, args ...interface{}) (*Value, error) {
	if v.IsObject() {
		_, argValues, err := v.prepareArguments(nil, args)
		if err != nil {
			return nil, err
		}
		val, err := v.val.Object().Call(name, argValues...)
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

func (v *Value) exportInto(val reflect.Value, jsVal otto.Value) error {
	switch val.Kind() {
	case reflect.Bool:
		vv, err := jsVal.ToBoolean()
		if err != nil {
			return fmt.Errorf("error converting to bool: %s", err)
		}
		val.SetBool(vv)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		vv, err := jsVal.ToInteger()
		if err != nil {
			return fmt.Errorf("error converting to integer: %s", err)
		}
		val.SetInt(vv)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		vv, err := jsVal.ToInteger()
		if err != nil {
			return fmt.Errorf("error converting to integer: %s", err)
		}
		val.SetUint(uint64(vv))
	case reflect.Float32, reflect.Float64:
		vv, err := jsVal.ToFloat()
		if err != nil {
			return fmt.Errorf("error converting to float: %s", err)
		}
		val.SetFloat(vv)
	case reflect.String:
		if !jsVal.IsUndefined() && !jsVal.IsNull() {
			vv, err := jsVal.ToString()
			if err != nil {
				return fmt.Errorf("error converting to string: %s", err)
			}
			val.SetString(vv)
		} else {
			val.SetString("")
		}
	case reflect.Struct:
		if !jsVal.IsObject() {
			return fmt.Errorf("can't export struct %T into non-object %+v", val.Interface(), jsVal)
		}
		obj := jsVal.Object()
		typ := val.Type()
		n := typ.NumField()
		for ii := 0; ii < n; ii++ {
			structField := typ.Field(ii)
			if structField.PkgPath != "" {
				continue
			}
			field, err := obj.Get(structField.Name)
			if err != nil || field.IsUndefined() {
				field, err = obj.Get(strings.ToLower(structField.Name))
			}
			fieldVal := val.Field(ii)
			if err := v.exportInto(fieldVal, field); err != nil {
				return err
			}
		}
	case reflect.Slice:
		if !jsVal.IsArray() {
			return fmt.Errorf("can't export %+v into slice", jsVal)
		}
		obj := jsVal.Object()
		length := jsVal.Length()
		val.Set(reflect.MakeSlice(val.Type(), length, length))
		for ii := 0; ii < length; ii++ {
			elemVal := val.Index(ii)
			if elemVal.Kind() == reflect.Ptr && elemVal.IsNil() {
				elemVal.Set(reflect.New(elemVal.Type().Elem()))
			}
			elem, err := obj.Get(strconv.Itoa(ii))
			if err != nil {
				return err
			}
			if err := v.exportInto(elemVal, elem); err != nil {
				return err
			}
		}
	case reflect.Ptr:
		if val.IsNil() {
			val.Set(reflect.New(val.Type().Elem()))
		}
		return v.exportInto(val.Elem(), jsVal)
	default:
		return fmt.Errorf("can't export into %T", val.Interface())
	}
	return nil
}

func (v *Value) Export(out interface{}) error {
	if v != nil {
		val := reflect.ValueOf(out)
		if val.Kind() != reflect.Ptr || val.IsNil() {
			return fmt.Errorf("can't export to non-pointer %T", out)
		}
		val = reflect.Indirect(val)
		return v.exportInto(val, v.val)
	}
	return nil
}
