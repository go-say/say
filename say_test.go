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

var testTraces string

func init() {
	setPackageErrorStack()
	runtimeStack = func(traces []byte, _ bool) int {
		for i := range traces {
			if i == len(testTraces) {
				return i
			}
			traces[i] = testTraces[i]
		}
		return len(traces)
	}
	exit = func(int) {}
}

func setPackageErrorStack() {
	testTraces = "goroutine 1 [running]:\n" +
		"say.writeStackTrace()\n" +
		"	say.go:6 +0x0\n" +
		"say.(*Logger).error()\n" +
		"	say.go:5 +0x0\n" +
		"say.(*Logger).Error()\n" +
		"	say.go:4 +0x0\n" +
		"say.Error()\n" +
		"	say.go:3 +0x0\n" +
		"github.com/me/myapp.myFunc()\n" +
		"	main.go:2 +0x0\n" +
		"github.com/me/myapp.main()\n" +
		"	main.go:1 +0x0\n"
}

func setLoggerErrorStack() {
	testTraces = "goroutine 1 [running]:\n" +
		"say.writeStackTrace()\n" +
		"	say.go:5 +0x0\n" +
		"say.(*Logger).error()\n" +
		"	say.go:4 +0x0\n" +
		"say.(*Logger).Error()\n" +
		"	say.go:3 +0x0\n" +
		"github.com/me/myapp.myFunc()\n" +
		"	main.go:2 +0x0\n" +
		"github.com/me/myapp.main()\n" +
		"	main.go:1 +0x0\n"
}

func setLoggerDataStack() {
	testTraces = "goroutine 1 [running]:\n" +
		"say.writeStackTrace()\n" +
		"	say.go:5 +0x0\n" +
		"say.(*Logger).sayError()\n" +
		"	say.go:4 +0x0\n" +
		"say.(*Logger).Event()\n" +
		"	say.go:3 +0x0\n" +
		"github.com/me/myapp.myFunc()\n" +
		"	main.go:2 +0x0\n" +
		"github.com/me/myapp.main()\n" +
		"	main.go:1 +0x0\n"
}

func expect(t *testing.T, f func(), lines []string) {
	buf := new(bytes.Buffer)
	w := Redirect(buf)
	defer Redirect(w)

	f()
	SetData()
	SkipStackFrames(0)

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
		"ERROR Test error ",
		"       ",
		"      github.com/me/myapp.myFunc() ",
		"      	main.go:2 +0x0 ",
		"      github.com/me/myapp.main() ",
		"      	main.go:1 +0x0",
		"ERROR  ",
		"       ",
		"      github.com/me/myapp.myFunc() ",
		"      	main.go:2 +0x0 ",
		"      github.com/me/myapp.main() ",
		"      	main.go:1 +0x0",
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
		"FATAL Test fatal ",
		"       ",
		"      github.com/me/myapp.myFunc() ",
		"      	main.go:2 +0x0 ",
		"      github.com/me/myapp.main() ",
		"      	main.go:1 +0x0",
	})
}

func TestInit(t *testing.T) {
	expect(t, func() {
		Init("my_app")
	}, []string{
		"INIT  my_app",
	})
}

func TestMultiline(t *testing.T) {
	expect(t, func() {
		Info("foo\nbar \nbaz ")
		Info("\n")
		Info("\n\n")
	}, []string{
		"INFO  foo ",
		"      bar  ",
		"      baz",
		"INFO   ",
		"      ",
		"INFO   ",
		"       ",
		"      ",
	})
}

func TestEscape(t *testing.T) {
	expect(t, func() {
		Info("foo\t| i=1")
	}, []string{
		"INFO  foo | i=1",
	})
}

func TestNewLogger(t *testing.T) {
	expect(t, func() {
		SetData("foo", "bar")
		SkipStackFrames(-1)
		log := NewLogger()
		log.Error("oops")
	}, []string{
		`ERROR oops	| foo="bar"`,
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

func TestTimeHook(t *testing.T) {
	expect(t, func() {
		Info("foo", "timestamp", TimeHook("2006-01-02 15:04:05"))
	}, []string{
		`INFO  foo	| timestamp="2015-09-01 21:37:00"`,
	})
}

func TestInvalidData(t *testing.T) {
	setLoggerErrorStack()
	defer setPackageErrorStack()

	expect(t, func() {
		log := new(Logger)
		log.SetData("foo")
		log.Info("foo", "a")
		log.SkipStackFrames(-1)
		log.SetData(true, "foo")
		log.SetData("foo\n", 1)
		log.SetData("", 1)
		log.AddData("foo\t", 1)
		log.AddData("", 1)
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
		"ERROR " + errOddNumArgs.Error() + " ",
		"       ",
		"      github.com/me/myapp.myFunc() ",
		"      	main.go:2 +0x0 ",
		"      github.com/me/myapp.main() ",
		"      	main.go:1 +0x0",
		"INFO  foo",
		"ERROR " + errOddNumArgs.Error() + " ",
		"       ",
		"      github.com/me/myapp.myFunc() ",
		"      	main.go:2 +0x0 ",
		"      github.com/me/myapp.main() ",
		"      	main.go:1 +0x0",
		"ERROR " + errKeyNotString.Error(),
		"ERROR " + errKeyInvalid.Error(),
		"ERROR " + errKeyEmpty.Error(),
		"ERROR " + errKeyInvalid.Error(),
		"ERROR " + errKeyEmpty.Error(),
		"ERROR " + errKeyEmpty.Error(),
		"ERROR " + errKeyInvalid.Error(),
		"ERROR " + errKeyInvalid.Error(),
		"ERROR " + errKeyInvalid.Error(),
		"ERROR " + errKeyInvalid.Error(),
		"ERROR " + errKeyInvalid.Error(),
		"ERROR " + errKeyInvalid.Error(),
		"INFO  foo",
		"ERROR " + errOddNumArgs.Error(),
		"INFO  foo",
		"ERROR " + errKeyNotString.Error(),
		"INFO  foo",
		"ERROR " + errKeyInvalid.Error(),
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
	expect(t, func() {
		Error("foo")
		Fatal("bar")
		SkipStackFrames(-1)
		Error("foo")
		Fatal("bar")
		SkipStackFrames(1)
		Error("foo")
		Fatal("bar")
		SkipStackFrames(2)
		Error("foo")
		Fatal("bar")
	}, []string{
		"ERROR foo ",
		"       ",
		"      github.com/me/myapp.myFunc() ",
		"      	main.go:2 +0x0 ",
		"      github.com/me/myapp.main() ",
		"      	main.go:1 +0x0",
		"FATAL bar ",
		"       ",
		"      github.com/me/myapp.myFunc() ",
		"      	main.go:2 +0x0 ",
		"      github.com/me/myapp.main() ",
		"      	main.go:1 +0x0",
		"ERROR foo",
		"FATAL bar",
		"ERROR foo ",
		"       ",
		"      github.com/me/myapp.main() ",
		"      	main.go:1 +0x0",
		"FATAL bar ",
		"       ",
		"      github.com/me/myapp.main() ",
		"      	main.go:1 +0x0",
		"ERROR foo",
		"FATAL bar",
	})
}

func TestCaptureStandardLog(t *testing.T) {
	expect(t, func() {
		CaptureStandardLog()
		log.Print("foo")
	}, []string{
		"INFO  foo",
	})
}

func TestCapturePanic(t *testing.T) {
	expect(t, func() {
		defer CapturePanic()
		panic("hello")
	}, []string{
		"FATAL hello ",
		"       ",
		"      github.com/me/myapp.myFunc() ",
		"      	main.go:2 +0x0 ",
		"      github.com/me/myapp.main() ",
		"      	main.go:1 +0x0",
	})
}

func TestLoggerCapturePanic(t *testing.T) {
	setLoggerErrorStack()
	defer setPackageErrorStack()

	expect(t, func() {
		defer new(Logger).CapturePanic()
		panic("hello")
	}, []string{
		"FATAL hello ",
		"       ",
		"      github.com/me/myapp.myFunc() ",
		"      	main.go:2 +0x0 ",
		"      github.com/me/myapp.main() ",
		"      	main.go:1 +0x0",
	})
}

func TestDataRace(t *testing.T) {
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
	Redirect(ioutil.Discard)
	for i := 0; i < b.N; i++ {
		Info("Test message!")
	}
}

func BenchmarkInfoData(b *testing.B) {
	Redirect(ioutil.Discard)
	for i := 0; i < b.N; i++ {
		Info("Test message!", "foo", "bar", "i", 42)
	}
}

func BenchmarkWarning(b *testing.B) {
	Redirect(ioutil.Discard)
	for i := 0; i < b.N; i++ {
		Warning("Test message!")
	}
}

func BenchmarkError(b *testing.B) {
	Redirect(ioutil.Discard)
	err := errors.New("bench error")
	for i := 0; i < b.N; i++ {
		Error(err)
	}
}

func BenchmarkEvent(b *testing.B) {
	Redirect(ioutil.Discard)
	for i := 0; i < b.N; i++ {
		Event("bench_event")
	}
}

func BenchmarkEvents(b *testing.B) {
	Redirect(ioutil.Discard)
	for i := 0; i < b.N; i++ {
		Events("bench_event", 17)
	}
}

func BenchmarkTiming(b *testing.B) {
	Redirect(ioutil.Discard)
	for i := 0; i < b.N; i++ {
		NewTiming().Say("timing")
	}
}

func BenchmarkGauge(b *testing.B) {
	Redirect(ioutil.Discard)
	for i := 0; i < b.N; i++ {
		Gauge("bench_event", 17.06)
	}
}

func BenchmarkData2(b *testing.B) {
	Redirect(ioutil.Discard)
	for i := 0; i < b.N; i++ {
		Info("Test message!", "a", "b", "i", 57)
	}
}

func BenchmarkData3(b *testing.B) {
	Redirect(ioutil.Discard)
	for i := 0; i < b.N; i++ {
		Info("Test message!", "a", "b", "i", 57, "d", true)
	}
}

func BenchmarkData4(b *testing.B) {
	Redirect(ioutil.Discard)
	for i := 0; i < b.N; i++ {
		Info("Test message!", "a", "b", "i", 57, "d", true, "e", "lol")
	}
}

func BenchmarkData5(b *testing.B) {
	Redirect(ioutil.Discard)
	for i := 0; i < b.N; i++ {
		Info("Test message!", "a", "b", "i", 57, "d", true, "e", "lol", "j", 45)
	}
}
