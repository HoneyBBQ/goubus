package rpc

import (
	"context"
	"encoding/json"
	"slices"

	"github.com/honeybbq/goubus/v2"
	"github.com/honeybbq/goubus/v2/internal/errs"
)

// List queries the sessionless JSON-RPC introspection method on uhttpd-mod-ubus.
func (c *Client) List(ctx context.Context, patterns ...string) (map[string]goubus.ObjectSignatures, error) {
	ctx = normalizeContext(ctx)
	if c.isClosed() {
		return nil, goubus.ErrClosed
	}
	if slices.Contains(patterns, "") {
		return nil, goubus.ErrInvalidParameter
	}
	if len(patterns) == 0 || slices.Contains(patterns, "*") {
		patterns = []string{"*"}
	}

	payload, err := c.requestBody("list", patterns)
	if err != nil {
		return nil, err
	}
	body, err := c.post(ctx, payload, false)
	if err != nil {
		return nil, err
	}

	var response struct {
		Error  *jsonRPCError   `json:"error"`
		Result json.RawMessage `json:"result"`
	}
	decodeErr := json.Unmarshal(body, &response)
	if decodeErr != nil {
		return nil, errs.Wrapf(goubus.ErrInvalidResponse, "decode list response: %v", decodeErr)
	}
	if response.Error != nil {
		return nil, &goubus.CallError{
			Err: mapJSONRPCError(response.Error.Code), Method: "list",
			Message: response.Error.Message, Code: response.Error.Code,
		}
	}

	objects := make(map[string]goubus.ObjectSignatures)
	if len(response.Result) == 0 || string(response.Result) == "null" {
		return objects, nil
	}
	decodeErr = json.Unmarshal(response.Result, &objects)
	if decodeErr != nil {
		return nil, errs.Wrapf(goubus.ErrInvalidResponse, "decode list signatures: %v", decodeErr)
	}

	return objects, nil
}
