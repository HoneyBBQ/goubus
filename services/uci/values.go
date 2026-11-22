package uci

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/honeybbq/goubus/v2"
)

type sectionValueKind uint8

const (
	sectionValueScalar sectionValueKind = iota
	sectionValueList
)

type sectionValue struct {
	values []string
	kind   sectionValueKind
}

// SectionValues stores scalar or list UCI option values.
type SectionValues map[string]sectionValue

// NewSectionValues creates an initialized SectionValues value.
func NewSectionValues() SectionValues {
	return make(SectionValues)
}

// MarshalJSON encodes values using UCI scalar and list forms.
func (values SectionValues) MarshalJSON() ([]byte, error) {
	return json.Marshal(values.toUbusValues())
}

// UnmarshalJSON decodes UCI scalar and list values.
func (values *SectionValues) UnmarshalJSON(data []byte) error {
	if values == nil {
		return nil
	}
	if len(data) == 0 || string(data) == "null" {
		*values = NewSectionValues()

		return nil
	}
	var decoded map[string]any
	err := json.Unmarshal(data, &decoded)
	if err != nil {
		return err
	}
	*values = SectionValuesFromAny(decoded)

	return nil
}

// Set replaces an option's values, using a list when multiple values are given.
func (values SectionValues) Set(option string, items ...string) {
	if values == nil {
		return
	}
	kind := sectionValueScalar
	if len(items) > 1 {
		kind = sectionValueList
	}
	values[option] = sectionValue{kind: kind, values: append([]string(nil), items...)}
}

// SetList replaces an option's values and forces list serialization.
func (values SectionValues) SetList(option string, items ...string) {
	if values == nil {
		return
	}
	values[option] = sectionValue{kind: sectionValueList, values: append([]string(nil), items...)}
}

// SetScalar stores one scalar option value.
func (values SectionValues) SetScalar(option, value string) {
	values.Set(option, value)
}

// Append adds values to an option without discarding its existing values.
func (values SectionValues) Append(option string, items ...string) {
	if values == nil || len(items) == 0 {
		return
	}
	current, ok := values[option]
	if !ok {
		values.Set(option, items...)

		return
	}
	merged := append(append([]string(nil), current.values...), items...)
	kind := current.kind
	if len(merged) > 1 {
		kind = sectionValueList
	}
	values[option] = sectionValue{kind: kind, values: merged}
}

// Delete removes an option from the value set.
func (values SectionValues) Delete(option string) {
	delete(values, option)
}

// First returns the first option value and whether the option has a value.
func (values SectionValues) First(option string) (string, bool) {
	item, ok := values[option]
	if !ok || len(item.values) == 0 {
		return "", false
	}

	return item.values[0], true
}

// Get returns a copy of all values for an option.
func (values SectionValues) Get(option string) []string {
	item, ok := values[option]
	if !ok {
		return nil
	}

	return append([]string(nil), item.values...)
}

// All returns a deep copy of every option's values.
func (values SectionValues) All() map[string][]string {
	if len(values) == 0 {
		return nil
	}
	result := make(map[string][]string, len(values))
	for name, item := range values {
		result[name] = append([]string(nil), item.values...)
	}

	return result
}

// Len returns the number of stored options.
func (values SectionValues) Len() int {
	return len(values)
}

// Clone returns a deep copy of the value set.
func (values SectionValues) Clone() SectionValues {
	clone := NewSectionValues()
	for name, item := range values {
		clone[name] = sectionValue{kind: item.kind, values: append([]string(nil), item.values...)}
	}

	return clone
}

// SectionValuesFromStrings converts a map of scalar values into SectionValues.
func SectionValuesFromStrings(items map[string]string) SectionValues {
	result := NewSectionValues()
	for name, value := range items {
		result.Set(name, value)
	}

	return result
}

// SectionValuesFromAny converts a map of JSON-decoded values into SectionValues.
func SectionValuesFromAny(items map[string]any) SectionValues {
	result := NewSectionValues()
	for name, item := range items {
		setSectionValueFromAny(&result, name, item)
	}

	return result
}

func (values SectionValues) toUbusValues() map[string]any {
	result := make(map[string]any, len(values))
	for name, item := range values {
		if item.kind == sectionValueList || len(item.values) > 1 {
			result[name] = append([]string(nil), item.values...)

			continue
		}
		if len(item.values) == 0 {
			result[name] = ""

			continue
		}
		result[name] = item.values[0]
	}

	return result
}

// Section represents parsed UCI section values and metadata.
type Section struct {
	// Values contains the section's options.
	Values SectionValues `json:"values"`
	// Name is the section name.
	Name string `json:"name"`
	// Type is the section type.
	Type string `json:"type"`
	// Metadata contains read-only section information.
	Metadata Metadata `json:"metadata"`
}

// Get returns the values for one section option.
func (section *Section) Get(option string) []string {
	if section == nil {
		return nil
	}

	return section.Values.Get(option)
}

// First returns the first value for one section option.
func (section *Section) First(option string) (string, bool) {
	if section == nil {
		return "", false
	}

	return section.Values.First(option)
}

func newSectionFromRaw(name string, raw map[string]any) *Section {
	values := NewSectionValues()
	for key, value := range raw {
		setSectionValueFromAny(&values, key, value)
	}
	metadata := parseMetadata(raw)
	if metadata.Name == "" {
		metadata.Name = name
	}

	return &Section{Name: name, Type: metadata.Type, Values: values, Metadata: metadata}
}

func setSectionValueFromAny(destination *SectionValues, key string, raw any) {
	if strings.HasPrefix(key, ".") {
		return
	}
	switch value := raw.(type) {
	case nil:
		destination.Delete(key)
	case string:
		destination.Set(key, value)
	case []string:
		destination.SetList(key, value...)
	case []any:
		items := make([]string, 0, len(value))
		for _, entry := range value {
			items = append(items, fmt.Sprint(entry))
		}
		destination.SetList(key, items...)
	default:
		destination.Set(key, fmt.Sprint(value))
	}
}

func parseMetadata(data map[string]any) Metadata {
	metadata := Metadata{}
	if name, ok := data[".name"].(string); ok {
		metadata.Name = name
	}
	if sectionType, ok := data[".type"].(string); ok {
		metadata.Type = sectionType
	}
	if index, ok := parseIndex(data[".index"]); ok {
		metadata.Index = &index
	}
	metadata.Anonymous = goubus.Bool(parseAnonymous(data[".anonymous"]))

	return metadata
}

func parseAnonymous(value any) bool {
	switch typed := value.(type) {
	case bool:
		return typed
	case goubus.Bool:
		return bool(typed)
	case int64:
		return typed != 0
	case float64:
		return typed != 0
	case uint64:
		return typed != 0
	case string:
		return typed == "1" || typed == "true"
	default:
		return false
	}
}

func parseIndex(value any) (int, bool) {
	switch typed := value.(type) {
	case float64:
		return int(typed), true
	case json.Number:
		index, err := strconv.Atoi(typed.String())

		return index, err == nil
	case string:
		index, err := strconv.Atoi(typed)

		return index, err == nil
	default:
		return 0, false
	}
}
