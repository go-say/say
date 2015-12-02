package listen

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"time"
)

var (
	errorHandler = func(errMsg string) {
		log.Print(errMsg)
	}
	in  *listener
	msg *Message
)

func init() {
	SetInput(os.Stdin)
}

// SetInput defines the reader that is listened to by the Listen function.
// By default it listens to the standard input.
//
// This function is useful to have a program that listens to itself.
func SetInput(r io.Reader) {
	in = &listener{s: bufio.NewScanner(r)}
	in.s.Split(scanSayLines)
}

// SetErrorHandler sets the function handling error messages.
//
// The default handler uses log.Print on errMsg.
func SetErrorHandler(h func(errMsg string)) {
	errorHandler = h
}

// Init returns the INIT message. It should be used before Listen to configure
// the listener.
//
// If the first message received is not an INIT message, Init returns nil.
//
// If Init is not used before Listen and the first message received is an INIT
// message, the handler passed to Listen will receive that INIT message.
func Init() *Message {
	var err error
	msg, err = in.next()
	if err != nil {
		return nil
	}
	if msg.Type() != TypeInit {
		return nil
	}
	return msg
}

// Listen listens to the input and executes the given function on each
// incoming message. It listens as long as the input is not closed.
//
// When the input is closed, for example when the left hand program of the pipe
// exits, Listen exits after the last message.
//
// It returns an error only if there is an error reading from the input. All Say
// formatting errors are handled by the function passed to SetErrorHandler.
func Listen(handler func(*Message)) error {
	var err error

	// If Init() was run and the first message was not an INIT message, handle
	// it now.
	if msg != nil && msg.Type() != TypeInit {
		handler(msg)
	}

	for {
		msg, err = in.next()
		if err != nil {
			return err
		}
		if msg == nil {
			return nil
		}

		handler(msg)
	}
}

type listener struct {
	s *bufio.Scanner
}

func (l *listener) next() (*Message, error) {
	for l.s.Scan() {
		if m := getMessage(l.s.Bytes()); m != nil {
			return m, nil
		}
	}
	return nil, l.s.Err()
}

func getMessage(raw []byte) *Message {
	t := getType(raw)
	if t == "" {
		errorf("invalid line: %q", raw)
		return nil
	}

	content, data := parseMessage(raw[typeLen+1:])
	return &Message{
		typ:     t,
		content: content,
		rawData: data,
	}
}

func parseMessage(raw []byte) (string, string) {
	i := bytes.Index(raw, []byte("\t|"))
	if i == -1 {
		return parseContent(raw), ""
	}
	return parseContent(raw[:i]), string(raw[i+3:])
}

func parseContent(raw []byte) string {
	return string(bytes.Replace(raw, blankPrefix, []byte{'\n'}, -1))
}

var blankPrefix = []byte(" \n      ")

func getType(raw []byte) Type {
	v := Type(raw[:typeLen])
	for _, t := range types {
		if v == t {
			return t
		}
	}
	return ""
}

func scanSayLines(data []byte, atEOF bool) (int, []byte, error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}

	discarded := 0
	i := 0
	for {
		j := bytes.IndexByte(data[i:], '\n')
		if j == -1 {
			break
		}
		i += j

		if j < typeLen+1 {
			// The line is too short to be valid. Ignore it.
			errorf("invalid line: %q", string(data[:i]))
			data = data[i+1:]
			discarded += i + 1
			i = 0
			continue
		}

		if j > typeLen+1 && data[i-1] == ' ' {
			// This is not a terminal newline. Keep looking.
			i++
			continue
		}

		// We have a full line.
		return discarded + i + 1, data[:i], nil
	}

	// If we're at EOF, we have a final, non-terminated line. Ignore it.
	if atEOF {
		errorf("invalid line: %q", string(data))
		return discarded + len(data), nil, nil
	}

	if len(data) < typeLen+1 {
		// Request more data.
		return discarded, nil, nil
	}

	if getType(data[:typeLen]) == "" || data[typeLen] != ' ' {
		// This is not a valid say line and it is probably some other output.
		// Ignore it.
		errorf("invalid line: %q", string(data))
		return discarded + len(data), nil, nil
	}

	if len(data) >= bufio.MaxScanTokenSize {
		// The line is too long. Return what we have now.
		errorf("line is way too long")
		return discarded + len(data), data, nil
	}

	// It looks like an unfinished valid say line. Request more data.
	return discarded, nil, nil
}

func errorf(format string, args ...interface{}) {
	errorHandler(fmt.Sprintf("listen: "+format, args...))
}

var now = time.Now
