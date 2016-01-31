package say_test

import (
	"log"
	"os"
	"regexp"
	"runtime"

	"gopkg.in/say.v0"
)

// A trick to allow examples to correctly capture stdout.
var ew = exampleWriter{w: &os.Stdout}

type exampleWriter struct {
	w **os.File
}

var findDuration = regexp.MustCompile(`\d+ms`)

func (w exampleWriter) Write(p []byte) (int, error) {
	// Replace durations.
	p = findDuration.ReplaceAll(p, []byte("17ms"))

	if _, err := (*w.w).Write(p); err != nil {
		return 0, err
	}
	return len(p), nil
}

func init() {
	say.Redirect(ew)
	say.SkipStackFrames(-1)
}

func Example() {
	// Capture panics as FATAL.
	defer say.CapturePanic()

	say.Info("Getting list of users...")
	say.Value("user_found", 42)
	// Output:
	// INFO  Getting list of users...
	// VALUE user_found:42
}

func ExampleNewLogger() {
	say.SetData("weather", "sunny")
	log := say.NewLogger()
	log.Info("hello") // INFO  hello	| weather="sunny"
}

func ExampleLogger_NewLogger() {
	log := new(say.Logger) // Create a clean Logger.
	log.SetData("id", 5)
	log2 := log.NewLogger() // log2 inherits its parent settings.
	log2.AddData("age", 53)
	log2.Info("hello")
	// Output:
	// INFO  hello	| id=5 age=53
}

func ExampleLogger_SetData() {
	log := new(say.Logger)
	log.SetData("id", 5, "foo", "bar")
	log.Info("hello")
	// Output:
	// INFO  hello	| id=5 foo="bar"
}

func ExampleLogger_AddData() {
	log := new(say.Logger)
	log.AddData("id", 5)
	log.Info("hello")
	log.AddData("foo", "bar")
	log.Info("dear")
	// Output:
	// INFO  hello	| id=5
	// INFO  dear	| id=5 foo="bar"
}

func ExampleLogger_SkipStackFrames() {
	log := new(say.Logger)
	log.SkipStackFrames(-1) // Disable stack traces.
	log.Error("Oops")
	// Output:
	// ERROR Oops
}

func ExampleLogger_Event() {
	log := new(say.Logger)
	log.Event("new_user", "id", 7654)
	// Output:
	// EVENT new_user	| id=7654
}

func ExampleEvent() {
	say.Event("new_user", "id", 7654)
	// Output:
	// EVENT new_user	| id=7654
}

func ExampleLogger_Events() {
	log := new(say.Logger)
	log.Events("file_uploaded", 3)
	// Output:
	// EVENT file_uploaded:3
}

func ExampleEvents() {
	say.Events("file_uploaded", 3)
	// Output:
	// EVENT file_uploaded:3
}

func ExampleLogger_Value() {
	log := new(say.Logger)
	log.Value("search_items", 117)
	// Output:
	// VALUE search_items:117
}

func ExampleValue() {
	say.Value("search_items", 117)
	// Output:
	// VALUE search_items:117
}

func ExampleTiming_Say() {
	t := say.NewTiming()
	// Do some stuff.
	t.Say("duration")
	// Output:
	// VALUE duration:17ms
}

func ExampleLogger_Gauge() {
	log := new(say.Logger)
	log.Gauge("connected_users", 73)
	// Output:
	// GAUGE connected_users:73
}

func ExampleGauge() {
	say.Gauge("connected_users", 73)
	// Output:
	// GAUGE connected_users:73
}

func ExampleLogger_Debug() {
	log := new(say.Logger)
	say.SetDebug(false)
	log.Debug("foo")
	say.SetDebug(true)
	log.Debug("bar")
	// Output:
	// DEBUG bar
}

func ExampleDebug() {
	say.SetDebug(false)
	say.Debug("foo")
	say.SetDebug(true)
	say.Debug("bar")
	// Output:
	// DEBUG bar
}

func ExampleLogger_Info() {
	log := new(say.Logger)
	log.Info("Connecting to server...", "ip", "127.0.0.1")
	// Output:
	// INFO  Connecting to server...	| ip="127.0.0.1"
}

func ExampleInfo() {
	say.Info("Connecting to server...", "ip", "127.0.0.1")
	// Output:
	// INFO  Connecting to server...	| ip="127.0.0.1"
}

func ExampleLogger_Warning() {
	log := new(say.Logger)
	log.Warning("Could not connect to host", "host", "example.com")
	// Output:
	// WARN  Could not connect to host	| host="example.com"
}

func ExampleWarning() {
	say.Warning("Could not connect to host", "host", "example.com")
	// Output:
	// WARN  Could not connect to host	| host="example.com"
}

func ExampleLogger_Error() {
	f, err := os.Open("foo.txt")
	log := new(say.Logger)
	log.Error(err)           // Print an error only if err is not nil.
	defer log.Error(f.Close) // Call Close and print the error only if not nil.
}

func ExampleError() {
	f, err := os.Open("foo.txt")
	say.Error(err)           // Print an error only if err is not nil.
	defer say.Error(f.Close) // Call Close and print the error only if not nil.
}

func ExampleLogger_Fatal() {
	f, err := os.Open("foo.txt")
	log := new(say.Logger)
	log.Fatal(err)           // Print an error only if err is not nil.
	defer log.Fatal(f.Close) // Call Close and print the error only if not nil.
}

func ExampleFatal() {
	f, err := os.Open("foo.txt")
	say.Fatal(err)           // Print an error only if err is not nil.
	defer say.Fatal(f.Close) // Call Close and print the error only if not nil.
}

func ExampleLogger_CapturePanic() {
	log := new(say.Logger)
	defer log.CapturePanic()

	panic("oops!") // The panic message will be printed with a FATAL severity.
}

func ExampleCapturePanic() {
	defer say.CapturePanic()

	panic("oops!") // The panic message will be printed with a FATAL severity.
}

func ExampleLogger_CaptureStandardLog() {
	l := new(say.Logger)
	l.CaptureStandardLog()
	log.Print("Hello from the standard library!")
	// Output:
	// INFO  Hello from the standard library!
}

func ExampleCaptureStandardLog() {
	say.CaptureStandardLog()
	log.Print("Hello from the standard library!")
	// Output:
	// INFO  Hello from the standard library!
}

func ExampleHook() {
	goroutinesHook := say.Hook(func() interface{} {
		return runtime.NumGoroutine
	})
	// Print the current number of goroutines with each message.
	say.SetData("num_goroutine", goroutinesHook)
}

func ExampleDebugHook() {
	query := "SELECT * FROM users WHERE id = ?"
	say.SetDebug(true)
	say.Event("db.get_user", "query", say.DebugHook(query)) // Print the query.
	say.SetDebug(false)
	say.Event("db.get_user", "query", say.DebugHook(query)) // Omit the query.
	// Output:
	// EVENT db.get_user	| query="SELECT * FROM users WHERE id = ?"
	// EVENT db.get_user
}

func ExampleTimeHook() {
	// Print the current timestamp with each message.
	say.SetData("num_goroutine", say.TimeHook("2006-01-02 15:04:05"))
}
