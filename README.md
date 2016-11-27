# rlog - A simple Golang logger without external dependencies

Rlog is a simple logging package. It is configurable 'from the outside' via
environment variables and has no dependencies other than the standard Golang
library.

## Features

* Offers familiar and easy to use log functions for the usual levels: Debug,
  Info, Warn, Error and Critical.
* Every log function comes in a 'plain' version (to be used like Println)
  and in a formatted version (to be used like Printf). For example, there
  is Debug() and Debugf(), which takes a format string as first parameter.
* Offers an additional multi level logging facility with arbitrary depth,
  called "Trace".
* Can be configured to print caller info (filename and line, function name).
* Has NO external dependencies, except things contained in the standard Go
  library.

Rlog is configured via the following environment variables:

* RLOG_LOG_LEVEL:   Set to "DEBUG", "INFO", "WARN", "ERROR", "CRITICAL"
                    or "NONE".
                    Any message of a level >= than what is configured will
                    be printed. If this is not defined it will default to
                    "INFO". If it is set to "NONE" then all logging is
                    disabled, except Trace logs, which are controlled via a
                    separate variable.
                    Default: INFO
* RLOG_TRACE_LEVEL: "Trace" log messages take an additional numeric level as
                    first parameter. The user can specify an arbitrary
                    number of levels. Set RLOG_TRACE_LEVEL to a number. All
                    Trace messages with a level <= RLOG_TRACE_LEVEL will be
                    printed. If this variable is undefined, or set to -1
                    then no Trace messages are printed. The idea is that the
                    higher the RLOG_TRACE_LEVEL value, the more 'chatty' and
                    verbose the Trace message output becomes.
                    Default: -1
* RLOG_CALLER_INFO: If this variable is set to "1", "yes" or something else
                    that evaluates to 'true' then the message also contains
                    the caller information, consisting of the file and line
                    number as well as function name from which the log
                    message was called.
                    Default: no

Please note! If these environment variables have incorrect or misspelled
values then they will be silently ignored and a default value will be used.

## Usage example

    import "github.com/romana/rlog"
    func main() {
 	   rlog.Debug("A debug message: For the developer")
 	   rlog.Info("An info message: Normal operation messages")
 	   rlog.Warn("A warning message: Intermittend issues, high load, etc.")
 	   rlog.Error("An error message: An error occurred, I will recover.")
 	   rlog.Critical("A critical message: That's it! I give up!")
 	   rlog.Trace(2, "A trace message")
 	   rlog.Trace(3, "An even deeper trace message")
    }


