package listen

import (
	"io"
	"strconv"
	"strings"
	"sync"
	"time"
)

// A Type represents a message type.
type Type string

// All the available message types.
const (
	TypeEvent   Type = "EVENT"
	TypeValue   Type = "VALUE"
	TypeGauge   Type = "GAUGE"
	TypeDebug   Type = "DEBUG"
	TypeInfo    Type = "INFO "
	TypeWarning Type = "WARN "
	TypeError   Type = "ERROR"
	TypeFatal   Type = "FATAL"
	TypeInit    Type = "INIT "
	typeLen          = 5
)

var (
	types = []Type{TypeEvent, TypeValue, TypeGauge, TypeDebug, TypeInfo,
		TypeWarning, TypeError, TypeFatal, TypeInit}
)

// A Message represents a Say message.
type Message struct {
	typ     Type
	content string
	rawData string
}

// Type returns the type of a message.
func (m *Message) Type() Type { return m.typ }

// Content returns the content of a message (i.e. the message without the prefix
// and without the associated data).
func (m *Message) Content() string { return m.content }

// Key returns the key of an EVENT, VALUE or GAUGE message.
func (m *Message) Key() string {
	i := strings.IndexByte(m.content, ':')
	if i == -1 {
		return m.content
	}
	return m.content[:i]
}

// Value returns the value of an EVENT, VALUE or GAUGE message.
func (m *Message) Value() string {
	i := strings.IndexByte(m.content, ':')
	if i == -1 {
		return ""
	}
	return m.content[i+1:]
}

// Int returns the value as an integer. If the value is not an integer, ok is
// false. If the value is a duration in milliseconds, return the number of
// milliseconds. It returns 1 if the message is an EVENT without an increment.
func (m *Message) Int() (n int, ok bool) {
	v := m.Value()
	if v == "" {
		if m.Type() == TypeEvent {
			return 1, true
		}
		return 0, false
	}
	if strings.HasSuffix(v, "ms") {
		v = v[:len(v)-2]
	}
	if i, err := strconv.Atoi(v); err == nil {
		return i, true
	}
	if f, err := strconv.ParseFloat(v, 64); err == nil {
		return int(f), true
	}
	return 0, false
}

// Float64 returns the value as an float64. If the value is not a float64, ok is
// false. If the value is a duration in milliseconds, return the number of
// milliseconds. It returns 1 if the message is an EVENT without an increment.
func (m *Message) Float64() (float64, bool) {
	v := m.Value()
	if v == "" {
		if m.Type() == TypeEvent {
			return 1, true
		}
		return 0, false
	}
	if strings.HasSuffix(v, "ms") {
		v = v[:len(v)-2]
	}
	if f, err := strconv.ParseFloat(v, 64); err == nil {
		return f, true
	}
	return 0, false
}

// Duration returns the duration of a VALUE message. If the value is not a
// duration, ok is false.
func (m *Message) Duration() (time.Duration, bool) {
	v := m.Value()
	if v == "" {
		return 0, false
	}
	d, err := time.ParseDuration(v)
	return d, err == nil
}

// Error returns the error message of an ERROR or FATAL message.
func (m *Message) Error() string {
	if m.Type() != TypeError && m.Type() != TypeFatal {
		return ""
	}

	i := strings.LastIndex(m.content, "\n\n")
	if i == -1 {
		return m.content
	}
	return m.content[:i]
}

// StackTrace returns the stack trace of an ERROR or FATAL message.
func (m *Message) StackTrace() string {
	i := strings.LastIndex(m.content, "\n\n")
	if i == -1 {
		return ""
	}
	return m.content[i+2:]
}

// DataString returns the raw data string associated with the message.
func (m *Message) DataString() string { return m.rawData }

// Write writes the Message to w.
func (m *Message) Write(w io.Writer) (int, error) {
	t := now()
	buf := getBuffer()
	defer putBuffer(buf)

	// Print the timestamp.
	buf = appendDigits(buf, t.Year(), 4)
	buf = append(buf, '-')
	buf = appendDigits(buf, int(t.Month()), 2)
	buf = append(buf, '-')
	buf = appendDigits(buf, t.Day(), 2)
	buf = append(buf, ' ')
	buf = appendDigits(buf, t.Hour(), 2)
	buf = append(buf, ':')
	buf = appendDigits(buf, t.Minute(), 2)
	buf = append(buf, ':')
	buf = appendDigits(buf, t.Second(), 2)
	buf = append(buf, '.')
	buf = appendDigits(buf, t.Nanosecond()/int(time.Millisecond), 3)

	// Print the message.
	buf = append(buf, ' ')
	buf = append(buf, m.typ...)
	buf = append(buf, ' ')
	buf = append(buf, m.content...)
	if len(m.rawData) > 0 {
		buf = append(buf, "\t| "...)
		buf = append(buf, m.rawData...)
	}
	buf = append(buf, '\n')

	return w.Write(buf)
}

func appendDigits(buf []byte, n, length int) []byte {
	for i := 0; i < length; i++ {
		buf = append(buf, '0')
	}
	for i := 0; i < length && n > 0; i++ {
		buf[len(buf)-i-1] = digits[n%10]
		n /= 10
	}
	return buf
}

const digits = "0123456789"

// WriteJSON writes the JSON-encoded form of the Message to w.
func (m *Message) WriteJSON(w io.Writer) (int, error) {
	buf := getBuffer()
	defer putBuffer(buf)

	buf = append(buf, `{"timestamp": "`...)
	buf = append(buf, now().Format(time.RFC3339Nano)...)
	buf = append(buf, `", "type": "`...)
	buf = append(buf, strings.TrimSuffix(string(m.typ), " ")...)
	buf = append(buf, `", "content": `...)
	buf = strconv.AppendQuote(buf, m.content)

	data := m.Data()
	if len(data) > 0 {
		start := len(buf)
		written := false

		for i, kv := range data {
			if m.skipKey(data, i) {
				continue
			}
			buf = append(buf, ", "...)
			buf = strconv.AppendQuote(buf, kv.Key)
			buf = append(buf, ": "...)
			buf = append(buf, kv.Value...)
			written = true
		}

		if !written {
			buf = buf[:start]
		}
	}
	buf = append(buf, "}\n"...)

	return w.Write(buf)
}

func (m *Message) skipKey(d Data, i int) bool {
	key := d[i].Key
	if key == "timestamp" || key == "type" || key == "content" {
		return true
	}
	for _, kv := range d[i+1:] {
		if key == kv.Key {
			return true
		}
	}
	return false
}

var bufPool = sync.Pool{
	New: func() interface{} {
		return make([]byte, 0, 200)
	},
}

func getBuffer() []byte {
	return bufPool.Get().([]byte)
}

func putBuffer(buf []byte) {
	bufPool.Put(buf)
}

// Stubbed out for testing.
var now = time.Now
