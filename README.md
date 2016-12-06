# rlog - A simple Golang logger with lots of features and no external dependencies

Rlog is a simple logging package, rich in features. It is configurable 'from
the outside' via environment variables and has no dependencies other than the
standard Golang library.

It is called "rlog", because it was originally written for the Romana project
(https://github.com/romana/romana).


## Features

* Is configured through environment variables: No need to call a special
  init function of some kind to initialize and configure the logger.
* Offers familiar and easy to use log functions for the usual levels: Debug,
  Info, Warn, Error and Critical.
* Offers an additional multi level logging facility with arbitrary depth,
  called Trace.
* Log and trace levels can be configured separately for the individual files
  that make up your executable.
* Every log function comes in a 'plain' version (to be used like Println)
  and in a formatted version (to be used like Printf). For example, there
  is Debug() and Debugf(), which takes a format string as first parameter.
* Can be configured to print caller info (module filename and line, function
  name).
* Has NO external dependencies, except things contained in the standard Go
  library.
* Fully configurable date/time format.
* Logging of date and time can be disabled (useful in case of systemd, which
  adds its own time stamps in its log database).
* By default logs to stderr or stdout. A logfile can be configured via
  environment variable. Output may happen exclusively to the logfile or in
  addition to the output on stderr/stdout. Also, a different output stream
  or file can be specified from within your programs at any time.


## Defaults

Rlog comes with reasonable defaults, so you can just start using it without any
configuration at all. By default:

* Log level set to INFO.
* Trace messages are not logged.
* Time stamps are logged with each message.
* No caller information.
* Output is sent to stderr.
  
All those defaults can easily be changed through environment variables.


## Controlling rlog through environment variables

Rlog is configured via the following environment variables:

* RLOG_LOG_LEVEL:   Set to "DEBUG", "INFO", "WARN", "ERROR", "CRITICAL"
                    or "NONE".
                    Any message of a level >= than what's configured will
                    be printed. If this is not defined it will default to
                    "INFO". If it is set to "NONE" then all logging is
                    disabled, except Trace logs, which are controlled via a
                    separate variable. In addition, log levels can be set
                    for individual files (see below for more information).
                    Default: INFO - meaning that INFO and higher is logged.
* RLOG_TRACE_LEVEL: "Trace" log messages take an additional numeric level as
                    first parameter. The user can specify an arbitrary
                    number of levels. Set RLOG_TRACE_LEVEL to a number. All
                    Trace messages with a level <= RLOG_TRACE_LEVEL will be
                    printed. If this variable is undefined, or set to -1
                    then no Trace messages are printed. The idea is that the
                    higher the RLOG_TRACE_LEVEL value, the more 'chatty' and
                    verbose the Trace message output becomes. In addition,
                    trace levels can be set for individual files (see below
                    for more information).
                    Default: Not set - meaning that no trace messages are
                    logged.
* RLOG_CALLER_INFO: If this variable is set to "1", "yes" or something else
                    that evaluates to 'true' then the message also contains
                    the caller information, consisting of the file and line
                    number as well as function name from which the log
                    message was called.
                    Default: No - meaning that no caller info is logged.
* RLOG_TIME_FORMAT: Use this variable to customize the date/time format. The
                    format is specified either by the well known formats
                    listed in https://golang.org/src/time/format.go (for
                    example "UnixDate" or "RFC3339") or as an example
                    date/time output, which is described here:
                    https://golang.org/pkg/time/#Time.Format
                    Default: Not set - formatted according to RFC3339.
* RLOG_LOG_NOTIME:  If this variable is set to "1", "yes" or something else
                    that evaluates to 'true' then no date/time stamp is
                    logged with each log message. This is useful in
                    environments that use systemd where access to the logs
                    via their logging tools already gives you time stamps.
                    Default: No - meaning that time/date is logged.
* RLOG_LOG_FILE:    Provide a filename here to determine if the logfile
                    should be written to a file, in addition to the output
                    stream specified in RLOG_LOG_STREAM.
                    Default: Not set - meaning that output is not written to
                    a file.
* RLOG_LOG_STREAM:  Use this to direct the log output to a different output
                    stream, instead of stderr. This accepts three values:
                    "stderr", "stdout" or "none". If either stderr or stdout
                    is defined here AND a logfile is specified via
                    RLOG_LOG_FILE then the output is sent to both.
                    Default: Not set - meaning the output goes to stderr.

Please note! If these environment variables have incorrect or misspelled
values then they will be silently ignored and a default value will be used.


## Per file level log and trace levels

In most cases you might want to set just a single log or trace level, which is
then applied to all log messages in your program:

    export RLOG_LOG_LEVEL=INFO
    export RLOG_TRACE_LEVEL=3

However, with rlog the log and trace levels can not only be configured
'globally' with a single value, but can also independently be set for the
individual module files that were compiled into your executable. This is useful
if enabling high trace levels or DEBUG logging for the entire executable would
fill up logs or consume too many resources.

For example, if your executable is compiled out of several files and one of
those files is called 'example.go' then you could set log levels like this:

    export RLOG_LOG_LEVEL=INFO,example.go=DEBUG

This sets the global log level to INFO, but for the messages originating from
the module file 'example.go' it is DEBUG.

Similarly, you can set trace levels for individual module files:

    export RLOG_TRACE_LEVEL=example.go=5,2

This sets a trace level of 5 for example.go and 2 for everyone else.

Note that as before, if in RLOG_LOG_LEVEL no global log level is specified then
INFO is assumed to be the global log level. If in RLOG_TRACE_LEVEL no global
trace level is specified then -1 (no trace output) is assumed as the global
trace level.

More examples:

    # DEBUG level for all files whose name starts with 'ex', WARNING level for
    # everyone else.
    export RLOG_LOG_LEVEL=WARN,ex*=DEBUG

    # DEBUG level for example.go, INFO for everyone else, since INFO is the
    # default level if nothing is specified.
    export RLOG_LOG_LEVEL=example.go=DEBUG

    # DEBUG level for example.go, no logging for anyone else.
    export RLOG_LOG_LEVEL=NONE,example.go=DEBUG

    # Multiple files' levels can be specified at once.
    export RLOG_LOG_LEVEL=NONE,example.go=DEBUG,foo.go=INFO

    # The default log level can appear anywhere in the list.
    export RLOG_LOG_LEVEL=example.go=DEBUG,INFO,foo.go=WARN


## Usage example

    import "github.com/romana/rlog"

    func main() {
 	   rlog.Debug("A debug message: For the developer")
 	   rlog.Info("An info message: Normal operation messages")
 	   rlog.Warn("A warning message: Intermittent issues, high load, etc.")
 	   rlog.Error("An error message: An error occurred, I will recover.")
 	   rlog.Critical("A critical message: That's it! I give up!")
 	   rlog.Trace(2, "A trace message")
 	   rlog.Trace(3, "An even deeper trace message")
    }

For a more interesting example, please check out 'examples/example.go'.


## Sample output

With time stamp, trace to level 2, log level WARNING, no caller info:

    $ export RLOG_LOG_LEVEL=WARN
    $ export RLOG_TRACE_LEVEL=2
    $ go run examples/example.go

    2016-12-05T12:03:41+13:00 WARN     : Warning level log message
    2016-12-05T12:03:41+13:00 ERROR    : Error level log message
    2016-12-05T12:03:41+13:00 CRITICAL : Critical level log message
    2016-12-05T12:03:41+13:00 TRACE(1) : Trace messages have their own numeric levels
    2016-12-05T12:03:41+13:00 TRACE(1) : To see them set RLOG_TRACE_LEVEL to the cut-off number
    2016-12-05T12:03:41+13:00 TRACE(1) : We're 1 levels down now...
    2016-12-05T12:03:41+13:00 TRACE(2) : We're 2 levels down now...

With time stamp, log level INFO, no trace logging (switched off by unsetting
the variable), but with caller info:

    $ export RLOG_CALLER_INFO=yes
    $ export RLOG_LOG_LEVEL=INFO
    $ export RLOG_TRACE_LEVEL=
    $ go run examples/example.go

    2016-12-05T12:04:33+13:00 INFO     : [examples/example.go:22 (main.main)] Start of program
    2016-12-05T12:04:33+13:00 INFO     : [examples/example.go:23 (main.main)] rlog is controlled via environment variables.
    2016-12-05T12:04:33+13:00 INFO     : [examples/example.go:24 (main.main)] Try the following settings:
    2016-12-05T12:04:33+13:00 INFO     : [examples/example.go:25 (main.main)]    export RLOG_LOG_LEVEL=DEBUG
    2016-12-05T12:04:33+13:00 INFO     : [examples/example.go:26 (main.main)]    export RLOG_TRACE_LEVEL=5
    2016-12-05T12:04:33+13:00 INFO     : [examples/example.go:27 (main.main)]    export RLOG_CALLER_INFO=yes
    2016-12-05T12:04:33+13:00 INFO     : [examples/example.go:29 (main.main)] Format strings are possible 123
    2016-12-05T12:04:33+13:00 WARN     : [examples/example.go:30 (main.main)] Warning level log message
    2016-12-05T12:04:33+13:00 ERROR    : [examples/example.go:31 (main.main)] Error level log message
    2016-12-05T12:04:33+13:00 CRITICAL : [examples/example.go:32 (main.main)] Critical level log message
    2016-12-05T12:04:33+13:00 INFO     : [examples/example.go:16 (main.someRecursiveFunction)] Reached end of recursion at level 10
    2016-12-05T12:04:33+13:00 INFO     : [examples/example.go:42 (main.main)] About to change log output. Check /tmp/rlog-output.log...

Without time stamp, no trace logging, no caller info:

    $ export RLOG_LOG_NOTIME=yes
    $ export RLOG_CALLER_INFO=no
    $ go run examples/example.go

    INFO     : Start of program
    INFO     : rlog is controlled via environment variables.
    INFO     : Try the following settings:
    INFO     :    export RLOG_LOG_LEVEL=DEBUG
    INFO     :    export RLOG_TRACE_LEVEL=5
    INFO     :    export RLOG_CALLER_INFO=yes
    INFO     : Format strings are possible 123
    WARN     : Warning level log message
    ERROR    : Error level log message
    CRITICAL : Critical level log message
    INFO     : Reached end of recursion at level 10
    INFO     : About to change log output. Check /tmp/rlog-output.log...

With time stamp in RFC822 format.

    $ export RLOG_LOG_NOTIME=no
    $ export RLOG_TIME_FORMAT=RFC822
    $ go run examples/example.go

    05 Dec 16 12:07 NZDT INFO     : Start of program
    05 Dec 16 12:07 NZDT INFO     : rlog is controlled via environment variables.
    05 Dec 16 12:07 NZDT INFO     : Try the following settings:
    05 Dec 16 12:07 NZDT INFO     :    export RLOG_LOG_LEVEL=DEBUG
    05 Dec 16 12:07 NZDT INFO     :    export RLOG_TRACE_LEVEL=5
    05 Dec 16 12:07 NZDT INFO     :    export RLOG_CALLER_INFO=yes
    05 Dec 16 12:07 NZDT INFO     : Format strings are possible 123
    05 Dec 16 12:07 NZDT WARN     : Warning level log message
    05 Dec 16 12:07 NZDT ERROR    : Error level log message
    05 Dec 16 12:07 NZDT CRITICAL : Critical level log message
    05 Dec 16 12:07 NZDT INFO     : Reached end of recursion at level 10
    05 Dec 16 12:07 NZDT INFO     : About to change log output. Check /tmp/rlog-output.log...

With custom time stamp:

    $ export RLOG_TIME_FORMAT="2006/01/06 15:04:05"
    $ go run examples/example.go

    2016/12/16 12:09:03 INFO     : Start of program
    2016/12/16 12:09:03 INFO     : rlog is controlled via environment variables.
    2016/12/16 12:09:03 INFO     : Try the following settings:
    2016/12/16 12:09:03 INFO     :    export RLOG_LOG_LEVEL=DEBUG
    2016/12/16 12:09:03 INFO     :    export RLOG_TRACE_LEVEL=5
    2016/12/16 12:09:03 INFO     :    export RLOG_CALLER_INFO=yes
    2016/12/16 12:09:03 INFO     : Format strings are possible 123
    2016/12/16 12:09:03 WARN     : Warning level log message
    2016/12/16 12:09:03 ERROR    : Error level log message
    2016/12/16 12:09:03 CRITICAL : Critical level log message
    2016/12/16 12:09:03 INFO     : Reached end of recursion at level 10
    2016/12/16 12:09:03 INFO     : About to change log output. Check /tmp/rlog-output.log...


## Links

* Goreportcard.com: https://goreportcard.com/report/github.com/romana/rlog
* Godoc.com: https://godoc.org/github.com/romana/rlog
* Gocover.io: http://gocover.io/github.com/romana/rlog



