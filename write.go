package say

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strconv"
	"unicode/utf8"
)

// Global buffer. If there are performance issues with highly-concurrent
// applications, use a sync.Pool instead.
var buf = make([]byte, 0, 1024)

func writeInt(i int64) {
	buf = strconv.AppendInt(buf, i, 10)
}

func writeUint(i uint64) {
	buf = strconv.AppendUint(buf, i, 10)
}

func writeFloat64(f float64) {
	buf = strconv.AppendFloat(buf, f, 'g', -1, 64)
}

func writeFloat32(f float32) {
	buf = strconv.AppendFloat(buf, float64(f), 'g', -1, 32)
}

func writeByte(b byte) {
	buf = append(buf, b)
}

func writeString(s string) {
	buf = append(buf, s...)
}

func writeBool(b bool) {
	if b {
		writeString("true")
		return
	}
	writeString("false")
}

func writeEscapeByte(b byte) {
	switch b {
	case '\n':
		writeString(" \n" + prefixBlank)
	case '|':
		if len(buf) > 0 && buf[len(buf)-1] == '\t' {
			buf[len(buf)-1] = ' '
		}
		writeByte('|')
	default:
		writeByte(b)
	}
}

func writeEscapeBytes(p []byte) {
	for _, b := range p {
		writeEscapeByte(b)
	}
}

func writeEscapeString(s string) {
	for i := 0; i < len(s); i++ {
		writeEscapeByte(s[i])
	}
}

func writeValue(v interface{}) {
	switch t := v.(type) {
	case string:
		writeEscapeString(t)
	case error:
		writeEscapeString(t.Error())
	case fmt.Stringer:
		writeEscapeString(t.String())
	case int:
		writeInt(int64(t))
	case int64:
		writeInt(t)
	case uint64:
		writeUint(t)
	case int32:
		writeInt(int64(t))
	case uint32:
		writeUint(uint64(t))
	case int16:
		writeInt(int64(t))
	case uint16:
		writeUint(uint64(t))
	case int8:
		writeInt(int64(t))
	case uint8:
		writeUint(uint64(t))
	case float64:
		writeFloat64(t)
	case float32:
		writeFloat32(t)
	default:
		writeInterface(v)
	}
}

type buffWriter struct{}

func (buffWriter) Write(p []byte) (int, error) {
	writeEscapeBytes(p)
	return len(p), nil
}

var buffW buffWriter

func writeInterface(v interface{}) {
	fmt.Fprint(buffW, v)
}

func writeDataValue(v interface{}) bool {
	switch t := v.(type) {
	case string:
		writeQuoteString(t)
	case error:
		writeQuoteString(t.Error())
	case fmt.Stringer:
		writeQuoteString(t.String())
	case func() string:
		writeQuoteString(t())
	case Hook:
		if v := t(); v != nil {
			return writeDataValue(v)
		}
		return false
	case int:
		writeInt(int64(t))
	case uint:
		writeUint(uint64(t))
	case int64:
		writeInt(t)
	case uint64:
		writeUint(t)
	case int32:
		writeInt(int64(t))
	case uint32:
		writeUint(uint64(t))
	case int16:
		writeInt(int64(t))
	case uint16:
		writeUint(uint64(t))
	case int8:
		writeInt(int64(t))
	case uint8:
		writeUint(uint64(t))
	case bool:
		writeBool(t)
	case float64:
		writeFloat64(t)
	case float32:
		writeFloat32(t)
	default:
		writeQuoteString(fmt.Sprint(v))
	}
	return true
}

const (
	quote    = '"'
	lowerhex = "0123456789abcdef"
)

// A slightly adapted version of strconv.quoteWith from the standard library.
func writeQuoteString(s string) {
	var runeTmp [utf8.UTFMax]byte
	buf = append(buf, quote)
	for width := 0; len(s) > 0; s = s[width:] {
		r := rune(s[0])
		width = 1
		if r >= utf8.RuneSelf {
			r, width = utf8.DecodeRuneInString(s)
		}
		if width == 1 && r == utf8.RuneError {
			buf = append(buf, `\x`...)
			buf = append(buf, lowerhex[s[0]>>4])
			buf = append(buf, lowerhex[s[0]&0xF])
			continue
		}
		if r == rune(quote) || r == '\\' { // always backslashed
			buf = append(buf, '\\')
			buf = append(buf, byte(r))
			continue
		}
		if strconv.IsPrint(r) {
			n := utf8.EncodeRune(runeTmp[:], r)
			buf = append(buf, runeTmp[:n]...)
			continue
		}
		switch r {
		case '\a':
			buf = append(buf, `\a`...)
		case '\b':
			buf = append(buf, `\b`...)
		case '\f':
			buf = append(buf, `\f`...)
		case '\n':
			buf = append(buf, `\n`...)
		case '\r':
			buf = append(buf, `\r`...)
		case '\t':
			buf = append(buf, `\t`...)
		case '\v':
			buf = append(buf, `\v`...)
		default:
			switch {
			case r < ' ':
				buf = append(buf, `\x`...)
				buf = append(buf, lowerhex[s[0]>>4])
				buf = append(buf, lowerhex[s[0]&0xF])
			case r > utf8.MaxRune:
				r = 0xFFFD
				fallthrough
			case r < 0x10000:
				buf = append(buf, `\u`...)
				for s := 12; s >= 0; s -= 4 {
					buf = append(buf, lowerhex[r>>uint(s)&0xF])
				}
			default:
				buf = append(buf, `\U`...)
				for s := 28; s >= 0; s -= 4 {
					buf = append(buf, lowerhex[r>>uint(s)&0xF])
				}
			}
		}
	}
	buf = append(buf, quote)
}

func (l *Logger) writeData(d []interface{}) {
	if len(l.data) == 0 && len(d) == 0 {
		return
	}

	start := len(buf)
	writeString("\t|")

	written1 := l.writeLoggerData()
	written2, err := writeParameterData(d)
	if !written1 && !written2 {
		buf = buf[:start]
	}
	if err != nil {
		writeByte('\n')
		l.sayError(err)
	}
}

func (l *Logger) writeLoggerData() bool {
	written := false
	for _, kv := range l.data {
		i := len(buf)
		writeByte(' ')
		writeString(kv.Key)
		writeByte('=')
		if ok := writeDataValue(kv.Value); ok {
			written = true
		} else {
			buf = buf[:i]
		}
	}
	return written
}

func writeParameterData(d []interface{}) (bool, error) {
	if len(d) == 0 {
		return false, nil
	}
	if len(d)%2 != 0 {
		return false, errOddNumArgs
	}

	written := false
	for i := 0; i < len(d)/2; i++ {
		// Write Key.
		start := len(buf)
		key, ok := d[2*i].(string)
		if !ok {
			return written, errKeyNotString
		}
		if err := isKeyValid(key); err != nil {
			return written, err
		}
		writeByte(' ')
		writeString(key)
		writeByte('=')

		// Write Value.
		if ok := writeDataValue(d[2*i+1]); ok {
			written = true
		} else {
			buf = buf[:start]
		}
	}
	return written, nil
}

func flush() {
	if err := flushTo(output); err != nil {
		writeString(prefixError)
		writeEscapeString(err.Error())
		_ = flushTo(os.Stderr)
	}
}

func flushTo(w io.Writer) error {
	trimSpaces()
	writeByte('\n')
	_, err := w.Write(buf)
	buf = buf[:0]
	return err
}

func trimSpaces() {
	for len(buf) > prefixLen && buf[len(buf)-1] == ' ' {
		buf = buf[:len(buf)-1]
	}
	if buf[len(buf)-1] == '\n' {
		// Oops we erased too much. Insert a blank prefix.
		writeString(prefixBlank)
	}
}

const maxStackSize = 4000

var stackTraces = make([]byte, maxStackSize)

func writeStackTrace(skip int) {
	n := runtimeStack(stackTraces, false)
	var t []byte
	if n < maxStackSize {
		t = stackTraces[:n-1] // Remove the last newline
	} else {
		t = stackTraces
		t[n-3] = '.'
		t[n-2] = '.'
		t[n-1] = '.'
	}

	for i := 0; i < 2*skip+3; i++ {
		n := bytes.IndexByte(t, '\n')
		if n == -1 {
			return
		}
		t = t[n+1:]
	}

	writeEscapeString("\n\n")
	writeEscapeBytes(t)
}
