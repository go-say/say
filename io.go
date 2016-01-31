package say

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
)

var (
	listener  func(*Message)
	ch        chan *Message
	waitFlush = make(chan struct{})
	closed    = make(chan struct{})
)

// SetListener sets the function that is applied to each message.
//
// SetListener(nil) restores the default behavior wich is printing messages to
// the standard output.
func SetListener(f func(*Message)) {
	switch {
	// If old is nil and new non-nil, start the listening daemon.
	case listener == nil && f != nil:
		listener = f
		ch = make(chan *Message, 1000)
		go func() {
			for {
				msg, ok := <-ch
				if !ok {
					closed <- struct{}{}
					return
				}
				if msg == nil {
					waitFlush <- struct{}{}
					continue
				}
				listener(msg)
				putMessage(msg)
			}
		}()
	// If old is non-nil and new is nil, stop the listening daemon.
	case listener != nil && f == nil:
		close(ch)
		<-closed
		listener = nil
	// If old and new are non-nil, replace old by new.
	case listener != nil && f != nil:
		listener = f
	}
}

// Flush flushes the message queue. It is a no-op when SetListener has not been
// used.
func Flush() {
	if listener != nil {
		ch <- nil
		<-waitFlush
	}
}

func (l *Logger) send(typ Type, content string, data []interface{}) {
	msg := getMessage()
	msg.Type = typ
	msg.Content = content

	mu.RLock()
	msg.Data = append(msg.Data, l.data...)
	mu.RUnlock()
	if len(data) > 0 {
		if err := msg.Data.appendData(data); err != nil {
			l.error(TypeError, err, nil, 2)
		}
	}

	if listener == nil {
		printMessage(msg)
		putMessage(msg)
	} else {
		ch <- msg
	}
}

var out io.Writer = os.Stdout

func printMessage(msg *Message) {
	buf := getBuffer()
	buf.appendString(string(msg.Type))
	buf.appendByte(' ')
	buf.appendEscapeString(msg.Content)
	buf.appendData(msg.Data)
	buf.appendByte('\n')

	mu.RLock()
	if _, err := out.Write(buf.buf); err != nil {
		_, err := fmt.Fprintf(os.Stderr, "say: cannot write to output: %v", err)
		if err != nil {
			// This isn't our lucky day. Panics since stderr is not writable.
			mu.RUnlock()
			panic(fmt.Sprintf("say: cannot write to stderr: %v", err))
		}
	}
	mu.RUnlock()

	putBuffer(buf)
}

// Redirect redirects the output to the given writer. It returns the writer
// where outputs were previously redirected to.
//
// It is only effective when SetListener has not been used.
func Redirect(w io.Writer) (oldW io.Writer) {
	mu.Lock()
	oldW, out = out, w
	mu.Unlock()
	return oldW
}

// Mute disables any output. It is the same as Redirect(ioutil.Discard).
func Mute() io.Writer {
	return Redirect(ioutil.Discard)
}

// CapturePanic captures panic values as FATAL messages.
func (l *Logger) CapturePanic() {
	l.capturePanic(recover())
}

// CapturePanic captures panic values as FATAL messages.
func CapturePanic() {
	defaultLogger.capturePanic(recover())
}

func (l *Logger) capturePanic(err interface{}) {
	if err != nil {
		l.error(TypeFatal, err, nil, 2)
	}

	Flush()

	if err != nil {
		exit(2)
	}
}

// Stubbed out for testing.
var exit = os.Exit
