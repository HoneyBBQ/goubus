package goubus

import (
	"encoding/json"
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
		Action: ActionStart,
	})
}

// stopService stops a system service using rc.init.
func (u *Client) stopService(serviceName string) error {
	return u.rcInit(UbusRcInitRequest{
		Name:   serviceName,
		Action: ActionStop,
	})
}

// restartService restarts a system service using rc.init.
func (u *Client) restartService(serviceName string) error {
	return u.rcInit(UbusRcInitRequest{
		Name:   serviceName,
		Action: ActionRestart,
	})
}

// reloadService reloads a system service using rc.init.
func (u *Client) reloadService(serviceName string) error {
	return u.rcInit(UbusRcInitRequest{
		Name:   serviceName,
		Action: ActionReload,
	})
}

// enableService enables a system service using rc.init.
func (u *Client) enableService(serviceName string) error {
	return u.rcInit(UbusRcInitRequest{
		Name:   serviceName,
		Action: ActionEnable,
	})
}

// disableService disables a system service using rc.init.
func (u *Client) disableService(serviceName string) error {
	return u.rcInit(UbusRcInitRequest{
		Name:   serviceName,
		Action: ActionDisable,
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
		params[ParamName] = request.Name
	}
	if request.Verbose {
		params[ParamVerbose] = true
	}

	jsonStr := u.buildUbusCall(ServiceService, MethodList, params)
	call, err := u.Call(jsonStr)
	if err != nil {
		return ServiceListResponse{}, err
	}
	ubusData := ServiceListResponse{}

	ubusDataByte, err := json.Marshal(call.Result.([]interface{})[1])
	if err != nil {
		return ServiceListResponse{}, ErrDataParsingError
	}
	json.Unmarshal(ubusDataByte, &ubusData)
	return ubusData, nil
}
