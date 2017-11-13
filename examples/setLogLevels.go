package main

import (
	"errors"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"

	"github.com/romana/rlog"
	"gopkg.in/gin-gonic/gin.v1"
)

// It is necessary to have a rlog configuration file with entries for both: RLOG_LOG_LEVEL and RLOG_TRACE_LEVEL
// the const variables were copied from rlog
//
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

// Path to the config file. Defined here for standalone example purpose
const (
	rlogConfigFile      = "/tmp/rlog.conf"
	rlogConfigFileUmask = 0644
)

// LogConfHandler is a handler function for the gin-gonic framework to change trace and log level
func LogConfHandler(c *gin.Context) {

	// get HTTP GET params
	level := c.Query("level")
	trace := c.Query("trace")

	traceInt, err := strconv.Atoi(trace)

	// continue if trace is an integer
	if err == nil {
		err := setGlobalLogConf(level, traceInt)

		if err != nil {
			c.String(http.StatusInternalServerError, err.Error())
			return
		}
	}

	c.String(http.StatusOK, "setting log level to "+level+" trace "+trace)
}

// setGlobalLogConf to change logging settings while running
func setGlobalLogConf(level string, trace int) error {

	// check if specified log level is within allowed values
	if _, ok := levelNumbers[level]; ok {
		// check for config file
		configFile, err := ioutil.ReadFile(rlogConfigFile)
		if err != nil {
			return errors.New("could not read config file: " + err.Error())
		}

		// replace config lines
		lines := strings.Split(string(configFile), "\n")

		// check for pre-existing values in config, override or append
		hasLogLevelEntry := false
		hasTraceLevelEntry := false

		for i, line := range lines {
			if strings.Contains(line, "RLOG_LOG_LEVEL") {
				lines[i] = "RLOG_LOG_LEVEL = " + level
				hasLogLevelEntry = true
				continue
			}

			if strings.Contains(line, "RLOG_TRACE_LEVEL") {
				lines[i] = "RLOG_TRACE_LEVEL = " + strconv.Itoa(trace)
				hasTraceLevelEntry = true
			}
		}

		// append new options if neccessary
		if !hasLogLevelEntry {
			lines = append(lines, "RLOG_LOG_LEVEL = "+level)
		}
		if !hasTraceLevelEntry {
			lines = append(lines, "RLOG_TRACE_LEVEL = "+strconv.Itoa(trace))
		}

		output := strings.Join(lines, "\n")
		err = ioutil.WriteFile(rlogConfigFile, []byte(output), rlogConfigFileUmask)

		if err != nil {
			return errors.New("could not replace config file: " + err.Error())

		}

		// set new config
		rlog.SetConfFile(rlogConfigFile)

		return nil
	}

	return errors.New("invalid value for level, must be valid rlog log level")
}

func main() {
	// check file contents of rlogConfFile to see results e.g. with: "cat /tmp/rlog.conf" before and after
	setGlobalLogConf("DEBUG", 1)
	rlog.Trace(3, "this will not appear in the log")
	setGlobalLogConf("DEBUG", 3)
	rlog.Trace(3, "this will appear in the log")
}
