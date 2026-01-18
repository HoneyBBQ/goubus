// Copyright (c) 2026 honeybbq
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package uci

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/honeybbq/goubus/v2"
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
	values []string
	kind   sectionValueKind
}

// NewSectionValues creates an initialized SectionValues.
func NewSectionValues() SectionValues {
	return SectionValues{
		values: make(map[string]sectionValue),
	}
}

// MarshalJSON ensures consistent boolean encoding.
func (sv *SectionValues) MarshalJSON() ([]byte, error) {
	return json.Marshal(sv.toUbusValues())
}

// UnmarshalJSON implements json.Unmarshaler.
func (sv *SectionValues) UnmarshalJSON(data []byte) error {
	if sv == nil {
		return nil
	}

	if len(data) == 0 || string(data) == "null" {
		*sv = NewSectionValues()

		return nil
	}

	var values map[string]any

	err := json.Unmarshal(data, &values)
	if err != nil {
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
func (sv *SectionValues) First(option string) (string, bool) {
	v, ok := sv.values[option]
	if !ok || len(v.values) == 0 {
		return "", false
	}

	return v.values[0], true
}

// Get returns the values for a given option.
func (sv *SectionValues) Get(option string) []string {
	if sv.values == nil {
		return nil
	}

	v, ok := sv.values[option]
	if !ok {
		return nil
	}

	return append([]string(nil), v.values...)
}

// All returns all values as a map.
func (sv *SectionValues) All() map[string][]string {
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
func (sv *SectionValues) Len() int {
	return len(sv.values)
}

// Clone returns a deep copy of the values.
func (sv *SectionValues) Clone() SectionValues {
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

func (sv *SectionValues) ensure() {
	if sv.values == nil {
		sv.values = make(map[string]sectionValue)
	}
}

func (sv *SectionValues) toUbusValues() map[string]any {
	if len(sv.values) == 0 {
		return map[string]any{}
	}

	serialized := make(map[string]any, len(sv.values))

	for key, value := range sv.values {
		if value.kind == sectionValueKindList {
			serialized[key] = append([]string(nil), value.values...)

			continue
		}

		switch len(value.values) {
		case 0:
			serialized[key] = ""
		case 1:
			serialized[key] = value.values[0]
		default:
			serialized[key] = append([]string(nil), value.values...)
		}
	}

	return serialized
}

// Section represents a parsed UCI section along with its metadata.
type Section struct {
	Values   SectionValues `json:"values"`
	Name     string        `json:"name"`
	Type     string        `json:"type"`
	Metadata Metadata      `json:"metadata"`
}

// Get returns the values for a given option.
func (s *Section) Get(option string) []string {
	if s == nil {
		return nil
	}

	return s.Values.Get(option)
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

	meta := parseMetadata(raw)
	if meta.Name == "" {
		meta.Name = name
	}

	return &Section{
		Name:     name,
		Type:     meta.Type,
		Values:   values,
		Metadata: meta,
	}
}

func setSectionValueFromAny(dst *SectionValues, key string, raw any) {
	if dst == nil || strings.HasPrefix(key, ".") {
		return
	}

	switch rawValue := raw.(type) {
	case nil:
		dst.Delete(key)
	case string:
		dst.Set(key, rawValue)
	case []string:
		dst.SetList(key, rawValue...)
	case []any:
		var entries []string
		for _, item := range rawValue {
			entries = append(entries, fmt.Sprint(item))
		}

		dst.SetList(key, entries...)
	default:
		dst.Set(key, fmt.Sprint(raw))
	}
}

func parseMetadata(data map[string]any) Metadata {
	meta := Metadata{}

	if name, ok := data[".name"].(string); ok {
		meta.Name = name
	}

	if typ, ok := data[".type"].(string); ok {
		meta.Type = typ
	}

	if indexVal, ok := data[".index"]; ok {
		meta.Index = parseIndex(indexVal)
	}

	if anonVal, ok := data[".anonymous"]; ok {
		meta.Anonymous = parseAnonymous(anonVal)
	}

	return meta
}

func parseIndex(value any) *int {
	switch _value := value.(type) {
	case string:
		index, err := strconv.Atoi(_value)
		if err == nil {
			return &index
		}
	case float64:
		index := int(_value)

		return &index
	case json.Number:
		idx, err := strconv.Atoi(_value.String())
		if err == nil {
			return &idx
		}
	}

	return nil
}

func parseAnonymous(value any) goubus.Bool {
	switch _value := value.(type) {
	case string:
		anon, err := strconv.ParseBool(_value)
		if err == nil {
			return goubus.Bool(anon)
		}
	case bool:
		return goubus.Bool(_value)
	}

	return false
}
