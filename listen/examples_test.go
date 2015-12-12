package listen_test

import (
	"fmt"
	"log"
	"os"

	"gopkg.in/say.v0"
	"gopkg.in/say.v0/listen"
)

func Example() {
	appName := "noname"
	if msg := listen.Init(); msg != nil {
		appName = msg.Content()
	}

	f, err := os.Create(fmt.Sprintf("/var/log/%s.log", appName))
	if err != nil {
		log.Printf("error: cannot open file: %v", err)
		log.Print("printing to stdout")
		f = os.Stdout
	} else {
		defer f.Close()
	}

	err = listen.Listen(func(m *listen.Message) {
		switch m.Type() {
		case listen.TypeError, listen.TypeFatal:
			// Email the errors or send them to your favorite webservice.
			fallthrough
		case listen.TypeInfo, listen.TypeWarning:
			_, err = m.WriteTo(f)
			if err != nil {
				_ = f.Close()
				log.Printf("error: cannot write to file: %v", err)
				log.Print("printing to stdout")
				f = os.Stdout
			}
		}
	})
	if err != nil {
		panic(err)
	}
}

// An example of a program listening to itself.
func ExampleSetInput() {
	r, w, err := os.Pipe()
	if err != nil {
		panic(err)
	}
	listen.SetInput(r)
	say.Redirect(w)

	ch := make(chan bool)
	go func() {
		err = listen.Listen(func(m *listen.Message) {
			fmt.Printf("Received message: %s %s", m.Type(), m.Content())
		})
		if err != nil {
			panic(err)
		}
		ch <- true
	}()

	say.Info("Hello!")
	// Closing the writer makes Listen exit when it finishes
	// processing all messages.
	w.Close()
	<-ch

	// Output:
	// Received message: INFO  Hello!
}

func ExampleSetErrorHandler() {
	listen.SetErrorHandler(func(errMsg string) {
		log.Print(errMsg)
	})
}
