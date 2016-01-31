# Say
[![Build Status](https://travis-ci.org/go-say/say.svg?branch=v0)](https://travis-ci.org/go-say/say)
[![Code Coverage](http://gocover.io/_badge/gopkg.in/say.v0)](http://gocover.io/gopkg.in/say.v0)
[![Documentation](https://godoc.org/gopkg.in/say.v0?status.svg)](https://godoc.org/gopkg.in/say.v0)


## Introduction

Say is a logging and metrics-reporting library that increases developer
productivity.

The idea is to print everything - logs and metrics - to standard output by
default. So that applications using Say are verbose and easier to debug while
developing.

In production a listener function is usually set so that log lines and metrics
are handled asynchronously in a goroutine exactly as the developer wants.
It makes Say extremely fast and flexible.


## Download

    go get gopkg.in/say.v0


## Example

```go
func main() {
	defer say.CapturePanic() // Catch panics as FATAL messages.

	today := time.Now().Format("01/02")

	say.Info("Getting number of users having their birthday", "date", today)
	n, err := countBirthdays(today)
	if err != nil {
		panic(err)
	}
	say.Value("birthdays", n)
}

func countBirthdays(birthday string) (int, error) {
	defer say.NewTiming().Say("query.duration") // Time the function.

	var n int
	query := "SELECT count(id) FROM users WHERE birthday=?"

	err := db.QueryRow(query, id).Scan(&n)
	say.Event("query_user", "query", say.DebugHook(query))

	return n, err
}
```

This code will output:
```
INFO  Getting number of users having their birthday	| date="11/19"
EVENT query_user
VALUE query.duration:17ms
VALUE birthdays:6
```

Using say.Debug(true), the SQL query would be displayed:
```
...
EVENT query_user	| query="SELECT count(id) FROM users WHERE birthday=?"
...
```

If an error happens, we have the stack trace:
```
INFO  Getting number of users having their birthday	| date="11/19"
EVENT query_user
VALUE query.duration:17ms
FATAL sql: database is closed

      main.main()
      	/home/me/go/src/main.go:22 +0x269
```

In production you will usually want to set a listener:

```go
var prod bool

func init() {
	flag.BoolVar(&prod, "prod", false, "Set to production mode.")
}

func main() {
	defer say.CapturePanic()

	f, err := os.Create("my_app.log")
	if err != nil {
		panic(err)
	}

	say.SetListener(func(m *say.Message) {
		switch m.Type {
		case say.TypeError, say.TypeFatal:
			// Send errors by email or to your favorite webservice.
			fallthrough
		case say.TypeInfo, say.TypeWarning:
			// Log to a file.
			if _, err = m.WriteTo(f); err != nil {
				panic(err)
			}
		}
	})

	// Your code...
}
```


## Features

### Developer friendly

Applications using Say are verbose since metrics are printed to standard output
by default which tremendously helps debugging.

Say also has a cool API with many nice things that makes the developer's life
easier: errors' stack traces are printed by default, one-liner to time a
function, simple debugging functions, etc.


### Flexible

With Say, it is very easy to handle logs and metrics exactly as you want:
writing logs only if an error happens, sending errors by email, sending
metrics to a StatsD backend or your favorite webservice, etc.


### Lightweight and Fast

Say has been carefully written to be fast and with a low memory footprint. As a
result, Say is way faster than most logging libraries and is even faster than
the standard library:

```
BenchmarkStd-4        2000000     694 ns/op     32 B/op    2 allocs/op
BenchmarkSay-4        5000000     334 ns/op      0 B/op    0 allocs/op

BenchmarkStdData-4    1000000    1693 ns/op    112 B/op    6 allocs/op
BenchmarkSayData-4    1000000    1742 ns/op     96 B/op    6 allocs/op
```


### Simple

Say's output is often deterministic (since there is no timestamp by default).
So a simple diff of the output of two versions of an application running the
same tests can provides quick insights of what changed in the behavior of the
application.


## License

[MIT](LICENSE)


## Contribute

Do you have any question the documentation does not answer? Is there a use case
that you feel is common and is not well-addressed by the current API?

Then you are more than welcome to open an issue or send a pull-request.
See [CONTRIBUTING.md](CONTRIBUTING.md) for more info.
