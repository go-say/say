package say

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
)

// A Message represents a log line or a metric.
type Message struct {
	Type    Type
	Content string
	Data    Data
}

// Key returns the key of an EVENT, VALUE or GAUGE message.
func (m *Message) Key() string {
	i := strings.IndexByte(m.Content, ':')
	if i == -1 {
		return m.Content
	}
	return m.Content[:i]
}

// Value returns the value of an EVENT, VALUE or GAUGE message.
func (m *Message) Value() string {
	i := strings.IndexByte(m.Content, ':')
	if i == -1 {
		return ""
	}
	return m.Content[i+1:]
}

// Int returns the value as an integer. If the value is not an integer, ok is
// false. If the value is a duration in milliseconds, return the number of
// milliseconds. It returns 1 if the message is an EVENT without an increment.
func (m *Message) Int() (n int, ok bool) {
	v := m.Value()
	if v == "" {
		if m.Type == TypeEvent {
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
		if m.Type == TypeEvent {
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
	if m.Type != TypeError && m.Type != TypeFatal {
		return ""
	}

	i := strings.LastIndex(m.Content, "\n\n")
	if i == -1 {
		return m.Content
	}
	return m.Content[:i]
}

// StackTrace returns the stack trace of an ERROR or FATAL message.
func (m *Message) StackTrace() string {
	i := strings.LastIndex(m.Content, "\n\n")
	if i == -1 {
		return ""
	}
	return m.Content[i+2:]
}

// DataString returns the raw data string associated with the message.
// func (m *Message) DataString() string { return m.rawData }

// WriteTo writes the Message to w.
func (m *Message) WriteTo(w io.Writer) (int64, error) {
	t := now()
	buf := getBuffer()

	// Print the timestamp.
	buf.appendDigits(t.Year(), 4)
	buf.appendByte('-')
	buf.appendDigits(int(t.Month()), 2)
	buf.appendByte('-')
	buf.appendDigits(t.Day(), 2)
	buf.appendByte(' ')
	buf.appendDigits(t.Hour(), 2)
	buf.appendByte(':')
	buf.appendDigits(t.Minute(), 2)
	buf.appendByte(':')
	buf.appendDigits(t.Second(), 2)
	buf.appendByte('.')
	buf.appendDigits(t.Nanosecond()/int(time.Millisecond), 3)

	// Print the message.
	buf.appendByte(' ')
	buf.appendString(string(m.Type))
	buf.appendByte(' ')
	buf.appendString(m.Content)
	if len(m.Data) > 0 {
		buf.appendData(m.Data)
	}
	buf.appendByte('\n')

	n, err := w.Write(buf.buf)
	putBuffer(buf)
	return int64(n), err
}

// WriteJSONTo writes the JSON-encoded form of the Message to w.
func (m *Message) WriteJSONTo(w io.Writer) (int, error) {
	buf := getBuffer()

	buf.appendString(`{"timestamp": "`)
	buf.appendString(now().Format(time.RFC3339Nano))
	buf.appendString(`", "type": "`)
	buf.appendString(strings.TrimSuffix(string(m.Type), " "))
	buf.appendString(`", "content": `)
	buf.appendQuote(m.Content)

	data := m.Data
	if len(data) > 0 {
		start := len(buf.buf)
		written := false

		for i, kv := range data {
			if m.skipKey(data, i) {
				continue
			}
			buf.appendString(", ")
			buf.appendQuote(kv.Key)
			buf.appendString(": ")
			buf.appendDataValue(kv.Value)
			written = true
		}

		if !written {
			buf.buf = buf.buf[:start]
		}
	}
	buf.appendString("}\n")

	n, err := w.Write(buf.buf)
	putBuffer(buf)
	return n, err
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

var msgPool = sync.Pool{
	New: func() interface{} {
		return new(Message)
	},
}

func getMessage() *Message {
	return msgPool.Get().(*Message)
}

func putMessage(msg *Message) {
	msg.Data = msg.Data[:0]
	msgPool.Put(msg)
}
