package say

import (
	"fmt"
	"testing"
)

func TestSetDataError(t *testing.T) {
	tests := []struct {
		data []interface{}
		err  error
	}{
		{[]interface{}{"foo"}, errOddNumArgs},
		{[]interface{}{true, "foo"}, errKeyNotString},
		{[]interface{}{"foo\n", 1}, errKeyInvalid},
		{[]interface{}{"", 1}, errKeyEmpty},
	}

	for _, tt := range tests {
		func() {
			defer func() {
				err := recover()
				if err != tt.err {
					t.Errorf("SetData(%v) = %v, want %v", tt.data, err, tt.err)
				}
			}()
			SetData(tt.data...)
		}()
	}
}

func TestAddDataError(t *testing.T) {
	tests := []struct {
		key   string
		value interface{}
		err   error
	}{
		{"foo\t", 1, errKeyInvalid},
		{"", 1, errKeyEmpty},
	}

	for _, tt := range tests {
		func() {
			defer func() {
				err := recover()
				if err != tt.err {
					t.Errorf("AddData(%s, %v) = %v, want %v",
						tt.key, tt.value, err, tt.err)
				}
			}()
			AddData(tt.key, tt.value)
		}()
	}
}

func TestDataFormat(t *testing.T) {
	expect(t, func() {
		Value("foo", float32(-.61))
		Value("foo", true)
		Value("foo", []int{1, 2, 3})
	}, []string{
		"VALUE foo:-0.61",
		"VALUE foo:true",
		"VALUE foo:[1 2 3]",
	})
}

func TestMessageData(t *testing.T) {
	tests := []test{
		{func() { Info("", "a", 5) }, Data{{"a", 5}}},
		{func() { Info("", "a", "b") }, Data{{"a", "b"}}},
		{func() { Info("", "5", "b\nb") }, Data{{"5", "b\nb"}}},
		{func() { Info("", "5", true) }, Data{{"5", true}}},
		{func() { Info("", "5", false) }, Data{{"5", false}}},
		{func() { Info("", "a", -10.3) }, Data{{"a", -10.3}}},
		{func() { Info("", "foo ", " bar ") }, Data{{"foo ", " bar "}}},
		{func() { Info("", "a", "b", "c", "d") }, Data{{"a", "b"}, {"c", "d"}}},
	}

	testMessage(t, tests, func(m *Message, want interface{}) {
		wantString := toString(want.(Data))
		gotString := toString(m.Data)
		if gotString != wantString {
			t.Errorf("Data = %s, want %s", gotString, wantString)
			return
		}
	})
}

func toString(d Data) string {
	if len(d) == 0 {
		return `()`
	}
	var s string
	for _, kv := range d {
		s += fmt.Sprintf(",(K:%q, V:%q)", kv.Key, kv.Value)
	}
	return s[1:]
}

func TestDataGet(t *testing.T) {
	d := Data{
		{"string", 5},
		{"string", 7},
		{"bool", true},
		{"int", 5},
		{"float", 17.6},
	}

	tests := []struct {
		key  string
		want interface{}
		ok   bool
	}{
		{"string", 7, true},
		{"bool", true, true},
		{"int", 5, true},
		{"float", 17.6, true},
		{"foo", nil, false},
		{"", nil, false},
	}

	for _, tt := range tests {
		got, ok := d.Get(tt.key)
		if got != tt.want || ok != tt.ok {
			t.Errorf("Data.Get(%q) = (%v, %v), want (%v, %v)",
				got, ok, tt.want, tt.ok)
		}
	}
}
