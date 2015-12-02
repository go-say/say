package listen

import (
	"fmt"
	"testing"

	"gopkg.in/say.v0"
)

func TestData(t *testing.T) {
	tests := []test{
		{func() { say.Info("", "a", 5) }, Data{{"a", "5"}}},
		{func() { say.Info("", "a", "b") }, Data{{"a", `"b"`}}},
		{func() { say.Info("", "5", "b\nb") }, Data{{"5", `"b\nb"`}}},
		{func() { say.Info("", "5", true) }, Data{{"5", "true"}}},
		{func() { say.Info("", "5", false) }, Data{{"5", "false"}}},
		{func() { say.Info("", "a", -10.3) }, Data{{"a", "-10.3"}}},
		{func() { say.Info("", "foo ", " bar ") }, Data{{"foo ", `" bar "`}}},
		{func() { say.Info("", "a", "b", "c", "d") }, Data{{"a", `"b"`}, {"c", `"d"`}}},
	}

	testMessage(t, tests, func(m *Message, want interface{}) {
		wantString := toString(want.(Data))
		gotString := toString(m.Data())
		if gotString != wantString {
			t.Errorf("Data() = %s, want %s", gotString, wantString)
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

type dataTest struct {
	data []interface{}
	want []dataWant
}

type dataWant struct {
	key  string
	want interface{}
}

func testData(t *testing.T, dataTests []dataTest, h func(Data, string, interface{})) {
	tests := make([]test, len(dataTests))
	for i, dt := range dataTests {
		dt := dt
		tests[i] = test{
			f:    func() { say.Info("", dt.data...) },
			want: dt.want,
		}
	}
	testMessage(t, tests, func(m *Message, want interface{}) {
		dws := want.([]dataWant)
		data := m.Data()
		for _, dw := range dws {
			h(data, dw.key, dw.want)
		}
	})
}

func TestDataHas(t *testing.T) {
	tests := []dataTest{
		{[]interface{}{"foo", "bar"}, []dataWant{
			{"foo", true},
			{"bar", false},
			{"", false},
		}},
	}

	testData(t, tests, func(d Data, key string, want interface{}) {
		ok := want.(bool)
		got := d.Has(key)
		if got != ok {
			t.Errorf("Data.Has(%q) = %t, want %t", key, got, ok)
		}
	})
}

func TestDataGetString(t *testing.T) {
	type result struct {
		s  string
		ok bool
	}

	tests := []dataTest{
		{[]interface{}{"foo", "bar"}, []dataWant{
			{"foo", result{"bar", true}},
			{"bar", result{"", false}},
			{"", result{"", false}},
		}},
		{[]interface{}{"i", 5, "f", 3.5}, []dataWant{
			{"i", result{"", false}},
			{"f", result{"", false}},
		}},
		{[]interface{}{"ok", true, "ko", false}, []dataWant{
			{"ok", result{"", false}},
			{"ko", result{"", false}},
		}},
	}

	testData(t, tests, func(d Data, key string, want interface{}) {
		res := want.(result)
		s, ok := d.GetString(key)
		if s != res.s || ok != res.ok {
			t.Errorf("Data.GetString(%q) = (%q, %t), want (%q, %t)",
				key, s, ok, res.s, res.ok)
		}
	})
}

func TestDataGetInt(t *testing.T) {
	type result struct {
		i  int
		ok bool
	}

	tests := []dataTest{
		{[]interface{}{"foo", "bar"}, []dataWant{
			{"foo", result{0, false}},
			{"bar", result{0, false}},
			{"", result{0, false}},
		}},
		{[]interface{}{"a", 7}, []dataWant{
			{"a", result{7, true}},
			{"foo", result{0, false}},
		}},
		{[]interface{}{"a", -3}, []dataWant{
			{"a", result{-3, true}},
			{"foo", result{0, false}},
		}},
		{[]interface{}{"a", 4.5}, []dataWant{
			{"a", result{4, true}},
			{"foo", result{0, false}},
		}},
	}

	testData(t, tests, func(d Data, key string, want interface{}) {
		res := want.(result)
		i, ok := d.GetInt(key)
		if i != res.i || ok != res.ok {
			t.Errorf("Data.GetInt(%q) = (%d, %t), want (%d, %t)",
				key, i, ok, res.i, res.ok)
		}
	})
}

func TestDataGetFloat64(t *testing.T) {
	type result struct {
		f  float64
		ok bool
	}

	tests := []dataTest{
		{[]interface{}{"foo", "bar"}, []dataWant{
			{"foo", result{0, false}},
			{"bar", result{0, false}},
			{"", result{0, false}},
		}},
		{[]interface{}{"a", 7}, []dataWant{
			{"a", result{7, true}},
			{"foo", result{0, false}},
		}},
		{[]interface{}{"a", -3}, []dataWant{
			{"a", result{-3, true}},
			{"foo", result{0, false}},
		}},
		{[]interface{}{"a", 4.5}, []dataWant{
			{"a", result{4.5, true}},
			{"foo", result{0, false}},
		}},
	}

	testData(t, tests, func(d Data, key string, want interface{}) {
		res := want.(result)
		f, ok := d.GetFloat64(key)
		if f != res.f || ok != res.ok {
			t.Errorf("Data.GetFloat64(%q) = (%g, %t), want (%g, %t)",
				key, f, ok, res.f, res.ok)
		}
	})
}

func TestDataGetBool(t *testing.T) {
	type result struct {
		b  bool
		ok bool
	}

	tests := []dataTest{
		{[]interface{}{"foo", "bar"}, []dataWant{
			{"foo", result{false, false}},
			{"bar", result{false, false}},
			{"", result{false, false}},
		}},
		{[]interface{}{"i", 5, "f", 3.5}, []dataWant{
			{"i", result{false, false}},
			{"f", result{false, false}},
		}},
		{[]interface{}{"ok", true, "ko", false}, []dataWant{
			{"ok", result{true, true}},
			{"ko", result{false, true}},
		}},
	}

	testData(t, tests, func(d Data, key string, want interface{}) {
		res := want.(result)
		b, ok := d.GetBool(key)
		if b != res.b || ok != res.ok {
			t.Errorf("Data.GetBool(%q) = (%t, %t), want (%t, %t)",
				key, b, ok, res.b, res.ok)
		}
	})
}
