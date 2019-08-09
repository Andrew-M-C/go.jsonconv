package jsonconv

import "testing"

const raw = `{
	"a-string": "这是一个string",
	"an-int": 12345678,
	"a-float": 12345.12345678,
	"a-true": true,
	"a-false": false,
	"a-null": null,
	"an-object": {
		"sub-string": "string in an object",
		"sub-object": {
			"another-sub-string": "\"string\" in an object in an object",
			"another-sub-array": [1, "string in sub
array", true, null],
			"complex":"\u4e2d\t\u6587<<&&&>>%s\r\n"
		}
	},
	"escaping": "\\\\",
	"an-array": [
		{
			"sub-string": "string in an object in an array",
			"sub-sub-array": [
				{
					"sub-sub-string": "string in an object in an array in an object in an <string>"
				}
			]
		},
		56789,
		123.456,
		false,
		null
	]
}`

func TestNewFromBytes(t *testing.T) {
	first, err := NewFromBytes([]byte(raw))
	if err != nil || first == nil {
		t.Error("NewFromBytes failed")
		return
	}
}

func TestNewFromString(t *testing.T) {
	// normal object
	o, err := NewFromString(raw)
	if err != nil || o == nil {
		t.Error("NewFromString failed")
	}
	if false == o.IsObject() {
		t.Error("NewFromString failed")
	}

	// normal array
	a, err := NewFromString("[]")
	if err != nil || a == nil {
		t.Error("NewFromString failed")
	}
	if false == a.IsArray() {
		t.Error("NewFromString failed")
	}

	// normal string
	var s string
	s = "\\g\\"
	a, err = NewFromString(`"` + s + `"`)
	if err != nil || a == nil {
		t.Error("NewFromString failed")
	}
	if false == a.IsString() {
		t.Error("NewFromString failed")
	}
	if a.String() != s {
		t.Errorf("%s != %s", a.String(), s)
	}

	// number
	a, err = NewFromString("12345")
	if err != nil || a == nil {
		t.Error("NewFromString failed")
	}
	if false == a.IsNumber() {
		t.Error("NewFromString failed")
	}
	if a.Int() != 12345 {
		t.Errorf("number illegal %d", a.Int())
	}

	// invalid strings
	func_test_err := func(s string) {
		o, err := NewFromString(s)
		if err == nil {
			t.Errorf("error not detected, orig: %v", s)
			r, _ := o.MarshalToString()
			t.Errorf("object parsed: %s", r)
		} else {
			// t.Logf("got expected message %v", err)
		}
	}

	func_test_err(`{"int": 123\\}`)
	func_test_err(`{"foat": 123.\\}`)
	func_test_err(`{"string\": ""}`)
	func_test_err(`{"string": "\"}`)
	func_test_err(`{"string": "\uoooo"}`)
	func_test_err(`{"string": "\uo"}`)
	func_test_err(`[hahaha`)
	func_test_err(`[`)
	func_test_err(`["string", 12345\, ]`)
	func_test_err(`["string", 123.45\, ]`)
	func_test_err(`["\u\\"]`)
	func_test_err(`{"string}`)
	func_test_err(`[{"string}]`)
}
