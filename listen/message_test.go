package listen

import (
	"bytes"
	"testing"
	"time"

	"gopkg.in/say.v0"
)

func init() {
	now = func() time.Time {
		return time.Date(2015, 11, 25, 15, 47, 0, 0, time.UTC)
	}
}

func TestType(t *testing.T) {
	tests := []test{
		{func() { say.Init("foo") }, TypeInit},
		{func() { say.Event("foo") }, TypeEvent},
		{func() { say.Events("foo", 42) }, TypeEvent},
		{func() { say.Value("foo", 42) }, TypeValue},
		{func() { say.NewTiming().Say("foo") }, TypeValue},
		{func() { say.Gauge("foo", 42) }, TypeGauge},
		{func() { say.Debug("foo") }, TypeDebug},
		{func() { say.Info("foo") }, TypeInfo},
		{func() { say.Warning("foo") }, TypeWarning},
		{func() { say.Error("foo") }, TypeError},
		{func() { say.Fatal("foo") }, TypeFatal},
	}

	testMessage(t, tests, func(m *Message, want interface{}) {
		typ := want.(Type)
		if m.Type() != typ {
			t.Errorf("Message.Type() = %q, want %q", m.Type(), typ)
		}
	})
}

func TestContent(t *testing.T) {
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
		{"\t|", " |"},
	}

	tests := make([]test, len(strings))
	for i, s := range strings {
		s := s
		tests[i] = test{func() { say.Info(s.input) }, s.want}
	}
	testMessage(t, tests, func(m *Message, want interface{}) {
		content := want.(string)
		if m.Content() != content {
			t.Errorf("Message.Content() = %q, want %q", m.Content(), content)
		}
	})
}

func TestKey(t *testing.T) {
	tests := []test{
		{func() { say.Event("foo") }, "foo"},
		{func() { say.Events(`fo"o`, 42) }, `fo"o`},
		{func() { say.Value("foo bar", 42) }, "foo bar"},
		{func() { say.NewTiming().Say("app.host.key") }, "app.host.key"},
		{func() { say.Gauge("#!€", 42) }, "#!€"},
	}

	testMessage(t, tests, func(m *Message, want interface{}) {
		key := want.(string)
		if m.Key() != key {
			t.Errorf("Message.Key() = %q, want %q", m.Key(), key)
		}
	})
}

func TestValue(t *testing.T) {
	tests := []test{
		{func() { say.Event("foo") }, ""},
		{func() { say.Events(`fo"o`, 42) }, "42"},
		{func() { say.Value("foo bar", 17.6) }, "17.6"},
		{func() { say.NewTiming().Say("app.host.key") }, "5ms"},
		{func() { say.Gauge("#!€", -25.5) }, "-25.5"},
	}

	testMessage(t, tests, func(m *Message, want interface{}) {
		value := want.(string)
		if m.Value() != value {
			t.Errorf("Message.Key() = %q, want %q", m.Value(), value)
		}
	})
}

func TestInt(t *testing.T) {
	type result struct {
		i int
		b bool
	}

	tests := []test{
		{func() { say.Event("foo") }, result{1, true}},
		{func() { say.Events(`fo"o`, 42) }, result{42, true}},
		{func() { say.Value("foo bar", 17.6) }, result{17, true}},
		{func() { say.NewTiming().Say("app.host.key") }, result{5, true}},
		{func() { say.Gauge("#!€", -25.5) }, result{-25, true}},
		{func() { say.Info("hello") }, result{0, false}},
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

func TestFloat64(t *testing.T) {
	type result struct {
		f float64
		b bool
	}

	tests := []test{
		{func() { say.Event("foo") }, result{1, true}},
		{func() { say.Events(`fo"o`, 42) }, result{42, true}},
		{func() { say.Value("foo bar", 17.6) }, result{17.6, true}},
		{func() { say.NewTiming().Say("app.host.key") }, result{5, true}},
		{func() { say.Gauge("#!€", -25.5) }, result{-25.5, true}},
		{func() { say.Info("hello") }, result{0, false}},
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

func TestDuration(t *testing.T) {
	type result struct {
		d time.Duration
		b bool
	}

	tests := []test{
		{func() { say.Event("foo") }, result{0, false}},
		{func() { say.Events(`fo"o`, 42) }, result{0, false}},
		{func() { say.Value("foo bar", 17.6) }, result{0, false}},
		{func() { say.NewTiming().Say("app.host.key") }, result{5 * time.Millisecond, true}},
		{func() { say.Gauge("#!€", -25.5) }, result{0, false}},
		{func() { say.Info("hello") }, result{0, false}},
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

func TestError(t *testing.T) {
	log := new(say.Logger)

	tests := []test{
		{func() { log.Event("foo") }, ""},
		{func() { log.Info("hello") }, ""},
		{func() { log.Error("foo") }, "foo"},
		{func() { log.Fatal("foo\n\nbar\n") }, "foo\n\nbar\n"},
		{func() { log.SkipStackFrames(-1); log.Error("foo") }, "foo"},
	}

	testMessage(t, tests, func(m *Message, want interface{}) {
		err := want.(string)
		got := m.Error()
		if got != err {
			t.Errorf("Message.Error() = %q, want %q", got, err)
		}
	})
}

func TestStackTrace(t *testing.T) {
	tests := []test{
		{func() { say.Event("foo") }, false},
		{func() { say.Info("hello") }, false},
		{func() { say.Error("foo") }, true},
		{func() { say.Fatal("foo\n\nbar\n") }, true},
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

func TestDataString(t *testing.T) {
	tests := []test{
		{func() { say.Info("foo", "int", 1) }, "int=1"},
		{func() { say.Info("foo", "foo", "bar") }, `foo="bar"`},
		{func() { say.Info("foo", "a", `b="c"`) }, `a="b=\"c\""`},
		{func() { say.Info("foo", "a", "\n") }, "a=\"\\n\""},
		{func() { say.Info("foo", "ok", true) }, "ok=true"},
		{func() { say.Info("foo", "ko", false) }, "ko=false"},
		{func() { say.Info("foo", "i", 5, "f", -12.5) }, "i=5 f=-12.5"},
	}

	testMessage(t, tests, func(m *Message, want interface{}) {
		data := want.(string)
		got := m.DataString()
		if got != data {
			t.Errorf("Message.DataString() = %q, want %q", got, data)
		}
	})
}

func TestWriteMessage(t *testing.T) {
	log := new(say.Logger)
	log.SkipStackFrames(-1)
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
			"2015-11-25 15:47:00.000 VALUE foo:5ms\n"},
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
		{func() { log.Init("my_app", "version", 1.1) },
			"2015-11-25 15:47:00.000 INIT  my_app\t| version=1.1\n"},
	}

	buf := new(bytes.Buffer)
	testMessage(t, tests, func(m *Message, want interface{}) {
		out := want.(string)
		n, err := m.Write(buf)
		got := buf.String()
		if n != len(got) || err != nil {
			t.Errorf("Message.Write = (%d, %v), want (%d, %v)",
				n, err, len(got), nil)
		}
		if got != out {
			t.Errorf("Invalid Message.Write output\n got: %q\nwant: %q",
				got, out)
		}
		buf.Reset()
	})
}

func TestWriteJSONMessage(t *testing.T) {
	log := new(say.Logger)
	log.SkipStackFrames(-1)
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
			"{\"timestamp\": \"2015-11-25T15:47:00Z\", \"type\": \"VALUE\", \"content\": \"foo:5ms\"}\n"},
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
		{func() { log.Init("my_app", "version", 1.1) },
			"{\"timestamp\": \"2015-11-25T15:47:00Z\", \"type\": \"INIT\", \"content\": \"my_app\", \"version\": 1.1}\n"},
	}

	buf := new(bytes.Buffer)
	testMessage(t, tests, func(m *Message, want interface{}) {
		out := want.(string)
		n, err := m.WriteJSON(buf)
		got := buf.String()
		if n != len(got) || err != nil {
			t.Errorf("Message.Write = (%d, %v), want (%d, %v)",
				n, err, len(got), nil)
		}
		if got != out {
			t.Errorf("Invalid Message.WriteJSON output\n got: %s\nwant: %s",
				got, out)
		}
		buf.Reset()
	})
}
