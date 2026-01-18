package goubus_test

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/honeybbq/goubus/v2"
)

func TestBool_UnmarshalJSON(t *testing.T) {
	t.Helper()

	tests := []struct {
		input    string
		expected bool
		wantErr  bool
	}{
		// Standard booleans
		{`true`, true, false},
		{`false`, false, false},
		{`null`, false, false},

		// Numeric representations
		{`1`, true, false},
		{`0`, false, false},
		{`1.0`, true, false},
		{`0.0`, false, false},

		// String representations
		{`"true"`, true, false},
		{`"false"`, false, false},
		{`"1"`, true, false},
		{`"0"`, false, false},
		{`"yes"`, true, false},
		{`"no"`, false, false},
		{`"on"`, true, false},
		{`"off"`, false, false},
		{`""`, false, false},

		// Invalid inputs
		{`"maybe"`, false, true},
		{`{ "foo": "bar" }`, false, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			var boolValue goubus.Bool

			err := json.Unmarshal([]byte(tt.input), &boolValue)
			if (err != nil) != tt.wantErr {
				t.Errorf("UnmarshalJSON() error = %v, wantErr %v", err, tt.wantErr)

				return
			}

			if !tt.wantErr && bool(boolValue) != tt.expected {
				t.Errorf("UnmarshalJSON() got = %v, want %v", boolValue, tt.expected)
			}
		})
	}
}

func TestBool_MarshalJSON(t *testing.T) {
	tests := []struct {
		expected string
		input    goubus.Bool
	}{
		{`true`, goubus.Bool(true)},
		{`false`, goubus.Bool(false)},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("%v", test.input), func(t *testing.T) {
			data, err := json.Marshal(test.input)
			if err != nil {
				t.Fatalf("MarshalJSON() error = %v", err)
			}

			if string(data) != test.expected {
				t.Errorf("MarshalJSON() got = %s, want %s", string(data), test.expected)
			}
		})
	}
}

func TestBoolValue(t *testing.T) {
	if !goubus.BoolValue(goubus.Bool(true)) {
		t.Error("BoolValue(true) should be true")
	}

	if goubus.BoolValue(goubus.Bool(false)) {
		t.Error("BoolValue(false) should be false")
	}
}

func TestBool_Format(t *testing.T) {
	boolValue := goubus.Bool(true)

	str := fmt.Sprintf("%t", &boolValue)
	if str != "true" {
		t.Errorf("fmt.Sprintf(%%t) got = %s, want true", str)
	}

	boolValue = goubus.Bool(false)

	str = fmt.Sprintf("%t", &boolValue)
	if str != "false" {
		t.Errorf("fmt.Sprintf(%%t) got = %s, want false", str)
	}

	var nb *goubus.Bool

	str = fmt.Sprintf("%t", nb)
	if str != "false" {
		t.Errorf("fmt.Sprintf(%%t) for nil got = %s, want false", str)
	}
}
