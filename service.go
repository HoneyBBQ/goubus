package goubus

import (
	"encoding/json"
	"errors"
)

type Respawn struct {
	Threshold int `json:"threshold"`
	Timeout   int `json:"timeout"`
	Retries   int `json:"retries"`
}

// ServiceInstance represents an instance of a service.
type ServiceInstance struct {
	Running     bool     `json:"running"`
	Command     []string `json:"command"`
	TermTimeout int      `json:"term_timeout"`
	ExitCode    int      `json:"exit_code"`
	Respawn     Respawn  `json:"respawn"`
}

type ServiceInstanceList struct {
	Instances []ServiceInstance `json:"instances"`
}

// ServiceStatus represents the status of a service.
type ServiceStatus struct {
	Service ServiceInstanceList `json:"service"`
}

// ServiceListResponse represents the response from service list operations.
type ServiceListResponse map[string]interface{}

type ServiceListRequest struct {
	Name    string `json:"name"`
	Verbose bool   `json:"verbose"`
}

type ServiceActionRequest struct {
	Name string `json:"name"`
}

// startService starts a system service using rc.init.
func (u *Client) startService(serviceName string) error {
	return u.rcInit(UbusRcInitRequest{
		Name:   serviceName,
		Action: "start",
	})
}

// stopService stops a system service using rc.init.
func (u *Client) stopService(serviceName string) error {
	return u.rcInit(UbusRcInitRequest{
		Name:   serviceName,
		Action: "stop",
	})
}

// restartService restarts a system service using rc.init.
func (u *Client) restartService(serviceName string) error {
	return u.rcInit(UbusRcInitRequest{
		Name:   serviceName,
		Action: "restart",
	})
}

// getServiceList retrieves the status of services.
func (u *Client) getServiceList(request ServiceListRequest) (ServiceListResponse, error) {
	errLogin := u.LoginCheck()
	if errLogin != nil {
		return ServiceListResponse{}, errLogin
	}

	params := make(map[string]interface{})
	if request.Name != "" {
		params["name"] = request.Name
	}
	if request.Verbose {
		params["verbose"] = true
	}

	jsonStr := u.buildUbusCall("service", "list", params)
	call, err := u.Call(jsonStr)
	if err != nil {
		return ServiceListResponse{}, err
	}
	ubusData := ServiceListResponse{}

	ubusDataByte, err := json.Marshal(call.Result.([]interface{})[1])
	if err != nil {
		return ServiceListResponse{}, errors.New("data error")
	}
	json.Unmarshal(ubusDataByte, &ubusData)
	return ubusData, nil
}
