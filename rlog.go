// Copyright (c) 2016 Pani Networks
// All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
// WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
// License for the specific language governing permissions and limitations
// under the License.

package rlog

//
// Rlog is a simple logging package. It is configurable 'from the outside' via
// environment variables.
//
// Features:
//
// * Offers familiar and easy to use log functions for the usual levels: Debug,
//   Info, Warn, Error and Critical.
// * Every log function comes in a 'plain' version (to be used like Println)
//   and in a formatted version (to be used like Printf). For example, there
//   is Debug() and Debugf(), which takes a format string as first parameter.
// * Offers an additional multi level logging facility with arbitrary depth,
//   called "Trace".
// * Can be configured to print caller info (filename and line, function name).
// * Has NO external dependencies, except things contained in the standard Go
//   library.
//
// Rlog is configured via the following environment variables:
//
// * RLOG_LOG_LEVEL:   Set to "DEBUG", "INFO", "WARN", "ERROR", "CRITICAL"
//                     or "NONE".
//                     Any message of a level >= than what is configured will
//                     be printed. If this is not defined it will default to
//                     "INFO". If it is set to "NONE" then all logging is
//                     disabled, except Trace logs, which are controlled via a
//                     separate variable.
//                     Default: INFO
// * RLOG_TRACE_LEVEL: "Trace" log messages take an additional numeric level as
//                     first parameter. The user can specify an arbitrary
//                     number of levels. Set RLOG_TRACE_LEVEL to a number. All
//                     Trace messages with a level <= RLOG_TRACE_LEVEL will be
//                     printed. If this variable is undefined, or set to -1
//                     then no Trace messages are printed. The idea is that the
//                     higher the RLOG_TRACE_LEVEL value, the more 'chatty' and
//                     verbose the Trace message output becomes.
//                     Default: -1
// * RLOG_CALLER_FILTER:
//                     Comma separated list of filters that allow enabling
//                     trace logs for different files.
//                     
//                     RLOG_CALLER_FILTER="<filter>,<filter>"
//                     filter:
//                       <pattern=Level>
//                     pattern:
//                       shell glob to match caller filename
//                     Level:
//                       trace level of the logs to enable in mathed files
//
//                     e.g.
//                        RLOG_CALLER_FILTER="client.go=1,ip*=5"
//                     enables trace 1 for client.go and trace 5 for all files
//                     which names starts with "ip"
//                    
// * RLOG_CALLER_INFO: If this variable is set to "1", "yes" or something else
//                     that evaluates to 'true' then the message also contains
//                     the caller information, consisting of the file and line
//                     number as well as function name from which the log
//                     message was called.
//                     Default: no
//
// Please note! If these environment variables have incorrect or misspelled
// values then they will be silently ignored and a default value will be used.
//
// Usage example:
//
//	   import "github.com/romana/core/common/rlog"
//	   func main() {
//		   rlog.Debug("A debug message: For the developer")
//		   rlog.Info("An info message: Normal operation messages")
//		   rlog.Warn("A warning message: Intermittend issues, high load, etc.")
//		   rlog.Error("An error message: An error occurred, I will recover.")
//		   rlog.Critical("A critical message: That's it! I give up!")
//		   rlog.Trace(2, "A trace message")
//		   rlog.Trace(3, "An even deeper trace message")
//	   }
//

import (
	"fmt"
	"log"
	"os"
	"runtime"
	"strconv"
	"strings"
	"path/filepath"
)

// The known log levels
const (
	levelTrace = iota
	levelDebug
	levelInfo
	levelWarn
	levelErr
	levelCrit
	levelNone
)

// Translation map from level to string representation
var levelStrings = map[int]string{
	levelTrace: "TRACE",
	levelDebug: "DEBUG",
	levelInfo:  "INFO",
	levelWarn:  "WARN",
	levelErr:   "ERROR",
	levelCrit:  "CRITICAL",
	levelNone:  "NONE",
}

// Translation from level string to number.
var levelNumbers = map[string]int{
	"TRACE":    levelTrace,
	"DEBUG":    levelDebug,
	"INFO":     levelInfo,
	"WARN":     levelWarn,
	"ERROR":    levelErr,
	"CRITICAL": levelCrit,
	"NONE":     levelNone,
}

// FilterSpec holds a list of filters.
type FilterSpec struct {
	Filters []Filter
}

// FromString initializes FilterSpec from string.
// format "<filter>,<filter>,[<filter>]..."
//     filter:
//       <pattern=Level>
//     pattern:
//       shell glob to match caller filename
//     Level:
//       trace level of the logs to enable in mathed files
//
//     example:
//        "client.go=1,ip*=5"
func (spec *FilterSpec) FromString(s string) {
	fields := strings.Split(s, ",")

	for _, f := range fields {
		// tokens should contain two elements: The filename and the trace level.
		tokens := strings.Split(f, "=")
		if len(tokens) != 2 {
			return
		}

		if filterLevel, err := strconv.Atoi(tokens[1]); err == nil {
			spec.Filters = append(spec.Filters, Filter{tokens[0], filterLevel})
		} else {
			return
		}
	}

	return
}

// matchFilters checks if given filename and trace level are accepted
// by any of the filters
func (spec *FilterSpec) matchFilters(filename string, level int) bool {

	// If no filters don't match anything.
	if len(spec.Filters) == 0 {
		return false
	}

	// If at least one filter matches.
	for _, filter := range spec.Filters {
		if filter.match(filename, level) {
			return true
		}
	}

	return false
}

// Filter holds filename and trace level to match logs.
type Filter struct {
	Pattern string
	Level   int
}

// match checks if given filename and trace level are matched by
// this filter.
func (f Filter) match(filename string, traceLevel int) bool {
	match, _ := filepath.Match(f.Pattern, filepath.Base(filename))
	if match && f.Level >= traceLevel {
		return true
	}

	return false
}


// Rlog is controlled via environment variables. Those things won't change on
// us. Therefore, we can look them up once and store them in module level
// global variables.
var settingTraceLevel int = -1        // -1 indicates 'not set' -> no tracing
var settingLogLevel int = levelInfo   // by default we log INFO or higher
var settingGetCallerInfo bool = false // whether we log info about calling function
var settingCallerFilter string = ""
var filterSpec = new(FilterSpec)

// init extracts settings for our logger from environment variables when the
// module is imported.
func init() {
	var err error

	logLevelEnv := os.Getenv("RLOG_LOG_LEVEL")
	callerInfoEnv := os.Getenv("RLOG_CALLER_INFO")
	traceLevelEnv := os.Getenv("RLOG_TRACE_LEVEL")
	settingCallerFilter := os.Getenv("RLOG_CALLER_FILTER")

	// Evaluating the desired log level
	levelVal, ok := levelNumbers[logLevelEnv]
	if ok {
		if levelVal != levelTrace {
			// User can't set things to 'Trace', so we would leave it at
			// the default, which is 'Info'. But other than that the user
			// has specified a valid level value, so we can set this now.
			settingLogLevel = levelVal
		}
	}

	// Evaluating the caller info variable
	var getCallerInfo bool
	if getCallerInfo, err = strconv.ParseBool(callerInfoEnv); err == nil {
		settingGetCallerInfo = getCallerInfo
	}

	// Initialize filters.
	filterSpec.FromString(settingCallerFilter)

	// Evaluating the trace level variable
	if traceLevelEnv != "" {
		var traceLevel int
		if traceLevel, err = strconv.Atoi(traceLevelEnv); err == nil {
			if traceLevel >= -1 {
				settingTraceLevel = traceLevel
			}
		}
	}
}


// basicLog is called by all the 'level 'functions.
// It checks what is configured to be included in the log message,
// decorates it accordingly and assembles the entire line. It then
// uses the standard log package to finally output the message.
func basicLog(logLevel int, logTraceLevel int, format string, prefixAddition string, a ...interface{}) {

	// Extract information about the caller of the log function, if requested.
	var callingFuncName string
	pc, filename, line, callerInfoAvailable := runtime.Caller(2)
	if callerInfoAvailable {
		callingFuncName = runtime.FuncForPC(pc).Name()
	}

	// Perform tests to see if we should log this message.
	var allowLog bool
	if logLevel != levelTrace {
		// If log is not a trace log then check log level.
		if logLevel >= settingLogLevel {
			allowLog = true
		}
	} else {
		// Trace logs are allowed if either global tracel level matches
		// or one of filters matches.
		if logTraceLevel <= settingTraceLevel && logTraceLevel >= 0 {
			allowLog = true
		}

		if  filterSpec.matchFilters(filename, logTraceLevel) {
			allowLog = true
		}
	}
	if !allowLog {
		return
	}

	callerInfo := ""
	if settingGetCallerInfo && callerInfoAvailable {
		callerInfo = fmt.Sprintf("[%s:%d (%s)] ", filename, line, callingFuncName)
	}

	// Assemble the actual log line
	var msg string
	if format != "" {
		msg = fmt.Sprintf(format, a...)
	} else {
		msg = fmt.Sprintln(a...)
	}
	levelDecoration := levelStrings[logLevel]
	log.Printf("%s%s: %s%s", levelDecoration, prefixAddition, callerInfo, msg)
}

// Trace is for low level tracing of activities. It takes an additional 'level'
// parameter. The RLOG_TRACE_LEVEL variable is used to determine which levels
// of trace message are output: Every message with a level lower or equal to
// what is specified in RLOG_TRACE_LEVEL. If RLOG_TRACE_LEVEL is not defined at
// all then no trace messages are printed.
func Trace(traceLevel int, a ...interface{}) {
	prefixAddition := fmt.Sprintf("(%d)", traceLevel)
	basicLog(levelTrace, traceLevel, "", prefixAddition, a...)
}

// Tracef prints trace messages, with formatting.
func Tracef(traceLevel int, format string, a ...interface{}) {
	prefixAddition := fmt.Sprintf("(%d)", traceLevel)
	basicLog(levelTrace, traceLevel, format, prefixAddition, a...)
}

const NotATrace = -1
// Debug prints a message if RLOG_LEVEL is set to DEBUG.
func Debug(a ...interface{}) {
	basicLog(levelDebug, NotATrace, "", "", a...)
}

// Debugf prints a message if RLOG_LEVEL is set to DEBUG, with formatting.
func Debugf(format string, a ...interface{}) {
	basicLog(levelDebug, NotATrace, format, "", a...)
}

// Info prints a message if RLOG_LEVEL is set to INFO or lower.
func Info(a ...interface{}) {
	basicLog(levelInfo, NotATrace, "", "", a...)
}

// Infof prints a message if RLOG_LEVEL is set to INFO or lower, with
// formatting.
func Infof(format string, a ...interface{}) {
	basicLog(levelInfo, NotATrace, format, "", a...)
}

// Warn prints a message if RLOG_LEVEL is set to WARN or lower.
func Warn(a ...interface{}) {
	basicLog(levelWarn, NotATrace, "", "", a...)
}

// Warnf prints a message if RLOG_LEVEL is set to WARN or lower, with
// formatting.
func Warnf(format string, a ...interface{}) {
	basicLog(levelWarn, NotATrace, format, "", a...)
}

// Error prints a message if RLOG_LEVEL is set to ERROR or lower.
func Error(a ...interface{}) {
	basicLog(levelErr, NotATrace, "", "", a...)
}

// Errorf prints a message if RLOG_LEVEL is set to ERROR or lower, with
// formatting.
func Errorf(format string, a ...interface{}) {
	basicLog(levelErr, NotATrace, format, "", a...)
}

// Critical prints a message if RLOG_LEVEL is set to CRITICAL or lower.
func Critical(a ...interface{}) {
	basicLog(levelCrit, NotATrace, "", "", a...)
}

// Criticalf prints a message if RLOG_LEVEL is set to CRITICAL or lower, with
// formatting.
func Criticalf(format string, a ...interface{}) {
	basicLog(levelCrit, NotATrace, format, "", a...)
}
