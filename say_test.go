package say

import (
	"bytes"
	"errors"
	"io/ioutil"
	"log"
	"strings"
	"sync"
	"testing"
	"time"
)

func init() {
	SkipStackFrames(-1)
	exit = func(int) {}
}

func expect(t *testing.T, f func(), lines []string) {
	buf := new(bytes.Buffer)
	w := Redirect(buf)
	defer Redirect(w)

	f()
	SetData()

	want := strings.Join(lines, "\n") + "\n"
	got := buf.String()

	if got != want {
		t.Errorf("invalid output, got:\n%s\nwant:\n%s", got, want)
	}
}

func TestEvent(t *testing.T) {
	expect(t, func() {
		Event("test.event")
	}, []string{
		"EVENT test.event",
	})
}

func TestEvents(t *testing.T) {
	expect(t, func() {
		Events("test.event", 5)
	}, []string{
		"EVENT test.event:5",
	})
}

func TestValue(t *testing.T) {
	expect(t, func() {
		Value("test.value", 10)
	}, []string{
		"VALUE test.value:10",
	})
}

func TestTiming(t *testing.T) {
	i := 0
	date := time.Date(2015, 9, 1, 21, 37, 0, 0, time.UTC)
	now = func() time.Time {
		i++
		switch i {
		default:
			return date
		case 2:
			return date.Add(100 * time.Millisecond)
		case 3:
			return date.Add(time.Second)
		}
	}

	expect(t, func() {
		timing := NewTiming()
		timing.Say("test.timing")
		if timing.Get() != time.Second {
			t.Errorf("Timing.Get() = %v, want %v", timing.Get(), time.Second)
		}
	}, []string{
		"VALUE test.timing:100ms",
	})
}

func TestGauge(t *testing.T) {
	expect(t, func() {
		Gauge("test.gauge", 10)
	}, []string{
		"GAUGE test.gauge:10",
	})
}

func TestDebug(t *testing.T) {
	expect(t, func() {
		Debug("foo")
		Info("foo", "debug", DebugHook(45))
		SetData("foo", DebugHook("bar"))
		Info("bar")
		Info("baz", "debug", DebugHook(45))
		SetDebug(true)
		Debug("bar")
		Info("bar", "debug", DebugHook(45))
		Debug("")
		SetDebug(false)
		Debug("baz")
	}, []string{
		"INFO  foo",
		"INFO  bar",
		"INFO  baz",
		`DEBUG bar	| foo="bar"`,
		`INFO  bar	| foo="bar" debug=45`,
		`DEBUG 	| foo="bar"`,
	})
}

func TestInfo(t *testing.T) {
	expect(t, func() {
		Info("Test message!")
		Info("")
	}, []string{
		"INFO  Test message!",
		"INFO  ",
	})
}

func TestWarning(t *testing.T) {
	expect(t, func() {
		Warning("Test message!")
		Warning("")
	}, []string{
		"WARN  Test message!",
		"WARN  ",
	})
}

func TestError(t *testing.T) {
	expect(t, func() {
		Error("Test error")
		Error("")
	}, []string{
		"ERROR Test error",
		"ERROR ",
	})
}

func TestLoggerError(t *testing.T) {
	expect(t, func() {
		log := new(Logger)
		log.SkipStackFrames(-1)
		var err error
		log.Error(err)
		err = errors.New("Test error")
		log.Error(err)
		log.Error(func() error { return err })
		log.Error(func() error { return nil })
	}, []string{
		"ERROR Test error",
		"ERROR Test error",
	})
}

func TestFatal(t *testing.T) {
	expect(t, func() {
		Fatal("Test fatal")
	}, []string{
		"FATAL Test fatal",
	})
}

func TestMultiline(t *testing.T) {
	expect(t, func() {
		Info("foo\nbar \nbaz ")
		Info("\n")
		Info("\n\n")
	}, []string{
		"INFO  foo",
		"      bar ",
		"      baz ",
		"INFO  ",
		"      ",
		"INFO  ",
		"      ",
		"      ",
	})
}

func TestData(t *testing.T) {
	expect(t, func() {
		Info("foo")
		AddData("pi", 3.1415)
		Info("foo")
		SetData("ok", true, "ko", false)
		Info("foo", "i", uint(15))
	}, []string{
		"INFO  foo",
		"INFO  foo	| pi=3.1415",
		"INFO  foo	| ok=true ko=false i=15",
	})
}

func TestNewLogger(t *testing.T) {
	expect(t, func() {
		SetData("foo", "bar")
		log := NewLogger()
		log.Error("oops")
	}, []string{
		`ERROR oops	| foo="bar"`,
	})
}

func TestTimeHook(t *testing.T) {
	expect(t, func() {
		Info("foo", "timestamp", TimeHook("2006-01-02 15:04:05"))
	}, []string{
		`INFO  foo	| timestamp="2015-09-01 21:37:00"`,
	})
}

func TestInvalidData(t *testing.T) {
	expect(t, func() {
		log := new(Logger)
		log.SkipStackFrames(-1)
		log.Info("foo", "a")
		log.Event("")
		log.Event("f\noo:")
		log.Event("foo:")
		log.Events("foo=", 2)
		log.Gauge("\tfoo", 2)
		log.Value("=foo", 2)
		log.NewTiming().Say(":foo")
		log.Info("foo", "a")
		log.Info("foo", true, "foo")
		log.Info("foo", "foo\t", "bar")
	}, []string{
		"ERROR " + errOddNumArgs.Error(),
		"INFO  foo",
		"ERROR " + errKeyEmpty.Error(),
		"ERROR " + errKeyInvalid.Error(),
		"ERROR " + errKeyInvalid.Error(),
		"ERROR " + errKeyInvalid.Error(),
		"ERROR " + errKeyInvalid.Error(),
		"ERROR " + errKeyInvalid.Error(),
		"ERROR " + errKeyInvalid.Error(),
		"ERROR " + errOddNumArgs.Error(),
		"INFO  foo",
		"ERROR " + errKeyNotString.Error(),
		"INFO  foo",
		"ERROR " + errKeyInvalid.Error(),
		"INFO  foo",
	})
}

func TestInvalidKeys(t *testing.T) {
	expect(t, func() {
		log := new(Logger)
		log.SkipStackFrames(-1)
		log.Event("")
		log.Event("\n")
		log.Event("foo\t")
		log.Event("foo:bar")
		log.Event("=bar")
	}, []string{
		"ERROR " + errKeyEmpty.Error(),
		"ERROR " + errKeyInvalid.Error(),
		"ERROR " + errKeyInvalid.Error(),
		"ERROR " + errKeyInvalid.Error(),
		"ERROR " + errKeyInvalid.Error(),
	})
}

func TestSkipStackFrames(t *testing.T) {
	tests := []struct {
		mustHave, mustNotHave []string
	}{
		{
			[]string{"testStackFrames", "TestSkipStackFrames"},
			nil,
		},
		{
			[]string{"TestSkipStackFrames"},
			[]string{"testStackFrames"},
		},
		{
			nil,
			[]string{"TestSkipStackFrames", "testStackFrames"},
		},
	}

	for i, tt := range tests {
		testStackFrames(t, i, tt.mustHave, tt.mustNotHave)
	}
}

func testStackFrames(t *testing.T, skip int, mustHave, mustNotHave []string) {
	buf := new(bytes.Buffer)
	w := Redirect(buf)
	defer Redirect(w)

	log := new(Logger)
	log.SkipStackFrames(skip)
	log.Error("foo")
	log.Fatal("bar")

	SkipStackFrames(skip)
	Error("foo")
	Fatal("bar")
	SkipStackFrames(-1)

	mustNotHave = append(mustNotHave, []string{
		"goroutine",
		"/say.go:",
	}...)

	got := buf.String()
	for _, s := range mustHave {
		if !strings.Contains(got, s) {
			t.Errorf("%q does not appear in the stack frames (skip=%d):\n%s",
				s, skip, got)
		}
	}
	for _, s := range mustNotHave {
		if strings.Contains(got, s) {
			t.Errorf("%q should not appear in the stack frames (skip=%d):\n%s",
				s, skip, got)
		}
	}
}

func TestCaptureStandardLog(t *testing.T) {
	expect(t, func() {
		CaptureStandardLog()
		log.Print("foo")
	}, []string{
		"INFO  foo",
	})
}

func TestRace(t *testing.T) {
	w := Redirect(ioutil.Discard)
	defer Redirect(w)

	var wg sync.WaitGroup
	log := new(Logger)

	Info("foo")

	wg.Add(1)
	go func() {
		Info("foo")
		wg.Done()
	}()

	wg.Add(1)
	go func() {
		log.Info("foo")
		wg.Done()
	}()

	wg.Add(1)
	go func() {
		log.AddData("foo", "bar")
		wg.Done()
	}()

	wg.Add(1)
	go func() {
		log.SetData("foo", "bar")
		wg.Done()
	}()

	Info("foo")
	log.Info("foo")
	wg.Wait()
}

func BenchmarkStd(b *testing.B) {
	log.SetOutput(ioutil.Discard)
	for i := 0; i < b.N; i++ {
		log.Print("Test message!")
	}
}

func BenchmarkStdData(b *testing.B) {
	log.SetOutput(ioutil.Discard)
	for i := 0; i < b.N; i++ {
		log.Printf("Test message!	| foo=%q i=%d", "b", 42)
	}
}

func BenchmarkInfo(b *testing.B) {
	out = ioutil.Discard
	for i := 0; i < b.N; i++ {
		Info("Test message!")
	}
}

func BenchmarkInfoData(b *testing.B) {
	out = ioutil.Discard
	for i := 0; i < b.N; i++ {
		Info("Test message!", "foo", "bar", "i", 42)
	}
}

func BenchmarkWarning(b *testing.B) {
	out = ioutil.Discard
	for i := 0; i < b.N; i++ {
		Warning("Test message!")
	}
}

func BenchmarkError(b *testing.B) {
	out = ioutil.Discard
	err := errors.New("bench error")
	for i := 0; i < b.N; i++ {
		Error(err)
	}
}

func BenchmarkEvent(b *testing.B) {
	out = ioutil.Discard
	for i := 0; i < b.N; i++ {
		Event("bench_event")
	}
}

func BenchmarkEvents(b *testing.B) {
	out = ioutil.Discard
	for i := 0; i < b.N; i++ {
		Events("bench_event", 17)
	}
}

func BenchmarkTiming(b *testing.B) {
	out = ioutil.Discard
	for i := 0; i < b.N; i++ {
		NewTiming().Say("timing")
	}
}

func BenchmarkGauge(b *testing.B) {
	out = ioutil.Discard
	for i := 0; i < b.N; i++ {
		Gauge("bench_event", 17.06)
	}
}

func BenchmarkData2(b *testing.B) {
	out = ioutil.Discard
	for i := 0; i < b.N; i++ {
		Info("Test message!", "a", "b", "i", 57)
	}
}

func BenchmarkData3(b *testing.B) {
	out = ioutil.Discard
	for i := 0; i < b.N; i++ {
		Info("Test message!", "a", "b", "i", 57, "d", true)
	}
}

func BenchmarkData4(b *testing.B) {
	out = ioutil.Discard
	for i := 0; i < b.N; i++ {
		Info("Test message!", "a", "b", "i", 57, "d", true, "e", "lol")
	}
}

func BenchmarkData5(b *testing.B) {
	out = ioutil.Discard
	for i := 0; i < b.N; i++ {
		Info("Test message!", "a", "b", "i", 57, "d", true, "e", "lol", "j", 45)
	}
}
