// package goubus provides a client for the ubus RPC interface.
package goubus

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/honeybbq/goubus/api"
	"github.com/honeybbq/goubus/errdefs"
	"github.com/honeybbq/goubus/types"
)

// SectionValues represents raw UCI option data. Each key maps to one or more string values.
type SectionValues struct {
	values map[string]sectionValue
}

type sectionValueKind uint8

const (
	sectionValueKindScalar sectionValueKind = iota
	sectionValueKindList
)

type sectionValue struct {
	kind   sectionValueKind
	values []string
}

// NewSectionValues creates an initialized SectionValues.
func NewSectionValues() SectionValues {
	return SectionValues{
		values: make(map[string]sectionValue),
	}
}

func (sv *SectionValues) ensure() {
	if sv.values == nil {
		sv.values = make(map[string]sectionValue)
	}
}

func (sv SectionValues) MarshalJSON() ([]byte, error) {
	return json.Marshal(sv.toUbusValues())
}

func (sv *SectionValues) UnmarshalJSON(data []byte) error {
	if sv == nil {
		return nil
	}
	if len(data) == 0 || string(data) == "null" {
		*sv = NewSectionValues()
		return nil
	}
	var values map[string]any
	if err := json.Unmarshal(data, &values); err != nil {
		return err
	}
	*sv = SectionValuesFromAny(values)
	return nil
}

// Set replaces the values associated with an option.
func (sv *SectionValues) Set(option string, values ...string) {
	sv.ensure()
	copied := append([]string(nil), values...)
	kind := sectionValueKindScalar
	if len(copied) > 1 {
		kind = sectionValueKindList
	}
	sv.values[option] = sectionValue{kind: kind, values: copied}
}

// SetList replaces the values associated with an option and forces it to be serialized as a list.
// Even if there's only one value, it will be serialized as an array.
func (sv *SectionValues) SetList(option string, values ...string) {
	sv.ensure()
	copied := append([]string(nil), values...)
	sv.values[option] = sectionValue{kind: sectionValueKindList, values: copied}
}

// SetScalar is a convenience for setting a single value.
func (sv *SectionValues) SetScalar(option, value string) {
	if value == "" {
		sv.Set(option)
		return
	}
	sv.Set(option, value)
}

// Append adds values to an option without overwriting existing ones.
func (sv *SectionValues) Append(option string, values ...string) {
	sv.ensure()
	if len(values) == 0 {
		return
	}

	current, ok := sv.values[option]
	if !ok {
		sv.Set(option, values...)
		return
	}

	merged := append([]string(nil), current.values...)
	merged = append(merged, values...)

	kind := current.kind
	if kind == sectionValueKindScalar && len(merged) > 1 {
		kind = sectionValueKindList
	}
	sv.values[option] = sectionValue{kind: kind, values: merged}
}

// Delete removes an option from the set.
func (sv *SectionValues) Delete(option string) {
	if sv.values == nil {
		return
	}
	delete(sv.values, option)
}

// First returns the first value of an option.
func (sv SectionValues) First(option string) (string, bool) {
	v, ok := sv.values[option]
	if !ok || len(v.values) == 0 {
		return "", false
	}
	return v.values[0], true
}

// Clone returns a deep copy of the values.
func (sv SectionValues) Clone() SectionValues {
	cloned := NewSectionValues()
	for key, v := range sv.values {
		cloned.values[key] = sectionValue{
			kind:   v.kind,
			values: append([]string(nil), v.values...),
		}
	}
	return cloned
}

// SectionValuesFromStrings converts string values into SectionValues.
func SectionValuesFromStrings(values map[string]string) SectionValues {
	if len(values) == 0 {
		return NewSectionValues()
	}
	result := NewSectionValues()
	for key, value := range values {
		if value == "" {
			result.Set(key)
			continue
		}
		result.Set(key, value)
	}
	return result
}

// SectionValuesFromAny converts a map containing strings or slices into SectionValues.
func SectionValuesFromAny(values map[string]any) SectionValues {
	if len(values) == 0 {
		return NewSectionValues()
	}
	result := NewSectionValues()
	for key, raw := range values {
		setSectionValueFromAny(&result, key, raw)
	}
	return result
}

func (sv SectionValues) toUbusValues() map[string]any {
	if len(sv.values) == 0 {
		return map[string]any{}
	}
	serialized := make(map[string]any, len(sv.values))
	for key, v := range sv.values {
		// List: always serialize as array (even with single value).
		if v.kind == sectionValueKindList {
			serialized[key] = append([]string(nil), v.values...)
			continue
		}
		// Scalar: single value as string, multiple as array (for compatibility).
		switch len(v.values) {
		case 0:
			serialized[key] = ""
		case 1:
			serialized[key] = v.values[0]
		default:
			serialized[key] = append([]string(nil), v.values...)
		}
	}
	return serialized
}

// Section represents a parsed UCI section along with its metadata.
type Section struct {
	Name     string            `json:"name"`
	Type     string            `json:"type"`
	Values   SectionValues     `json:"values"`
	Metadata types.UciMetadata `json:"metadata"`
}

// Get returns the values for a given option.
func (s *Section) Get(option string) []string {
	if s == nil {
		return nil
	}
	return s.Values.Get(option)
}

// Get returns the values for a given option.
func (sv SectionValues) Get(option string) []string {
	if sv.values == nil {
		return nil
	}
	v, ok := sv.values[option]
	if !ok {
		return nil
	}
	return append([]string(nil), v.values...)
}

// All returns all values as a map (for backward compatibility).
func (sv SectionValues) All() map[string][]string {
	if sv.values == nil {
		return nil
	}
	result := make(map[string][]string, len(sv.values))
	for k, v := range sv.values {
		result[k] = append([]string(nil), v.values...)
	}
	return result
}

// Len returns the number of options.
func (sv SectionValues) Len() int {
	return len(sv.values)
}

// GetFirst returns the first value for a given option.
func (s *Section) GetFirst(option string) (string, bool) {
	if s == nil {
		return "", false
	}
	return s.Values.First(option)
}

func newSectionFromRaw(name string, raw map[string]any) *Section {
	values := NewSectionValues()
	for key, rawValue := range raw {
		setSectionValueFromAny(&values, key, rawValue)
	}

	meta := api.ParseUciMetadata(raw)
	if meta.Name == "" {
		meta.Name = name
	}

	sectionType := meta.Type

	return &Section{
		Name:     name,
		Type:     sectionType,
		Values:   values,
		Metadata: meta,
	}
}

func setSectionValueFromAny(dst *SectionValues, key string, raw any) {
	if dst == nil || strings.HasPrefix(key, ".") {
		return
	}

	switch v := raw.(type) {
	case nil:
		dst.Delete(key)
	case string:
		dst.Set(key, v)
	case []string:
		dst.SetList(key, v...)
	case []any:
		var entries []string
		for _, item := range v {
			entries = append(entries, fmt.Sprint(item))
		}
		dst.SetList(key, entries...)
	default:
		dst.Set(key, fmt.Sprint(raw))
	}
}

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
		Rollback: types.Bool(rollback),
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
//
// Corresponds to `ubus call uci get '{"config":"<package_name>"}'`.
func (pc *UciPackageContext) GetAll() (map[string]*Section, error) {
	req := types.UbusUciGetRequest{
		UbusUciRequestGeneric: types.UbusUciRequestGeneric{
			Config: pc.name,
		},
	}
	raw, err := api.GetAllUci(pc.client.caller, req)
	if err != nil {
		return nil, err
	}

	sections := make(map[string]*Section, len(raw))
	for name, data := range raw {
		sections[name] = newSectionFromRaw(name, data)
	}
	return sections, nil
}

// SectionsOfType returns the names of all sections of a given type.
func (pc *UciPackageContext) SectionsOfType(sectionType string) ([]string, error) {
	return api.GetUciSections(pc.client.caller, pc.name, sectionType)
}

// Add creates a new section within the configuration file.
// The `sectionType` parameter determines the type of section.
//
// Corresponds to `ubus call uci add '{"config":"<package_name>", "type":"<section_type>"}'`.
func (pc *UciPackageContext) Add(sectionType, name string, values SectionValues) error {
	req := types.UbusUciRequest{
		UbusUciRequestGeneric: types.UbusUciRequestGeneric{
			Config: pc.name,
			Name:   name,
			Type:   sectionType,
		},
	}
	if values.Len() > 0 {
		req.Values = values.toUbusValues()
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
// Corresponds to `ubus call uci order '{"config":"<package_name>", "sections":[...]}'.
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

// Get retrieves all option values from the section.
//
// Corresponds to `ubus call uci get '{"config":"<pkg>", "section":"<sec>"}'`.
func (sc *UciSectionContext) Get() (*Section, error) {
	req := types.UbusUciGetRequest{
		UbusUciRequestGeneric: types.UbusUciRequestGeneric{
			Config:  sc.packageName,
			Section: sc.sectionName,
		},
	}
	resp, err := api.GetUci(sc.client.caller, req)
	if err != nil {
		return nil, err
	}
	return newSectionFromRaw(sc.sectionName, resp.Values), nil
}

// SetValues updates multiple option values in the section.
// Corresponds to `ubus call uci set '{"config":"<pkg>", "section":"<sec>", "values":{...}}'`.
func (sc *UciSectionContext) SetValues(values SectionValues) error {
	req := types.UbusUciRequest{
		UbusUciRequestGeneric: types.UbusUciRequestGeneric{
			Config:  sc.packageName,
			Section: sc.sectionName,
		},
	}
	if values.Len() > 0 {
		req.Values = values.toUbusValues()
	}
	return api.SetUci(sc.client.caller, req)
}

// SetStrings is a helper that accepts simple string values.
func (sc *UciSectionContext) SetStrings(values map[string]string) error {
	return sc.SetValues(SectionValuesFromStrings(values))
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
		if errdefs.IsNotFound(err) {
			return "", errdefs.Wrapf(err, "option '%s' not found in section '%s'", oc.optionName, oc.sectionName)
		}
		return "", err
	}
	return resp.Value, nil
}

// Set updates the value of the specific option.
func (oc *UciOptionContext) Set(value string) error {
	values := NewSectionValues()
	values.Set(oc.optionName, value)
	return oc.client.Uci().
		Package(oc.packageName).
		Section(oc.sectionName).
		SetValues(values)
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
func (oc *UciOptionContext) AddToList(value string) error {
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
