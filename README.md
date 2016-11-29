# rlog - A simple Golang logger with lots of features and no external dependencies

Rlog is a simple logging package. It is configurable 'from the outside' via
environment variables and has no dependencies other than the standard Golang
library.

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
* Logging of date and time can be disabled (useful in case of systemd, which
  adds its own time stamps in its log database).
* By default logs to stderr. A different logfile can be configured via
  environment variable. Also, a different io.Writer can be specified by the
  program at any time, thus redirecting log output on the fly.


## Defaults

By default rlog logs all messages of level INFO and up. Trace messages are
not logged. Messages contain time stamps and output is sent to stderr. All
those defaults can be changed through environment variables.


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
* RLOG_LOG_NOTIME:  If this variable is set to "1", "yes" or something else
                    that evaluates to 'true' then no date/time stamp is
                    logged with each log message. This is useful in
                    environments that use systemd where access to the logs
                    via their logging tools already gives you time stamps.
                    Default: No - meaning that time/date is logged.
* RLOG_LOG_FILE:    Provide a filename here to determine where the logfile
                    is written. By default (if this variable is not defined)
                    the log output is simply written to stderr.
                    Default: Not set - meaning that output goes to stderr.

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

With time stamp, trace to level 2, but without caller info:

    2016/11/30 07:38:57 INFO     : Start of program
    2016/11/30 07:38:57 WARN     : Warning level log message
    2016/11/30 07:38:57 ERROR    : Error level log message
    2016/11/30 07:38:57 CRITICAL : Critical level log message
    2016/11/30 07:38:57 TRACE(1) : Trace messages have their own numeric levels
    2016/11/30 07:38:57 TRACE(1) : To see them set RLOG_TRACE_LEVEL to the cut-off number
    2016/11/30 07:38:57 TRACE(1) : We're 1 levels down now...
    2016/11/30 07:38:57 TRACE(2) : We're 2 levels down now...
    2016/11/30 07:38:57 INFO     : Reached end of recursion at level 10

With time stamp, no trace logging, but with caller info:

    2016/11/30 07:41:33 INFO     : [examples/example.go:22 (main.main)] Start of program
    2016/11/30 07:41:33 WARN     : [examples/example.go:30 (main.main)] Warning level log message
    2016/11/30 07:41:33 ERROR    : [examples/example.go:31 (main.main)] Error level log message
    2016/11/30 07:41:33 CRITICAL : [examples/example.go:32 (main.main)] Critical level log message
    2016/11/30 07:41:33 INFO     : [examples/example.go:16 (main.someRecursiveFunction)] Reached end of recursion at level 10

Without time stamp, no trace logging, no caller info:

    INFO     : Start of program
    WARN     : Warning level log message
    ERROR    : Error level log message
    CRITICAL : Critical level log message
    INFO     : Reached end of recursion at level 10


