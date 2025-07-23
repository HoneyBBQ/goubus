// package goubus provides a client for the ubus RPC interface.
package goubus

import (
	"fmt"
	"strings"

	"github.com/honeybbq/goubus/api"
	"github.com/honeybbq/goubus/errdefs"
	"github.com/honeybbq/goubus/types"
	"github.com/honeybbq/goubus/uci"
)

// UciManager is the entry point for all UCI-related operations.
// It corresponds to the 'uci' ubus service.
type UciManager struct {
	client *Client
}

// Uci returns a new UciManager.
func (c *Client) Uci() *UciManager {
	return &UciManager{client: c}
}

// Package selects a specific UCI configuration file (package) to work with.
// e.g., "network", "wireless", "system".
func (um *UciManager) Package(name string) *UciPackageContext {
	return &UciPackageContext{
		client: um.client,
		name:   name,
	}
}

// Configs lists all available UCI configuration files.
// Corresponds to `ubus call uci configs`.
func (um *UciManager) Configs() ([]string, error) {
	req := types.UbusUciConfigsRequest{}
	resp, err := api.GetUciConfigs(um.client.caller, req)
	if err != nil {
		return nil, err
	}
	return resp.Configs, nil
}

// Apply activates changes, making them permanent after a timeout.
// This is a global operation, not tied to a specific config file.
func (um *UciManager) Apply(rollback bool, timeout int) error {
	req := types.UbusUciApplyRequest{
		Rollback: rollback,
		Timeout:  timeout,
	}
	return api.ApplyUci(um.client.caller, req)
}

// Confirm commits the changes applied with Apply.
func (um *UciManager) Confirm() error {
	return api.ConfirmUci(um.client.caller)
}

// Rollback reverts changes applied with Apply.
func (um *UciManager) Rollback() error {
	return api.RollbackUci(um.client.caller)
}

// ReloadConfig reloads a configuration file.
// Note: This seems to be a global operation in ubus, not tied to a specific config.
func (um *UciManager) ReloadConfig() error {
	return api.ReloadUciConfig(um.client.caller)
}

// UciPackageContext represents operations on a specific UCI configuration file (package).
type UciPackageContext struct {
	client *Client
	name   string
}

// Section selects a specific section within the configuration file.
func (pc *UciPackageContext) Section(name string) *UciSectionContext {
	return &UciSectionContext{
		client:      pc.client,
		packageName: pc.name,
		sectionName: name,
	}
}

// GetAll retrieves all sections from a UCI package.
// Returns a map where keys are section names and values contain section data.
//
// Note: Unlike individual section queries (Get method), this method returns
// complete section information including the .index field in metadata.
// If you need section index information, use this method instead of Get.
//
// Example:
//
//	sections, err := client.Uci().Package("network").GetAll()
//	for sectionName, sectionData := range sections {
//	    // sectionData will include .index field when parsed with config models
//	}
//
// Corresponds to `ubus call uci get '{"config":"<package_name>"}'`.
func (pc *UciPackageContext) GetAll() (map[string]map[string]any, error) {
	req := types.UbusUciGetRequest{
		UbusUciRequestGeneric: types.UbusUciRequestGeneric{
			Config: pc.name,
		},
	}
	return api.GetAllUci(pc.client.caller, req)
}

// SectionsOfType returns the names of all sections of a given type.
func (pc *UciPackageContext) SectionsOfType(sectionType string) ([]string, error) {
	return api.GetUciSections(pc.client.caller, pc.name, sectionType)
}

// Add creates a new section within the configuration file.
// The `sectionType` parameter determines the type of section.
// For example, "interface" for network interfaces, "device" for network devices.
//
// Returns the auto-generated name if the section is anonymous.
//
// Corresponds to `ubus call uci add '{"config":"<package_name>", "type":"<section_type>"}'`.
func (pc *UciPackageContext) Add(sectionType, name string, config ConfigModel) error {
	values, err := config.ToUCI()
	if err != nil {
		return err
	}

	// If model is empty (all zero values), start with an empty values map
	if len(values) == 0 {
		values = make(map[string]string)
	}

	req := types.UbusUciRequest{
		UbusUciRequestGeneric: types.UbusUciRequestGeneric{
			Config: pc.name,
			Name:   name,
			Type:   sectionType,
		},
		Values: values,
	}
	return api.AddUci(pc.client.caller, req)
}

// Commit commits all changes to the configuration file.
// Corresponds to `ubus call uci commit '{"config":"<package_name>"}'`.
func (pc *UciPackageContext) Commit() error {
	req := types.UbusUciRequestGeneric{Config: pc.name}
	return api.CommitUci(pc.client.caller, req)
}

// Revert reverts all uncommitted changes to the configuration file.
// Corresponds to `ubus call uci revert '{"config":"<package_name>"}'`.
func (pc *UciPackageContext) Revert() error {
	req := types.UbusUciRevertRequest{Config: pc.name}
	return api.RevertUci(pc.client.caller, req)
}

// Changes retrieves all uncommitted changes to the configuration file.
// Corresponds to `ubus call uci changes '{"config":"<package_name>"}'`.
func (pc *UciPackageContext) Changes() (*types.UbusUciChangesResponse, error) {
	req := types.UbusUciChangesRequest{Config: pc.name}
	return api.GetUciChanges(pc.client.caller, req)
}

// Order sets the order of sections within the configuration file.
// Corresponds to `ubus call uci order '{"config":"<package_name>", "sections":[...]}"`.
func (pc *UciPackageContext) Order(sections []string) error {
	req := types.UbusUciOrderRequest{
		Config:   pc.name,
		Sections: sections,
	}
	return api.OrderUci(pc.client.caller, req)
}

// Sections returns a list of all section names within the package.
func (pc *UciPackageContext) Sections() ([]string, error) {
	req := types.UbusUciGetRequest{
		UbusUciRequestGeneric: types.UbusUciRequestGeneric{
			Config: pc.name,
		},
	}
	return api.GetUciSectionsList(pc.client.caller, req)
}

// UciSectionContext represents operations on a specific section within a UCI configuration file.
type UciSectionContext struct {
	client      *Client
	packageName string
	sectionName string
}

// Option selects a specific option within the section.
func (sc *UciSectionContext) Option(name string) *UciOptionContext {
	return &UciOptionContext{
		client:      sc.client,
		packageName: sc.packageName,
		sectionName: sc.sectionName,
		optionName:  name,
	}
}

// Get retrieves all option values from the section and populates the provided model.
// The model must implement the ConfigModel interface.
//
// Corresponds to `ubus call uci get '{"config":"<pkg>", "section":"<sec>"}'`.
func (sc *UciSectionContext) Get(model ConfigModel) error {
	req := types.UbusUciGetRequest{
		UbusUciRequestGeneric: types.UbusUciRequestGeneric{
			Config:  sc.packageName,
			Section: sc.sectionName,
		},
	}
	resp, err := api.GetUci(sc.client.caller, req)
	if err != nil {
		return err
	}

	// Convert response values to map[string]string
	uciData := make(map[string]string)
	for key, value := range resp.Values {
		// Handle different types from JSON response
		switch v := value.(type) {
		case string:
			uciData[key] = v
		case []interface{}:
			// Convert slice to space-separated string
			var parts []string
			for _, item := range v {
				if str, ok := item.(string); ok {
					parts = append(parts, str)
				}
			}
			uciData[key] = strings.Join(parts, " ")
		default:
			// Convert other types to string
			uciData[key] = fmt.Sprintf("%v", v)
		}
	}

	// Use the model's FromUCI method to populate data and metadata
	return model.FromUCI(uciData)
}

// Set updates multiple option values in the section from the provided model.
// Corresponds to `ubus call uci set '{"config":"<pkg>", "section":"<sec>", "values":{...}}'`.
func (sc *UciSectionContext) Set(model ConfigModel) error {
	// Use UCI serialization to get values
	values, err := model.ToUCI()
	if err != nil {
		return err
	}

	return sc.SetValues(values)
}

// SetValues updates multiple option values in the section from a map.
// Corresponds to `ubus call uci set '{"config":"<pkg>", "section":"<sec>", "values":{...}}'`.
func (sc *UciSectionContext) SetValues(values map[string]string) error {
	req := types.UbusUciRequest{
		UbusUciRequestGeneric: types.UbusUciRequestGeneric{
			Config:  sc.packageName,
			Section: sc.sectionName,
		},
		Values: values,
	}
	return api.SetUci(sc.client.caller, req)
}

// Delete removes the entire section.
// Corresponds to `ubus call uci delete '{"config":"<pkg>", "section":"<sec>"}'`.
func (sc *UciSectionContext) Delete() error {
	req := types.UbusUciRequestGeneric{
		Config:  sc.packageName,
		Section: sc.sectionName,
	}
	return api.DeleteUci(sc.client.caller, req)
}

// Rename renames the section.
// Corresponds to `ubus call uci rename '{"config":"<pkg>", "section":"<sec>", "name":"<new_name>"}'`.
func (sc *UciSectionContext) Rename(newName string) error {
	req := types.UbusUciRenameRequest{
		Config:  sc.packageName,
		Section: sc.sectionName,
		Name:    newName,
	}
	return api.RenameUci(sc.client.caller, req)
}

// UciOptionContext represents operations on a specific option within a UCI section.
type UciOptionContext struct {
	client      *Client
	packageName string
	sectionName string
	optionName  string
}

// Get retrieves the value of the specific option.
// Corresponds to `ubus call uci get '{"config":"<pkg>", "section":"<sec>", "option":"<opt>"}'`.
func (oc *UciOptionContext) Get() (string, error) {
	req := types.UbusUciGetRequest{
		UbusUciRequestGeneric: types.UbusUciRequestGeneric{
			Config:  oc.packageName,
			Section: oc.sectionName,
			Option:  oc.optionName,
		},
	}
	resp, err := api.GetUci(oc.client.caller, req)
	if err != nil {
		// Since ubus returns 'not found' for non-existent options,
		// we check for this specific error.
		if errdefs.IsNotFound(err) {
			return "", errdefs.Wrapf(err, "option '%s' not found in section '%s'", oc.optionName, oc.sectionName)
		}
		return "", err
	}
	return resp.Value, nil
}

// Set updates the value of the specific option.
// This is a convenience for setting a single value.
func (oc *UciOptionContext) Set(value string) error {
	return oc.client.Uci().
		Package(oc.packageName).
		Section(oc.sectionName).
		SetValues(map[string]string{oc.optionName: value})
}

// Delete removes the option from the section.
// Corresponds to `ubus call uci delete '{"config":"<pkg>", "section":"<sec>", "option":"<opt>"}'`.
func (oc *UciOptionContext) Delete() error {
	req := types.UbusUciRequestGeneric{
		Config:  oc.packageName,
		Section: oc.sectionName,
		Option:  oc.optionName,
	}
	return api.DeleteUci(oc.client.caller, req)
}

// AddToList adds a value to a list option.
// This is a convenience method that simplifies adding to a list.
func (oc *UciOptionContext) AddToList(value string) error {
	// ubus uci add_list '{"config":"<pkg>", "section":"<sec>", "option":"<opt>", "value":"<val>"}'
	// This is not a standard ubus method. `add_list` is often implemented in higher-level libraries.
	// The standard way is to set the option with the new full list string.
	// Let's implement this as a helper.
	return api.AddToUciList(oc.client.caller, oc.packageName, oc.sectionName, oc.optionName, value)
}

// DeleteFromList removes a value from a list option.
func (oc *UciOptionContext) DeleteFromList(value string) error {
	return api.DeleteFromUciList(oc.client.caller, oc.packageName, oc.sectionName, oc.optionName, value)
}

// Rename renames the option.
// Corresponds to `ubus call uci rename '{"config":"<pkg>", "section":"<sec>", "option":"<opt>", "name":"<new_name>"}'`.
func (oc *UciOptionContext) Rename(newName string) error {
	req := types.UbusUciRenameRequest{
		Config:  oc.packageName,
		Section: oc.sectionName,
		Option:  oc.optionName,
		Name:    newName,
	}
	return api.RenameUci(oc.client.caller, req)
}

// ConfigModel defines the public interface that all UCI configuration
// structures must implement to be managed by the UciManager.
type ConfigModel interface {
	// UCI serialization interfaces
	uci.Serializable

	// Metadata returns the read-only metadata associated with the UCI section.
	Metadata() types.UciMetadata
}

// BaseConfig can be embedded in UCI configuration models to automatically
// handle metadata and satisfy the ConfigModel interface's metadata methods.
type BaseConfig struct {
	metadata types.UciMetadata `uci:"-"` // Excluded from UCI serialization
}

// Metadata implements the public ConfigModel interface.
func (b *BaseConfig) Metadata() types.UciMetadata {
	return b.metadata
}

// ToUCI implements the UCISerializable interface.
// This method should be overridden by embedding structs for custom serialization.
func (b *BaseConfig) ToUCI() (map[string]string, error) {
	return uci.Marshal(b)
}

// FromUCI implements the UCISerializable interface.
// This method handles metadata extraction and delegates to UCI unmarshaling.
func (b *BaseConfig) FromUCI(data map[string]string) error {
	// Extract and set metadata
	b.parseAndSetMetadata(data)

	// Remove metadata from data for unmarshaling
	cleanData := make(map[string]string)
	for k, v := range data {
		if !strings.HasPrefix(k, ".") {
			cleanData[k] = v
		}
	}

	// Unmarshal clean data
	return uci.Unmarshal(cleanData, b)
}

// parseAndSetMetadata extracts metadata from UCI data and sets it on the BaseConfig
func (b *BaseConfig) parseAndSetMetadata(data map[string]string) {
	b.metadata = api.ParseUciMetadata(data)
}
