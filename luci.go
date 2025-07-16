package goubus

import (
	"time"
)

func (u *Client) luciGetTime() (time.Time, error) {
	jsonStr := u.buildUbusCall(ServiceLuci, MethodGetTime, nil)
	call, err := u.Call(jsonStr)
	if err != nil {
		return time.Time{}, err
	}
	resultArray := call.Result.([]interface{})
	resultMap, ok := resultArray[1].(map[string]interface{})
	if !ok {
		return time.Time{}, ErrLuciTimeError
	}

	timeFloat, ok := resultMap["result"].(float64)
	if !ok {
		return time.Time{}, ErrLuciTimeError
	}

	return time.Unix(int64(timeFloat), 0), nil
}

func (u *Client) luciSetTime(time time.Time) error {
	jsonStr := u.buildUbusCall(ServiceLuci, MethodSetTime, map[string]interface{}{
		ParamLocaltime: time.Unix(),
	})
	_, err := u.Call(jsonStr)
	return err
}
