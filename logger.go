package main

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"
)

type LogLevel int
const (
	LogLevel_VERBOSE = 5
	LogLevel_DEBUG = 4
	LogLevel_INFO = 3
	LogLevel_WARN = 2
	LogLevel_ERROR = 1
)

type LogMessage struct {
	Timestamp     time.Time
	Caller        string
	Level         LogLevel
	MessageFormat string
	Error         error
}



func LogForwarder(config *Config, msg LogMessage, messageObjects ...interface{}) {
	if config != nil && config.LogLevel < msg.Level {
		return
	}

	type logMessageRaw struct {
		Timestamp      time.Time
		Version        string
		LogLevelLabel  string
		Caller         string
		Level          LogLevel
		Message        string
		Error          error
	}

	jsonMessage := logMessageRaw{
		Timestamp:      msg.Timestamp,
		Version:        GetMyVersion(),
		LogLevelLabel:  "",
		Caller:         msg.Caller,
		Level:          msg.Level,
		Error:          msg.Error,
	}

	if messageObjects == nil || len(messageObjects) == 0 {
		jsonMessage.Message = msg.MessageFormat
	} else {
		jsonMessage.Message = fmt.Sprintf(msg.MessageFormat, messageObjects[:]...)
	}

	if jsonMessage.Timestamp.IsZero() {
		jsonMessage.Timestamp = time.Now()
	}

	switch jsonMessage.Level {
	case LogLevel_VERBOSE:
		jsonMessage.LogLevelLabel = "[VERBOSE]"
	case LogLevel_DEBUG:
		jsonMessage.LogLevelLabel = "[DEBUG  ]"
	case LogLevel_WARN:
		jsonMessage.LogLevelLabel = "[WARN   ]"
	case LogLevel_ERROR:
		jsonMessage.LogLevelLabel = "[ERROR  ]"
	case LogLevel_INFO:
		jsonMessage.LogLevelLabel = "[INFO   ]"
	default:
		panic("Unknown log level " + strconv.FormatInt(int64(msg.Level), 10) + " for message " + fmt.Sprintf("%v", msg.MessageFormat) + " from " + fmt.Sprintf("%v", msg.Caller))
	}

	jsonMessageJsonString, _ := json.Marshal(jsonMessage)
	fmt.Println(string(jsonMessageJsonString))
}