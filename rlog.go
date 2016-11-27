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
	"log"
	"os"
	"path"
	"runtime"
	"strconv"
	"strings"
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

// Rlog is controlled via environment variables. Those things won't change on
// us. Therefore, we can look them up once and store them in module level
// global variables.
var settingTraceLevel int = -1        // -1 indicates 'not set' -> no tracing
var settingLogLevel int = levelInfo   // by default we log INFO or higher
var settingGetCallerInfo bool = false // whether we log info about calling function

// init extracts settings for our logger from environment variables when the
// module is imported.
func init() {
	var err error

	logLevelEnv := strings.ToUpper(os.Getenv("RLOG_LOG_LEVEL"))
	callerInfoEnv := strings.ToUpper(os.Getenv("RLOG_CALLER_INFO"))
	traceLevelEnv := strings.ToUpper(os.Getenv("RLOG_TRACE_LEVEL"))

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

	// Evaluating the caller info variable. ParseBool unfortunately doesn't
	// recognize 'y', or 'yes' as 'true' values. So we are checking for those
	// manually.
	var getCallerInfo bool
	getCallerInfo, err = strconv.ParseBool(callerInfoEnv)
	if (err == nil && getCallerInfo) || callerInfoEnv == "Y" || callerInfoEnv == "YES" {
		settingGetCallerInfo = true
	}

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
func basicLog(logLevel int, format string, prefixAddition string, a ...interface{}) {
	// Should we even be logging this?
	// Note that Trace is a special case. We will never get a message with
	// logTrace unless tracing was specifically enabled.
	if logLevel < settingLogLevel && logLevel != levelTrace {
		return
	}

	// Extract information about the caller of the log function, if requested.
	callerInfo := ""
	if settingGetCallerInfo {
		if pc, fullFilePath, line, ok := runtime.Caller(2); ok {
			callingFuncName := runtime.FuncForPC(pc).Name()
			// We only want to print file and package name, so use the last two
			// elements of the full path. The path package deals with different
			// path formats on different systems, so we use that instead of
			// just string-split.
			dirPath, fileName := path.Split(fullFilePath)
			var moduleName string
			if dirPath != "" {
				dirPath = dirPath[:len(dirPath)-1]
				dirPath, moduleName = path.Split(dirPath)
			} else {
				moduleName = ""
			}

			callerInfo = fmt.Sprintf("[%s/%s:%d (%s)] ", moduleName, fileName,
				line, callingFuncName)
		}
	}

	// Assemble the actual log line
	var msg string
	if format != "" {
		msg = fmt.Sprintf(format, a...)
	} else {
		msg = fmt.Sprintln(a...)
	}
	levelDecoration := levelStrings[logLevel] + prefixAddition
	log.Printf("%-9s: %s%s", levelDecoration, callerInfo, msg)
}

// Trace is for low level tracing of activities. It takes an additional 'level'
// parameter. The RLOG_TRACE_LEVEL variable is used to determine which levels
// of trace message are output: Every message with a level lower or equal to
// what is specified in RLOG_TRACE_LEVEL. If RLOG_TRACE_LEVEL is not defined at
// all then no trace messages are printed.
func Trace(traceLevel int, a ...interface{}) {
	if traceLevel <= settingTraceLevel && traceLevel >= 0 {
		prefixAddition := fmt.Sprintf("(%d)", traceLevel)
		basicLog(levelTrace, "", prefixAddition, a...)
	}
}

// Tracef prints trace messages, with formatting.
func Tracef(traceLevel int, format string, a ...interface{}) {
	if traceLevel <= settingTraceLevel && traceLevel >= 0 {
		prefixAddition := fmt.Sprintf("(%d)", traceLevel)
		basicLog(levelTrace, format, prefixAddition, a...)
	}
}

// Debug prints a message if RLOG_LEVEL is set to DEBUG.
func Debug(a ...interface{}) {
	basicLog(levelDebug, "", "", a...)
}

// Debugf prints a message if RLOG_LEVEL is set to DEBUG, with formatting.
func Debugf(format string, a ...interface{}) {
	basicLog(levelDebug, format, "", a...)
}

// Info prints a message if RLOG_LEVEL is set to INFO or lower.
func Info(a ...interface{}) {
	basicLog(levelInfo, "", "", a...)
}

// Infof prints a message if RLOG_LEVEL is set to INFO or lower, with
// formatting.
func Infof(format string, a ...interface{}) {
	basicLog(levelInfo, format, "", a...)
}

// Warn prints a message if RLOG_LEVEL is set to WARN or lower.
func Warn(a ...interface{}) {
	basicLog(levelWarn, "", "", a...)
}

// Warnf prints a message if RLOG_LEVEL is set to WARN or lower, with
// formatting.
func Warnf(format string, a ...interface{}) {
	basicLog(levelWarn, format, "", a...)
}

// Error prints a message if RLOG_LEVEL is set to ERROR or lower.
func Error(a ...interface{}) {
	basicLog(levelErr, "", "", a...)
}

// Errorf prints a message if RLOG_LEVEL is set to ERROR or lower, with
// formatting.
func Errorf(format string, a ...interface{}) {
	basicLog(levelErr, format, "", a...)
}

// Critical prints a message if RLOG_LEVEL is set to CRITICAL or lower.
func Critical(a ...interface{}) {
	basicLog(levelCrit, "", "", a...)
}

// Criticalf prints a message if RLOG_LEVEL is set to CRITICAL or lower, with
// formatting.
func Criticalf(format string, a ...interface{}) {
	basicLog(levelCrit, format, "", a...)
}
