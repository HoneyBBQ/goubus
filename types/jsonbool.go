package types

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// Bool is a JSON boolean type that gracefully accepts numbers or string representations.
type Bool bool

// UnmarshalJSON implements json.Unmarshaler, accepting 0/1, "0"/"1", true/false, "true"/"false", and null.
func (b *Bool) UnmarshalJSON(data []byte) error {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
		*b = Bool(false)
		return nil
	}

	switch trimmed[0] {
	case 't', 'T', 'f', 'F':
		var v bool
		if err := json.Unmarshal(trimmed, &v); err != nil {
			return err
		}
		*b = Bool(v)
		return nil
	case '"':
		var s string
		if err := json.Unmarshal(trimmed, &s); err != nil {
			return err
		}
		return b.fromString(s)
	default:
		// Attempt numeric parsing
		var num json.Number
		if err := json.Unmarshal(trimmed, &num); err == nil {
			if n, err := num.Float64(); err == nil {
				*b = Bool(n != 0)
				return nil
			}
		}
		// Fall back to string parsing
		return b.fromString(string(trimmed))
	}
}

// MarshalJSON ensures consistent boolean encoding.
func (b Bool) MarshalJSON() ([]byte, error) {
	if b {
		return []byte("true"), nil
	}
	return []byte("false"), nil
}

// Bool converts the custom type back to the built-in bool.
func (b Bool) Bool() bool {
	return bool(b)
}

// BoolValue safely dereferences a *Bool, returning false when nil.
func BoolValue(b *Bool) bool {
	if b == nil {
		return false
	}
	return bool(*b)
}

// Format enables using Bool with fmt verbs like %t.
func (b Bool) Format(state fmt.State, verb rune) {
	fmt.Fprintf(state, "%t", bool(b))
}

func (b *Bool) fromString(value string) error {
	normalized := strings.TrimSpace(strings.ToLower(value))
	switch normalized {
	case "", "0", "false", "no", "off":
		*b = Bool(false)
		return nil
	case "1", "true", "yes", "on":
		*b = Bool(true)
		return nil
	default:
		parsed, err := strconv.ParseFloat(normalized, 64)
		if err != nil {
			return err
		}
		*b = Bool(parsed != 0)
		return nil
	}
}
