package jsonconv

import (
	"bytes"
	"reflect"
	"strconv"
	"strings"

	"github.com/buger/jsonparser"
)

// data definitions same as jsonparser
type ValueType int

const (
	NotExist ValueType = ValueType(jsonparser.NotExist)
	String   ValueType = ValueType(jsonparser.String)
	Number   ValueType = ValueType(jsonparser.Number)
	Object   ValueType = ValueType(jsonparser.Object)
	Array    ValueType = ValueType(jsonparser.Array)
	Boolean  ValueType = ValueType(jsonparser.Boolean)
	Null     ValueType = ValueType(jsonparser.Null)
	Unknown  ValueType = ValueType(jsonparser.Unknown)
)

type JsonValue struct {
	// type
	valueType ValueType
	// values
	stringValue string
	intValue    int64
	floatValue  float64
	boolValue   bool
	uintValue   uint64
	// object children
	objChildren map[string]*JsonValue
	// array children
	arrChildren []*JsonValue
	// number type judgement
	mustSigned   bool
	mustUnsigned bool
	mustFloat    bool
}

// ====================
// internal functions

var escapeMap = map[rune]rune{
	'"': '"',
	'/': '/',
	'b': '\b',
	'f': '\f',
	't': '\t',
	'n': '\n',
}

func stringFromEscapedBytes(input []byte) (string, error) {
	b := bytes.Buffer{}
	s := string(input)
	escaping := false
	skip := 0

	for i, chr := range s {
		if skip > 0 {
			skip--
		} else if escaping {
			escaping = false
			switch chr {
			case '"', '/', 'b', 'f', 't', 'n', 'r':
				write_chr, exist := escapeMap[chr]
				if exist {
					b.WriteRune(write_chr)
				}
			case 'u':
				// parse unicode
				if i+5 > len(s) {
					return "", JsonFormatError
				}
				sub_str := s[i+1 : i+5]
				unicode, err := strconv.ParseInt(sub_str, 16, 32)
				if err != nil {
					// err
					// log.Error("err: %s", err.Error())
					return "", err
				} else {
					skip = 4
					b.WriteRune(rune(unicode))
				}
			default:
				// the previous \ is just a simple character
				escaping = false
				b.WriteRune('\\')
				b.WriteRune(chr)
			}
		} else {
			switch chr {
			case '\\':
				// get ready to escape
				escaping = true
			default:
				b.WriteRune(chr)
			}
		}
	}
	if escaping {
		escaping = false
		b.WriteRune('\\')
	}
	return b.String(), nil
}

// ====================
// New() functions

func NewFromBytes(b []byte) (*JsonValue, error) {
	return NewFromString(string(b))
}

func NewFromString(s string) (*JsonValue, error) {
	var obj *JsonValue
	var err error

	// get first character
	for index, chr := range s {
		switch chr {
		case ' ', '\r', '\n', '\t':
			// continue
		case '{':
			obj = NewObject()
			err = obj.parseObject([]byte(s[index:]))
			if err != nil {
				return nil, err
			} else {
				return obj, nil
			}
		case '[':
			obj = NewArray()
			err = obj.parseArray([]byte(s[index:]))
			if err != nil {
				return nil, err
			} else {
				return obj, nil
			}
		case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9', '-':
			obj = new(JsonValue)
			obj.valueType = Number
			obj.mustSigned = (chr == '-')
			obj.floatValue, err = strconv.ParseFloat(s[index:], 64)
			if err != nil {
				return nil, err
			}
			obj.intValue, err = strconv.ParseInt(s[index:], 10, 64)
			if err != nil {
				// return nil, err
				// if parseFloat OK but parseInt failed, this may be a float
				// log.Debug("error: %s", err.Error())
				if strings.HasSuffix(err.Error(), "value out of range") {
					// this must be a unsigned integer
					obj.mustUnsigned = true
				} else {
					// log.Debug("MARK")
					obj.mustFloat = true
					obj.intValue = int64(obj.floatValue)
					obj.uintValue = uint64(obj.intValue)
				}
			}
			// log.Debug("mustUnsigned = %t, mustFloat = %t", obj.mustUnsigned, obj.mustFloat)

			if obj.mustUnsigned && false == obj.mustFloat {
				obj.uintValue, err = strconv.ParseUint(s[index:], 10, 64)
				if err != nil {
					return nil, err
				}
				if 0 != obj.uintValue&0x1000000000000000 {
					obj.mustUnsigned = true
				}
			} else {
				obj.uintValue = uint64(obj.intValue)
			}
			// log.Debug("orig str: %s", s[index:])
			// log.Debug("parsed i: %d", obj.intValue)
			// log.Debug("parsed u: %s", strconv.FormatUint(obj.uintValue, 10))
			// log.Debug("parsed x: 0x%x", obj.uintValue)
			return obj, nil
		case '"':
			obj = new(JsonValue)
			obj.valueType = String
			// search for next quote
			next := strings.IndexRune(s[index+1:], '"')
			if next < 0 {
				return nil, JsonFormatError
			} else {
				obj.stringValue = s[index+1 : index+1+next]
				return obj, nil
			}
		case 't':
			if s[index:] == "true" {
				obj = new(JsonValue)
				obj.valueType = Boolean
				obj.boolValue = true
				return obj, nil
			} else {
				return nil, JsonFormatError
			}
		case 'f':
			if s[index:] == "false" {
				obj = new(JsonValue)
				obj.valueType = Boolean
				obj.boolValue = false
				return obj, nil
			} else {
				return nil, JsonFormatError
			}
		case 'n':
			if s[index:] == "null" {
				obj = new(JsonValue)
				obj.valueType = Null
				return obj, nil
			} else {
				return nil, JsonFormatError
			}
		default:
			// log.Debug("Skip: %c", chr)
			// skip
		}
	}
	return nil, JsonFormatError
}

// ====================
// parse functions

func (obj *JsonValue) parseObject(data []byte) error {
	add_child := func(obj *JsonValue, key []byte, child *JsonValue) error {
		key_str, key_err := stringFromEscapedBytes(key)
		if key_err == nil {
			obj.objChildren[key_str] = child
		}
		return key_err
	}

	err := jsonparser.ObjectEach(data, func(key []byte, value []byte, dataType jsonparser.ValueType, _ int) error {
		// log.Debug("----------")
		// log.Debug("key: %s", string(key))
		// log.Debug("value: %s", string(value))
		switch dataType {
		case jsonparser.String:
			// log.Debug("string")
			str_value, err := stringFromEscapedBytes(value)
			if err != nil {
				return err
			}
			child := NewString(str_value)
			err = add_child(obj, key, child)
			if err != nil {
				return err
			}
		case jsonparser.Number:
			// log.Debug("number")
			var err error
			child := new(JsonValue)
			str_value := string(value)
			child.valueType = Number
			child.intValue, err = strconv.ParseInt(str_value, 10, 64)
			if err != nil {
				return nil
			}
			child.floatValue, err = strconv.ParseFloat(str_value, 64)
			if err != nil {
				return nil
			}
			err = add_child(obj, key, child)
			if err != nil {
				return err
			}
		case jsonparser.Object:
			// log.Debug("object")
			child := NewObject()
			err := child.parseObject(value)
			if err != nil {
				// log.Error("Failed to parse object: %s", err.Error())
				return err
			}
			// log.Debug("%s ---- object size: %d", string(key), child.Length())
			err = add_child(obj, key, child)
			if err != nil {
				return err
			}
		case jsonparser.Array:
			// log.Debug("array")
			child := NewArray()
			err := child.parseArray(value)
			if err != nil {
				// log.Error("Failed to parse array: %s", err.Error())
				return err
			}
			// log.Debug("%s ---- array size: %d", string(key), child.Length())
			err = add_child(obj, key, child)
			if err != nil {
				return err
			}
		case jsonparser.Boolean:
			// log.Debug("bool")
			b, err := strconv.ParseBool(string(value))
			if err != nil {
				return err
			}
			child := NewBool(b)
			err = add_child(obj, key, child)
			if err != nil {
				return err
			}
		case jsonparser.Null:
			// log.Debug("null")
			child := NewNull()
			err := add_child(obj, key, child)
			if err != nil {
				return err
			}
		default:
			// log.Debug("Invalid type: %d", int(dataType))
		}
		return nil
	})
	return err
}

func (obj *JsonValue) parseArray(data []byte) (err error) {
	// check ending quote
	{
		l := len(data)
		if l < 2 {
			return JsonFormatError
		}
		if data[l-1] != byte(']') {
			return JsonFormatError
		}
	}
	jsonparser.ArrayEach(data, func(value []byte, dataType jsonparser.ValueType, _ int, _ error) {
		switch dataType {
		case jsonparser.String:
			str_value, loc_err := stringFromEscapedBytes(value)
			if loc_err != nil {
				err = loc_err
				return
			}
			child := NewString(str_value)
			obj.arrChildren = append(obj.arrChildren, child)
		case jsonparser.Number:
			child := new(JsonValue)
			str_value := string(value)
			child.valueType = Number
			child.intValue, _ = strconv.ParseInt(str_value, 10, 64)
			child.floatValue, _ = strconv.ParseFloat(str_value, 64)
			obj.arrChildren = append(obj.arrChildren, child)
		case jsonparser.Object:
			child := NewObject()
			err = child.parseObject(value)
			if err != nil {
				return
			}
			obj.arrChildren = append(obj.arrChildren, child)
		case jsonparser.Array:
			child := NewArray()
			err = child.parseArray(value)
			if err != nil {
				return
			}
			obj.arrChildren = append(obj.arrChildren, child)
		case jsonparser.Boolean:
			b, loc_err := strconv.ParseBool(string(value))
			if loc_err != nil {
				err = loc_err
				return
			}
			child := NewBool(b)
			obj.arrChildren = append(obj.arrChildren, child)
		case jsonparser.Null:
			child := NewNull()
			obj.arrChildren = append(obj.arrChildren, child)
		}
		return
	})
	return
}

// ====================
// content access

// simple values
func (obj *JsonValue) String() string {
	if obj.valueType == String {
		return obj.stringValue
	}
	return ""
}

func (obj *JsonValue) Int64() int64 {
	if obj.valueType == Number {
		return obj.intValue
	}
	return 0
}

func (obj *JsonValue) Uint64() uint64 {
	if obj.valueType == Number {
		return obj.uintValue
	}
	return 0
}

func (obj *JsonValue) Int32() int32 {
	if obj.valueType == Number {
		return int32(obj.intValue)
	}
	return 0
}

func (obj *JsonValue) Uint32() uint32 {
	if obj.valueType == Number {
		return uint32(obj.uintValue)
	}
	return 0
}

func (obj *JsonValue) Int() int {
	if obj.valueType == Number {
		return int(obj.intValue)
	}
	return 0
}

func (obj *JsonValue) Uint() uint {
	if obj.valueType == Number {
		return uint(obj.uintValue)
	}
	return 0
}

func (obj *JsonValue) Float() float64 {
	if obj.valueType == Number {
		return obj.floatValue
	}
	return 0.0
}

func (obj *JsonValue) Bool() bool {
	if obj.valueType == Boolean {
		return obj.boolValue
	}
	return false
}

func (obj *JsonValue) Boolean() bool {
	if obj.valueType == Boolean {
		return obj.boolValue
	}
	return false
}

func (obj *JsonValue) Length() int {
	if obj.valueType == Array {
		return len(obj.arrChildren)
	} else if obj.valueType == Object {
		return len(obj.objChildren)
	} else {
		return 0
	}
}

func (obj *JsonValue) Len() int {
	if obj.valueType == Array {
		return len(obj.arrChildren)
	} else if obj.valueType == Object {
		return len(obj.objChildren)
	} else {
		return 0
	}
}

// types
func (obj *JsonValue) Type() ValueType {
	return ValueType(obj.valueType)
}

func (obj *JsonValue) TypeString() string {
	switch obj.valueType {
	case String:
		return "string"
	case Number:
		return "number"
	case Boolean:
		return "boolean"
	case Null:
		return "null"
	case Object:
		return "object"
	case Array:
		return "array"
	default:
		return "unknown"
	}
}

func (obj *JsonValue) IsNull() bool {
	return obj.valueType == Null
}

func (obj *JsonValue) IsString() bool {
	return obj.valueType == String
}

func (obj *JsonValue) IsNumber() bool {
	return obj.valueType == Number
}

func (obj *JsonValue) IsObject() bool {
	return obj.valueType == Object
}

func (obj *JsonValue) IsArray() bool {
	return obj.valueType == Array
}

func (obj *JsonValue) IsBoollean() bool {
	return obj.valueType == Boolean
}

func (obj *JsonValue) IsBool() bool {
	return obj.valueType == Boolean
}

// children access
func (obj *JsonValue) GetByKey(keys ...string) (*JsonValue, error) {
	if obj.valueType != Object {
		return nil, NotAnObjectError
	}
	if 0 == len(keys) {
		return obj, nil
	}
	if 1 == len(keys) {
		child, exist := obj.objChildren[keys[0]]
		if false == exist {
			return nil, ObjectNotFoundError
		} else {
			return child, nil
		}
	}
	// else
	first_child, exist := obj.objChildren[keys[0]]
	if false == exist {
		return nil, ObjectNotFoundError
	}
	child, err := first_child.GetByKey(keys[1:]...)
	if err != nil {
		return nil, ObjectNotFoundError
	} else {
		return child, nil
	}
}

func (obj *JsonValue) GetAtIndex(index int) (*JsonValue, error) {
	if obj.valueType == Array {
		if index >= 0 && index < obj.Length() {
			return obj.arrChildren[index], nil
		} else {
			// log.Error("request index %d, but length is %d", index, obj.Length())
			return nil, IndexOutOfBoundsError
		}
	} else {
		return nil, NotAnArrayError
	}
}

func (obj *JsonValue) Get(first interface{}, keys ...interface{}) (*JsonValue, error) {
	switch first.(type) {
	case string:
		child, err := obj.GetByKey(first.(string))
		if err != nil {
			return nil, err
		} else if len(keys) == 1 {
			return child.Get(keys[0])
		} else if len(keys) > 1 {
			return child.Get(keys[0], keys[1:]...)
		} else {
			return child, nil
		}
	case int8, uint8, int16, uint16, int32, uint32, int64, uint64, int, uint:
		value := reflect.ValueOf(first)
		index := int(value.Int())
		child, err := obj.GetAtIndex(index)
		if err != nil {
			return nil, err
		} else if len(keys) == 1 {
			return child.Get(keys[0])
		} else if len(keys) > 1 {
			return child.Get(keys[0], keys[1:]...)
		} else {
			return child, nil
		}
	default:
		return nil, ParaError
	}
}

func (obj *JsonValue) GetString(first interface{}, keys ...interface{}) (string, error) {
	child, err := obj.Get(first, keys...)
	if err != nil {
		return "", err
	}
	if false == child.IsString() {
		return "", NotAStringError
	}
	return child.String(), nil
}

// ====================
// Marshal
func (obj *JsonValue) MarshalToString(opts ...Option) (string, error) {
	buff := bytes.Buffer{}
	err := obj.marshalToBuffer(&buff, opts...)
	if err != nil {
		return "", err
	}
	return buff.String(), nil
}

func (obj *JsonValue) Marshal(opts ...Option) ([]byte, error) {
	buff := bytes.Buffer{}
	err := obj.marshalToBuffer(&buff, opts...)
	if err != nil {
		return nil, err
	}
	return buff.Bytes(), nil
}

func (obj *JsonValue) marshalToBuffer(buff *bytes.Buffer, opts ...Option) error {
	var opt *Option
	if len(opts) > 0 {
		opt = &(opts[0])
	} else {
		opt = &dftOption
	}

	switch obj.valueType {
	case String:
		s := `"` + escapeJsonString(obj.String(), true) + `"`
		buff.WriteString(s)
		return nil
	case Number:
		i := obj.intValue
		f := obj.floatValue
		if obj.mustFloat {
			s := convertFloatToString(f, opt.FloatDigits)
			buff.WriteString(s)
			return nil
		} else if obj.mustUnsigned {
			s := strconv.FormatUint(obj.uintValue, 10)
			buff.WriteString(s)
			return nil
		} else if float64(i) == f {
			s := strconv.FormatInt(i, 10)
			buff.WriteString(s)
			return nil
		} else {
			s := convertFloatToString(f, opt.FloatDigits)
			buff.WriteString(s)
			return nil
		}
	case Null:
		buff.WriteString("null")
		return nil
	case Boolean:
		if obj.Bool() {
			buff.WriteString("true")
			return nil
		} else {
			buff.WriteString("false")
			return nil
		}
	case Object:
		is_first := true
		buff.WriteRune('{')
		marshal_child_func := func(key string, child *JsonValue) {
			if child.IsNull() && false == opt.ShowNull {
				// do nothing
			} else {
				if is_first {
					is_first = false
				} else {
					buff.WriteRune(',')
				}
				buff.WriteRune('"')
				buff.WriteString(escapeJsonString(key, true))
				buff.WriteRune('"')
				buff.WriteRune(':')

				child.marshalToBuffer(buff, *opt)
			}
		}
		if Random != opt.SortMode {
			sorted := sortObjects(obj, opt.SortMode)
			for _, pair := range sorted {
				marshal_child_func(pair.K, pair.V)
			}
		} else {
			for key, child := range obj.objChildren {
				marshal_child_func(key, child)
			}
		}
		buff.WriteRune('}')
		return nil
	case Array:
		is_first := true
		buff.WriteRune('[')
		for _, child := range obj.arrChildren {
			if child.IsNull() && false == opt.ShowNull {
				// do nothing
			} else {
				if is_first {
					is_first = false
				} else {
					buff.WriteRune(',')
				}
				child.marshalToBuffer(buff, *opt)
			}
		}
		buff.WriteRune(']')
		return nil
	default:
		// do nothing
		return JsonTypeError
	}
}

// ====================
// object modification
func (obj *JsonValue) Delete(first interface{}, keys ...interface{}) error {
	var parent *JsonValue
	var last_key *interface{}
	var err error

	// get parent
	switch len(keys) {
	case 0:
		last_key = &first
		parent = obj
	case 1:
		last_key = &keys[0]
		parent, err = obj.Get(first)
		if err != nil {
			return ObjectNotFoundError
		}
	default:
		last_index := len(keys) - 1
		last_key = &keys[last_index]
		parent, err = obj.Get(first, keys[1:]...)
		if err != nil {
			return ObjectNotFoundError
		}
	}

	// get child
	switch (*last_key).(type) {
	case string:
		if false == parent.IsObject() {
			return ObjectNotFoundError
		}
		// delect object
		last_key_str := (*last_key).(string)
		// log.Debug("Delete key: %s", last_key_str)
		_, err = parent.Get(last_key_str)
		if err != nil {
			// log.Debug("key %s not found", last_key_str)
			return err
		}
		delete(parent.objChildren, last_key_str)
		return nil

	case uint8, int8, uint16, int16, uint32, int32, uint64, int64, int, uint:
		value := reflect.ValueOf(first)
		index := int(value.Int())
		arr_len := len(parent.arrChildren)
		if index >= 0 && index < arr_len {
			tail := parent.arrChildren[index+1:]
			parent.arrChildren = parent.arrChildren[0:index]
			parent.arrChildren = append(parent.arrChildren, tail...)
			return nil
		} else {
			return IndexOutOfBoundsError
		}

	default:
		return DataTypeError
	}
}

func (this *JsonValue) Append(newOne *JsonValue, keys ...interface{}) (*JsonValue, error) {
	if nil == newOne {
		return nil, ParaError
	}
	if 0 == len(keys) {
		if this.valueType == Array {
			this.arrChildren = append(this.arrChildren, newOne)
			return newOne, nil
		} else {
			return nil, NotAnArrayError
		}
	} else {
		child, err := this.Get(keys[0], keys[1:]...)
		if err != nil {
			return nil, err
		} else {
			return child.Append(newOne)
		}
	}
}

func (this *JsonValue) Insert(newOne *JsonValue, index interface{}, keys ...interface{}) (*JsonValue, error) {
	if nil == newOne {
		return nil, ParaError
	}
	keys_count := len(keys)
	if 0 == keys_count {
		if this.valueType != Array {
			return nil, NotAnArrayError
		}
		switch index.(type) {
		case uint8, int8, uint16, int16, uint32, int32, uint64, int64, int, uint:
			value := reflect.ValueOf(index)
			index := int(value.Int())
			arr_len := len(this.arrChildren)
			if index >= 0 && index < arr_len {
				// ref: [SliceTricks](https://github.com/golang/go/wiki/SliceTricks)
				a := this.arrChildren
				this.arrChildren = append(a[:index], append([]*JsonValue{newOne}, a[index:]...)...)
				return newOne, nil
			} else {
				return nil, IndexOutOfBoundsError
			}
		default:
			return nil, DataTypeError

		}
	} else {
		var err error
		var child *JsonValue
		if 1 == keys_count {
			child, err = this.Get(index, keys[0])
		} else {
			child, err = this.Get(index, keys[:keys_count-2]...)
		}
		if err != nil {
			return nil, err
		}
		return child.Insert(newOne, keys[keys_count-1])
	}
}

func (this *JsonValue) Swap(i, j int) error {
	if false == this.IsArray() {
		return NotAnArrayError
	}

	l := this.Length()
	if i >= l || j >= l {
		return IndexOutOfBoundsError
	}

	this.arrChildren[i], this.arrChildren[j] = this.arrChildren[j], this.arrChildren[i]
	return nil
}

func (this *JsonValue) Set(newOne *JsonValue, first interface{}, keys ...interface{}) (*JsonValue, error) {
	// log.Debug("Set \"%v\" (%v)", first, keys)
	keys_count := len(keys)
	switch keys_count {
	case 0:
		switch first.(type) {
		case string:
			if this.IsObject() {
				key := first.(string)
				this.objChildren[key] = newOne
				return newOne, nil
			} else {
				// log.Error("Not an object")
				return nil, NotAnObjectError
			}
		case uint8, int8, uint16, int16, uint32, int32, uint64, int64, int, uint:
			if this.IsArray() {
				value := reflect.ValueOf(first)
				index := int(value.Int())
				this.arrChildren[index] = newOne
				return newOne, nil
			} else {
				// log.Error("Not an array")
				return nil, NotAnArrayError
			}
		default:
			// log.Error("leaf not a string")
			return nil, DataTypeError
		}
	case 1:
		child, err := this.Get(first)
		if err != nil {
			// log.Error("Failed to get: %s", err.Error())
			return nil, err
		}
		return child.Set(newOne, keys[0])
	default:
		child, err := this.Get(first)
		if err != nil {
			// log.Error("Failed to get: %s", err.Error())
			return nil, err
		}
		return child.Set(newOne, keys[0], keys[1:]...)
	}
}

// ====================
// foreach
func (this *JsonValue) ArrayForeach(callback func(index int, value *JsonValue) error) error {
	if false == this.IsArray() {
		// log.Error("object is not an array")
		return NotAnArrayError
	}
	// log.Debug("array size: %d", len(this.arrChildren))
	for i, val := range this.arrChildren {
		err := callback(i, val)
		if err != nil {
			return err
		}
	}
	return nil
}

func (this *JsonValue) ObjectForeach(callback func(key string, value *JsonValue) error) error {
	if false == this.IsObject() {
		return NotAnObjectError
	}
	for k, v := range this.objChildren {
		err := callback(k, v)
		if err != nil {
			return err
		}
	}
	return nil
}
