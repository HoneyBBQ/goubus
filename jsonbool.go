package goubus

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

const (
	boolStrTrue  = "true"
	boolStrFalse = "false"
)

// Bool accepts numeric and string representations of a JSON boolean.
type Bool bool

// UnmarshalJSON decodes boolean, numeric, and string JSON values.
func (b *Bool) UnmarshalJSON(data []byte) error {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
		*b = false

		return nil
	}

	switch trimmed[0] {
	case 't', 'T', 'f', 'F':
		return b.unmarshalBool(trimmed)
	case '"':
		return b.unmarshalString(trimmed)
	default:
		return b.unmarshalNumberOrString(trimmed)
	}
}

// MarshalJSON encodes the boolean as a JSON boolean.
func (b Bool) MarshalJSON() ([]byte, error) {
	return []byte(strconv.FormatBool(bool(b))), nil
}

// BoolValue returns the boolean value.
func BoolValue(b Bool) bool {
	return bool(b)
}

// Format writes the boolean using the requested fmt state.
func (b Bool) Format(state fmt.State, _ rune) {
	_, _ = fmt.Fprintf(state, "%t", bool(b))
}

func (b *Bool) unmarshalBool(data []byte) error {
	var value bool
	err := json.Unmarshal(data, &value)
	if err != nil {
		return err
	}

	*b = Bool(value)

	return nil
}

func (b *Bool) unmarshalString(data []byte) error {
	var value string
	err := json.Unmarshal(data, &value)
	if err != nil {
		return err
	}

	parsed, err := parseBoolString(value)
	if err != nil {
		return err
	}

	*b = Bool(parsed)

	return nil
}

func (b *Bool) unmarshalNumberOrString(data []byte) error {
	var number json.Number
	err := json.Unmarshal(data, &number)
	if err == nil {
		value, floatErr := number.Float64()
		if floatErr == nil {
			*b = Bool(value != 0)

			return nil
		}
	}

	parsed, err := parseBoolString(string(data))
	if err != nil {
		return err
	}

	*b = Bool(parsed)

	return nil
}

func parseBoolString(value string) (bool, error) {
	normalized := strings.TrimSpace(strings.ToLower(value))
	switch normalized {
	case "", "0", boolStrFalse, "no", "off":
		return false, nil
	case "1", boolStrTrue, "yes", "on":
		return true, nil
	default:
		parsed, err := strconv.ParseFloat(normalized, 64)
		if err != nil {
			return false, err
		}

		return parsed != 0, nil
	}
}
