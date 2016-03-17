package say

import (
	"reflect"
	"testing"
)

func TestCapturePanic(t *testing.T) {
	expect(t, func() {
		defer CapturePanic()
		panic("oops")
	}, []string{
		"FATAL oops",
	})
}

func TestLoggerCapturePanic(t *testing.T) {
	expect(t, func() {
		log := NewLogger(SkipStackFrames(-1))
		defer log.CapturePanic()
		panic("oops")
	}, []string{
		"FATAL oops",
	})
}

func TestFlush(t *testing.T) {
	received := false
	SetListener(func(msg *Message) {
		received = true
	})
	defer SetListener(nil)
	Info("test")
	Flush()
	if !received {
		t.Error("Message not received")
	}
}

func TestFlushNoListener(t *testing.T) {
	Flush()
}

func TestSetListener(t *testing.T) {
	content := "hello"
	processed := false
	SetListener(func(msg *Message) {
		if msg.Type != TypeInfo {
			t.Errorf("Invalid message type, got %s, want %s",
				msg.Type, TypeInfo)
		}
		if msg.Content != content {
			t.Errorf("Invalid message type, got %s, want %s",
				msg.Content, content)
		}
		wantData := Data{{Key: "int", Value: 42}}
		if !reflect.DeepEqual(msg.Data, wantData) {
			t.Errorf("Invalid message data, got %#v, want %#v",
				msg.Data, wantData)
		}
		processed = true
	})
	defer SetListener(nil)
	Info(content, "int", 42)
	Flush()
	if !processed {
		t.Error("Message not processed")
	}
}

func TestPanicSetListener(t *testing.T) {
	content := "oops"
	processed := false
	SetListener(func(msg *Message) {
		if msg.Type != TypeFatal {
			t.Errorf("Invalid message type, got %s, want %s",
				msg.Type, TypeFatal)
		}
		if msg.Content != content {
			t.Errorf("Invalid message type, got %s, want %s",
				msg.Content, content)
		}
		if len(msg.Data) != 0 {
			t.Errorf("Invalid message data, got %#v, want empty", msg.Data)
		}
		processed = true
	})

	defer func() {
		SetListener(nil)
		if !processed {
			t.Error("Message not processed")
		}
	}()
	defer CapturePanic()
	panic(content)
}
