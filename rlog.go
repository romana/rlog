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

import (
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// A few constants, which are used more like flags
const notATrace = -1
const noTraceOutput = -1

// The known log levels
const (
	levelNone = iota
	levelCrit
	levelErr
	levelWarn
	levelInfo
	levelDebug
	levelTrace
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

// filterSpec holds a list of filters. These are applied to the 'caller'
// information of a log message (calling module and file) to see if this
// message should be logged. Different log or trace levels per file can
// therefore be maintained. For log messages this is the log level, for trace
// messages this is going to be the trace level.
type filterSpec struct {
	filters []filter
}

// filter holds filename and level to match logs against log messages.
type filter struct {
	Pattern string
	Level   int
}

// rlogConfig captures the entire configuration of rlog, as supplied by a user,
// for example via environment variables.
type rlogConfig struct {
	logLevel       string // What log level. String, since filters are allowed
	traceLevel     string // What trace level. String, since filters are allowed
	logTimeFormat  string // The time format spec for date/time stamps in output
	logFile        string // Name of logfile
	logStream      string // Name of logstream: stdout, stderr or NONE
	logTimeDate    bool   // Flag to determine if date/time is logged at all
	showCallerInfo bool   // Flag to determine if caller info is logged
}

// The configuration items in rlogConfig are what is supplied by the user (for
// example in environment variables). We interpret this and produce
// pre-processed configuration values, which are stored in those variables
// below.
var settingShowCallerInfo bool   // whether we log caller info
var settingDateTimeFormat string // flags for date/time output
var settingLogFile string        // logfile name

var logWriterStream *log.Logger // the first writer to which output is sent
var logWriterFile *log.Logger   // the second writer to which output is sent
var logFilterSpec *filterSpec   // filters for log messages
var traceFilterSpec *filterSpec // filters for trace messages

// fromString initializes filterSpec from string.
//
// Use the isTraceLevel flag to indicate whether the levels are numeric (for
// trace messages) or are level strings (for log messages).
//
// Format "<filter>,<filter>,[<filter>]..."
//     filter:
//       <pattern=level> | <level>
//     pattern:
//       shell glob to match caller file name
//     level:
//       log or trace level of the logs to enable in matched files.
//
//     Example:
//     - "RLOG_TRACE_LEVEL=3"
//       Just a global trace level of 3 for all files and modules.
//     - "RLOG_TRACE_LEVEL=client.go=1,ip*=5,3"
//       This enables trace level 1 in client.go, level 5 in all files whose
//       names start with 'ip', and level 3 for everyone else.
//     - "RLOG_LOG_LEVEL=DEBUG"
//       Global log level DEBUG for all files and modules.
//     - "RLOG_LOG_LEVEL=client.go=ERROR,INFO,ip*=WARN"
//       ERROR and higher for client.go, WARN or higher for all files whose
//       name starts with 'ip', INFO for everyone else.
func (spec *filterSpec) fromString(s string, isTraceLevels bool, globalLevelDefault int) {
	var globalLevel int = globalLevelDefault
	var levelToken string
	var matchToken string

	fields := strings.Split(s, ",")

	for _, f := range fields {
		var filterLevel int
		var err error
		var ok bool

		// Tokens should contain two elements: The filename and the trace
		// level. If there is only one token then we have to assume that this
		// is the 'global' filter (without filename component).
		tokens := strings.Split(f, "=")
		if len(tokens) == 1 {
			// Global level. We'll store this one for the end, since it needs
			// to sit last in the list of filters (during evaluation in gets
			// checked last).
			matchToken = ""
			levelToken = tokens[0]
		} else if len(tokens) == 2 {
			matchToken = tokens[0]
			levelToken = tokens[1]
		} else {
			// Skip anything else that's malformed
			continue
		}
		if isTraceLevels {
			// The level token should contain a numeric value
			if filterLevel, err = strconv.Atoi(levelToken); err != nil {
				continue
			}
		} else {
			// The level token should contain the name of a log level
			levelToken = strings.ToUpper(levelToken)
			filterLevel, ok = levelNumbers[levelToken]
			if !ok || filterLevel == levelTrace {
				// User not allowed to set trace log levels, so if that or
				// not a known log level then this specification will be
				// ignored.
				continue
			}

		}

		if matchToken == "" {
			// Global level just remembered for now, not yet added
			globalLevel = filterLevel
		} else {
			spec.filters = append(spec.filters, filter{matchToken, filterLevel})
		}
	}

	// Now add the global level, so that later it will be evaluated last.
	// For trace levels we do something extra: There are possibly many trace
	// messages, but most often trace level debugging is fully disabled. We
	// want to optimize this. Therefore, a globalLevel of -1 (no trace levels)
	// isn't stored in the filter chain. If no other trace filters were defined
	// then this means the filter chain is empty, which can be tested very
	// efficiently in the top-level trace functions for an early exit.
	if !isTraceLevels || globalLevel != noTraceOutput {
		spec.filters = append(spec.filters, filter{"", globalLevel})
	}

	return
}

// matchfilters checks if given filename and trace level are accepted
// by any of the filters
func (spec *filterSpec) matchfilters(filename string, level int) bool {

	// If there are no filters then we don't match anything.
	if len(spec.filters) == 0 {
		return false
	}

	// If at least one filter matches.
	for _, filter := range spec.filters {
		if matched, loggit := filter.match(filename, level); matched {
			return loggit
		}
	}

	return false
}

// match checks if given filename and level are matched by
// this filter. Returns two bools: One to indicate whether a filename match was
// made, and the second to indicate whether the message should be logged
// (matched the level).
func (f filter) match(filename string, level int) (bool, bool) {
	var match bool
	if f.Pattern != "" {
		match, _ = filepath.Match(f.Pattern, filepath.Base(filename))
	} else {
		match = true
	}
	if match {
		return true, level <= f.Level
	}

	return false, false
}

// init extracts settings for our logger from environment variables when the
// module is imported and calls actual initialization function with that
// configuration.
func init() {
	var config rlogConfig = rlogConfig{
		logLevel:       os.Getenv("RLOG_LOG_LEVEL"),
		traceLevel:     os.Getenv("RLOG_TRACE_LEVEL"),
		logTimeFormat:  os.Getenv("RLOG_TIME_FORMAT"),
		logFile:        os.Getenv("RLOG_LOG_FILE"),
		logStream:      strings.ToUpper(os.Getenv("RLOG_LOG_STREAM")),
		logTimeDate:    !isTrueBoolString(os.Getenv("RLOG_LOG_NOTIME")),
		showCallerInfo: isTrueBoolString(os.Getenv("RLOG_CALLER_INFO")),
	}
	Initialize(config)
}

// Initialize translates config items into initialized data structures,
// config values and freshly created or opened config files, if necessary.
// This function prepares everything for the fast and efficient processing of
// the actual log functions.
func Initialize(config rlogConfig) {
	settingShowCallerInfo = config.showCallerInfo

	// Initialize filters for trace (by default no trace output) and log levels
	// (by default INFO level).
	logFilterSpec = new(filterSpec)
	traceFilterSpec = new(filterSpec)
	traceFilterSpec.fromString(config.traceLevel, true, noTraceOutput)
	logFilterSpec.fromString(config.logLevel, false, levelInfo)

	// Evaluate the specified date/time format
	settingDateTimeFormat = ""
	if config.logTimeDate {
		// Store the format string for date/time logging. Allowed values are
		// all the constants specified in
		// https://golang.org/src/time/format.go.
		var f string
		switch config.logTimeFormat {
		case "ANSIC":
			f = time.ANSIC
		case "UnixDate":
			f = time.UnixDate
		case "RubyDate":
			f = time.RubyDate
		case "RFC822":
			f = time.RFC822
		case "RFC822Z":
			f = time.RFC822Z
		case "RFC1123":
			f = time.RFC1123
		case "RFC1123Z":
			f = time.RFC1123Z
		case "RFC3339":
			f = time.RFC3339
		case "RFC3339Nano":
			f = time.RFC3339Nano
		case "Kitchen":
			f = time.Kitchen
		default:
			if config.logTimeFormat != "" {
				f = config.logTimeFormat
			} else {
				f = time.RFC3339
			}
		}
		settingDateTimeFormat = f + " "
	}

	// By default we log to stderr...
	// Evaluating whether a different log stream should be used.
	// By default (if flag is not set) we want to log date and time.
	// Note that in our log writers we disable date/time loggin, since we will
	// take care of producing this ourselves.
	if config.logStream == "STDOUT" {
		logWriterStream = log.New(os.Stdout, "", 0)
	} else if config.logStream == "NONE" {
		logWriterStream = nil
	} else {
		logWriterStream = log.New(os.Stderr, "", 0)
	}

	// ... but if requested we'll also create and/or append to a logfile
	if config.logFile == "" {
		logWriterFile = nil
	} else {
		newLogFile, err := os.OpenFile(config.logFile,
			os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
		if err == nil {
			logWriterFile = log.New(newLogFile, "", 0)
		}
	}
}

// SetOutput re-wires the log output to a new io.Writer. By default rlog
// logs to os.Stderr, but this function can be used to direct the output
// somewhere else. If output to two destinations was specified via environment
// variables then this will change it back to just one output.
func SetOutput(writer io.Writer) {
	// Use the stored date/time flag settings
	logWriterStream = log.New(writer, "", 0)
	logWriterFile = nil
}

// isTrueBoolString tests a string to see if it represents a 'true' value.
// The ParseBool function unfortunately doesn't recognize 'y' or 'yes', which
// is why we added that test here as well.
func isTrueBoolString(str string) bool {
	str = strings.ToUpper(str)
	if str == "Y" || str == "YES" {
		return true
	}
	if isTrue, err := strconv.ParseBool(str); err == nil && isTrue {
		return true
	}
	return false
}

// basicLog is called by all the 'level' log functions.
// It checks what is configured to be included in the log message,
// decorates it accordingly and assembles the entire line. It then
// uses the standard log package to finally output the message.
func basicLog(logLevel int, traceLevel int, format string, prefixAddition string, a ...interface{}) {
	// Extract information about the caller of the log function, if requested.
	var callingFuncName string = ""
	var moduleAndFileName string = ""
	pc, fullFilePath, line, ok := runtime.Caller(2)
	if ok {
		callingFuncName = runtime.FuncForPC(pc).Name()
		// We only want to print or examine file and package name, so use the
		// last two elements of the full path. The path package deals with
		// different path formats on different systems, so we use that instead
		// of just string-split.
		dirPath, fileName := path.Split(fullFilePath)
		var moduleName string = ""
		if dirPath != "" {
			dirPath = dirPath[:len(dirPath)-1]
			dirPath, moduleName = path.Split(dirPath)
		}
		moduleAndFileName = moduleName + "/" + fileName
	}

	// Perform tests to see if we should log this message.
	var allowLog bool
	if traceLevel == notATrace {
		if logFilterSpec.matchfilters(moduleAndFileName, logLevel) {
			allowLog = true
		}
	} else {
		if traceFilterSpec.matchfilters(moduleAndFileName, traceLevel) {
			allowLog = true
		}
	}
	if !allowLog {
		return
	}

	callerInfo := ""
	if settingShowCallerInfo {
		callerInfo = fmt.Sprintf("[%s:%d (%s)] ", moduleAndFileName,
			line, callingFuncName)
	}

	// Assemble the actual log line
	var msg string
	if format != "" {
		msg = fmt.Sprintf(format, a...)
	} else {
		msg = fmt.Sprintln(a...)
	}
	levelDecoration := levelStrings[logLevel] + prefixAddition
	logLine := fmt.Sprintf("%s%-9s: %s%s",
		time.Now().Format(settingDateTimeFormat), levelDecoration,
		callerInfo, msg)
	if logWriterStream != nil {
		logWriterStream.Printf(logLine)
	}
	if logWriterFile != nil {
		logWriterFile.Printf(logLine)
	}
}

// Trace is for low level tracing of activities. It takes an additional 'level'
// parameter. The RLOG_TRACE_LEVEL variable is used to determine which levels
// of trace message are output: Every message with a level lower or equal to
// what is specified in RLOG_TRACE_LEVEL. If RLOG_TRACE_LEVEL is not defined at
// all then no trace messages are printed.
func Trace(traceLevel int, a ...interface{}) {
	// There are possibly many trace messages. If trace logging isn't enabled
	// then we want to get out of here as quickly as possible.
	if len(traceFilterSpec.filters) > 0 {
		prefixAddition := fmt.Sprintf("(%d)", traceLevel)
		basicLog(levelTrace, traceLevel, "", prefixAddition, a...)
	}
}

// Tracef prints trace messages, with formatting.
func Tracef(traceLevel int, format string, a ...interface{}) {
	// There are possibly many trace messages. If trace logging isn't enabled
	// then we want to get out of here as quickly as possible.
	if len(traceFilterSpec.filters) > 0 {
		prefixAddition := fmt.Sprintf("(%d)", traceLevel)
		basicLog(levelTrace, traceLevel, format, prefixAddition, a...)
	}
}

// Debug prints a message if RLOG_LEVEL is set to DEBUG.
func Debug(a ...interface{}) {
	basicLog(levelDebug, notATrace, "", "", a...)
}

// Debugf prints a message if RLOG_LEVEL is set to DEBUG, with formatting.
func Debugf(format string, a ...interface{}) {
	basicLog(levelDebug, notATrace, format, "", a...)
}

// Info prints a message if RLOG_LEVEL is set to INFO or lower.
func Info(a ...interface{}) {
	basicLog(levelInfo, notATrace, "", "", a...)
}

// Infof prints a message if RLOG_LEVEL is set to INFO or lower, with
// formatting.
func Infof(format string, a ...interface{}) {
	basicLog(levelInfo, notATrace, format, "", a...)
}

// Println prints a message if RLOG_LEVEL is set to INFO or lower.
// Println shouldn't be used except for backward compatibility
// with standard log package, directly using Info is preferred way.
func Println(a ...interface{}) {
	basicLog(levelInfo, notATrace, "", "", a...)
}

// Printf prints a message if RLOG_LEVEL is set to INFO or lower, with
// formatting.
// Printf shouldn't be used except for backward compatibility
// with standard log package, directly using Infof is preferred way.
func Printf(format string, a ...interface{}) {
	basicLog(levelInfo, notATrace, format, "", a...)
}

// Warn prints a message if RLOG_LEVEL is set to WARN or lower.
func Warn(a ...interface{}) {
	basicLog(levelWarn, notATrace, "", "", a...)
}

// Warnf prints a message if RLOG_LEVEL is set to WARN or lower, with
// formatting.
func Warnf(format string, a ...interface{}) {
	basicLog(levelWarn, notATrace, format, "", a...)
}

// Error prints a message if RLOG_LEVEL is set to ERROR or lower.
func Error(a ...interface{}) {
	basicLog(levelErr, notATrace, "", "", a...)
}

// Errorf prints a message if RLOG_LEVEL is set to ERROR or lower, with
// formatting.
func Errorf(format string, a ...interface{}) {
	basicLog(levelErr, notATrace, format, "", a...)
}

// Critical prints a message if RLOG_LEVEL is set to CRITICAL or lower.
func Critical(a ...interface{}) {
	basicLog(levelCrit, notATrace, "", "", a...)
}

// Criticalf prints a message if RLOG_LEVEL is set to CRITICAL or lower, with
// formatting.
func Criticalf(format string, a ...interface{}) {
	basicLog(levelCrit, notATrace, format, "", a...)
}
