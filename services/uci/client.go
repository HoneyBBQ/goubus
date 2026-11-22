// Package uci provides UCI configuration ubus operations.
package uci

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/honeybbq/goubus/v2"
)

// Client provides UCI configuration operations.
type Client struct {
	caller goubus.Caller
}

// New creates a UCI client.
func New(caller goubus.Caller) *Client {
	if caller == nil {
		panic("goubus/uci: nil caller")
	}

	return &Client{caller: caller}
}

// Package selects a configuration package for operations.
func (client *Client) Package(name string) *PackageContext {
	return &PackageContext{manager: client, name: name}
}

// Configs lists available configuration package names.
func (client *Client) Configs(ctx context.Context) ([]string, error) {
	result, err := goubus.Call[configsResponse](ctx, client.caller, "uci", "configs", nil)
	if err != nil {
		return nil, err
	}

	return result.Configs, nil
}

// State retrieves runtime state for a UCI request.
func (client *Client) State(ctx context.Context, request StateRequest) (*GetResponse, error) {
	return client.getRaw(ctx, "state", getRequest{requestGeneric: requestGeneric(request)})
}

// Apply activates staged configuration changes.
func (client *Client) Apply(ctx context.Context, rollback bool, timeout int) error {
	request := applyRequest{Rollback: goubus.Bool(rollback), Timeout: timeout}

	return client.call(ctx, "apply", request)
}

// Confirm commits changes applied with rollback enabled.
func (client *Client) Confirm(ctx context.Context) error {
	return client.call(ctx, "confirm", nil)
}

// Rollback manually reverts changes applied with rollback enabled.
func (client *Client) Rollback(ctx context.Context) error {
	return client.call(ctx, "rollback", nil)
}

// ReloadConfig reloads services affected by UCI changes.
func (client *Client) ReloadConfig(ctx context.Context) error {
	return client.call(ctx, "reload_config", nil)
}

// PackageContext provides operations on a selected UCI package.
type PackageContext struct {
	manager *Client
	name    string
}

// Section selects a section within the package.
func (packageContext *PackageContext) Section(name string) *SectionContext {
	return &SectionContext{pc: packageContext, name: name}
}

// All retrieves all sections and their values from the package.
func (packageContext *PackageContext) All(ctx context.Context) (map[string]*Section, error) {
	sections, err := packageContext.manager.getAllRaw(ctx, "get", packageContext.getRequest())
	if err != nil {
		return nil, err
	}
	result := make(map[string]*Section, len(sections))
	for name, values := range sections {
		result[name] = newSectionFromRaw(name, values)
	}

	return result, nil
}

// State retrieves all runtime state sections in the package.
func (packageContext *PackageContext) State(ctx context.Context) (map[string]*Section, error) {
	sections, err := packageContext.manager.getAllRaw(ctx, "state", packageContext.getRequest())
	if err != nil {
		return nil, err
	}
	result := make(map[string]*Section, len(sections))
	for name, values := range sections {
		result[name] = newSectionFromRaw(name, values)
	}

	return result, nil
}

// Sections returns the sorted names of all sections in the package.
func (packageContext *PackageContext) Sections(ctx context.Context) ([]string, error) {
	sections, err := packageContext.manager.getAllRaw(ctx, "get", packageContext.getRequest())
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(sections))
	for name := range sections {
		names = append(names, name)
	}
	slices.Sort(names)

	return names, nil
}

// SectionsOfType returns sorted section names matching a section type.
func (packageContext *PackageContext) SectionsOfType(ctx context.Context, sectionType string) ([]string, error) {
	sections, err := packageContext.manager.getAllRaw(ctx, "get", packageContext.getRequest())
	if err != nil {
		return nil, err
	}
	names := make([]string, 0)
	for name, values := range sections {
		if values[".type"] == sectionType {
			names = append(names, name)
		}
	}
	slices.Sort(names)

	return names, nil
}

// Add creates a section with optional initial values.
func (packageContext *PackageContext) Add(ctx context.Context, sectionType, name string, values SectionValues) error {
	request := writeRequest{requestGeneric: requestGeneric{Config: packageContext.name, Name: name, Type: sectionType}}
	if values.Len() > 0 {
		request.Values = values.toUbusValues()
	}

	return packageContext.manager.call(ctx, "add", request)
}

// Commit saves staged changes for the package.
func (packageContext *PackageContext) Commit(ctx context.Context) error {
	return packageContext.manager.call(ctx, "commit", requestGeneric{Config: packageContext.name})
}

// Revert discards staged changes for the package.
func (packageContext *PackageContext) Revert(ctx context.Context) error {
	return packageContext.manager.call(ctx, "revert", revertRequest{Config: packageContext.name})
}

// Changes lists staged changes for the package.
func (packageContext *PackageContext) Changes(ctx context.Context) (*ChangesResponse, error) {
	request := changesRequest{Config: packageContext.name}

	return goubus.Call[ChangesResponse](ctx, packageContext.manager.caller, "uci", "changes", request)
}

// Order rearranges sections in the package.
func (packageContext *PackageContext) Order(ctx context.Context, names []string) error {
	request := orderRequest{Config: packageContext.name, Sections: names}

	return packageContext.manager.call(ctx, "order", request)
}

func (packageContext *PackageContext) getRequest() getRequest {
	return getRequest{requestGeneric: requestGeneric{Config: packageContext.name}}
}

// SectionContext provides operations on one UCI section.
type SectionContext struct {
	pc   *PackageContext
	name string
}

// Option selects an option within the section.
func (sectionContext *SectionContext) Option(name string) *OptionContext {
	return &OptionContext{sc: sectionContext, name: name}
}

// Get retrieves the section's type, metadata, and values.
func (sectionContext *SectionContext) Get(ctx context.Context) (*Section, error) {
	response, err := sectionContext.get(ctx, "get")
	if err != nil {
		return nil, err
	}

	return newSectionFromRaw(sectionContext.name, response.Values), nil
}

// State retrieves runtime state for the section.
func (sectionContext *SectionContext) State(ctx context.Context) (*Section, error) {
	response, err := sectionContext.get(ctx, "state")
	if err != nil {
		return nil, err
	}

	return newSectionFromRaw(sectionContext.name, response.Values), nil
}

// SetValues updates multiple options in the section.
func (sectionContext *SectionContext) SetValues(ctx context.Context, values SectionValues) error {
	request := writeRequest{requestGeneric: requestGeneric{Config: sectionContext.pc.name, Section: sectionContext.name}}
	if values.Len() > 0 {
		request.Values = values.toUbusValues()
	}

	return sectionContext.pc.manager.call(ctx, "set", request)
}

// Delete removes the section from the package.
func (sectionContext *SectionContext) Delete(ctx context.Context) error {
	request := requestGeneric{Config: sectionContext.pc.name, Section: sectionContext.name}

	return sectionContext.pc.manager.call(ctx, "delete", request)
}

// Rename changes the section name.
func (sectionContext *SectionContext) Rename(ctx context.Context, newName string) error {
	request := renameRequest{Config: sectionContext.pc.name, Section: sectionContext.name, Name: newName}

	return sectionContext.pc.manager.call(ctx, "rename", request)
}

func (sectionContext *SectionContext) get(ctx context.Context, method string) (*GetResponse, error) {
	request := getRequest{requestGeneric: requestGeneric{Config: sectionContext.pc.name, Section: sectionContext.name}}

	return sectionContext.pc.manager.getRaw(ctx, method, request)
}

// OptionContext provides operations on one UCI option.
type OptionContext struct {
	sc   *SectionContext
	name string
}

// Get retrieves the first current value of the option.
func (optionContext *OptionContext) Get(ctx context.Context) (string, error) {
	values, err := optionContext.Values(ctx)
	if err != nil {
		return "", err
	}
	if len(values) == 0 {
		return "", nil
	}

	return values[0], nil
}

// Values retrieves all current values of the option.
func (optionContext *OptionContext) Values(ctx context.Context) ([]string, error) {
	response, err := optionContext.get(ctx, "get")
	if err != nil {
		return nil, err
	}

	return append([]string(nil), response.Value...), nil
}

// StateValues retrieves all runtime state values of the option.
func (optionContext *OptionContext) StateValues(ctx context.Context) ([]string, error) {
	response, err := optionContext.get(ctx, "state")
	if err != nil {
		return nil, err
	}

	return append([]string(nil), response.Value...), nil
}

// State retrieves the first runtime state value of the option.
func (optionContext *OptionContext) State(ctx context.Context) (string, error) {
	response, err := optionContext.get(ctx, "state")
	if err != nil {
		return "", err
	}

	return response.Value.First(), nil
}

// Set updates the option value.
func (optionContext *OptionContext) Set(ctx context.Context, value string) error {
	values := NewSectionValues()
	values.Set(optionContext.name, value)

	return optionContext.sc.SetValues(ctx, values)
}

// Delete removes the option from the section.
func (optionContext *OptionContext) Delete(ctx context.Context) error {
	return optionContext.sc.pc.manager.call(ctx, "delete", optionContext.request())
}

// AddToList appends a value to a list option without splitting its contents.
func (optionContext *OptionContext) AddToList(ctx context.Context, value string) error {
	current, err := optionContext.Values(ctx)
	if err != nil && !errors.Is(err, goubus.ErrNotFound) {
		return err
	}
	if slices.Contains(current, value) {
		return nil
	}
	current = append(current, value)

	return optionContext.setList(ctx, current)
}

// DeleteFromList removes a value from a list option.
func (optionContext *OptionContext) DeleteFromList(ctx context.Context, value string) error {
	current, err := optionContext.Values(ctx)
	if errors.Is(err, goubus.ErrNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	updated, found := removeValue(current, value)
	if !found {
		return nil
	}
	if len(updated) == 0 {
		return optionContext.Delete(ctx)
	}

	return optionContext.setList(ctx, updated)
}

// Rename changes the option name.
func (optionContext *OptionContext) Rename(ctx context.Context, newName string) error {
	request := renameRequest{
		Config:  optionContext.sc.pc.name,
		Section: optionContext.sc.name,
		Option:  optionContext.name,
		Name:    newName,
	}

	return optionContext.sc.pc.manager.call(ctx, "rename", request)
}

func (optionContext *OptionContext) get(ctx context.Context, method string) (*GetResponse, error) {
	request := getRequest{requestGeneric: optionContext.request()}

	return optionContext.sc.pc.manager.getRaw(ctx, method, request)
}

func (optionContext *OptionContext) request() requestGeneric {
	return requestGeneric{Config: optionContext.sc.pc.name, Section: optionContext.sc.name, Option: optionContext.name}
}

func (optionContext *OptionContext) setList(ctx context.Context, values []string) error {
	request := writeRequest{
		requestGeneric: requestGeneric{Config: optionContext.sc.pc.name, Section: optionContext.sc.name},
		Values:         map[string]any{optionContext.name: values},
	}

	return optionContext.sc.pc.manager.call(ctx, "set", request)
}

func removeValue(values []string, target string) ([]string, bool) {
	result := make([]string, 0, len(values))
	found := false
	for _, value := range values {
		if value == target {
			found = true

			continue
		}
		result = append(result, value)
	}

	return result, found
}

func (client *Client) getRaw(ctx context.Context, method string, request getRequest) (*GetResponse, error) {
	return goubus.Call[GetResponse](ctx, client.caller, "uci", method, request)
}

func (client *Client) getAllRaw(
	ctx context.Context,
	method string,
	request getRequest,
) (map[string]map[string]any, error) {
	response, err := client.getRaw(ctx, method, request)
	if err != nil {
		return nil, err
	}
	sections := make(map[string]map[string]any)
	for name, raw := range response.Values {
		values, ok := raw.(map[string]any)
		if ok {
			sections[name] = values
		}
	}

	return sections, nil
}

func (client *Client) call(ctx context.Context, method string, request any) error {
	_, err := client.caller.Call(ctx, "uci", method, request)
	if err != nil {
		return fmt.Errorf("uci %s: %w", method, err)
	}

	return nil
}
