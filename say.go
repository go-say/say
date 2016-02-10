package say

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"runtime"
	"sync"
	"time"
)

var (
	mu              sync.RWMutex
	defaultLogger   = &Logger{skipStackFrames: 1}
	errOddNumArgs   = errors.New("say: odd number of data arguments")
	errKeyNotString = errors.New("say: keys must be string")
	errKeyEmpty     = errors.New("say: key is empty")
	errKeyInvalid   = errors.New("say: keys must not contain ':', '=', tabs or newlines")
)

// Logger is the object that prints messages.
type Logger struct {
	skipStackFrames int
	data            Data
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
	if skip != -1 {
		skip++
	}
	defaultLogger.SkipStackFrames(skip)
}

// Event prints an EVENT message. Use it to track the occurence of a particular
// event (e.g. a user signs up, a database query fails).
func (l *Logger) Event(name string, data ...interface{}) {
	if err := isKeyValid(name); err != nil {
		l.sendError(err, 1)
		return
	}
	l.send(TypeEvent, name, data)
}

// Eventf prints a formatted EVENT message. Use it to track the occurence of a particular
// event (e.g. a user signs up, a database query fails).
func (l *Logger) Eventf(name string, data ...interface{}) {
	if err := isKeyValid(name); err != nil {
		l.sendError(err, 1)
		return
	}
	l.send(TypeEvent, fmt.Sprintf(name, data...), []interface{})
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

// Event prints an EVENT message. Use it to track the occurence of a particular
// event (e.g. a user signs up, a database query fails).
func Event(name string, data ...interface{}) {
	defaultLogger.Event(name, data...)
}

// Eventf prints a formatted EVENT message. Use it to track the occurence of a particular
// event (e.g. a user signs up, a database query fails).
func Eventf(name string, data ...interface{}) {
	defaultLogger.Eventf(name, data...)
}

// Events prints an EVENT message with an increment value. Use it to track the
// occurence of a batch of events (e.g. how many new files were uploaded).
func (l *Logger) Events(name string, incr int, data ...interface{}) {
	if err := isKeyValid(name); err != nil {
		l.sendError(err, 1)
		return
	}

	buf := getBuffer()
	buf.appendString(name)
	buf.appendByte(':')
	buf.appendInt(int64(incr))
	l.send(TypeEvent, buf.String(), data)
}

// Events prints a formatted EVENT message with an increment value. Use it to track the
// occurence of a batch of events (e.g. how many new files were uploaded).
func (l *Logger) Eventsf(name string, incr int, data ...interface{}) {
	if err := isKeyValid(name); err != nil {
		l.sendError(err, 1)
		return
	}

	buf := getBuffer()
	buf.appendString(fmt.Sprintf(name, data...))
	buf.appendByte(':')
	buf.appendInt(int64(incr))
	l.send(TypeEvent, buf.String(), []interface{}{})
}

// Events prints an EVENT message with an increment value. Use it to track the
// occurence of a batch of events (e.g. how many new files were uploaded).
func Events(name string, incr int, data ...interface{}) {
	defaultLogger.Events(name, incr, data...)
}

// Eventsf prints a formatted EVENT message with an increment value. Use it to track the
// occurence of a batch of events (e.g. how many new files were uploaded).
func Eventsf(name string, incr int, data ...interface{}) {
	defaultLogger.Eventsf(name, incr, data...)
}

// Value prints a VALUE message. Use it to measure a value associated with a
// particular event (e.g. number of items returned by a search).
func (l *Logger) Value(name string, value interface{}, data ...interface{}) {
	l.keyValue(TypeValue, name, value, data)
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
	if err := isKeyValid(name); err != nil {
		t.l.sendError(err, 1)
		return
	}

	buf := getBuffer()
	buf.appendString(name)
	buf.appendByte(':')
	buf.appendInt(n)
	buf.appendString("ms")
	t.l.send(TypeValue, buf.String(), data)
}

// Get returns the duration since the Timing has been created.
func (t Timing) Get() time.Duration {
	return now().Sub(t.start)
}

// Gauge prints a GAUGE message. Use it to capture the current value of
// something that changes over time (e.g. number of active goroutines, number of
// connected users)
func (l *Logger) Gauge(name string, value interface{}, data ...interface{}) {
	l.keyValue(TypeGauge, name, value, data)
}

// Gauge prints a GAUGE message. Use it to capture the current value of
// something that changes over time (e.g. number of active goroutines, number of
// connected users)
func Gauge(name string, value interface{}, data ...interface{}) {
	defaultLogger.Gauge(name, value, data...)
}

func (l *Logger) keyValue(typ Type, name string, value interface{}, data []interface{}) {
	if err := isKeyValid(name); err != nil {
		l.sendError(err, 1)
		return
	}

	buf := getBuffer()
	buf.appendString(name)
	buf.appendByte(':')
	buf.appendValue(value)
	l.send(typ, buf.String(), data)
}

// Debug prints a DEBUG message only if the debug mode is on.
func (l *Logger) Debug(msg string, data ...interface{}) {
	if !debug {
		return
	}
	l.send(TypeDebug, msg, data)
}

// Debug prints a formatted DEBUG message only if the debug mode is on.
func (l *Logger) Debugf(msg string, data ...interface{}) {
	if !debug {
		return
	}
	l.send(TypeDebug, fmt.Sprintf(msg, data...), []interface{}{})
}

// Debug prints a DEBUG message only if the debug mode is on.
func Debug(msg string, data ...interface{}) {
	defaultLogger.Debug(msg, data...)
}

// Debugf prints a formatted DEBUG message only if the debug mode is on.
func Debugf(msg string, data ...interface{}) {
	defaultLogger.Debugf(msg, data...)
}

// Info prints an INFO message.
func (l *Logger) Info(msg string, data ...interface{}) {
	l.send(TypeInfo, msg, data)
}

// Info prints a formatted INFO message.
func (l *Logger) Infof(msg string, data ...interface{}) {
	l.send(TypeInfo, fmt.Sprintf(msg, data...), []interface{}{})
}

// Info prints an INFO message.
func Info(msg string, data ...interface{}) {
	defaultLogger.Info(msg, data...)
}

// Infof prints a formatted INFO message.
func Infof(msg string, data ...interface{}) {
	defaultLogger.Infof(msg, data...)
}

// Warning prints a WARNING message.
func (l *Logger) Warning(v interface{}, data ...interface{}) {
	buf := getBuffer()
	buf.appendValue(v)
	l.send(TypeWarning, buf.String(), data)
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
	l.error(TypeError, v, data, 1)
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
	l.error(TypeFatal, v, data, 1)
}

// Fatal prints a FATAL message with the stack trace.
//
// If v is nil, nothing is printed. If v is a func() error, then Fatal runs v
// and prints an error only if v returns a non-nil error.
func Fatal(v interface{}, data ...interface{}) {
	defaultLogger.Fatal(v, data...)
}

func (l *Logger) sendError(err error, skip int) {
	l.error(TypeError, err, nil, skip+1)
}

func (l *Logger) error(typ Type, v interface{}, data []interface{}, skip int) {
	if v == nil {
		return
	}
	if f, ok := v.(func() error); ok {
		v = f()
		if v == nil {
			return
		}
	}

	buf := getBuffer()
	buf.appendValue(v)

	// Lock instead of RLock because getStackTrace is not concurrent-safe.
	mu.Lock()
	if l.skipStackFrames >= 0 {
		st := getStackTrace(l.skipStackFrames + skip + 1)
		buf.appendString("\n\n")
		buf.appendBytes(st)
	}
	mu.Unlock()

	l.send(typ, buf.String(), data)
}

const maxStackSize = 4000

var stBuf = make([]byte, maxStackSize)

// Be careful, getStackTrace is not concurrent-safe.
func getStackTrace(skip int) []byte {
	n := runtimeStack(stBuf, false)
	var tmp []byte
	if n < maxStackSize {
		tmp = stBuf[:n-1] // Remove the last newline
	} else {
		tmp = stBuf
		tmp[n-3] = '.'
		tmp[n-2] = '.'
		tmp[n-1] = '.'
	}

	for i := 0; i < 2*skip+3; i++ {
		n := bytes.IndexByte(tmp, '\n')
		if n == -1 {
			break
		}
		tmp = tmp[n+1:]
	}

	return tmp
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

// Stubbed out for testing.
var (
	now          = time.Now
	runtimeStack = runtime.Stack
)
