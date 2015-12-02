# Say
[![Build Status](https://travis-ci.org/go-say/say.svg?branch=v0)](https://travis-ci.org/go-say/say)
[![Code Coverage](http://gocover.io/_badge/gopkg.in/say.v0)](http://gocover.io/gopkg.in/say.v0)
[![Documentation](https://godoc.org/gopkg.in/say.v0?status.svg)](https://godoc.org/gopkg.in/say.v0)


## Introduction

Say is a logging and metrics-reporting library.

Basically logs and metrics are both a way to report how an application is
behaving. So the idea is to print both logs and metrics to standard output.

When developing, applications using Say are verbose and easier to debug.
In production, application's output is piped to a listener application that
handles the logs and metrics.

Package say provides functions to print logs and metrics while package listen
provides functions to build listener applications.

Say is particularly effective to manage the logs and metrics of a fleet of Go
applications. It is less suited for libraries or distributable applications.


## Example

```go
func main() {
	defer say.CapturePanic() // Catch panics and format the error message.

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

While developing, your applications are verbose and easier to debug.

In production, you pipe your application's output with a listener that handles
your logs and metrics.

Listeners are very easy to build thanks to the
[say/listen package](https://github.com/go-say/say/tree/v0/listen).


## Features

### Developer friendly

Applications using Say are verbose since metrics are printed to standard output
which tremendously helps debugging.

Say also has a cool API with many nice things that makes the developer's life
easier: errors' stack traces are printed by default, one-liner to time a
function, simple debugging functions, no boilerplate to define how your logs are
handled, etc.


### Separation of concerns

Say allows you to follow the best practices like the
[Twelve Factors App](http://12factor.net/logs): Applications using Say simply
print to standard output. You don't need to define how to handle your logs and
metrics in every application.

Instead, all your log and metric related logic is defined in your listener
application. If you want to change the way you handle logs or metrics you don't
need to update all your applications but just the listener.


### Flexible

With Say, it is very easy to handle logs and metrics exactly as you want:
writing logs only if an error happens, sending errors by email, sending
metrics to a StatsD backend or your favorite webservice, etc.


### Simple

Say's output is often deterministic (since there is no timestamp by default).
So a simple diff of the output of two versions of an application running the
same tests can provides quick insights of what changed in the behavior of the
application.

In short, Say's format is pretty simple which brings many advantages: it can be
implemented in other languages, tools can be built around Say, etc.


### Lightweight and Fast

Say just prints to standard output and the code has been carefully written to be
fast and with a low memory footprint. As a result, Say is way faster than most
logging libraries and is even faster than the standard library:

```
BenchmarkStd-4        2000000     629 ns/op     32 B/op    2 allocs/op
BenchmarkSay-4       10000000     163 ns/op      0 B/op    0 allocs/op

BenchmarkStdData-4    1000000    1492 ns/op    112 B/op    6 allocs/op
BenchmarkSayData-4    2000000     960 ns/op     64 B/op    4 allocs/op
```

## Drawback

Say's main drawback is that you must pipe a listener application to handle logs
and metrics. That is why Say might not be suited for libraries and distributable
applications.

Although it is not Say's target use case. It is still possible for an
application to listen to itself. See
[this example from the listen package](https://godoc.org/gopkg.in/say.v0/listen#example-SetInput).

However for this kind of use, you might prefer using a logger like
[log15](https://github.com/inconshreveable/log15) or
[logrus](https://github.com/Sirupsen/logrus).


## License

[MIT](LICENSE)


## Contribute

Do you have any question the documentation does not answer? Is there a use case
that you feel is common and is not well-addressed by the current API?

Then you are more than welcome to open an issue or send a pull-request.
See [CONTRIBUTING.md](CONTRIBUTING.md) for more info.
