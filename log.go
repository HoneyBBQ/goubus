package goubus

import (
	"encoding/json"
	"errors"
)

type UbusLog struct {
	Log []UbusLogData
}

type UbusLogData struct {
	Msg      string
	ID       int
	Priority int
	Source   int
	Time     int
}

// LogRead reads system log entries.
func (u *Client) logRead(lines int, stream bool, oneshot bool) (UbusLog, error) {
	errLogin := u.LoginCheck()
	if errLogin != nil {
		return UbusLog{}, errLogin
	}

	params := map[string]interface{}{
		"lines":   lines,
		"stream":  stream,
		"oneshot": oneshot,
	}

	jsonStr := u.buildUbusCall("log", "read", params)
	call, err := u.Call(jsonStr)
	if err != nil {
		return UbusLog{}, err
	}

	ubusData := UbusLog{}
	ubusDataByte, err := json.Marshal(call.Result.([]interface{})[1])
	if err != nil {
		return UbusLog{}, errors.New("data error")
	}
	json.Unmarshal(ubusDataByte, &ubusData)
	return ubusData, nil
}

// LogWrite writes an entry to the system log.
func (u *Client) logWrite(event string) error {
	errLogin := u.LoginCheck()
	if errLogin != nil {
		return errLogin
	}

	params := map[string]interface{}{
		"event": event,
	}

	jsonStr := u.buildUbusCall("log", "write", params)
	_, err := u.Call(jsonStr)
	return err
}
