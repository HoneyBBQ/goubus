package goubus

import (
	"encoding/json"
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
		ParamLines:   lines,
		ParamStream:  stream,
		ParamOneshot: oneshot,
	}

	jsonStr := u.buildUbusCall(ServiceLog, MethodRead, params)
	call, err := u.Call(jsonStr)
	if err != nil {
		return UbusLog{}, err
	}

	ubusData := UbusLog{}
	ubusDataByte, err := json.Marshal(call.Result.([]interface{})[1])
	if err != nil {
		return UbusLog{}, ErrDataParsingError
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
		ParamEvent: event,
	}

	jsonStr := u.buildUbusCall(ServiceLog, MethodWrite, params)
	_, err := u.Call(jsonStr)
	return err
}
