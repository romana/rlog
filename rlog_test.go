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
	"bufio"
	"fmt"
	"os"
	"path"
	"runtime"
	"testing"
	"time"
)

var logfile string
var removeLogfile bool = false
var fixedLogfileName bool = true

// setup is called at the start of each test and prepares a new log file. It
// also returns a new configuration, which can be used by this test.
func setup() rlogConfig {
	if fixedLogfileName {
		logfile = "/tmp/rlog-test.log"
	} else {
		logfile = fmt.Sprintf("/tmp/rlog-test-%d.log", time.Now().UnixNano())
	}
	os.Remove(logfile)

	return rlogConfig{
		logLevel:       "",
		traceLevel:     "",
		logTimeFormat:  "",
		logFile:        logfile,
		logStream:      "NONE",
		logTimeDate:    false,
		showCallerInfo: false,
	}
}

// cleanup is called at the end of each test.
func cleanup() {
	if removeLogfile {
		os.Remove(logfile)
	}
}

// fileMatch compares entries in the logfile with expected entries provided as
// a list of strings (one for each line)
func fileMatch(t *testing.T, checkLines []string, timeLayout string) {
	file, err := os.Open(logfile)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	i := 0
	for scanner.Scan() {
		line := scanner.Text()
		if timeLayout != "" {
			dateTime := line[:len(timeLayout)]
			line = line[len(timeLayout)+1:]
			_, err := time.Parse(timeLayout, dateTime)
			if err != nil {
				t.Fatal(err)
				t.Fatalf("Incorrect date/time format.\nSHOULD: %s\nIS:     %s\n", timeLayout, dateTime)
			}
		}
		if i >= len(checkLines) {
			t.Fatal("Not enough lines provided in checkLines.")
		}
		if line != checkLines[i] {
			t.Fatalf("Log line %d does not match check line.\nSHOULD: %s\nIS:     %s\n", i, checkLines[i], line)
		}
		i++
	}
	if len(checkLines) > i {
		t.Fatalf("Only %d of %d checklines found in output file.", i, len(checkLines))
	}
	if i == 0 {
		t.Fatal("No input scanned")
	}
}

func TestLogLevels(t *testing.T) {
	conf := setup()
	defer cleanup()

	conf.logLevel = "DEBUG"
	Initialize(conf)

	Debug("Test Debug")
	Info("Test Info")
	Warn("Test Warning")
	Error("Test Error")
	Critical("Test Critical")

	checkLines := []string{
		"DEBUG    : Test Debug",
		"INFO     : Test Info",
		"WARN     : Test Warning",
		"ERROR    : Test Error",
		"CRITICAL : Test Critical",
	}
	fileMatch(t, checkLines, "")
}

func TestLogLevelsLimited(t *testing.T) {
	conf := setup()
	defer cleanup()

	conf.logLevel = "WARN"
	conf.traceLevel = "3"
	Initialize(conf)

	Debug("Test Debug")
	Info("Test Info")
	Warn("Test Warning")
	Error("Test Error")
	Critical("Test Critical")
	Trace(1, "Trace 1")
	Trace(2, "Trace 2")
	Trace(3, "Trace 3")
	Trace(4, "Trace 4")
	checkLines := []string{
		"WARN     : Test Warning",
		"ERROR    : Test Error",
		"CRITICAL : Test Critical",
		"TRACE(1) : Trace 1",
		"TRACE(2) : Trace 2",
		"TRACE(3) : Trace 3",
	}
	fileMatch(t, checkLines, "")
}

func TestLogFormatted(t *testing.T) {
	conf := setup()
	defer cleanup()

	conf.logLevel = "DEBUG"
	conf.traceLevel = "1"
	Initialize(conf)

	Debugf("Test Debug %d", 123)
	Infof("Test Info %d", 123)
	Warnf("Test Warning %d", 123)
	Errorf("Test Error %d", 123)
	Criticalf("Test Critical %d", 123)
	Tracef(1, "Trace 1 %d", 123)
	checkLines := []string{
		"DEBUG    : Test Debug 123",
		"INFO     : Test Info 123",
		"WARN     : Test Warning 123",
		"ERROR    : Test Error 123",
		"CRITICAL : Test Critical 123",
		"TRACE(1) : Trace 1 123",
	}
	fileMatch(t, checkLines, "")
}

func TestLogTimestamp(t *testing.T) {
	conf := setup()
	defer cleanup()

	conf.logTimeFormat = "ANSIC"
	conf.logTimeDate = true
	Initialize(conf)

	Info("Test Info")
	checkLines := []string{
		"INFO     : Test Info",
	}
	fileMatch(t, checkLines, time.ANSIC)
}

func TestLogCallerInfo(t *testing.T) {
	// In this test we manually figure out the caller info, which we should
	// display
	conf := setup()
	defer cleanup()

	conf.showCallerInfo = true
	Initialize(conf)

	Info("Test Info")
	pc, fullFilePath, line, _ := runtime.Caller(0)
	line-- // The log was called in the line before, so... -1
	// The following lines simply format the caller info in the way that it
	// should be formatted by rlog
	callingFuncName := runtime.FuncForPC(pc).Name()
	dirPath, fileName := path.Split(fullFilePath)
	var moduleName string = ""
	if dirPath != "" {
		dirPath = dirPath[:len(dirPath)-1]
		dirPath, moduleName = path.Split(dirPath)
	}
	moduleAndFileName := moduleName + "/" + fileName
	shouldLine := fmt.Sprintf("INFO     : [%s:%d (%s)] Test Info",
		moduleAndFileName, line, callingFuncName)

	checkLines := []string{shouldLine}
	fileMatch(t, checkLines, "")
}

func TestLogLevelsFiltered(t *testing.T) {
	conf := setup()
	defer cleanup()

	conf.logLevel = "rlog_test.go=WARN"
	conf.traceLevel = "foobar.go=2" // should not see any of those
	Initialize(conf)

	Debug("Test Debug")
	Info("Test Info")
	Warn("Test Warning")
	Error("Test Error")
	Critical("Test Critical")
	Trace(1, "Trace 1")
	Trace(2, "Trace 2")
	Trace(3, "Trace 3")
	Trace(4, "Trace 4")
	checkLines := []string{
		"WARN     : Test Warning",
		"ERROR    : Test Error",
		"CRITICAL : Test Critical",
	}
	fileMatch(t, checkLines, "")
}
