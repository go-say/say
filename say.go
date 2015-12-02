package say

import (
	"errors"
	"io"
	"io/ioutil"
	"log"
	"os"
	"runtime"
	"sync"
	"time"
)

const (
	prefixEvent   = "EVENT "
	prefixValue   = "VALUE "
	prefixGauge   = "GAUGE "
	prefixDebug   = "DEBUG "
	prefixInfo    = "INFO  "
	prefixWarning = "WARN  "
	prefixError   = "ERROR "
	prefixFatal   = "FATAL "
	prefixInit    = "INIT  "
	prefixBlank   = "      "
	prefixLen     = 6
)

var (
	output          = io.Writer(os.Stdout)
	mu              sync.RWMutex
	defaultLogger   = &Logger{skipStackFrames: 1}
	errOddNumArgs   = errors.New("say: odd number of data arguments")
	errKeyNotString = errors.New("say: keys must be string")
	errKeyEmpty     = errors.New("say: key is empty")
	errKeyInvalid   = errors.New("say: keys must not contain ':', '=', tabs or  newlines")
)

// Logger is the object that prints messages.
type Logger struct {
	skipStackFrames int
	data            []*kvPair
}

type kvPair struct {
	Key   string
	Value interface{}
}

// NewLogger creates a new Logger from a parent Logger. The new Logger have the
// same Data and SkipStackFrames values than the parent Logger.
func (l *Logger) NewLogger() *Logger {
	log := new(Logger)
	mu.RLock()
	log.skipStackFrames = l.skipStackFrames
	log.data = l.data
	mu.RUnlock()
	return log
}

// NewLogger creates a new Logger that have the same Data and SkipStackFrames
// values than the package-level Logger.
func NewLogger() *Logger {
	return defaultLogger.NewLogger()
}

// SetData sets a key-value pair that will be printed along with all messages
// sent with this Logger.
func (l *Logger) SetData(data ...interface{}) {
	mu.Lock()
	if len(data)%2 != 0 {
		l.sayError(errOddNumArgs)
		flush()
		mu.Unlock()
		return
	}

	l.data = make([]*kvPair, 0, len(data)/2)
	for k := 0; k < len(data)/2; k++ {
		key, ok := data[2*k].(string)
		if !ok {
			l.sayError(errKeyNotString)
			flush()
			mu.Unlock()
			return
		}
		if err := isKeyValid(key); err != nil {
			l.sayError(err)
			flush()
			mu.Unlock()
			return
		}
		l.data = append(l.data, &kvPair{
			Key:   key,
			Value: data[2*k+1],
		})
	}
	mu.Unlock()
}

// SetData sets a key-value pair that will be printed along with all messages
// sent with the package-level functions.
func SetData(data ...interface{}) {
	defaultLogger.SetData(data...)
}

// AddData adds a key-value pair that will be printed along with all messages
// sent with this Logger.
func (l *Logger) AddData(key string, value interface{}) {
	mu.Lock()
	if err := isKeyValid(key); err != nil {
		l.sayError(err)
		flush()
	} else {
		l.data = append(l.data, &kvPair{Key: key, Value: value})
	}
	mu.Unlock()
}

// AddData adds a key-value pair that will be printed along with all messages
// sent with the package-level functions.
func AddData(key string, value interface{}) {
	defaultLogger.AddData(key, value)
}

func isKeyValid(key string) error {
	if key == "" {
		return errKeyEmpty
	}
	for i := 0; i < len(key); i++ {
		switch key[i] {
		case ':', '=', '\t', '\n':
			return errKeyInvalid
		}
	}
	return nil
}

// SkipStackFrames sets the number of stack frames to skip in the Error and
// Fatal methods. It is 0 by default.
//
// A value of -1 disable printing the stack traces with this Logger.
func (l *Logger) SkipStackFrames(skip int) {
	mu.Lock()
	l.skipStackFrames = skip
	mu.Unlock()
}

// SkipStackFrames sets the number of stack frames to skip in the Error and
// Fatal package-level functions. It is 0 by default.
//
// A value of -1 disable printing the stack traces with the package-level
// functions.
func SkipStackFrames(skip int) {
	mu.Lock()
	if skip == -1 {
		defaultLogger.skipStackFrames = -1
	} else {
		defaultLogger.skipStackFrames = skip + 1
	}
	mu.Unlock()
}

// Event prints an EVENT message. Use it to track the occurence of a particular
// event (e.g. a user signs up, a database query fails).
func (l *Logger) Event(name string, data ...interface{}) {
	mu.Lock()
	if err := isKeyValid(name); err != nil {
		l.sayError(err)
		flush()
		mu.Unlock()
		return
	}

	writeString(prefixEvent)
	writeString(name)
	l.writeData(data)
	flush()
	mu.Unlock()
}

// Event prints an EVENT message. Use it to track the occurence of a particular
// event (e.g. a user signs up, a database query fails).
func Event(name string, data ...interface{}) {
	defaultLogger.Event(name, data...)
}

// Events prints an EVENT message with an increment value. Use it to track the
// occurence of a batch of events (e.g. how many new files were uploaded).
func (l *Logger) Events(name string, incr int, data ...interface{}) {
	mu.Lock()
	if err := isKeyValid(name); err != nil {
		l.sayError(err)
		flush()
		mu.Unlock()
		return
	}

	writeString(prefixEvent)
	writeString(name)
	writeByte(':')
	writeInt(int64(incr))
	l.writeData(data)
	flush()
	mu.Unlock()
}

// Events prints an EVENT message with an increment value. Use it to track the
// occurence of a batch of events (e.g. how many new files were uploaded).
func Events(name string, incr int, data ...interface{}) {
	defaultLogger.Events(name, incr, data...)
}

// Value prints a VALUE message. Use it to measure a value associated with a
// particular event (e.g. number of items returned by a search).
func (l *Logger) Value(name string, value interface{}, data ...interface{}) {
	l.keyValue(prefixValue, name, value, data)
}

// Value prints a VALUE message. Use it to measure a value associated with a
// particular event (e.g. number of items returned by a search).
func Value(name string, value interface{}, data ...interface{}) {
	defaultLogger.Value(name, value, data...)
}

// A Timing helps printing a duration.
type Timing struct {
	l     *Logger
	start time.Time
}

// NewTiming returns a new Timing with the same associated data than the Logger.
func (l *Logger) NewTiming() Timing {
	return Timing{l: l, start: now()}
}

// NewTiming returns a new Timing with the package-level data.
func NewTiming() Timing {
	return defaultLogger.NewTiming()
}

// Say prints a VALUE message with the duration in milliseconds since the Timing
// has been created. Use it to measure a duration value (e.g. database query
// duration, webservice call duration).
func (t Timing) Say(name string, data ...interface{}) {
	n := int64(t.Get() / time.Millisecond)
	mu.Lock()
	if err := isKeyValid(name); err != nil {
		t.l.sayError(err)
		flush()
		mu.Unlock()
		return
	}

	writeString(prefixValue)
	writeString(name)
	writeByte(':')
	writeInt(n)
	writeString("ms")
	t.l.writeData(data)
	flush()
	mu.Unlock()
}

// Get returns the duration since the Timing has been created.
func (t Timing) Get() time.Duration {
	return now().Sub(t.start)
}

// Gauge prints a GAUGE message. Use it to capture the current value of
// something that changes over time (e.g. number of active goroutines, number of
// connected users)
func (l *Logger) Gauge(name string, value interface{}, data ...interface{}) {
	l.keyValue(prefixGauge, name, value, data)
}

// Gauge prints a GAUGE message. Use it to capture the current value of
// something that changes over time (e.g. number of active goroutines, number of
// connected users)
func Gauge(name string, value interface{}, data ...interface{}) {
	defaultLogger.Gauge(name, value, data...)
}

func (l *Logger) keyValue(prefix, name string, value interface{}, data []interface{}) {
	mu.Lock()
	if err := isKeyValid(name); err != nil {
		l.sayError(err)
		flush()
		mu.Unlock()
		return
	}

	writeString(prefix)
	writeString(name)
	writeByte(':')
	writeValue(value)
	l.writeData(data)
	flush()
	mu.Unlock()
}

// Debug prints a DEBUG message only if the debug mode is on.
func (l *Logger) Debug(msg string, data ...interface{}) {
	if !debug {
		return
	}
	l.message(prefixDebug, msg, data)
}

// Debug prints a DEBUG message only if the debug mode is on.
func Debug(msg string, data ...interface{}) {
	defaultLogger.Debug(msg, data...)
}

// Info prints an INFO message.
func (l *Logger) Info(msg string, data ...interface{}) {
	l.message(prefixInfo, msg, data)
}

// Info prints an INFO message.
func Info(msg string, data ...interface{}) {
	defaultLogger.Info(msg, data...)
}

func (l *Logger) message(prefix, msg string, data []interface{}) {
	mu.Lock()
	writeString(prefix)
	writeEscapeString(msg)
	l.writeData(data)
	flush()
	mu.Unlock()
}

// Warning prints a WARNING message.
func (l *Logger) Warning(v interface{}, data ...interface{}) {
	mu.Lock()
	writeString(prefixWarning)
	writeValue(v)
	l.writeData(data)
	flush()
	mu.Unlock()
}

// Warning prints a WARNING message.
func Warning(v interface{}, data ...interface{}) {
	defaultLogger.Warning(v, data...)
}

// Error prints an ERROR message with the stack trace.
//
// If v is nil, nothing is printed. If v is a func() error, then Error run v
// and prints an error only if v return a non-nil error.
func (l *Logger) Error(v interface{}, data ...interface{}) {
	l.error(prefixError, v, data...)
}

// Error prints an ERROR message with the stack trace.
//
// If v is nil, nothing is printed. If v is a func() error, then Error run v
// and prints an error only if v return a non-nil error.
func Error(v interface{}, data ...interface{}) {
	defaultLogger.Error(v, data...)
}

// Fatal prints a FATAL message with the stack trace.
//
// If v is nil, nothing is printed. If v is a func() error, then Fatal runs v
// and prints an error only if v returns a non-nil error.
func (l *Logger) Fatal(v interface{}, data ...interface{}) {
	l.error(prefixFatal, v, data...)
}

// Fatal prints a FATAL message with the stack trace.
//
// If v is nil, nothing is printed. If v is a func() error, then Fatal runs v
// and prints an error only if v returns a non-nil error.
func Fatal(v interface{}, data ...interface{}) {
	defaultLogger.Fatal(v, data...)
}

func (l *Logger) error(prefix string, v interface{}, data ...interface{}) {
	if v == nil {
		return
	}
	if f, ok := v.(func() error); ok {
		v = f()
		if v == nil {
			return
		}
	}

	mu.Lock()
	writeString(prefix)
	writeValue(v)
	if l.skipStackFrames >= 0 {
		writeStackTrace(l.skipStackFrames + 2)
	}
	l.writeData(data)
	flush()
	mu.Unlock()
}

// Init prints an INIT message. It is usually used at the start of a program to
// setup the listening program (e.g. choose which file to save logs, configure
// the metrics namespace).
func (l *Logger) Init(name string, data ...interface{}) *Logger {
	l.message(prefixInit, name, data)
	return l
}

// Init prints an INIT message. It is usually used at the start of a program to
// setup the listening program (e.g. choose which file to save logs, configure
// the metrics namespace).
func Init(name string, data ...interface{}) *Logger {
	return defaultLogger.Init(name, data...)
}

// CaptureStandardLog captures the log lines coming from the log package of the
// standard library. Captured lines are output with an INFO level.
func (l *Logger) CaptureStandardLog() {
	log.SetFlags(0)
	log.SetOutput(stdLogWriter{l})
}

// CaptureStandardLog captures the log lines coming from the log package of the
// standard library. Captured lines are output with an INFO level.
func CaptureStandardLog() {
	defaultLogger.CaptureStandardLog()
}

type stdLogWriter struct {
	*Logger
}

func (w stdLogWriter) Write(p []byte) (int, error) {
	w.Info(string(p[:len(p)-1])) // Remove the trailing newline.
	return len(p), nil
}

// Redirect redirects the output to the given writer. It returns the writer
// where outputs were previously redirected to.
func Redirect(w io.Writer) io.Writer {
	mu.Lock()
	oldW := output
	output = w
	mu.Unlock()
	return oldW
}

// Mute disables any output. It is the same as Redirect(ioutil.Discard).
func Mute() io.Writer {
	return Redirect(ioutil.Discard)
}

// CapturePanic captures panic values and prints them as FATAL messages.
func (l *Logger) CapturePanic() {
	if r := recover(); r != nil {
		l.Fatal(r)
		exit(2)
	}
}

// CapturePanic captures panic values and prints them as FATAL messages.
func CapturePanic() {
	if r := recover(); r != nil {
		defaultLogger.Fatal(r)
		exit(2)
	}
}

var debug bool

// SetDebug sets whether Say is in debug mode. The debug mode is off by default.
//
// This function must not be called concurrently with the other functions of
// this package.
func SetDebug(b bool) {
	debug = b
}

// A Hook is a function used to provide dynamic Data values.
type Hook func() interface{}

// DebugHook allows printing a key-value pairs only when Say is in debug mode.
func DebugHook(v interface{}) Hook {
	return Hook(func() interface{} {
		if debug {
			return v
		}
		return nil
	})
}

// TimeHook prints the current time.
func TimeHook(format string) Hook {
	return Hook(func() interface{} {
		return now().Format(format)
	})
}

func (l *Logger) sayError(err error) {
	writeString(prefixError)
	writeString(err.Error())
	if l.skipStackFrames >= 0 {
		writeStackTrace(l.skipStackFrames + 2)
	}
}

// Stubbed out for testing.
var (
	now          = time.Now
	runtimeStack = runtime.Stack
	exit         = os.Exit
)
