package goubus

import (
	"encoding/json"
	"errors"
	"strconv"
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

	streamStr := "false"
	if stream {
		streamStr = "true"
	}
	oneshotStr := "false"
	if oneshot {
		oneshotStr = "true"
	}

	var jsonStr = []byte(`
		{ 
			"jsonrpc": "2.0", 
			"id": ` + strconv.Itoa(u.id) + `, 
			"method": "call", 
			"params": [ 
				"` + u.AuthData.UbusRPCSession + `", 
				"log",
				"read", 
				{ 
					"lines": ` + strconv.Itoa(lines) + `,
					"stream": ` + streamStr + `,
					"oneshot": ` + oneshotStr + `
				} 
			] 
		}`)
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

	var jsonStr = []byte(`
		{ 
			"jsonrpc": "2.0", 
			"id": ` + strconv.Itoa(u.id) + `, 
			"method": "call", 
			"params": [ 
				"` + u.AuthData.UbusRPCSession + `", 
				"log",
				"write", 
				{ 
					"event": "` + event + `"
				} 
			] 
		}`)
	_, err := u.Call(jsonStr)
	return err
}
