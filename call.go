package goubus

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"maps"
	"slices"

	"github.com/honeybbq/goubus/v2/internal/blobmsg"
	"github.com/honeybbq/goubus/v2/internal/errs"
)

type socketResult struct {
	data map[string]any
}

func (r *socketResult) Unmarshal(target any) error {
	if len(r.data) == 0 {
		return ErrNoData
	}

	raw, err := json.Marshal(r.data)
	if err != nil {
		return errs.Wrapf(ErrInvalidResponse, "marshal result: %v", err)
	}

	err = json.Unmarshal(raw, target)
	if err != nil {
		return errs.Wrapf(ErrInvalidResponse, "unmarshal result: %v", err)
	}

	return nil
}

// Call invokes a method on a ubus object over the multiplexed socket transport.
func (c *Client) Call(ctx context.Context, service, method string, data any) (Result, error) {
	ctx = normalizeContext(ctx)
	if service == "" || method == "" {
		return nil, errs.Wrapf(ErrInvalidParameter, "service and method required")
	}

	args, err := blobmsg.NormalizeArgs(data)
	if err != nil {
		return nil, errs.Wrapf(ErrInvalidParameter, "normalize arguments: %v", err)
	}
	if c.isClosed() {
		return nil, ErrClosed
	}
	ctxErr := wrapCtxErr(ctx, "ubus call")
	if ctxErr != nil {
		return nil, ctxErr
	}

	return c.callWithRefresh(ctx, service, method, args)
}

// List returns registered ubus objects and their method argument signatures.
func (c *Client) List(ctx context.Context, patterns ...string) (map[string]ObjectSignatures, error) {
	ctx = normalizeContext(ctx)
	patterns, err := normalizePatterns(patterns)
	if err != nil {
		return nil, err
	}
	if c.isClosed() {
		return nil, ErrClosed
	}

	queries := patterns
	if len(queries) == 0 || slices.Contains(queries, "*") {
		queries = []string{""}
	}

	objects := make(map[string]ObjectSignatures)
	for _, pattern := range queries {
		if pattern == "*" {
			pattern = ""
		}
		entries, listErr := c.listObjects(ctx, pattern)
		if listErr != nil {
			code, ok, statusErr := lookupStatusError(listErr)
			if ok {
				if code == StatusNotFound && pattern != "" {
					continue
				}

				return nil, &CallError{Err: statusErr, Service: pattern, Method: "list", Code: code}
			}

			return nil, listErr
		}
		c.collectListedObjects(objects, entries)
	}

	return objects, nil
}

func (c *Client) collectListedObjects(objects map[string]ObjectSignatures, entries []map[string]any) {
	for _, entry := range entries {
		path, ok := entry["objpath"].(string)
		if !ok || path == "" {
			continue
		}
		if id, hasID := blobmsg.ReadUint(entry["objid"]); hasID {
			c.objectMu.Lock()
			c.objectCache[path] = id
			c.objectMu.Unlock()
		}
		objects[path] = decodeSignatureTable(entry["signature"])
	}
}

func (c *Client) callWithRefresh(
	ctx context.Context,
	service, method string,
	args map[string]any,
) (Result, error) {
	objectID, fromCache, err := c.resolveCallTarget(ctx, service, method)
	if err != nil {
		return nil, err
	}

	res, err := c.invokeByID(ctx, service, objectID, method, args)
	if fromCache && isNotFoundCallError(err) {
		c.evictObjectID(service, objectID)
		freshID, _, lookupErr := c.resolveCallTarget(ctx, service, method)
		if lookupErr != nil {
			return nil, lookupErr
		}

		return c.invokeByID(ctx, service, freshID, method, args)
	}

	return res, err
}

func (c *Client) resolveCallTarget(ctx context.Context, service, method string) (uint32, bool, error) {
	objectID, fromCache, err := c.getObjectID(ctx, service)
	if err != nil {
		code, ok, statusErr := lookupStatusError(err)
		if ok {
			return 0, false, &CallError{Err: statusErr, Service: service, Method: method, Code: code}
		}

		return 0, false, err
	}

	return objectID, fromCache, nil
}

func (c *Client) invokeByID(
	ctx context.Context,
	service string,
	objectID uint32,
	method string,
	args map[string]any,
) (Result, error) {
	body, err := c.createInvokeBody(objectID, method, args)
	if err != nil {
		return nil, err
	}

	c.logger.Debug("Invoke",
		slog.String("service", service),
		slog.String("method", method),
		slog.String("args", previewJSON(args, logBodyLimit)),
		slog.String("body", hexPreview(body, logLongHexLimit)))

	frames, status, err := c.sendRequest(ctx, "ubus call", blobmsg.UbusMsgInvoke, objectID, body)
	if err != nil {
		return nil, err
	}
	if status != StatusOK {
		return nil, &CallError{
			Err: StatusError(int(status)), Service: service, Method: method, Code: int(status),
		}
	}

	var merged map[string]any
	for _, attrs := range frames {
		extracted := blobmsg.ExtractDataSection(attrs)
		if len(extracted) == 0 {
			continue
		}
		if merged == nil {
			merged = make(map[string]any, len(extracted))
		}
		maps.Copy(merged, extracted)
	}

	return &socketResult{data: merged}, nil
}

func (c *Client) getObjectID(ctx context.Context, path string) (uint32, bool, error) {
	c.objectMu.RLock()
	if id, ok := c.objectCache[path]; ok {
		c.objectMu.RUnlock()

		return id, true, nil
	}
	c.objectMu.RUnlock()

	objects, err := c.listObjects(ctx, path)
	if err != nil {
		return 0, false, err
	}

	for _, obj := range objects {
		objPath, ok := obj["objpath"].(string)
		if !ok {
			continue
		}
		if id, ok := blobmsg.ReadUint(obj["objid"]); ok {
			c.objectMu.Lock()
			c.objectCache[objPath] = id
			c.objectMu.Unlock()

			if objPath == path {
				return id, false, nil
			}
		}
	}

	return 0, false, errs.Wrapf(ErrNotFound, "object %q not found", path)
}

func (c *Client) evictObjectID(path string, staleID uint32) {
	c.objectMu.Lock()
	if current, ok := c.objectCache[path]; ok && current == staleID {
		delete(c.objectCache, path)
	}
	c.objectMu.Unlock()
}

func (c *Client) listObjects(ctx context.Context, path string) ([]map[string]any, error) {
	attrs := map[uint32]any{}
	if path != "" {
		attrs[blobmsg.UbusAttrObjPath] = path
	}

	body, err := blobmsg.CreateBlobMessage(attrs, []uint32{blobmsg.UbusAttrObjPath})
	if err != nil {
		return nil, err
	}

	frames, statusCode, err := c.sendRequest(ctx, "ubus lookup", blobmsg.UbusMsgLookup, c.peerID, body)
	if err != nil {
		return nil, err
	}
	statusErr := StatusError(int(statusCode))
	if statusErr != nil {
		return nil, statusErr
	}

	return frames, nil
}

func (c *Client) createInvokeBody(objID uint32, method string, args map[string]any) ([]byte, error) {
	argData, err := blobmsg.CreateBlobmsgData(args)
	if err != nil {
		return nil, err
	}

	c.logger.Debug("Create invoke body",
		slog.String("args", previewJSON(args, logBodyLimit)),
		slog.String("blobmsg_data", hexPreview(argData, logLongHexLimit)))

	attrs := map[uint32]any{
		blobmsg.UbusAttrObjID:  objID,
		blobmsg.UbusAttrMethod: method,
	}
	if argData != nil {
		attrs[blobmsg.UbusAttrData] = argData
	}

	return blobmsg.CreateBlobMessage(attrs, []uint32{blobmsg.UbusAttrObjID, blobmsg.UbusAttrMethod, blobmsg.UbusAttrData})
}

func isNotFoundCallError(err error) bool {
	callErr, ok := errors.AsType[*CallError](err)

	return ok && callErr.Code == StatusNotFound
}

func decodeSignatureTable(raw any) ObjectSignatures {
	methods, ok := raw.(map[string]any)
	if !ok {
		return ObjectSignatures{}
	}

	out := make(ObjectSignatures, len(methods))
	for methodName, rawSignature := range methods {
		signature := MethodSignature{}
		args, ok := rawSignature.(map[string]any)
		if ok {
			for argName, rawType := range args {
				signature[argName] = blobmsgCodeToArgType(rawType)
			}
		}
		out[methodName] = signature
	}

	return out
}

func blobmsgCodeToArgType(raw any) ArgType {
	code, ok := blobmsg.ReadUint(raw)
	if !ok {
		if text, isString := raw.(string); isString {
			return normalizeArgType([]string{text})
		}

		return ArgTypeUnknown
	}

	switch code {
	case blobmsg.TypeString:
		return ArgTypeString
	case blobmsg.TypeInt64, blobmsg.TypeInt32, blobmsg.TypeInt16, blobmsg.TypeDouble:
		return ArgTypeNumber
	case blobmsg.TypeBool:
		return ArgTypeBoolean
	case blobmsg.TypeArray:
		return ArgTypeArray
	case blobmsg.TypeTable:
		return ArgTypeObject
	default:
		return ArgTypeUnknown
	}
}
