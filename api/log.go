package api

import (
	"github.com/honeybbq/goubus/types"
)

// Log service and method constants
const (
	ServiceLog = "log"

	LogMethodRead  = "read"
	LogMethodWrite = "write"
)

// Log parameter constants
const (
	LogParamLines   = "lines"
	LogParamStream  = "stream"
	LogParamOneshot = "oneshot"
	LogParamEvent   = "event"
)

// ReadLog reads system log entries.
func ReadLog(caller types.Transport, lines int, stream bool, oneshot bool) (*types.Log, error) {
	params := map[string]any{
		LogParamLines:   lines,
		LogParamStream:  stream,
		LogParamOneshot: oneshot,
	}

	resp, err := caller.Call(ServiceLog, LogMethodRead, params)
	if err != nil {
		return nil, err
	}

	var log types.Log
	if err := resp.Unmarshal(&log); err != nil {
		return nil, err
	}
	return &log, nil
}

// WriteLog writes an entry to the system log.
func WriteLog(caller types.Transport, event string) error {
	params := map[string]any{
		LogParamEvent: event,
	}

	_, err := caller.Call(ServiceLog, LogMethodWrite, params)
	return err
}
