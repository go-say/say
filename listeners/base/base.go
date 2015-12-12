// This program is a basic Say listener. It saves logs to a file and sends
// metrics to a StatsD-compatible daemon.
package main

import (
	"flag"
	"log"
	"os"
	"path"

	"gopkg.in/alexcesaro/statsd.v1"
	"gopkg.in/say.v0/listen"
)

var (
	dir        string
	json       bool
	statsdAddr string
)

func init() {
	flag.StringVar(&dir, "dir", ".", "Directory where logs are written.")
	flag.BoolVar(&json, "json", false, "Whether to print logs with the JSON format.")
	flag.StringVar(&statsdAddr, "statsd", ":8125", "The StatsD daemon address.")
}

func main() {
	flag.Parse()

	appName := "noname"
	if m := listen.Init(); m != nil {
		appName = m.Content()
	}

	// Configure the file where logs are written.
	filename := path.Join(dir, appName+".log")
	f, err := os.Create(filename)
	if err != nil {
		log.Printf("error: cannot open file %q: %v", filename, err)
		log.Print("printing to stdout")
		f = os.Stdout
	} else {
		defer f.Close()
	}

	// Initialize the StatsD client.
	client, err := statsd.New(statsdAddr)
	if err != nil {
		log.Printf("warning: cannot connect to StatsD: %v", err)
		// Create a muted client.
		client, _ = statsd.New("", statsd.Mute(true))
	}
	defer client.Close()

	// Handle messages.
	err = listen.Listen(func(m *listen.Message) {
		switch m.Type() {
		case listen.TypeEvent:
			if i, ok := m.Int(); ok {
				client.Count(m.Key(), i, 1)
			}
		case listen.TypeValue:
			if data := m.Data(); data != nil {
				// If a VALUE message has unique=true, treat the value as a
				// StatsD set.
				if ok, _ := data.GetBool("unique"); ok {
					client.Unique(m.Key(), m.Value())
				}
			}
			if i, ok := m.Int(); ok {
				client.Timing(m.Key(), i, 1)
			}
		case listen.TypeGauge:
			if i, ok := m.Int(); ok {
				client.Gauge(m.Key(), i)
			}
		case listen.TypeInfo, listen.TypeWarning,
			listen.TypeError, listen.TypeFatal:
			if json {
				_, err = m.WriteJSONTo(f)
			} else {
				_, err = m.WriteTo(f)
			}
			if err != nil {
				_ = f.Close()
				log.Printf("error: cannot write to file %q: %v", filename, err)
				log.Print("printing to stdout")
				f = os.Stdout
			}
		}
	})
	if err != nil {
		log.Fatal(err)
	}
}
