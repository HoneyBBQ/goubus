package goubus

import (
	"fmt"
	"strings"
)

type UbusUciRequestGeneric struct {
	Config  string `json:"config"`
	Section string `json:"section,omitempty"`
	Option  string `json:"option,omitempty"`
	Type    string `json:"type,omitempty"`
	Match   string `json:"match,omitempty"`
	Name    string `json:"name,omitempty"`
}

type UbusUciRequest struct {
	UbusUciRequestGeneric
	Values map[string]string `json:"values,omitempty"`
}

type UbusUciGetRequest struct {
	UbusUciRequestGeneric
}

type UbusUciGetResponse struct {
	Value  string            `json:"value"`
	Values map[string]string `json:"values"`
}

func (u *Client) uciGet(id int, request UbusUciGetRequest) (UbusUciGetResponse, error) {
	errLogin := u.LoginCheck()
	if errLogin != nil {
		return UbusUciGetResponse{}, errLogin
	}

	jsonStr := u.buildUbusCallWithID(id, ServiceUCI, MethodGet, request)
	call, err := u.Call(jsonStr)
	if err != nil {
		return UbusUciGetResponse{}, err
	}
	if len(call.Result.([]interface{})) < 2 {
		return UbusUciGetResponse{}, NewError(ErrorCodeInvalidResponse, "invalid uci get response")
	}

	// ubus 'uci get' can return either a single value or a map of values
	// We need to handle both cases
	var ubusData UbusUciGetResponse
	resultData := call.Result.([]interface{})[1]

	if value, ok := resultData.(string); ok {
		ubusData.Value = value
	} else if values, ok := resultData.(map[string]interface{}); ok {
		// The result is a section, so we get a map of options
		ubusData.Values = make(map[string]string)
		for k, v := range values {
			// Convert interface{} to string
			if val, ok := v.(string); ok {
				ubusData.Values[k] = val
			} else if val, ok := v.([]interface{}); ok {
				// Handle list options, join them into a space-separated string
				var strVals []string
				for _, item := range val {
					strVals = append(strVals, item.(string))
				}
				ubusData.Values[k] = strings.Join(strVals, " ")
			}
		}
	} else {
		return UbusUciGetResponse{}, NewError(ErrorCodeUnexpectedFormat, "unexpected type for uci get result")
	}

	return ubusData, nil
}

func (u *Client) uciSet(id int, request UbusUciRequest) error {
	errLogin := u.LoginCheck()
	if errLogin != nil {
		return errLogin
	}

	jsonStr := u.buildUbusCallWithID(id, ServiceUCI, MethodSet, request)
	_, err := u.Call(jsonStr)
	return err
}

func (u *Client) uciAdd(id int, request UbusUciRequest) error {
	errLogin := u.LoginCheck()
	if errLogin != nil {
		return errLogin
	}

	jsonStr := u.buildUbusCallWithID(id, ServiceUCI, MethodAdd, request)
	_, err := u.Call(jsonStr)
	return err
}

func (u *Client) uciAddToList(id int, request UbusUciRequest) error {
	errLogin := u.LoginCheck()
	if errLogin != nil {
		return errLogin
	}
	// For add_list, 'values' should contain the single key-value to add.
	// The key is the option, and the value is the string to add to the list.

	jsonStr := u.buildUbusCallWithID(id, ServiceUCI, MethodAddList, request)
	_, err := u.Call(jsonStr)
	return err
}

func (u *Client) uciDelete(id int, request UbusUciRequestGeneric) error {
	errLogin := u.LoginCheck()
	if errLogin != nil {
		return errLogin
	}

	jsonStr := u.buildUbusCallWithID(id, ServiceUCI, MethodDelete, request)
	_, err := u.Call(jsonStr)
	return err
}

func (u *Client) uciCommit(id int, request UbusUciRequestGeneric) error {
	errLogin := u.LoginCheck()
	if errLogin != nil {
		return errLogin
	}

	jsonStr := u.buildUbusCallWithID(id, ServiceUCI, MethodCommit, request)
	_, err := u.Call(jsonStr)
	return err
}

// UciDeleteFromList removes a specific value from a UCI list.
// It works by getting the list, finding the index of the value, and deleting by index.
func (u *Client) uciDeleteFromList(id int, config, section, option, value string) error {
	// 1. Get the current list
	getRequest := UbusUciGetRequest{
		UbusUciRequestGeneric: UbusUciRequestGeneric{
			Config:  config,
			Section: section,
			Option:  option,
		},
	}
	getResponse, err := u.uciGet(id, getRequest)
	if err != nil {
		return fmt.Errorf("could not get list to delete from: %w", err)
	}

	// 2. Find the index of the value
	list := strings.Split(getResponse.Value, " ")
	indexToDelete := -1
	for i, item := range list {
		if item == value {
			indexToDelete = i
			break
		}
	}

	if indexToDelete == -1 {
		return fmt.Errorf("value '%s' not found in list for option '%s'", value, option)
	}

	// 3. Delete by index
	// UCI uses a zero-based index for list deletion.
	// The option needs to be formatted as "option[index]"
	deleteRequest := UbusUciRequestGeneric{
		Config:  config,
		Section: section,
		Option:  fmt.Sprintf("%s[%d]", option, indexToDelete),
	}

	return u.uciDelete(id, deleteRequest)
}
