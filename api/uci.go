package api

import (
	"encoding/json"
	"errors"
	"strconv"
	"strings"

	"github.com/honeybbq/goubus/errdefs"
	"github.com/honeybbq/goubus/types"
)

// UCI service constants
const (
	ServiceUCI            = "uci"
	UciMethodGet          = "get"
	UciMethodSet          = "set"
	UciMethodAdd          = "add"
	UciMethodDelete       = "delete"
	UciMethodCommit       = "commit"
	UciMethodConfigs      = "configs"
	UciMethodState        = "state"
	UciMethodRename       = "rename"
	UciMethodOrder        = "order"
	UciMethodChanges      = "changes"
	UciMethodRevert       = "revert"
	UciMethodApply        = "apply"
	UciMethodConfirm      = "confirm"
	UciMethodReloadConfig = "reload_config"
)

// UCI metadata field constants
const (
	UciMetaType      = ".type"
	UciMetaName      = ".name"
	UciMetaIndex     = ".index"
	UciMetaAnonymous = ".anonymous"
)

// UCI parameter constants
const (
	UciParamValues = "values"
)

// GetUci retrieves UCI configuration data.
func GetUci(caller types.Transport, request types.UbusUciGetRequest) (*types.UbusUciGetResponse, error) {
	resp, err := caller.Call(ServiceUCI, UciMethodGet, request)
	if err != nil {
		return nil, err
	}

	var ubusData types.UbusUciGetResponse
	if err := resp.Unmarshal(&ubusData); err != nil {
		return nil, errdefs.Wrapf(err, "failed to unmarshal uci get result")
	}

	dataBytes, err := json.Marshal(ubusData)
	if err != nil {
		return nil, errdefs.Wrapf(err, "failed to marshal uci get result")
	}

	var singleValue string
	if err := json.Unmarshal(dataBytes, &singleValue); err == nil {
		ubusData.Value = singleValue
		return &ubusData, nil
	}

	var responseMap map[string]any
	if err := json.Unmarshal(dataBytes, &responseMap); err != nil {
		return nil, errdefs.Wrapf(err, "unexpected type for uci get result, could not unmarshal to map")
	}

	valuesData, ok := responseMap[UciParamValues]
	if !ok {
		return &ubusData, nil
	}

	valuesMap, ok := valuesData.(map[string]any)
	if !ok {
		return &ubusData, nil
	}

	ubusData.Values = valuesMap
	return &ubusData, nil
}

// GetAllUci retrieves all sections from a UCI package.
func GetAllUci(caller types.Transport, request types.UbusUciGetRequest) (map[string]map[string]any, error) {
	resp, err := GetUci(caller, request)
	if err != nil {
		return nil, err
	}

	allSections := make(map[string]map[string]any)
	for sectionName, sectionDataStr := range resp.Values {
		// The value is now a map, not a JSON string
		if sectionData, ok := sectionDataStr.(map[string]any); ok {
			allSections[sectionName] = sectionData
		}
	}
	return allSections, nil
}

// SetUci sets UCI configuration values.
func SetUci(caller types.Transport, request types.UbusUciRequest) error {
	_, err := caller.Call(ServiceUCI, UciMethodSet, request)
	return err
}

// AddUci adds new UCI configuration sections.
func AddUci(caller types.Transport, request types.UbusUciRequest) error {
	_, err := caller.Call(ServiceUCI, UciMethodAdd, request)
	return err
}

// DeleteUci deletes UCI configuration sections or options.
func DeleteUci(caller types.Transport, request types.UbusUciRequestGeneric) error {
	_, err := caller.Call(ServiceUCI, UciMethodDelete, request)
	return err
}

// AddToUciList adds a value to a UCI list.
func AddToUciList(caller types.Transport, config, section, option, value string) error {
	// There is no direct "add_list" in ubus's uci object.
	// We need to get the list, append the new value, and set it back.
	getRequest := types.UbusUciGetRequest{
		UbusUciRequestGeneric: types.UbusUciRequestGeneric{
			Config:  config,
			Section: section,
			Option:  option,
		},
	}
	getResponse, err := GetUci(caller, getRequest)
	if err != nil {
		// If the option doesn't exist, it's not an error. Start a new list.
		if !errors.Is(err, errdefs.ErrNotFound) {
			return errdefs.Wrapf(err, "could not get list to add to")
		}
	}

	currentList := []string{}
	if getResponse != nil && getResponse.Value != "" {
		// Filter out empty strings that might result from splitting a single-item list that has spaces
		for _, item := range strings.Split(getResponse.Value, " ") {
			if item != "" {
				currentList = append(currentList, item)
			}
		}
	}

	// Avoid adding duplicate values
	for _, item := range currentList {
		if item == value {
			return nil // Value already exists
		}
	}

	newList := append(currentList, value)
	newValue := strings.Join(newList, " ")

	setRequest := types.UbusUciRequest{
		UbusUciRequestGeneric: types.UbusUciRequestGeneric{
			Config:  config,
			Section: section,
		},
		Values: map[string]any{option: newValue},
	}

	return SetUci(caller, setRequest)
}

// DeleteFromUciList removes a specific value from a UCI list.
func DeleteFromUciList(caller types.Transport, config, section, option, value string) error {
	// 1. Get the current list
	getRequest := types.UbusUciGetRequest{
		UbusUciRequestGeneric: types.UbusUciRequestGeneric{
			Config:  config,
			Section: section,
			Option:  option,
		},
	}
	getResponse, err := GetUci(caller, getRequest)
	if err != nil {
		if errors.Is(err, errdefs.ErrNotFound) {
			return nil // Option doesn't exist, nothing to do.
		}
		return errdefs.Wrapf(err, "could not get list to delete from")
	}

	if getResponse.Value == "" {
		return nil // Option is empty, nothing to do.
	}

	// 2. Filter out the value
	list := strings.Split(getResponse.Value, " ")
	newList := []string{}
	found := false
	for _, item := range list {
		if item == value {
			found = true
			continue
		}
		if item != "" {
			newList = append(newList, item)
		}
	}

	if !found {
		return nil // Value not in list, nothing to do.
	}

	// 3. Set the new list back
	newValue := strings.Join(newList, " ")
	setRequest := types.UbusUciRequest{
		UbusUciRequestGeneric: types.UbusUciRequestGeneric{
			Config:  config,
			Section: section,
		},
		Values: map[string]any{option: newValue},
	}
	return SetUci(caller, setRequest)
}

// CommitUci commits UCI configuration changes.
func CommitUci(caller types.Transport, request types.UbusUciRequestGeneric) error {
	_, err := caller.Call(ServiceUCI, UciMethodCommit, request)
	return err
}

// GetUciConfigs retrieves the list of available UCI configuration files.
func GetUciConfigs(caller types.Transport, request types.UbusUciConfigsRequest) (*types.UbusUciConfigsResponse, error) {
	resp, err := caller.Call(ServiceUCI, UciMethodConfigs, request)
	if err != nil {
		return nil, err
	}

	var ubusData types.UbusUciConfigsResponse
	if err := resp.Unmarshal(&ubusData); err != nil {
		return nil, errdefs.Wrapf(err, "failed to unmarshal uci configs result")
	}

	return &ubusData, nil
}

// GetUciState retrieves UCI state information.
func GetUciState(caller types.Transport, request types.UbusUciStateRequest) (*types.UbusUciStateResponse, error) {
	resp, err := caller.Call(ServiceUCI, UciMethodState, request)
	if err != nil {
		return nil, err
	}

	var ubusData types.UbusUciStateResponse
	if err := resp.Unmarshal(&ubusData); err != nil {
		return nil, errdefs.Wrapf(err, "failed to unmarshal uci state result")
	}

	return &ubusData, nil
}

// RenameUci renames UCI sections or options.
func RenameUci(caller types.Transport, request types.UbusUciRenameRequest) error {
	_, err := caller.Call(ServiceUCI, UciMethodRename, request)
	return err
}

// OrderUci reorders UCI sections.
func OrderUci(caller types.Transport, request types.UbusUciOrderRequest) error {
	_, err := caller.Call(ServiceUCI, UciMethodOrder, request)
	return err
}

// GetUciChanges retrieves pending UCI changes.
func GetUciChanges(caller types.Transport, request types.UbusUciChangesRequest) (*types.UbusUciChangesResponse, error) {
	resp, err := caller.Call(ServiceUCI, UciMethodChanges, request)
	if err != nil {
		return nil, err
	}

	var ubusData types.UbusUciChangesResponse
	if err := resp.Unmarshal(&ubusData); err != nil {
		return nil, errdefs.Wrapf(err, "failed to unmarshal uci changes result")
	}

	return &ubusData, nil
}

// RevertUci reverts UCI configuration changes.
func RevertUci(caller types.Transport, request types.UbusUciRevertRequest) error {
	_, err := caller.Call(ServiceUCI, UciMethodRevert, request)
	return err
}

// ApplyUci applies UCI configuration changes.
func ApplyUci(caller types.Transport, request types.UbusUciApplyRequest) error {
	_, err := caller.Call(ServiceUCI, UciMethodApply, request)
	return err
}

// ConfirmUci confirms UCI configuration changes.
func ConfirmUci(caller types.Transport) error {
	_, err := caller.Call(ServiceUCI, UciMethodConfirm, nil)
	return err
}

// RollbackUci rolls back UCI configuration changes.
func RollbackUci(caller types.Transport) error {
	_, err := caller.Call(ServiceUCI, UciMethodRevert, nil)
	return err
}

// ReloadUciConfig reloads UCI configuration.
func ReloadUciConfig(caller types.Transport) error {
	_, err := caller.Call(ServiceUCI, UciMethodReloadConfig, nil)
	return err
}

// GetUciSections is a helper to get all sections of a certain type from a config.
func GetUciSections(caller types.Transport, config, sectionType string) ([]string, error) {
	// Request the entire configuration file
	req := types.UbusUciGetRequest{
		UbusUciRequestGeneric: types.UbusUciRequestGeneric{
			Config: config,
		},
	}
	// Instead of calling the low-level GetUci, we use the new GetAllUci
	// which handles the parsing into a map[string]map[string]string structure.
	allSections, err := GetAllUci(caller, req)
	if err != nil {
		return nil, errdefs.Wrapf(err, "failed to get config '%s'", config)
	}

	var sectionNames []string
	for sectionName, options := range allSections {
		// Check if the section has a ".type" field matching what we want
		if t, ok := options[UciMetaType].(string); ok && t == sectionType {
			sectionNames = append(sectionNames, sectionName)
		}
	}

	return sectionNames, nil
}

// GetUciSectionsList retrieves a list of sections from UCI.
func GetUciSectionsList(caller types.Transport, req types.UbusUciGetRequest) ([]string, error) {
	resp, err := caller.Call(ServiceUCI, "get", req)
	if err != nil {
		return nil, err
	}

	ubusData := map[string]any{}
	if err := resp.Unmarshal(&ubusData); err != nil {
		return nil, errdefs.Wrapf(err, "failed to unmarshal uci sections result")
	}

	if sections, ok := ubusData["sections"].(map[string]any); ok {
		var names []string
		for name := range sections {
			names = append(names, name)
		}
		return names, nil
	}
	// // The result is nested, like: [0, {"sections": {...}}]
	// // We need to extract the "sections" map.
	// if resArray, ok := resp.Result.([]any); ok && len(resArray) > 1 {
	// 	if data, ok := resArray[1].(map[string]any); ok {
	// 		if sections, ok := data["sections"].(map[string]any); ok {
	// 			var names []string
	// 			for name := range sections {
	// 				names = append(names, name)
	// 			}
	// 			return names, nil
	// 		}
	// 	}
	// }
	return nil, errdefs.Wrapf(errdefs.ErrInvalidResponse, "could not parse uci sections from response: %+v", ubusData)
}

// ParseUciMetadata extracts metadata from UCI data.
func ParseUciMetadata(data map[string]any) types.UciMetadata {
	meta := types.UciMetadata{}

	if name, ok := data[UciMetaName]; ok {
		if str, ok := name.(string); ok {
			meta.Name = str
		}
	}
	if typ, ok := data[UciMetaType]; ok {
		if str, ok := typ.(string); ok {
			meta.Type = str
		}
	}
	if indexVal, ok := data[UciMetaIndex]; ok {
		switch v := indexVal.(type) {
		case string:
			if index, err := strconv.Atoi(v); err == nil {
				meta.Index = &index
			}
		case float64:
			index := int(v)
			meta.Index = &index
		case json.Number:
			if idx, err := strconv.Atoi(v.String()); err == nil {
				meta.Index = &idx
			}
		}
	}
	if anonVal, ok := data[UciMetaAnonymous]; ok {
		switch v := anonVal.(type) {
		case string:
			if anon, err := strconv.ParseBool(v); err == nil {
				meta.Anonymous = types.Bool(anon)
			}
		case bool:
			meta.Anonymous = types.Bool(v)
		}
	}

	return meta
}
