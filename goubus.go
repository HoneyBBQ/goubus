package goubus

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"slices"
	"strings"

	"github.com/honeybbq/goubus/v2/internal/blobmsg"
	"github.com/honeybbq/goubus/v2/internal/errs"
)

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

var discardLogger = slog.New(slog.DiscardHandler)

// Caller executes a single ubus method call.
type Caller interface {
	// Call invokes one service method and returns its un-decoded result.
	Call(ctx context.Context, service, method string, data any) (Result, error)
}

// CallerFunc adapts a function to the Caller interface.
type CallerFunc func(ctx context.Context, service, method string, data any) (Result, error)

// CallMiddleware wraps a Caller with call-specific behavior.
type CallMiddleware func(next Caller) Caller

// Transport combines a Caller with resource cleanup.
type Transport interface {
	Caller
	// Close releases resources owned by the transport.
	Close() error
}

// Result decodes the response data into a target value.
type Result interface {
	// Unmarshal decodes the response data into target.
	Unmarshal(target any) error
}

// ArgType identifies the type of a ubus method argument.
type ArgType string

// MethodSignature maps argument names to their normalized types.
type MethodSignature map[string]ArgType

// ObjectSignatures maps method names to their argument signatures.
type ObjectSignatures map[string]MethodSignature

// Lister lists ubus objects and their method signatures.
type Lister interface {
	// List returns matching ubus objects and their method signatures.
	List(ctx context.Context, patterns ...string) (map[string]ObjectSignatures, error)
}

type jsonResult struct {
	raw json.RawMessage
}

// NewJSONResult creates a Result that decodes raw JSON response data.
func NewJSONResult(raw json.RawMessage) Result {
	return jsonResult{raw: slices.Clone(raw)}
}

// Call invokes the adapted function.
func (f CallerFunc) Call(ctx context.Context, service, method string, data any) (Result, error) {
	return f(ctx, service, method, data)
}

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

// Unmarshal decodes a JSON response into target.
func (r jsonResult) Unmarshal(target any) error {
	trimmed := bytes.TrimSpace(r.raw)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
		return ErrNoData
	}

	decodeErr := json.Unmarshal(trimmed, target)
	if decodeErr != nil {
		return errs.Wrapf(ErrInvalidResponse, "decode JSON result: %v", decodeErr)
	}

	return nil
}

// ChainMiddleware applies middleware from first to last around caller.
func ChainMiddleware(caller Caller, middlewares ...CallMiddleware) Caller {
	for _, middleware := range slices.Backward(middlewares) {
		if middleware != nil {
			caller = middleware(caller)
		}
	}

	return caller
}

// Call invokes caller and decodes its result into T.
// For status-only methods that return no payload (such as "uci commit"), call Caller.Call directly.
func Call[T any](ctx context.Context, caller Caller, service, method string, data any) (T, error) {
	var target T

	resp, err := caller.Call(ctx, service, method, data)
	if err != nil {
		return target, err
	}

	err = resp.Unmarshal(&target)
	if err != nil {
		var zero T

		return zero, err
	}

	return target, nil
}

// List queries a caller that implements Lister for matching ubus objects and signatures.
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

func argTypeToBlobmsg(argType ArgType) (int32, bool) {
	switch argType {
	case ArgTypeString:
		return blobmsg.TypeString, true
	case ArgTypeNumber:
		return blobmsg.TypeInt32, true
	case ArgTypeBoolean:
		return blobmsg.TypeBool, true
	case ArgTypeArray:
		return blobmsg.TypeArray, true
	case ArgTypeObject:
		return blobmsg.TypeTable, true
	case ArgTypeUnknown:
		return blobmsg.TypeUnspec, true
	default:
		return 0, false
	}
}

func normalizePatterns(patterns []string) ([]string, error) {
	if slices.Contains(patterns, "") {
		return nil, ErrInvalidParameter
	}

	return patterns, nil
}
