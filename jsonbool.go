// Copyright (c) 2026 honeybbq
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

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

// Bool is a JSON boolean type that gracefully accepts numbers or string representations.
// It is useful for handling inconsistent boolean representations in ubus responses
// (e.g., 1/0, "1"/"0", "true"/"false").
type Bool bool

// UnmarshalJSON implements json.Unmarshaler, accepting 0/1, "0"/"1", true/false, "true"/"false", and null.
func (b *Bool) UnmarshalJSON(data []byte) error {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
		*b = false

		return nil
	}

	switch trimmed[0] {
	case 't', 'T', 'f', 'F':
		var value bool

		err := json.Unmarshal(trimmed, &value)
		if err != nil {
			return err
		}

		*b = Bool(value)

		return nil
	case '"':
		var str string

		err := json.Unmarshal(trimmed, &str)
		if err != nil {
			return err
		}

		return b.fromString(str)
	default:
		// Attempt numeric parsing
		var num json.Number

		err := json.Unmarshal(trimmed, &num)
		if err == nil {
			n, err := num.Float64()
			if err == nil {
				*b = Bool(n != 0)

				return nil
			}
		}

		// Fall back to string parsing
		return b.fromString(string(trimmed))
	}
}

// MarshalJSON ensures consistent boolean encoding.
func (b *Bool) MarshalJSON() ([]byte, error) {
	if b == nil {
		return []byte(boolStrFalse), nil
	}

	if *b {
		return []byte(boolStrTrue), nil
	}

	return []byte(boolStrFalse), nil
}

// BoolValue safely dereferences a Bool, returning false when nil.
func BoolValue(b Bool) bool {
	return bool(b)
}

// Format enables using Bool with fmt verbs like %t.
func (b *Bool) Format(state fmt.State, _ rune) {
	val := false
	if b != nil {
		val = bool(*b)
	}

	_, _ = fmt.Fprintf(state, "%t", val)
}

func (b *Bool) fromString(value string) error {
	normalized := strings.TrimSpace(strings.ToLower(value))
	switch normalized {
	case "", "0", boolStrFalse, "no", "off":
		*b = false

		return nil
	case "1", boolStrTrue, "yes", "on":
		*b = true

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
