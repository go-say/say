package say

import (
	"fmt"
	"strconv"
	"sync"
	"unicode/utf8"
)

type buffer struct {
	buf []byte
}

var bufPool = sync.Pool{
	New: func() interface{} {
		return &buffer{buf: make([]byte, 0, 200)}
	},
}

func getBuffer() *buffer {
	return bufPool.Get().(*buffer)
}

func putBuffer(b *buffer) {
	b.buf = b.buf[:0]
	bufPool.Put(b)
}

func (b *buffer) String() string {
	s := string(b.buf)
	putBuffer(b)
	return s
}

func (b *buffer) appendInt(i int64) {
	b.buf = strconv.AppendInt(b.buf, i, 10)
}

func (b *buffer) appendUint(i uint64) {
	b.buf = strconv.AppendUint(b.buf, i, 10)
}

func (b *buffer) appendFloat64(f float64) {
	b.buf = strconv.AppendFloat(b.buf, f, 'g', -1, 64)
}

func (b *buffer) appendFloat32(f float32) {
	b.buf = strconv.AppendFloat(b.buf, float64(f), 'g', -1, 32)
}

func (b *buffer) appendByte(v byte) {
	b.buf = append(b.buf, v)
}

func (b *buffer) appendBytes(p []byte) {
	b.buf = append(b.buf, p...)
}

func (b *buffer) appendString(s string) {
	b.buf = append(b.buf, s...)
}

func (b *buffer) appendBool(v bool) {
	if v {
		b.appendString("true")
		return
	}
	b.appendString("false")
}

func (b *buffer) appendQuote(s string) {
	b.buf = strconv.AppendQuote(b.buf, s)
}

func (b *buffer) appendEscapeByte(c byte) {
	switch c {
	case '\n':
		b.appendString("\n      ")
	default:
		b.appendByte(c)
	}
}

func (b *buffer) appendEscapeString(s string) {
	for i := 0; i < len(s); i++ {
		b.appendEscapeByte(s[i])
	}
}

func (b *buffer) appendValue(v interface{}) {
	switch t := v.(type) {
	case string:
		b.appendString(t)
	case []byte:
		b.appendBytes(t)
	case error:
		b.appendString(t.Error())
	case fmt.Stringer:
		b.appendString(t.String())
	case int:
		b.appendInt(int64(t))
	case int64:
		b.appendInt(t)
	case uint64:
		b.appendUint(t)
	case int32:
		b.appendInt(int64(t))
	case uint32:
		b.appendUint(uint64(t))
	case int16:
		b.appendInt(int64(t))
	case uint16:
		b.appendUint(uint64(t))
	case int8:
		b.appendInt(int64(t))
	case uint8:
		b.appendUint(uint64(t))
	case float64:
		b.appendFloat64(t)
	case float32:
		b.appendFloat32(t)
	default:
		b.appendString(fmt.Sprint(v))
	}
}

func (b *buffer) appendDataValue(v interface{}) bool {
	switch t := v.(type) {
	case string:
		b.appendQuoteString(t)
	case error:
		b.appendQuoteString(t.Error())
	case fmt.Stringer:
		b.appendQuoteString(t.String())
	case func() string:
		b.appendQuoteString(t())
	case Hook:
		if v := t(); v != nil {
			return b.appendDataValue(v)
		}
		return false
	case int:
		b.appendInt(int64(t))
	case uint:
		b.appendUint(uint64(t))
	case int64:
		b.appendInt(t)
	case uint64:
		b.appendUint(t)
	case int32:
		b.appendInt(int64(t))
	case uint32:
		b.appendUint(uint64(t))
	case int16:
		b.appendInt(int64(t))
	case uint16:
		b.appendUint(uint64(t))
	case int8:
		b.appendInt(int64(t))
	case uint8:
		b.appendUint(uint64(t))
	case bool:
		b.appendBool(t)
	case float64:
		b.appendFloat64(t)
	case float32:
		b.appendFloat32(t)
	default:
		b.appendQuoteString(fmt.Sprint(v))
	}
	return true
}

func (b *buffer) Write(p []byte) (int, error) {
	b.appendBytes(p)
	return len(p), nil
}

func (b *buffer) appendInterface(v interface{}) {
	fmt.Fprint(b, v)
}

const digits = "0123456789"

func (b *buffer) appendDigits(n, length int) {
	for i := 0; i < length; i++ {
		b.buf = append(b.buf, '0')
	}
	for i := 0; i < length && n > 0; i++ {
		b.buf[len(b.buf)-i-1] = digits[n%10]
		n /= 10
	}
}

func (b *buffer) appendData(data Data) {
	if len(data) == 0 {
		return
	}

	start := len(b.buf)

	b.appendString("\t|")
	written := false
	for _, kv := range data {
		i := len(b.buf)
		b.appendByte(' ')
		b.appendString(kv.Key)
		b.appendByte('=')
		if ok := b.appendDataValue(kv.Value); ok {
			written = true
		} else {
			b.buf = b.buf[:i]
		}
	}

	if !written {
		b.buf = b.buf[:start]
	}
}

const (
	quote    = '"'
	lowerhex = "0123456789abcdef"
)

// A slightly adapted version of strconv.quoteWith from the standard library.
func (b *buffer) appendQuoteString(s string) {
	var runeTmp [utf8.UTFMax]byte
	b.buf = append(b.buf, quote)
	for width := 0; len(s) > 0; s = s[width:] {
		r := rune(s[0])
		width = 1
		if r >= utf8.RuneSelf {
			r, width = utf8.DecodeRuneInString(s)
		}
		if width == 1 && r == utf8.RuneError {
			b.buf = append(b.buf, `\x`...)
			b.buf = append(b.buf, lowerhex[s[0]>>4])
			b.buf = append(b.buf, lowerhex[s[0]&0xF])
			continue
		}
		if r == rune(quote) || r == '\\' { // always backslashed
			b.buf = append(b.buf, '\\')
			b.buf = append(b.buf, byte(r))
			continue
		}
		if strconv.IsPrint(r) {
			n := utf8.EncodeRune(runeTmp[:], r)
			b.buf = append(b.buf, runeTmp[:n]...)
			continue
		}
		switch r {
		case '\a':
			b.buf = append(b.buf, `\a`...)
		case '\b':
			b.buf = append(b.buf, `\b`...)
		case '\f':
			b.buf = append(b.buf, `\f`...)
		case '\n':
			b.buf = append(b.buf, `\n`...)
		case '\r':
			b.buf = append(b.buf, `\r`...)
		case '\t':
			b.buf = append(b.buf, `\t`...)
		case '\v':
			b.buf = append(b.buf, `\v`...)
		default:
			switch {
			case r < ' ':
				b.buf = append(b.buf, `\x`...)
				b.buf = append(b.buf, lowerhex[s[0]>>4])
				b.buf = append(b.buf, lowerhex[s[0]&0xF])
			case r > utf8.MaxRune:
				r = 0xFFFD
				fallthrough
			case r < 0x10000:
				b.buf = append(b.buf, `\u`...)
				for s := 12; s >= 0; s -= 4 {
					b.buf = append(b.buf, lowerhex[r>>uint(s)&0xF])
				}
			default:
				b.buf = append(b.buf, `\U`...)
				for s := 28; s >= 0; s -= 4 {
					b.buf = append(b.buf, lowerhex[r>>uint(s)&0xF])
				}
			}
		}
	}
	b.buf = append(b.buf, quote)
}
