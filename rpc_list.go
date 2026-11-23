package goubus

import (
	"context"
	"encoding/json"

	"github.com/honeybbq/goubus/v2/internal/errs"
	"github.com/honeybbq/goubus/v2/internal/rpc"
)

// List queries the sessionless JSON-RPC introspection method.
func (client *RPCClient) List(ctx context.Context, patterns ...string) (map[string]ObjectSignatures, error) {
	if client.isClosed() {
		return nil, ErrClosed
	}

	patterns, err := normalizePatterns(patterns)
	if err != nil {
		return nil, err
	}
	if len(patterns) == 0 || containsPattern(patterns, "*") {
		patterns = []string{"*"}
	}

	payload, err := client.requestBody("list", patterns)
	if err != nil {
		return nil, err
	}
	body, err := client.post(ctx, payload, false)
	if err != nil {
		return nil, err
	}

	var response struct {
		Error  *rpc.UbusJSONRPCError `json:"error"`
		Result json.RawMessage       `json:"result"`
	}
	decodeErr := json.Unmarshal(body, &response)
	if decodeErr != nil {
		return nil, errs.Wrapf(ErrInvalidResponse, "decode list response: %v", decodeErr)
	}
	if response.Error != nil {
		return nil, &CallError{
			Err: mapJSONRPCError(response.Error.Code), Method: "list",
			Message: response.Error.Message, Code: response.Error.Code,
		}
	}

	objects := make(map[string]ObjectSignatures)
	if len(response.Result) == 0 || string(response.Result) == "null" {
		return objects, nil
	}
	decodeErr = json.Unmarshal(response.Result, &objects)
	if decodeErr != nil {
		return nil, errs.Wrapf(ErrInvalidResponse, "decode list signatures: %v", decodeErr)
	}
	if objects == nil {
		objects = make(map[string]ObjectSignatures)
	}

	return objects, nil
}
