package goubus

import (
	"context"
	"encoding/json"
	"slices"
	"strings"
)

// ArgType identifies the type of a ubus method argument.
type ArgType string

const (
	// ArgTypeString identifies string arguments.
	ArgTypeString ArgType = "string"
	// ArgTypeNumber identifies numeric arguments.
	ArgTypeNumber ArgType = "number"
	// ArgTypeBoolean identifies boolean arguments.
	ArgTypeBoolean ArgType = "boolean"
	// ArgTypeArray identifies array arguments.
	ArgTypeArray ArgType = "array"
	// ArgTypeObject identifies object arguments.
	ArgTypeObject ArgType = "object"
	// ArgTypeUnknown identifies arguments with an unrecognized type.
	ArgTypeUnknown ArgType = "unknown"
)

// UnmarshalJSON decodes and normalizes a ubus argument type.
func (argType *ArgType) UnmarshalJSON(data []byte) error {
	var value any
	err := json.Unmarshal(data, &value)
	if err != nil {
		*argType = ArgTypeUnknown

		return err
	}

	var kinds []string
	switch typed := value.(type) {
	case string:
		kinds = []string{typed}
	case []any:
		for _, item := range typed {
			if item, ok := item.(string); ok {
				kinds = append(kinds, item)
			}
		}
	}
	*argType = normalizeArgType(kinds)

	return nil
}

func normalizeArgType(kinds []string) ArgType {
	for _, kind := range kinds {
		switch strings.ToLower(kind) {
		case "string":
			return ArgTypeString
		case "number", "integer", "int32", "int64", "int16", "double":
			return ArgTypeNumber
		case "boolean", "bool", "int8":
			return ArgTypeBoolean
		case "array":
			return ArgTypeArray
		case "object", "table":
			return ArgTypeObject
		}
	}

	return ArgTypeUnknown
}

// MethodSignature maps argument names to their normalized types.
type MethodSignature map[string]ArgType

// ObjectSignatures maps method names to their argument signatures.
type ObjectSignatures map[string]MethodSignature

// Lister lists ubus objects and their method signatures.
type Lister interface {
	// List returns matching ubus objects and their method signatures.
	List(ctx context.Context, patterns ...string) (map[string]ObjectSignatures, error)
}

func normalizePatterns(patterns []string) ([]string, error) {
	if slices.Contains(patterns, "") {
		return nil, ErrInvalidParameter
	}

	return patterns, nil
}

func containsPattern(patterns []string, target string) bool {
	return slices.Contains(patterns, target)
}

// List asks a caller that supports introspection for matching ubus objects.
func List(ctx context.Context, caller Caller, patterns ...string) (map[string]ObjectSignatures, error) {
	patterns, err := normalizePatterns(patterns)
	if err != nil {
		return nil, err
	}
	lister, ok := caller.(Lister)
	if !ok {
		return nil, ErrNotSupported
	}

	return lister.List(ctx, patterns...)
}
