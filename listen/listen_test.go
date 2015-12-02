package listen

import (
	"fmt"
	"io"
	"os"
	"regexp"
	"testing"

	"gopkg.in/say.v0"
)

func init() {
	say.SetDebug(true)
}

var outputApp io.Closer

func resetListener() {
	msg = nil
	r, w, err := os.Pipe()
	if err != nil {
		panic(err)
	}
	dr := &durationRewriter{r}
	SetInput(dr)
	say.Redirect(w)
	outputApp = w
}

func closeOutputApp() {
	outputApp.Close()
}

type durationRewriter struct {
	io.Reader
}

var findDuration = regexp.MustCompile(`\d+ms`)

func (r *durationRewriter) Read(p []byte) (int, error) {
	n, err := r.Reader.Read(p)
	v := findDuration.ReplaceAll(p[:n], []byte("5ms"))
	k := copy(p, v)
	return k, err
}

func TestInit(t *testing.T) {
	resetListener()
	appName := "my_app"
	say.Info("foo")
	msg := Init()
	if msg != nil {
		t.Error("Init() should return nil")
	}
	say.Init(appName)
	msg = Init()
	if msg == nil {
		t.Fatal("Init() should not return nil")
	}
	if msg.Type() != TypeInit {
		t.Errorf("Wrong message type, got %q, want %q", msg.Type(), TypeInit)
	}
	if msg.Content() != appName {
		t.Errorf("Wrong message type, got %q, want %q", msg.Content(), appName)
	}
	say.Info("bar")
	closeOutputApp()

	received := false
	err := Listen(func(m *Message) {
		if m.Type() != TypeInfo {
			t.Errorf("Wrong message type, got %q, want %q",
				msg.Type(), TypeInfo)
		}
		received = true
	})
	if err != nil {
		t.Fatalf("Listen(): %v", err)
	}
	if received != true {
		t.Error("INFO message not received in Listen")
	}
}

func TestNonInit(t *testing.T) {
	resetListener()
	say.Info("foo")
	msg := Init()
	if msg != nil {
		t.Error("Init() should return nil")
	}
	closeOutputApp()

	received := false
	err := Listen(func(m *Message) {
		if m.Type() != TypeInfo {
			t.Errorf("Wrong message type, got %q, want %q",
				msg.Type(), TypeInfo)
		}
		if m.Content() != "foo" {
			t.Errorf("Wrong message content, got %q, want %q",
				msg.Content(), "foo")
		}
		received = true
	})
	if err != nil {
		t.Fatalf("Listen(): %v", err)
	}
	if received != true {
		t.Error("Message not received in Listen")
	}
}

func TestInvalidLines(t *testing.T) {
	msg = nil
	r, w, err := os.Pipe()
	if err != nil {
		panic(err)
	}
	dr := &durationRewriter{r}
	SetInput(dr)

	go func() {
		w.WriteString("\n")
		w.WriteString("foo\n")
		w.WriteString("INFO\n")
		w.WriteString("INFO \n")
		w.WriteString("INF   foo\n")
		w.WriteString("INFO  foo")
		w.WriteString("\n")
		w.WriteString("foo bar baz")
		w.Close()
	}()

	n := 0
	SetErrorHandler(func(got string) {
		n++
		var invalid string
		switch n {
		case 1:
			invalid = ""
		case 2:
			invalid = "foo"
		case 3:
			invalid = "INFO"
		case 4:
			invalid = "INFO "
		case 5:
			invalid = "INF   foo"
		case 6:
			invalid = "foo bar baz"
		default:
			t.Errorf("Got at least %d errors, want %d", n, 6)
		}
		want := fmt.Sprintf("listen: invalid line: %q", invalid)
		if got != want {
			t.Errorf("Wrong error received\n got: %q\nwant: %q", got, want)
		}
	})

	if err = Listen(func(m *Message) {}); err != nil {
		t.Fatalf("Listen(): %v", err)
	}

	if n != 6 {
		t.Errorf("Got %d errors, want %d", n, 6)
	}
}

type test struct {
	f    func()
	want interface{}
}

func testMessage(t *testing.T, tests []test, h func(*Message, interface{})) {
	resetListener()

	go func() {
		for _, test := range tests {
			test.f()
		}
	}()

	i := 0
	err := Listen(func(m *Message) {
		if i >= len(tests) {
			t.Fatal("Listen received too many messages.")
		}
		h(m, tests[i].want)
		i++
		if i == len(tests) {
			closeOutputApp()
		}
	})
	if err != nil {
		t.Fatalf("Listen(): %v", err)
	}
}
