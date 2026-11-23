package goubus

import (
	"bytes"
	"encoding/json"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/honeybbq/goubus/v2/internal/errs"
)

// StringOrSlice accepts either one string or a list of strings.
type StringOrSlice []string

// UnmarshalJSON decodes either one string or a JSON array of strings.
func (s *StringOrSlice) UnmarshalJSON(data []byte) error {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
		*s = nil

		return nil
	}

	if trimmed[0] == '"' {
		var value string
		err := json.Unmarshal(trimmed, &value)
		if err != nil {
			return err
		}
		if value == "" {
			*s = nil
		} else {
			*s = StringOrSlice{value}
		}

		return nil
	}

	var values []string
	err := json.Unmarshal(trimmed, &values)
	if err != nil {
		return err
	}

	*s = StringOrSlice(values)

	return nil
}

// MarshalJSON encodes the values as a JSON string array.
func (s StringOrSlice) MarshalJSON() ([]byte, error) {
	return json.Marshal([]string(s))
}

// First returns the first string, or an empty string when the slice is empty.
func (s StringOrSlice) First() string {
	if len(s) == 0 {
		return ""
	}

	return s[0]
}

// StringInt accepts an integer encoded as a JSON number or string.
type StringInt int64

// UnmarshalJSON decodes an integer from a JSON number or string.
func (i *StringInt) UnmarshalJSON(data []byte) error {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) || bytes.Equal(trimmed, []byte(`""`)) {
		*i = 0

		return nil
	}

	var text string
	if trimmed[0] == '"' {
		err := json.Unmarshal(trimmed, &text)
		if err != nil {
			return errs.Wrapf(ErrInvalidResponse, "decode integer string: %v", err)
		}
	} else {
		text = string(trimmed)
	}

	value, err := strconv.ParseInt(text, 10, 64)
	if err != nil {
		return errs.Wrapf(ErrInvalidResponse, "parse integer %q: %v", text, err)
	}

	*i = StringInt(value)

	return nil
}

// MarshalJSON encodes the integer as a JSON number.
func (i StringInt) MarshalJSON() ([]byte, error) {
	return json.Marshal(int64(i))
}

// Int returns the value as an int.
func (i StringInt) Int() int {
	return int(i)
}

// Int64 returns the value as an int64.
func (i StringInt) Int64() int64 {
	return int64(i)
}

// UnixTime accepts Unix seconds or RFC3339 timestamps.
type UnixTime struct {
	// Time is the timestamp value.
	time.Time
}

// IsZero reports whether the time is zero.
func (t UnixTime) IsZero() bool {
	return t.Time.IsZero()
}

// MarshalJSON encodes the timestamp as Unix seconds.
func (t UnixTime) MarshalJSON() ([]byte, error) {
	if t.IsZero() {
		return []byte("0"), nil
	}

	return json.Marshal(t.Unix())
}

// UnmarshalJSON decodes Unix seconds or an RFC3339 timestamp.
func (t *UnixTime) UnmarshalJSON(data []byte) error {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) || bytes.Equal(trimmed, []byte(`""`)) {
		*t = UnixTime{}

		return nil
	}

	var text string
	if trimmed[0] == '"' {
		parsedText, parsed, direct, parseErr := parseUnixTimeString(trimmed)
		if parseErr != nil {
			return parseErr
		}
		if direct {
			*t = UnixTime{Time: parsed}

			return nil
		}
		text = parsedText
	} else {
		text = string(trimmed)
	}

	seconds, err := strconv.ParseFloat(strings.TrimSpace(text), 64)
	if err != nil || math.IsNaN(seconds) || math.IsInf(seconds, 0) ||
		seconds >= math.Exp2(63) || seconds < -math.Exp2(63) {
		return errs.Wrapf(ErrInvalidResponse, "parse Unix timestamp %q", text)
	}
	if seconds == 0 {
		*t = UnixTime{}

		return nil
	}

	whole, fraction := math.Modf(seconds)
	nanoseconds := int64(math.Round(fraction * float64(time.Second)))
	wholeSeconds := int64(whole)
	if nanoseconds == int64(time.Second) {
		if wholeSeconds == math.MaxInt64 {
			return errs.Wrapf(ErrInvalidResponse, "parse Unix timestamp %q", text)
		}
		wholeSeconds++
		nanoseconds = 0
	}
	*t = UnixTime{Time: time.Unix(wholeSeconds, nanoseconds).UTC()}

	return nil
}

func parseUnixTimeString(data []byte) (string, time.Time, bool, error) {
	var text string
	decodeErr := json.Unmarshal(data, &text)
	if decodeErr != nil {
		return "", time.Time{}, false, errs.Wrapf(ErrInvalidResponse, "decode timestamp string: %v", decodeErr)
	}
	if text == "" || text == "0" {
		return "", time.Time{}, true, nil
	}
	parsed, err := time.Parse(time.RFC3339Nano, text)
	if err == nil {
		return "", parsed.UTC(), true, nil
	}

	return text, time.Time{}, false, nil
}
