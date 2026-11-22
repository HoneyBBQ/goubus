package goubus

import (
	"context"
	"errors"

	"github.com/honeybbq/goubus/v2/internal/blobmsg"
)

// List queries object and method signatures from the ubus socket.
func (client *SocketClient) List(ctx context.Context, patterns ...string) (map[string]ObjectSignatures, error) {
	patterns, err := normalizePatterns(patterns)
	if err != nil {
		return nil, err
	}

	objects := make(map[string]ObjectSignatures)
	if len(patterns) == 0 || containsPattern(patterns, "*") {
		entries, err := client.listObjects(ctx, "")
		if err != nil {
			return nil, err
		}
		client.mergeListedObjects(objects, entries)

		return objects, nil
	}

	for _, pattern := range patterns {
		entries, err := client.listObjects(ctx, pattern)
		if errors.Is(err, ErrNotFound) {
			continue
		}
		client.mergeListedObjects(objects, entries)
	}

	return objects, nil
}

func (client *SocketClient) mergeListedObjects(destination map[string]ObjectSignatures, entries []map[string]any) {
	for _, entry := range entries {
		path, ok := entry["objpath"].(string)
		if !ok || path == "" {
			continue
		}
		if objectID, ok := blobmsg.ReadUint(entry["objid"]); ok {
			client.objectMu.Lock()
			client.objectCache[path] = objectID
			client.objectMu.Unlock()
		}

		signatures := ObjectSignatures{}
		if rawSignature, ok := entry["signature"].(map[string]any); ok {
			for method, rawArguments := range rawSignature {
				arguments := MethodSignature{}
				if args, ok := rawArguments.(map[string]any); ok {
					for name, rawType := range args {
						arguments[name] = socketArgType(rawType)
					}
				}
				signatures[method] = arguments
			}
		}
		destination[path] = signatures
	}
}

func socketArgType(value any) ArgType {
	code, ok := blobmsg.ReadUint(value)
	if !ok {
		return ArgTypeUnknown
	}

	switch code {
	case blobmsg.TypeString:
		return ArgTypeString
	case blobmsg.TypeInt64, blobmsg.TypeInt32, blobmsg.TypeInt16, blobmsg.TypeDouble:
		return ArgTypeNumber
	case blobmsg.TypeInt8:
		return ArgTypeBoolean
	case blobmsg.TypeArray:
		return ArgTypeArray
	case blobmsg.TypeTable:
		return ArgTypeObject
	default:
		return ArgTypeUnknown
	}
}
