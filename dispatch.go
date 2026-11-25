package goubus

import (
	"context"
	"encoding/json"
	"log/slog"
	"maps"
	"slices"

	"github.com/honeybbq/goubus/v2/internal/blobmsg"
)

func (c *Client) handleIncomingInvoke(connCtx context.Context, hdr *blobmsg.UbusMessageHeader, attrs map[string]any) {
	objID, ok := blobmsg.ReadUint(attrs["objid"])
	if !ok {
		return
	}
	method, _ := attrs["method"].(string)
	noReply, _ := attrs["no_reply"].(bool)

	c.localMu.RLock()
	loc := c.localObjects[objID]
	c.localMu.RUnlock()

	if loc == nil {
		if !noReply {
			c.sendInvokeReply(hdr.Peer, hdr.Seq, objID, nil, StatusNotFound)
		}

		return
	}

	rawJSON := marshalAttrData(attrs["data"])
	if loc.sub != nil {
		if !noReply {
			c.sendInvokeReply(hdr.Peer, hdr.Seq, objID, nil, StatusOK)
		}
		loc.sub.enqueue(method, rawJSON)

		return
	}

	handler, exists := loc.methods[method]
	if !exists {
		if !noReply {
			c.sendInvokeReply(hdr.Peer, hdr.Seq, objID, nil, StatusMethodNotFound)
		}

		return
	}

	user, _ := attrs["user"].(string)
	group, _ := attrs["group"].(string)
	req := CallRequest{
		Object: loc.name,
		Method: method,
		User:   user,
		Group:  group,
		Data:   rawJSON,
	}
	go c.runMethodHandler(connCtx, loc.pub, hdr.Peer, hdr.Seq, objID, noReply, handler, req)
}

func (c *Client) runMethodHandler(
	connCtx context.Context,
	pub *RegisteredObject,
	callerPeer uint32,
	callerSeq uint16,
	objID uint32,
	noReply bool,
	handler MethodHandler,
	req CallRequest,
) {
	reqCtx, cancel := context.WithCancel(connCtx)
	defer cancel()
	if pub != nil {
		if !pub.trackCall(&cancel) {
			return
		}
		defer pub.untrackCall(&cancel)
	}

	resp, err := handler(reqCtx, req)
	if noReply || c.isClosed() {
		return
	}
	if err != nil {
		c.sendInvokeReply(callerPeer, callerSeq, objID, nil, wireStatusFromError(err))

		return
	}

	replyData, normErr := blobmsg.NormalizeArgs(resp)
	if normErr != nil {
		c.logger.Debug("marshal object handler response failed", slog.Any("error", normErr))
		c.sendInvokeReply(callerPeer, callerSeq, objID, nil, StatusInvalidParameter)

		return
	}
	c.sendInvokeReply(callerPeer, callerSeq, objID, replyData, StatusOK)
}

func (c *Client) sendInvokeReply(peer uint32, seq uint16, objID uint32, data map[string]any, status uint32) {
	var dataFrame []byte
	if len(data) > 0 {
		encoded, err := blobmsg.CreateBlobmsgData(data)
		if err == nil {
			body, bodyErr := blobmsg.CreateBlobMessage(map[uint32]any{
				blobmsg.UbusAttrObjID: objID,
				blobmsg.UbusAttrData:  encoded,
			}, []uint32{blobmsg.UbusAttrObjID, blobmsg.UbusAttrData})
			if bodyErr == nil {
				dataFrame = encodeWireFrame(blobmsg.UbusMsgData, seq, peer, body)
			}
		}
	}

	statusBody := blobmsg.EncodeUintAttributes(
		[2]uint32{blobmsg.UbusAttrStatus, status},
		[2]uint32{blobmsg.UbusAttrObjID, objID},
	)
	statusFrame := encodeWireFrame(blobmsg.UbusMsgStatus, seq, peer, statusBody)

	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	if c.isClosed() {
		return
	}
	if len(dataFrame) > 0 {
		writeErr := c.writeLocked(dataFrame)
		if writeErr != nil {
			_ = c.closeWithError(writeErr)

			return
		}
	}
	writeErr := c.writeLocked(statusFrame)
	if writeErr != nil {
		_ = c.closeWithError(writeErr)
	}
}

func (c *Client) handleIncomingNotify(attrs map[string]any) {
	objID, ok := blobmsg.ReadUint(attrs["objid"])
	if !ok {
		return
	}
	active, hasActive := attrs["active"].(bool)
	if !hasActive {
		return
	}

	c.localMu.RLock()
	loc := c.localObjects[objID]
	c.localMu.RUnlock()
	if loc != nil && loc.pub != nil {
		loc.pub.notifySubscriberState(active)
	}
}

func (c *Client) handleIncomingUnsubscribe(connCtx context.Context, attrs map[string]any) {
	objID, ok := blobmsg.ReadUint(attrs["objid"])
	if !ok {
		return
	}

	c.localMu.RLock()
	loc := c.localObjects[objID]
	c.localMu.RUnlock()
	if loc != nil && loc.sub != nil {
		loc.sub.handleTargetRemoved(connCtx)
	}
}

func (c *Client) removeLocalObject(objID uint32) {
	c.localMu.Lock()
	delete(c.localObjects, objID)
	c.localMu.Unlock()
}

func (c *Client) shutdownLocalObjects() {
	c.localMu.Lock()
	snapshot := slices.Collect(maps.Values(c.localObjects))
	clear(c.localObjects)
	c.localMu.Unlock()

	for _, loc := range snapshot {
		if loc.sub != nil {
			loc.sub.markClosed()
		}
		if loc.pub != nil {
			loc.pub.markClosed()
		}
	}
}

func marshalAttrData(raw any) json.RawMessage {
	mapped, ok := raw.(map[string]any)
	if !ok || len(mapped) == 0 {
		return nil
	}
	encoded, err := json.Marshal(maps.Clone(mapped))
	if err != nil {
		return nil
	}

	return encoded
}
