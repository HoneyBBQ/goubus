package uci

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/honeybbq/goubus/errdefs"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// UCITag represents the structure tag for UCI serialization
type Tag struct {
	Name         string        // UCI field name
	Omitempty    bool          // Omit if empty
	Flatten      bool          // Flatten embedded struct
	Join         string        // Join slice with separator (default: space)
	BoolMapping  *BoolMapping  // Boolean value mapping
	EnumMapping  *EnumMapping  // Enum value mapping
	UnitMapping  *UnitMapping  // Unit conversion mapping
	RangeMapping *RangeMapping // Range validation mapping
	CaseMapping  *CaseMapping  // String case mapping
}

// BoolMapping defines how boolean values should be serialized/deserialized
type BoolMapping struct {
	FalseValue string // String representation for false
	TrueValue  string // String representation for true
}

// EnumMapping defines mapping between integers and string values
type EnumMapping struct {
	Values  map[int]string // int -> string mapping
	Reverse map[string]int // string -> int mapping (for deserialization)
}

// UnitMapping defines unit conversion for numeric values
type UnitMapping struct {
	Unit       string  // Unit type (kb, mb, gb, seconds, minutes, hours, etc.)
	Multiplier float64 // Conversion multiplier
}

// RangeMapping defines valid range for numeric values
type RangeMapping struct {
	Min *int // Minimum value (nil = no limit)
	Max *int // Maximum value (nil = no limit)
}

// CaseMapping defines string case conversion
type CaseMapping struct {
	Type string // "lower", "upper", "title"
}

// parseUCITag parses UCI struct tag
func parseTag(tag string) Tag {
	uciTag := Tag{Join: " "} // Default join with space

	if tag == "" || tag == "-" {
		return Tag{Name: "-"} // Skip field
	}

	parts := strings.Split(tag, ",")
	if len(parts) > 0 {
		uciTag.Name = parts[0]
	}

	for i := 1; i < len(parts); i++ {
		option := strings.TrimSpace(parts[i])
		switch {
		case option == "omitempty":
			uciTag.Omitempty = true
		case option == "flatten":
			uciTag.Flatten = true
		case strings.HasPrefix(option, "join="):
			uciTag.Join = strings.TrimPrefix(option, "join=")
		case strings.HasPrefix(option, "bool="):
			// Parse boolean mapping: bool=false_value/true_value
			mapping := strings.TrimPrefix(option, "bool=")
			if boolMap := parseBoolMapping(mapping); boolMap != nil {
				uciTag.BoolMapping = boolMap
			}
		case strings.HasPrefix(option, "enum="):
			// Parse enum mapping: enum=low,medium,high
			mapping := strings.TrimPrefix(option, "enum=")
			if enumMap := parseEnumMapping(mapping); enumMap != nil {
				uciTag.EnumMapping = enumMap
			}
		case strings.HasPrefix(option, "unit="):
			// Parse unit mapping: unit=kb, unit=mb, unit=seconds, etc.
			mapping := strings.TrimPrefix(option, "unit=")
			if unitMap := parseUnitMapping(mapping); unitMap != nil {
				uciTag.UnitMapping = unitMap
			}
		case strings.HasPrefix(option, "range="):
			// Parse range mapping: range=1-10, range=0-100
			mapping := strings.TrimPrefix(option, "range=")
			if rangeMap := parseRangeMapping(mapping); rangeMap != nil {
				uciTag.RangeMapping = rangeMap
			}
		case strings.HasPrefix(option, "case="):
			// Parse case mapping: case=lower, case=upper, case=title
			mapping := strings.TrimPrefix(option, "case=")
			if caseMap := parseCaseMapping(mapping); caseMap != nil {
				uciTag.CaseMapping = caseMap
			}
		}
	}

	return uciTag
}

// parseBoolMapping parses boolean mapping from string like "0/1" or "no/yes"
func parseBoolMapping(mapping string) *BoolMapping {
	parts := strings.Split(mapping, "/")
	if len(parts) != 2 {
		return nil // Invalid format
	}

	return &BoolMapping{
		FalseValue: strings.TrimSpace(parts[0]),
		TrueValue:  strings.TrimSpace(parts[1]),
	}
}

// parseEnumMapping parses enum mapping from string like "low,medium,high"
func parseEnumMapping(mapping string) *EnumMapping {
	values := strings.Split(mapping, ",")
	if len(values) == 0 {
		return nil
	}

	enumMap := &EnumMapping{
		Values:  make(map[int]string),
		Reverse: make(map[string]int),
	}

	for i, value := range values {
		trimmed := strings.TrimSpace(value)
		enumMap.Values[i] = trimmed
		enumMap.Reverse[trimmed] = i
	}

	return enumMap
}

// parseUnitMapping parses unit mapping from string like "kb" or "seconds"
func parseUnitMapping(mapping string) *UnitMapping {
	unit := strings.TrimSpace(strings.ToLower(mapping))

	var multiplier float64
	switch unit {
	// Data units
	case "b", "byte", "bytes":
		multiplier = 1
	case "kb", "kilobyte", "kilobytes":
		multiplier = 1024
	case "mb", "megabyte", "megabytes":
		multiplier = 1024 * 1024
	case "gb", "gigabyte", "gigabytes":
		multiplier = 1024 * 1024 * 1024

	// Time units
	case "s", "sec", "second", "seconds":
		multiplier = 1
	case "m", "min", "minute", "minutes":
		multiplier = 60
	case "h", "hour", "hours":
		multiplier = 3600
	case "d", "day", "days":
		multiplier = 86400

	// Percentage
	case "%", "percent", "percentage":
		multiplier = 0.01

	default:
		return nil // Unsupported unit
	}

	return &UnitMapping{
		Unit:       unit,
		Multiplier: multiplier,
	}
}

// parseRangeMapping parses range mapping from string like "1-10" or "0-100"
func parseRangeMapping(mapping string) *RangeMapping {
	parts := strings.Split(mapping, "-")
	if len(parts) != 2 {
		return nil
	}

	rangeMap := &RangeMapping{}

	if minStr := strings.TrimSpace(parts[0]); minStr != "" && minStr != "*" {
		if min, err := strconv.Atoi(minStr); err == nil {
			rangeMap.Min = &min
		}
	}

	if maxStr := strings.TrimSpace(parts[1]); maxStr != "" && maxStr != "*" {
		if max, err := strconv.Atoi(maxStr); err == nil {
			rangeMap.Max = &max
		}
	}

	return rangeMap
}

// parseCaseMapping parses case mapping from string like "lower" or "upper"
func parseCaseMapping(mapping string) *CaseMapping {
	caseType := strings.TrimSpace(strings.ToLower(mapping))

	switch caseType {
	case "lower", "lowercase":
		return &CaseMapping{Type: "lower"}
	case "upper", "uppercase":
		return &CaseMapping{Type: "upper"}
	case "title", "titlecase":
		return &CaseMapping{Type: "title"}
	default:
		return nil
	}
}

// Serializer provides UCI-specific serialization capabilities
type Serializer struct {
	// TagName specifies the struct tag to use (default: "uci")
	TagName string
}

// NewUCISerializer creates a new UCI serializer
func NewSerializer() *Serializer {
	return &Serializer{
		TagName: "uci",
	}
}

// Marshal converts a struct to UCI format (map[string]string)
func (s *Serializer) Marshal(v interface{}) (map[string]string, error) {
	result := make(map[string]string)

	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Ptr {
		if rv.IsNil() {
			return result, nil
		}
		rv = rv.Elem()
	}

	if rv.Kind() != reflect.Struct {
		return nil, errdefs.Wrapf(errdefs.ErrInvalidParameter, "UCI serialization only supports structs, got %T", v)
	}

	return s.marshalStruct(rv, result, "")
}

// marshalStruct marshals a struct value to UCI format
func (s *Serializer) marshalStruct(rv reflect.Value, result map[string]string, prefix string) (map[string]string, error) {
	rt := rv.Type()

	for i := 0; i < rv.NumField(); i++ {
		field := rt.Field(i)
		fieldValue := rv.Field(i)

		// Skip unexported fields
		if !field.IsExported() {
			continue
		}

		// Get UCI tag
		tag := parseTag(field.Tag.Get(s.TagName))
		if tag.Name == "-" {
			continue // Skip this field
		}

		// Determine field name
		fieldName := tag.Name
		if fieldName == "" {
			fieldName = strings.ToLower(field.Name)
		}

		// Build full name with prefix
		var fullName string
		if prefix != "" {
			fullName = prefix + "_" + fieldName
		} else {
			fullName = fieldName
		}

		// Marshal field
		if err := s.marshalField(fieldValue, tag, fullName, result); err != nil {
			return nil, errdefs.Wrapf(err, "failed to marshal field %s", field.Name)
		}
	}

	return result, nil
}

// marshalField marshals a single field to UCI format
func (s *Serializer) marshalField(fieldValue reflect.Value, tag Tag, fullName string, result map[string]string) error {
	// Handle pointer types
	if fieldValue.Kind() == reflect.Ptr {
		if fieldValue.IsNil() {
			if !tag.Omitempty {
				result[fullName] = ""
			}
			return nil
		}
		fieldValue = fieldValue.Elem()
	}

	// Handle different field types
	switch fieldValue.Kind() {
	case reflect.String:
		value := fieldValue.String()

		// Apply case mapping if specified
		if tag.CaseMapping != nil {
			switch tag.CaseMapping.Type {
			case "lower":
				value = strings.ToLower(value)
			case "upper":
				value = strings.ToUpper(value)
			case "title":
				value = strings.Title(value)
			}
		}

		if value != "" || !tag.Omitempty {
			result[fullName] = value
		}

	case reflect.Bool:
		// Handle boolean mapping
		if tag.BoolMapping != nil {
			value := tag.BoolMapping.FalseValue
			if fieldValue.Bool() {
				value = tag.BoolMapping.TrueValue
			}
			result[fullName] = value
		} else {
			// Default boolean mapping (false="0", true="1")
			value := "0"
			if fieldValue.Bool() {
				value = "1"
			}
			result[fullName] = value
		}

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		intValue := fieldValue.Int()

		// Handle enum mapping
		if tag.EnumMapping != nil {
			if enumStr, ok := tag.EnumMapping.Values[int(intValue)]; ok {
				result[fullName] = enumStr
			} else {
				return errdefs.Wrapf(errdefs.ErrInvalidParameter, "invalid enum value %d for field %s", intValue, fullName)
			}
		} else if tag.UnitMapping != nil {
			// Apply unit conversion (divide by multiplier when serializing)
			convertedValue := float64(intValue) / tag.UnitMapping.Multiplier
			result[fullName] = strconv.FormatFloat(convertedValue, 'f', -1, 64)
		} else {
			// Validate range if specified
			if tag.RangeMapping != nil {
				if tag.RangeMapping.Min != nil && intValue < int64(*tag.RangeMapping.Min) {
					return errdefs.Wrapf(errdefs.ErrInvalidParameter, "value %d below minimum %d for field %s", intValue, *tag.RangeMapping.Min, fullName)
				}
				if tag.RangeMapping.Max != nil && intValue > int64(*tag.RangeMapping.Max) {
					return errdefs.Wrapf(errdefs.ErrInvalidParameter, "value %d above maximum %d for field %s", intValue, *tag.RangeMapping.Max, fullName)
				}
			}
			result[fullName] = strconv.FormatInt(intValue, 10)
		}

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		uintValue := fieldValue.Uint()

		// Similar handling as int types
		if tag.EnumMapping != nil {
			if enumStr, ok := tag.EnumMapping.Values[int(uintValue)]; ok {
				result[fullName] = enumStr
			} else {
				return errdefs.Wrapf(errdefs.ErrInvalidParameter, "invalid enum value %d for field %s", uintValue, fullName)
			}
		} else if tag.UnitMapping != nil {
			convertedValue := float64(uintValue) / tag.UnitMapping.Multiplier
			result[fullName] = strconv.FormatFloat(convertedValue, 'f', -1, 64)
		} else {
			result[fullName] = strconv.FormatUint(uintValue, 10)
		}

	case reflect.Float32, reflect.Float64:
		floatValue := fieldValue.Float()

		if tag.UnitMapping != nil {
			// Apply unit conversion
			convertedValue := floatValue / tag.UnitMapping.Multiplier
			result[fullName] = strconv.FormatFloat(convertedValue, 'f', -1, 64)
		} else {
			result[fullName] = strconv.FormatFloat(floatValue, 'f', -1, 64)
		}

	case reflect.Slice:
		if fieldValue.Len() > 0 || !tag.Omitempty {
			var parts []string
			for j := 0; j < fieldValue.Len(); j++ {
				elem := fieldValue.Index(j)
				str, err := s.valueToString(elem)
				if err != nil {
					return err
				}
				parts = append(parts, str)
			}
			result[fullName] = strings.Join(parts, tag.Join)
		}

	case reflect.Struct:
		if tag.Flatten {
			// Flatten embedded struct
			_, err := s.marshalStruct(fieldValue, result, "")
			return err
		} else {
			// Marshal nested struct with prefix
			_, err := s.marshalStruct(fieldValue, result, fullName)
			return err
		}

	case reflect.Map:
		if fieldValue.Type().Key().Kind() == reflect.String &&
			fieldValue.Type().Elem().Kind() == reflect.String {
			// Handle map[string]string - merge into result
			for _, key := range fieldValue.MapKeys() {
				value := fieldValue.MapIndex(key)
				result[key.String()] = value.String()
			}
		}

	default:
		// Try to convert to string
		str, err := s.valueToString(fieldValue)
		if err != nil {
			return err
		}
		if str != "" || !tag.Omitempty {
			result[fullName] = str
		}
	}

	return nil
}

// valueToString converts a reflect.Value to string
func (s *Serializer) valueToString(v reflect.Value) (string, error) {
	switch v.Kind() {
	case reflect.String:
		return v.String(), nil
	case reflect.Bool:
		return strconv.FormatBool(v.Bool()), nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return strconv.FormatInt(v.Int(), 10), nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return strconv.FormatUint(v.Uint(), 10), nil
	case reflect.Float32, reflect.Float64:
		return strconv.FormatFloat(v.Float(), 'f', -1, 64), nil
	default:
		return fmt.Sprintf("%v", v.Interface()), nil
	}
}

// Unmarshal converts UCI format (map[string]string) to a struct
func (s *Serializer) Unmarshal(data map[string]string, v interface{}) error {
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Ptr || rv.IsNil() {
		return errdefs.Wrapf(errdefs.ErrInvalidParameter, "UCI unmarshal requires a non-nil pointer")
	}

	rv = rv.Elem()
	if rv.Kind() != reflect.Struct {
		return errdefs.Wrapf(errdefs.ErrInvalidParameter, "UCI unmarshal only supports struct pointers, got %T", v)
	}

	return s.unmarshalStruct(data, rv, "")
}

// unmarshalStruct unmarshals UCI data into a struct
func (s *Serializer) unmarshalStruct(data map[string]string, rv reflect.Value, prefix string) error {
	rt := rv.Type()

	for i := 0; i < rv.NumField(); i++ {
		field := rt.Field(i)
		fieldValue := rv.Field(i)

		// Skip unexported fields
		if !field.IsExported() {
			continue
		}

		// Get UCI tag
		tag := parseTag(field.Tag.Get(s.TagName))
		if tag.Name == "-" {
			continue // Skip this field
		}

		// Determine field name
		fieldName := tag.Name
		if fieldName == "" {
			fieldName = strings.ToLower(field.Name)
		}

		// Build full name with prefix
		var fullName string
		if prefix != "" {
			fullName = prefix + "_" + fieldName
		} else {
			fullName = fieldName
		}

		// Unmarshal field
		if err := s.unmarshalField(data, fieldValue, tag, fullName); err != nil {
			return errdefs.Wrapf(err, "failed to unmarshal field %s", field.Name)
		}
	}

	return nil
}

// unmarshalField unmarshals a single field from UCI data
func (s *Serializer) unmarshalField(data map[string]string, fieldValue reflect.Value, tag Tag, fullName string) error {
	// Handle pointer types
	if fieldValue.Kind() == reflect.Ptr {
		if _, exists := data[fullName]; exists {
			// Create new value for pointer
			newVal := reflect.New(fieldValue.Type().Elem())
			fieldValue.Set(newVal)
			fieldValue = fieldValue.Elem()
		} else {
			// Leave as nil
			return nil
		}
	}

	switch fieldValue.Kind() {
	case reflect.String:
		if val, exists := data[fullName]; exists {
			// Apply case mapping if specified
			if tag.CaseMapping != nil {
				switch tag.CaseMapping.Type {
				case "lower":
					val = strings.ToLower(val)
				case "upper":
					val = strings.ToUpper(val)
				case "title":
					val = cases.Title(language.English).String(val)
				}
			}
			fieldValue.SetString(val)
		}

	case reflect.Bool:
		if val, exists := data[fullName]; exists {
			// Handle boolean mapping
			if tag.BoolMapping != nil {
				switch val {
				case tag.BoolMapping.FalseValue:
					fieldValue.SetBool(false)
				case tag.BoolMapping.TrueValue:
					fieldValue.SetBool(true)
				default:
					// Fallback to standard parsing
					if boolVal, err := strconv.ParseBool(val); err == nil {
						fieldValue.SetBool(boolVal)
					}
				}
			} else {
				// Default boolean mapping
				switch val {
				case "0", "false", "no", "off", "disabled":
					fieldValue.SetBool(false)
				case "1", "true", "yes", "on", "enabled":
					fieldValue.SetBool(true)
				default:
					// Fallback to standard parsing
					if boolVal, err := strconv.ParseBool(val); err == nil {
						fieldValue.SetBool(boolVal)
					}
				}
			}
		}

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if val, exists := data[fullName]; exists {
			// Handle enum mapping
			if tag.EnumMapping != nil {
				if enumValue, ok := tag.EnumMapping.Reverse[val]; ok {
					fieldValue.SetInt(int64(enumValue))
				} else {
					return errdefs.Wrapf(errdefs.ErrInvalidParameter, "invalid enum string '%s' for field %s", val, fullName)
				}
			} else if tag.UnitMapping != nil {
				// Apply unit conversion (multiply by multiplier when deserializing)
				if floatVal, err := strconv.ParseFloat(val, 64); err == nil {
					convertedValue := floatVal * tag.UnitMapping.Multiplier
					intValue := int64(convertedValue)

					// Validate range if specified
					if tag.RangeMapping != nil {
						if tag.RangeMapping.Min != nil && intValue < int64(*tag.RangeMapping.Min) {
							return errdefs.Wrapf(errdefs.ErrInvalidParameter, "value %d below minimum %d for field %s", intValue, *tag.RangeMapping.Min, fullName)
						}
						if tag.RangeMapping.Max != nil && intValue > int64(*tag.RangeMapping.Max) {
							return errdefs.Wrapf(errdefs.ErrInvalidParameter, "value %d above maximum %d for field %s", intValue, *tag.RangeMapping.Max, fullName)
						}
					}

					fieldValue.SetInt(intValue)
				}
			} else {
				if intVal, err := strconv.ParseInt(val, 10, 64); err == nil {
					// Validate range if specified
					if tag.RangeMapping != nil {
						if tag.RangeMapping.Min != nil && intVal < int64(*tag.RangeMapping.Min) {
							return errdefs.Wrapf(errdefs.ErrInvalidParameter, "value %d below minimum %d for field %s", intVal, *tag.RangeMapping.Min, fullName)
						}
						if tag.RangeMapping.Max != nil && intVal > int64(*tag.RangeMapping.Max) {
							return errdefs.Wrapf(errdefs.ErrInvalidParameter, "value %d above maximum %d for field %s", intVal, *tag.RangeMapping.Max, fullName)
						}
					}
					fieldValue.SetInt(intVal)
				}
			}
		}

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if val, exists := data[fullName]; exists {
			// Similar handling as int types
			if tag.EnumMapping != nil {
				if enumValue, ok := tag.EnumMapping.Reverse[val]; ok {
					fieldValue.SetUint(uint64(enumValue))
				} else {
					return errdefs.Wrapf(errdefs.ErrInvalidParameter, "invalid enum string '%s' for field %s", val, fullName)
				}
			} else if tag.UnitMapping != nil {
				if floatVal, err := strconv.ParseFloat(val, 64); err == nil {
					convertedValue := floatVal * tag.UnitMapping.Multiplier
					fieldValue.SetUint(uint64(convertedValue))
				}
			} else {
				if uintVal, err := strconv.ParseUint(val, 10, 64); err == nil {
					fieldValue.SetUint(uintVal)
				}
			}
		}

	case reflect.Float32, reflect.Float64:
		if val, exists := data[fullName]; exists {
			if floatVal, err := strconv.ParseFloat(val, 64); err == nil {
				if tag.UnitMapping != nil {
					// Apply unit conversion
					convertedValue := floatVal * tag.UnitMapping.Multiplier
					fieldValue.SetFloat(convertedValue)
				} else {
					fieldValue.SetFloat(floatVal)
				}
			}
		}

	case reflect.Slice:
		if val, exists := data[fullName]; exists && val != "" {
			parts := strings.Split(val, tag.Join)
			slice := reflect.MakeSlice(fieldValue.Type(), len(parts), len(parts))

			for i, part := range parts {
				elem := slice.Index(i)
				if err := s.setStringValue(elem, strings.TrimSpace(part)); err != nil {
					return err
				}
			}

			fieldValue.Set(slice)
		}

	case reflect.Struct:
		if tag.Flatten {
			// Unmarshal flattened struct
			return s.unmarshalStruct(data, fieldValue, "")
		} else {
			// Unmarshal nested struct with prefix
			return s.unmarshalStruct(data, fieldValue, fullName)
		}

	case reflect.Map:
		if fieldValue.Type().Key().Kind() == reflect.String &&
			fieldValue.Type().Elem().Kind() == reflect.String {
			// Handle map[string]string - copy all remaining data
			if fieldValue.IsNil() {
				fieldValue.Set(reflect.MakeMap(fieldValue.Type()))
			}

			// // Find all fields that don't match known struct fields
			// for key, value := range data {
			// 	if !s.isKnownField(key, fieldValue.Type()) {
			// 		fieldValue.SetMapIndex(reflect.ValueOf(key), reflect.ValueOf(value))
			// 	}
			// }
		}
	}

	return nil
}

// setStringValue sets a reflect.Value from a string
func (s *Serializer) setStringValue(v reflect.Value, str string) error {
	switch v.Kind() {
	case reflect.String:
		v.SetString(str)
	case reflect.Bool:
		if boolVal, err := strconv.ParseBool(str); err == nil {
			v.SetBool(boolVal)
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if intVal, err := strconv.ParseInt(str, 10, 64); err == nil {
			v.SetInt(intVal)
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if uintVal, err := strconv.ParseUint(str, 10, 64); err == nil {
			v.SetUint(uintVal)
		}
	case reflect.Float32, reflect.Float64:
		if floatVal, err := strconv.ParseFloat(str, 64); err == nil {
			v.SetFloat(floatVal)
		}
	default:
		return errdefs.Wrapf(errdefs.ErrInvalidParameter, "unsupported type for string conversion: %s", v.Type())
	}
	return nil
}

// // isKnownField checks if a field name is a known struct field
// func (s *Serializer) isKnownField(name string, structType reflect.Type) bool {
// 	// This is a simplified implementation
// 	// In a real implementation, you'd want to build a map of known fields
// 	return false
// }

// Serializable interface for UCI serialization
type Serializable interface {
	ToUCI() (map[string]string, error)
	FromUCI(data map[string]string) error
}

// Global serializer instance
var defaultSerializer = NewSerializer()

// Marshal is a convenience function using the default serializer
func Marshal(v interface{}) (map[string]string, error) {
	return defaultSerializer.Marshal(v)
}

// Unmarshal is a convenience function using the default serializer
func Unmarshal(data map[string]string, v interface{}) error {
	return defaultSerializer.Unmarshal(data, v)
}
