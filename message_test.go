package say

import (
	"bytes"
	"sync"
	"testing"
	"time"
)

func TestMessageType(t *testing.T) {
	tests := []test{
		{func() { Event("foo") }, TypeEvent},
		{func() { Events("foo", 42) }, TypeEvent},
		{func() { Value("foo", 42) }, TypeValue},
		{func() { NewTiming().Say("foo") }, TypeValue},
		{func() { Gauge("foo", 42) }, TypeGauge},
		{func() { Debug("foo") }, TypeDebug},
		{func() { Info("foo") }, TypeInfo},
		{func() { Warning("foo") }, TypeWarning},
		{func() { Error("foo") }, TypeError},
		{func() { Fatal("foo") }, TypeFatal},
	}

	testMessage(t, tests, func(m *Message, want interface{}) {
		typ := want.(Type)
		if m.Type != typ {
			t.Errorf("Message.Type = %q, want %q", m.Type, typ)
		}
	})
}

func TestMessageContent(t *testing.T) {
	strings := []struct {
		input, want string
	}{
		{"foo", "foo"},
		{"foo\n", "foo\n"},
		{"foo\nbar", "foo\nbar"},
		{"foo \nbar", "foo \nbar"},
		{"foo\n\nbar", "foo\n\nbar"},
		{"foo\n \nbar", "foo\n \nbar"},
		{"", ""},
		{"\n", "\n"},
		{"\n\n", "\n\n"},
		{"\t", "\t"},
		{"|", "|"},
		{"\t|", "\t|"},
	}

	tests := make([]test, len(strings))
	for i, s := range strings {
		s := s
		tests[i] = test{func() { Info(s.input) }, s.want}
	}
	testMessage(t, tests, func(m *Message, want interface{}) {
		content := want.(string)
		if m.Content != content {
			t.Errorf("Message.Content = %q, want %q", m.Content, content)
		}
	})
}

func TestMessageKey(t *testing.T) {
	tests := []test{
		{func() { Event("foo") }, "foo"},
		{func() { Events(`fo"o`, 42) }, `fo"o`},
		{func() { Value("foo bar", 42) }, "foo bar"},
		{func() { NewTiming().Say("app.host.key") }, "app.host.key"},
		{func() { Gauge("#!€", 42) }, "#!€"},
	}

	testMessage(t, tests, func(m *Message, want interface{}) {
		key := want.(string)
		if m.Key() != key {
			t.Errorf("Message.Key() = %q, want %q", m.Key(), key)
		}
	})
}

func TestMessageValue(t *testing.T) {
	tests := []test{
		{func() { Event("foo") }, ""},
		{func() { Events(`fo"o`, 42) }, "42"},
		{func() { Value("foo bar", 17.6) }, "17.6"},
		{func() { NewTiming().Say("app.host.key") }, "0ms"},
		{func() { Gauge("#!€", -25.5) }, "-25.5"},
	}

	testMessage(t, tests, func(m *Message, want interface{}) {
		value := want.(string)
		if m.Value() != value {
			t.Errorf("Message.Key() = %q, want %q", m.Value(), value)
		}
	})
}

func TestMessageInt(t *testing.T) {
	type result struct {
		i int
		b bool
	}

	tests := []test{
		{func() { Event("foo") }, result{1, true}},
		{func() { Events(`fo"o`, 42) }, result{42, true}},
		{func() { Value("foo bar", 17.6) }, result{17, true}},
		{func() { NewTiming().Say("app.host.key") }, result{0, true}},
		{func() { Gauge("#!€", -25.5) }, result{-25, true}},
		{func() { Info("hello") }, result{0, false}},
	}

	testMessage(t, tests, func(m *Message, want interface{}) {
		res := want.(result)
		i, ok := m.Int()
		if i != res.i || ok != res.b {
			t.Errorf("Message.Int() = (%d, %t), want (%d, %t)",
				i, ok, res.i, res.b)
		}
	})
}

func TestMessageFloat64(t *testing.T) {
	type result struct {
		f float64
		b bool
	}

	tests := []test{
		{func() { Event("foo") }, result{1, true}},
		{func() { Events(`fo"o`, 42) }, result{42, true}},
		{func() { Value("foo bar", 17.6) }, result{17.6, true}},
		{func() { NewTiming().Say("app.host.key") }, result{0, true}},
		{func() { Gauge("#!€", -25.5) }, result{-25.5, true}},
		{func() { Info("hello") }, result{0, false}},
	}

	testMessage(t, tests, func(m *Message, want interface{}) {
		res := want.(result)
		f, ok := m.Float64()
		if f != res.f || ok != res.b {
			t.Errorf("Message.Int() = (%g, %t), want (%g, %t)",
				f, ok, res.f, res.b)
		}
	})
}

func TestMessageDuration(t *testing.T) {
	type result struct {
		d time.Duration
		b bool
	}

	tests := []test{
		{func() { Event("foo") }, result{0, false}},
		{func() { Events(`fo"o`, 42) }, result{0, false}},
		{func() { Value("foo bar", 17.6) }, result{0, false}},
		{func() { NewTiming().Say("app.host.key") }, result{0, true}},
		{func() { Gauge("#!€", -25.5) }, result{0, false}},
		{func() { Info("hello") }, result{0, false}},
	}

	testMessage(t, tests, func(m *Message, want interface{}) {
		res := want.(result)
		d, ok := m.Duration()
		if d != res.d || ok != res.b {
			t.Errorf("Message.Duration() = (%d, %t), want (%d, %t)",
				d, ok, res.d, res.b)
		}
	})
}

func TestMessageError(t *testing.T) {
	log := new(Logger)

	tests := []test{
		{func() { log.Event("foo") }, ""},
		{func() { log.Info("hello") }, ""},
		{func() { log.Error("foo") }, "foo"},
		{func() { log.Fatal("foo\n\nbar\n") }, "foo\n\nbar\n"},
		{func() { log.Error("foo") }, "foo"},
	}

	testMessage(t, tests, func(m *Message, want interface{}) {
		err := want.(string)
		got := m.Error()
		if got != err {
			t.Errorf("Message.Error() = %q, want %q", got, err)
		}
	})
}

func TestMessageStackTrace(t *testing.T) {
	DisableStackTraces(false)
	defer DisableStackTraces(true)
	tests := []test{
		{func() { Event("foo") }, false},
		{func() { Info("hello") }, false},
		{func() { Error("foo") }, true},
		{func() { Fatal("foo\n\nbar\n") }, true},
	}

	testMessage(t, tests, func(m *Message, want interface{}) {
		hasST := want.(bool)
		got := m.StackTrace()
		if hasST && (got == "") {
			t.Errorf("Message.StackTrace() empty")
		} else if !hasST && (got != "") {
			t.Errorf("Message.StackTrace() = %q, want empty string", got)
		}
	})
}

func TestMessageWriteTo(t *testing.T) {
	log := NewLogger(SkipStackFrames(-1))
	tests := []test{
		{func() { log.Event("foo") },
			"2015-11-25 15:47:00.000 EVENT foo\n"},
		{func() { log.Events("foo", 5) },
			"2015-11-25 15:47:00.000 EVENT foo:5\n"},
		{func() { log.Value("foo", 17.6) },
			"2015-11-25 15:47:00.000 VALUE foo:17.6\n"},
		{func() { log.Gauge(`foo"`, -35) },
			"2015-11-25 15:47:00.000 GAUGE foo\":-35\n"},
		{func() { log.NewTiming().Say("foo") },
			"2015-11-25 15:47:00.000 VALUE foo:0ms\n"},
		{func() { log.Debug("foo") },
			"2015-11-25 15:47:00.000 DEBUG foo\n"},
		{func() { log.Info("foo", "a", "b") },
			"2015-11-25 15:47:00.000 INFO  foo\t| a=\"b\"\n"},
		{func() { log.Warning("foo", "i", 1, "f", 3.5) },
			"2015-11-25 15:47:00.000 WARN  foo\t| i=1 f=3.5\n"},
		{func() { log.Error("foo\nbar", "ok", true, "ko", false) },
			"2015-11-25 15:47:00.000 ERROR foo\nbar\t| ok=true ko=false\n"},
		{func() { log.Fatal("foo\tbar\n") },
			"2015-11-25 15:47:00.000 FATAL foo\tbar\n\n"},
	}

	buf := new(bytes.Buffer)
	testMessage(t, tests, func(m *Message, want interface{}) {
		out := want.(string)
		n, err := m.WriteTo(buf)
		got := buf.String()
		if int(n) != len(got) || err != nil {
			t.Errorf("Message.WriteTo = (%d, %v), want (%d, %v)",
				n, err, len(got), nil)
		}
		if got != out {
			t.Errorf("Invalid Message.WriteTo output\n got: %q\nwant: %q",
				got, out)
		}
		buf.Reset()
	})
}

func TestMessageWriteJSONTo(t *testing.T) {
	log := NewLogger(SkipStackFrames(-1))
	tests := []test{
		{func() { log.Event("foo") },
			"{\"timestamp\": \"2015-11-25T15:47:00Z\", \"type\": \"EVENT\", \"content\": \"foo\"}\n"},
		{func() { log.Events("foo", 5) },
			"{\"timestamp\": \"2015-11-25T15:47:00Z\", \"type\": \"EVENT\", \"content\": \"foo:5\"}\n"},
		{func() { log.Value("foo", 17.6) },
			"{\"timestamp\": \"2015-11-25T15:47:00Z\", \"type\": \"VALUE\", \"content\": \"foo:17.6\"}\n"},
		{func() { log.Gauge(`foo"`, -35, "foo", "bar", "foo", "baz") },
			"{\"timestamp\": \"2015-11-25T15:47:00Z\", \"type\": \"GAUGE\", \"content\": \"foo\\\":-35\", \"foo\": \"baz\"}\n"},
		{func() { log.NewTiming().Say("foo", "timestamp", "skip") },
			"{\"timestamp\": \"2015-11-25T15:47:00Z\", \"type\": \"VALUE\", \"content\": \"foo:0ms\"}\n"},
		{func() { log.Debug("foo", "type", "skip") },
			"{\"timestamp\": \"2015-11-25T15:47:00Z\", \"type\": \"DEBUG\", \"content\": \"foo\"}\n"},
		{func() { log.Info("foo", "a", "b") },
			"{\"timestamp\": \"2015-11-25T15:47:00Z\", \"type\": \"INFO\", \"content\": \"foo\", \"a\": \"b\"}\n"},
		{func() { log.Warning("foo", "i", 1, "f", 3.5) },
			"{\"timestamp\": \"2015-11-25T15:47:00Z\", \"type\": \"WARN\", \"content\": \"foo\", \"i\": 1, \"f\": 3.5}\n"},
		{func() { log.Error("foo\nbar", "ok", true, "ko", false) },
			"{\"timestamp\": \"2015-11-25T15:47:00Z\", \"type\": \"ERROR\", \"content\": \"foo\\nbar\", \"ok\": true, \"ko\": false}\n"},
		{func() { log.Fatal("foo\tbar\n") },
			"{\"timestamp\": \"2015-11-25T15:47:00Z\", \"type\": \"FATAL\", \"content\": \"foo\\tbar\\n\"}\n"},
	}

	buf := new(bytes.Buffer)
	testMessage(t, tests, func(m *Message, want interface{}) {
		out := want.(string)
		n, err := m.WriteJSONTo(buf)
		got := buf.String()
		if n != len(got) || err != nil {
			t.Errorf("Message.WriteJSONTo = (%d, %v), want (%d, %v)",
				n, err, len(got), nil)
		}
		if got != out {
			t.Errorf("Invalid Message.WriteJSONTo output\n got: %s\nwant: %s",
				got, out)
		}
		buf.Reset()
	})
}

type test struct {
	f    func()
	want interface{}
}

func testMessage(t *testing.T, tests []test, h func(*Message, interface{})) {
	now = func() time.Time {
		return time.Date(2015, 11, 25, 15, 47, 0, 0, time.UTC)
	}

	var wg sync.WaitGroup
	n := 0
	SetDebug(true)
	SetListener(func(m *Message) {
		if n >= len(tests) {
			t.Fatal("Listen received too many messages.")
		}
		h(m, tests[n].want)
		n++
		wg.Done()
	})
	defer SetListener(nil)
	defer SetDebug(false)

	for _, test := range tests {
		wg.Add(1)
		test.f()
	}
	wg.Wait()
}
