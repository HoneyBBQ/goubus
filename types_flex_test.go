package goubus_test

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/honeybbq/goubus/v2"
)

func TestFlexibleJSONTypes(t *testing.T) {
	t.Run("StringOrSlice", testStringOrSliceJSON)
	t.Run("StringInt", testStringIntJSON)
	t.Run("UnixTime", testUnixTimeJSON)
}

func testStringOrSliceJSON(t *testing.T) {
	cases := []struct {
		input string
		want  goubus.StringOrSlice
	}{
		{input: `"eth0"`, want: goubus.StringOrSlice{"eth0"}},
		{input: `["eth0","eth1"]`, want: goubus.StringOrSlice{"eth0", "eth1"}},
		{input: `""`, want: nil},
		{input: `null`, want: nil},
	}
	for _, test := range cases {
		var got goubus.StringOrSlice
		err := json.Unmarshal([]byte(test.input), &got)
		if err != nil {
			t.Fatalf("Unmarshal(%s): %v", test.input, err)
		}
		if len(got) != len(test.want) {
			t.Errorf("Unmarshal(%s) = %#v, want %#v", test.input, got, test.want)
		}
	}
	if got := (goubus.StringOrSlice{"first", "second"}).First(); got != "first" {
		t.Errorf("First() = %q, want first", got)
	}
	if got := (goubus.StringOrSlice{}).First(); got != "" {
		t.Errorf("First() for empty slice = %q, want empty", got)
	}
}

func testStringIntJSON(t *testing.T) {
	for _, input := range []string{`4014`, `"4014"`} {
		var value goubus.StringInt
		err := json.Unmarshal([]byte(input), &value)
		if err != nil || value.Int64() != 4014 || value.Int() != 4014 {
			t.Errorf("Unmarshal(%s) = %d, %v", input, value, err)
		}
	}
	for _, input := range []string{`""`, `null`} {
		var value goubus.StringInt
		err := json.Unmarshal([]byte(input), &value)
		if err != nil || value != 0 {
			t.Errorf("Unmarshal(%s) = %d, %v", input, value, err)
		}
	}
	var invalid goubus.StringInt
	err := json.Unmarshal([]byte(`"nope"`), &invalid)
	if !errors.Is(err, goubus.ErrInvalidResponse) {
		t.Errorf("invalid integer error = %v, want ErrInvalidResponse", err)
	}
}

func testUnixTimeJSON(t *testing.T) {
	want := time.Unix(1_700_000_000, 0).UTC()
	for _, input := range []string{`1700000000`, `"1700000000"`, `"2023-11-14T22:13:20Z"`} {
		var value goubus.UnixTime
		err := json.Unmarshal([]byte(input), &value)
		if err != nil || !value.Equal(want) {
			t.Errorf("Unmarshal(%s) = %v, %v", input, value.Time, err)
		}
	}
	var fractional goubus.UnixTime
	err := json.Unmarshal([]byte(`1.5`), &fractional)
	if err != nil || fractional.Unix() != 1 || fractional.Nanosecond() != 500_000_000 {
		t.Errorf("fractional timestamp = %v, %v", fractional.Time, err)
	}
	for _, input := range []string{`0`, `"0"`, `""`, `null`} {
		var value goubus.UnixTime
		err = json.Unmarshal([]byte(input), &value)
		if err != nil || !value.IsZero() {
			t.Errorf("zero Unmarshal(%s) = %v, %v", input, value.Time, err)
		}
	}
	var outOfRange goubus.UnixTime
	err = json.Unmarshal([]byte(`9223372036854775808`), &outOfRange)
	if !errors.Is(err, goubus.ErrInvalidResponse) {
		t.Errorf("MaxInt64 boundary error = %v, want ErrInvalidResponse", err)
	}
	encoded, err := json.Marshal(goubus.UnixTime{})
	if err != nil || string(encoded) != "0" {
		t.Errorf("Marshal zero = %s, %v", encoded, err)
	}
}
