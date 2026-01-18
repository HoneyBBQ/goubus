// Copyright (c) 2026 honeybbq
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package uci

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"strings"

	"github.com/honeybbq/goubus/v2"
	"github.com/honeybbq/goubus/v2/errdefs"
)

// Dialect defines the differences in UCI ubus calls.
type Dialect any

// Manager is the entry point for all UCI-related operations.
type Manager struct {
	caller  goubus.Transport
	dialect Dialect
}

// New creates a new base UCI Manager.
func New(t goubus.Transport, d Dialect) *Manager {
	return &Manager{caller: t, dialect: d}
}

// Package selects a specific UCI configuration file (package) for operations.
func (m *Manager) Package(name string) *PackageContext {
	return &PackageContext{
		manager: m,
		name:    name,
	}
}

// Configs lists all available UCI configuration files on the system.
func (m *Manager) Configs(ctx context.Context) ([]string, error) {
	resp, err := goubus.Call[ConfigsResponse](ctx, m.caller, "uci", "configs", nil)
	if err != nil {
		return nil, errdefs.Wrapf(err, "failed to call uci configs")
	}

	return resp.Configs, nil
}

// State retrieves runtime state information.
func (m *Manager) State(ctx context.Context, req StateRequest) (*GetResponse, error) {
	return m.getRaw(ctx, "state", GetRequest(req))
}

// Apply activates staged changes.
func (m *Manager) Apply(ctx context.Context, rollback bool, timeout int) error {
	req := ApplyRequest{
		Rollback: goubus.Bool(rollback),
		Timeout:  timeout,
	}

	_, err := m.caller.Call(ctx, "uci", "apply", req)
	if err != nil {
		return errdefs.Wrapf(err, "failed to apply uci changes")
	}

	return nil
}

// Confirm commits changes that were applied with rollback enabled.
func (m *Manager) Confirm(ctx context.Context) error {
	_, err := m.caller.Call(ctx, "uci", "confirm", nil)
	if err != nil {
		return errdefs.Wrapf(err, "failed to confirm uci changes")
	}

	return nil
}

// Rollback manually reverts changes that were applied with Apply.
func (m *Manager) Rollback(ctx context.Context) error {
	_, err := m.caller.Call(ctx, "uci", "rollback", nil)
	if err != nil {
		return errdefs.Wrapf(err, "failed to rollback uci changes")
	}

	return nil
}

// ReloadConfig reloads the system configuration services.
func (m *Manager) ReloadConfig(ctx context.Context) error {
	_, err := m.caller.Call(ctx, "uci", "reload_config", nil)
	if err != nil {
		return errdefs.Wrapf(err, "failed to reload uci config")
	}

	return nil
}

// PackageContext represents operations on a specific UCI configuration file.
type PackageContext struct {
	manager *Manager
	name    string
}

// Section selects a specific section within the package.
func (pc *PackageContext) Section(name string) *SectionContext {
	return &SectionContext{
		pc:   pc,
		name: name,
	}
}

// GetAll retrieves all sections and their values from the package.
func (pc *PackageContext) GetAll(ctx context.Context) (map[string]*Section, error) {
	req := GetRequest{
		RequestGeneric: RequestGeneric{Config: pc.name},
	}

	raw, err := pc.manager.getAllRaw(ctx, "get", req)
	if err != nil {
		return nil, err
	}

	sections := make(map[string]*Section, len(raw))
	for name, data := range raw {
		sections[name] = newSectionFromRaw(name, data)
	}

	return sections, nil
}

// State retrieves all runtime state sections from the package.
func (pc *PackageContext) State(ctx context.Context) (map[string]*Section, error) {
	req := GetRequest{
		RequestGeneric: RequestGeneric{Config: pc.name},
	}

	raw, err := pc.manager.getAllRaw(ctx, "state", req)
	if err != nil {
		return nil, err
	}

	sections := make(map[string]*Section, len(raw))
	for name, data := range raw {
		sections[name] = newSectionFromRaw(name, data)
	}

	return sections, nil
}

// SectionsOfType returns the names of all sections in the package that match the given type.
func (pc *PackageContext) SectionsOfType(ctx context.Context, sectionType string) ([]string, error) {
	req := GetRequest{
		RequestGeneric: RequestGeneric{Config: pc.name},
	}

	allSections, err := pc.manager.getAllRaw(ctx, "get", req)
	if err != nil {
		return nil, err
	}

	var sectionNames []string

	for sectionName, options := range allSections {
		if t, ok := options[".type"].(string); ok && t == sectionType {
			sectionNames = append(sectionNames, sectionName)
		}
	}

	return sectionNames, nil
}

// Add creates a new section of sectionType with the given name and initial values.
func (pc *PackageContext) Add(ctx context.Context, sectionType, name string, values SectionValues) error {
	req := Request{
		RequestGeneric: RequestGeneric{
			Config: pc.name,
			Name:   name,
			Type:   sectionType,
		},
	}
	if values.Len() > 0 {
		req.Values = values.toUbusValues()
	}

	_, err := pc.manager.caller.Call(ctx, "uci", "add", req)

	return err
}

// Commit saves staged changes for the package.
func (pc *PackageContext) Commit(ctx context.Context) error {
	req := RequestGeneric{Config: pc.name}
	_, err := pc.manager.caller.Call(ctx, "uci", "commit", req)

	return err
}

// Revert discards staged changes for the package.
func (pc *PackageContext) Revert(ctx context.Context) error {
	req := RevertRequest{Config: pc.name}
	_, err := pc.manager.caller.Call(ctx, "uci", "revert", req)

	return err
}

// Changes lists the staged changes for the package.
func (pc *PackageContext) Changes(ctx context.Context) (*ChangesResponse, error) {
	req := ChangesRequest{Config: pc.name}

	return goubus.Call[ChangesResponse](ctx, pc.manager.caller, "uci", "changes", req)
}

// Order rearranges the sections in the package.
func (pc *PackageContext) Order(ctx context.Context, sections []string) error {
	req := OrderRequest{
		Config:   pc.name,
		Sections: sections,
	}
	_, err := pc.manager.caller.Call(ctx, "uci", "order", req)

	return err
}

// Sections returns the names of all sections currently defined in the package.
func (pc *PackageContext) Sections(ctx context.Context) ([]string, error) {
	req := GetRequest{
		RequestGeneric: RequestGeneric{Config: pc.name},
	}

	ubusData, err := goubus.Call[map[string]any](ctx, pc.manager.caller, "uci", "get", req)
	if err != nil {
		return nil, err
	}

	if sections, ok := (*ubusData)["sections"].(map[string]any); ok {
		var names []string
		for name := range sections {
			names = append(names, name)
		}

		return names, nil
	}

	return nil, errdefs.Wrapf(errdefs.ErrInvalidResponse, "could not parse uci sections from response")
}

// SectionContext represents operations on a specific section within a package.
type SectionContext struct {
	pc   *PackageContext
	name string
}

// Option selects a specific option within the section.
func (sc *SectionContext) Option(name string) *OptionContext {
	return &OptionContext{
		sc:   sc,
		name: name,
	}
}

// Get retrieves the section's type and all its current values.
func (sc *SectionContext) Get(ctx context.Context) (*Section, error) {
	req := GetRequest{
		RequestGeneric: RequestGeneric{
			Config:  sc.pc.name,
			Section: sc.name,
		},
	}

	resp, err := sc.pc.manager.getRaw(ctx, "get", req)
	if err != nil {
		return nil, err
	}

	return newSectionFromRaw(sc.name, resp.Values), nil
}

// State retrieves the runtime state of the section.
func (sc *SectionContext) State(ctx context.Context) (*Section, error) {
	req := GetRequest{
		RequestGeneric: RequestGeneric{
			Config:  sc.pc.name,
			Section: sc.name,
		},
	}

	resp, err := sc.pc.manager.getRaw(ctx, "state", req)
	if err != nil {
		return nil, err
	}

	return newSectionFromRaw(sc.name, resp.Values), nil
}

// SetValues updates multiple options in the section simultaneously.
func (sc *SectionContext) SetValues(ctx context.Context, values SectionValues) error {
	req := Request{
		RequestGeneric: RequestGeneric{
			Config:  sc.pc.name,
			Section: sc.name,
		},
	}
	if values.Len() > 0 {
		req.Values = values.toUbusValues()
	}

	_, err := sc.pc.manager.caller.Call(ctx, "uci", "set", req)

	return err
}

// Delete removes the section from the package.
func (sc *SectionContext) Delete(ctx context.Context) error {
	req := RequestGeneric{
		Config:  sc.pc.name,
		Section: sc.name,
	}
	_, err := sc.pc.manager.caller.Call(ctx, "uci", "delete", req)

	return err
}

// Rename changes the name of the section.
func (sc *SectionContext) Rename(ctx context.Context, newName string) error {
	req := RenameRequest{
		Config:  sc.pc.name,
		Section: sc.name,
		Name:    newName,
	}
	_, err := sc.pc.manager.caller.Call(ctx, "uci", "rename", req)

	return err
}

// OptionContext represents operations on a specific option within a section.
type OptionContext struct {
	sc   *SectionContext
	name string
}

// Get retrieves the current value of the option.
func (oc *OptionContext) Get(ctx context.Context) (string, error) {
	req := GetRequest{
		RequestGeneric: RequestGeneric{
			Config:  oc.sc.pc.name,
			Section: oc.sc.name,
			Option:  oc.name,
		},
	}

	resp, err := oc.sc.pc.manager.getRaw(ctx, "get", req)
	if err != nil {
		if errdefs.IsNotFound(err) {
			return "", errdefs.Wrapf(err, "option '%s' not found in section '%s'", oc.name, oc.sc.name)
		}

		return "", err
	}

	return resp.Value, nil
}

// State retrieves the runtime state of the option.
func (oc *OptionContext) State(ctx context.Context) (string, error) {
	req := GetRequest{
		RequestGeneric: RequestGeneric{
			Config:  oc.sc.pc.name,
			Section: oc.sc.name,
			Option:  oc.name,
		},
	}

	resp, err := oc.sc.pc.manager.getRaw(ctx, "state", req)
	if err != nil {
		if errdefs.IsNotFound(err) {
			return "", errdefs.Wrapf(err, "option '%s' not found in section '%s'", oc.name, oc.sc.name)
		}

		return "", err
	}

	return resp.Value, nil
}

// Set updates the value of the option.
func (oc *OptionContext) Set(ctx context.Context, value string) error {
	values := NewSectionValues()
	values.Set(oc.name, value)

	return oc.sc.SetValues(ctx, values)
}

// Delete removes the option from the section.
func (oc *OptionContext) Delete(ctx context.Context) error {
	req := RequestGeneric{
		Config:  oc.sc.pc.name,
		Section: oc.sc.name,
		Option:  oc.name,
	}
	_, err := oc.sc.pc.manager.caller.Call(ctx, "uci", "delete", req)

	return err
}

// AddToList appends a value to a list option.
func (oc *OptionContext) AddToList(ctx context.Context, value string) error {
	config, section, option := oc.sc.pc.name, oc.sc.name, oc.name
	getRequest := GetRequest{
		RequestGeneric: RequestGeneric{
			Config:  config,
			Section: section,
			Option:  option,
		},
	}

	getResponse, err := oc.sc.pc.manager.getRaw(ctx, "get", getRequest)
	if err != nil && !errors.Is(err, errdefs.ErrNotFound) {
		return errdefs.Wrapf(err, "could not get list to add to")
	}

	currentList := []string{}

	if getResponse != nil && getResponse.Value != "" {
		for item := range strings.SplitSeq(getResponse.Value, " ") {
			if item != "" {
				currentList = append(currentList, item)
			}
		}
	}

	if slices.Contains(currentList, value) {
		return nil
	}

	currentList = append(currentList, value)
	setRequest := Request{
		RequestGeneric: RequestGeneric{
			Config:  config,
			Section: section,
		},
		Values: map[string]any{option: currentList},
	}
	_, err = oc.sc.pc.manager.caller.Call(ctx, "uci", "set", setRequest)

	return err
}

// DeleteFromList removes a value from a list option.
func (oc *OptionContext) DeleteFromList(ctx context.Context, value string) error {
	config, section, option := oc.sc.pc.name, oc.sc.name, oc.name
	getRequest := GetRequest{
		RequestGeneric: RequestGeneric{Config: config, Section: section, Option: option},
	}

	getResponse, err := oc.sc.pc.manager.getRaw(ctx, "get", getRequest)
	if err != nil {
		if errors.Is(err, errdefs.ErrNotFound) {
			return nil
		}

		return errdefs.Wrapf(err, "could not get list to delete from")
	}

	if getResponse.Value == "" {
		return nil
	}

	newList := []string{}
	found := false

	for item := range strings.SplitSeq(getResponse.Value, " ") {
		if item == value {
			found = true

			continue
		}

		if item != "" {
			newList = append(newList, item)
		}
	}

	if !found {
		return nil
	}

	if len(newList) == 0 {
		delRequest := RequestGeneric{Config: config, Section: section, Option: option}
		_, err = oc.sc.pc.manager.caller.Call(ctx, "uci", "delete", delRequest)

		return err
	}

	setRequest := Request{
		RequestGeneric: RequestGeneric{Config: config, Section: section},
		Values:         map[string]any{option: newList},
	}
	_, err = oc.sc.pc.manager.caller.Call(ctx, "uci", "set", setRequest)

	return err
}

// Rename changes the name of the option.
func (oc *OptionContext) Rename(ctx context.Context, newName string) error {
	req := RenameRequest{Config: oc.sc.pc.name, Section: oc.sc.name, Option: oc.name, Name: newName}
	_, err := oc.sc.pc.manager.caller.Call(ctx, "uci", "rename", req)

	return err
}

func (m *Manager) getRaw(ctx context.Context, method string, req GetRequest) (*GetResponse, error) {
	ubusData, err := goubus.Call[GetResponse](ctx, m.caller, "uci", method, req)
	if err != nil {
		return nil, err
	}

	dataBytes, err := json.Marshal(ubusData)
	if err != nil {
		return nil, errdefs.Wrapf(err, "failed to marshal uci %s result", method)
	}

	var singleValue string

	err = json.Unmarshal(dataBytes, &singleValue)
	if err == nil {
		ubusData.Value = singleValue

		return ubusData, nil
	}

	var responseMap map[string]any

	err = json.Unmarshal(dataBytes, &responseMap)
	if err != nil {
		return nil, errdefs.Wrapf(err, "unexpected type for uci %s result", method)
	}

	if valuesData, ok := responseMap["values"]; ok {
		if valuesMap, ok := valuesData.(map[string]any); ok {
			ubusData.Values = valuesMap
		}
	}

	return ubusData, nil
}

func (m *Manager) getAllRaw(ctx context.Context, method string, req GetRequest) (map[string]map[string]any, error) {
	resp, err := m.getRaw(ctx, method, req)
	if err != nil {
		return nil, err
	}

	allSections := make(map[string]map[string]any)

	for sectionName, sectionDataStr := range resp.Values {
		if sectionData, ok := sectionDataStr.(map[string]any); ok {
			allSections[sectionName] = sectionData
		}
	}

	return allSections, nil
}
