/*
Package say is a logging and metrics-reporting library.

See https://github.com/go-say/say for a presentation of the project.


Introduction

By default, Say prints all messages (logs and metrics) to standard output.

When a listener is set using SetListener(), messages are handler by the listener
in a goroutine.


Logging functions

Say provides 5 severity levels:
 - Debug
 - Info
 - Warning
 - Error
 - Fatal


Metrics functions

Say provides 4 metrics-reporting functions:
 - Event: track the occurence of a particular event (user sign-up, query to the
   database)
 - Value: measure a value associated with a particular event (number of items
   returned by a search)
 - Timing.Say: measure a duration value (database query duration, webservice
   call duration)
 - Gauge: capture the current value of something that changes over time (number
   of active goroutines, number of connected users)

These metrics are directly inspired from StatsD metrics:
 - Event: counter
 - Value: histogram / timing
 - Timing.Say: timing
 - Gauge: gauge

See the function's descriptions below for more info.

Links:
 - StatsD metrics: https://github.com/etsy/statsd/blob/master/docs/metric_types.md
 - Datadog documentation: http://docs.datadoghq.com/guides/metrics/#counters


Package-level or methods

These functions can be called at a package-level or you can create a Logger and
use the associated methods:

	log := new(say.Logger)
	log.Info("Hello!")


Data

The point of using a Logger is to associate key-value pairs to it:

	log := new(say.Logger)
	log.AddData("request_id", requestID)
	log.Info("Hello!")
	// Output:
	INFO   Hello!  | request_id=3

All logging and metric-reporting functions also accept key-value pairs:

	Info("Hello!", "name", "Bob", "age", 30)
	// Output:
	INFO  Hello!  | name="Bob" age=30
*/
package say
