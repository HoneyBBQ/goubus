package goubus

import (
	"encoding/json"
)

type UbusRcInitRequest struct {
	Name   string `json:"name"`
	Action string `json:"action"`
}

type UbusRcListResponse struct {
	Start   int  `json:"start"`
	Running bool `json:"running"`
	Enabled bool `json:"enabled"`
}

// rcList retrieves a list of available init scripts.
func (u *Client) rcList() (map[string]UbusRcListResponse, error) {
	errLogin := u.LoginCheck()
	if errLogin != nil {
		return nil, errLogin
	}

	jsonStr := u.buildUbusCall(ServiceRC, MethodList, nil)
	call, err := u.Call(jsonStr)
	if err != nil {
		return nil, err
	}
	ubusData := make(map[string]UbusRcListResponse)

	ubusDataByte, err := json.Marshal(call.Result.([]interface{})[1])
	if err != nil {
		return nil, ErrDataParsingError
	}
	json.Unmarshal(ubusDataByte, &ubusData)
	return ubusData, nil
}

// rcInit controls an init script (start, stop, restart, etc.).
func (u *Client) rcInit(request UbusRcInitRequest) error {
	errLogin := u.LoginCheck()
	if errLogin != nil {
		return errLogin
	}

	jsonStr := u.buildUbusCall(ServiceRC, MethodInit, request)
	_, err := u.Call(jsonStr)
	return err
}
