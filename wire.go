package goubus

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/honeybbq/goubus/v2/internal/blobmsg"
	"github.com/honeybbq/goubus/v2/internal/errs"
)

const (
	logBodyLimit    = 512
	logJSONLimit    = 256
	logHexLimit     = 64
	logLongHexLimit = 128
)

type pendingRequest struct {
	onData     func(map[string]any)
	done       chan struct{}
	dataFrames []map[string]any
	status     uint32
}

func (c *Client) sendRequest(
	ctx context.Context,
	action string,
	msgType uint8,
	peer uint32,
	body []byte,
) ([]map[string]any, uint32, error) {
	return c.sendRequestWithHook(ctx, action, msgType, peer, body, nil)
}

func (c *Client) sendRequestWithHook(
	ctx context.Context,
	action string,
	msgType uint8,
	peer uint32,
	body []byte,
	onData func(map[string]any),
) ([]map[string]any, uint32, error) {
	ctxErr := wrapCtxErr(ctx, action)
	if ctxErr != nil {
		return nil, 0, ctxErr
	}
	if c.isClosed() {
		return nil, 0, ErrClosed
	}

	seq, req, err := c.registerPending(onData)
	if err != nil {
		return nil, 0, err
	}
	defer c.unregisterPending(seq)

	frame, err := encodeWireFrame(msgType, seq, peer, body)
	if err != nil {
		return nil, 0, err
	}

	writeErr := c.writeFrame(frame, msgType, seq, body)
	if writeErr != nil {
		return nil, 0, writeErr
	}

	return c.awaitPending(ctx, action, req)
}

func (c *Client) awaitPending(
	ctx context.Context,
	action string,
	req *pendingRequest,
) ([]map[string]any, uint32, error) {
	var timeoutCh <-chan time.Time
	if c.readTimeout > 0 {
		timer := time.NewTimer(c.readTimeout)
		defer timer.Stop()
		timeoutCh = timer.C
	}

	select {
	case <-req.done:
		return req.dataFrames, req.status, nil
	case <-ctx.Done():
		if c.isClosed() {
			return nil, 0, c.closedReason()
		}

		return nil, 0, wrapCtxErr(ctx, action)
	case <-timeoutCh:
		return nil, 0, errs.Wrapf(ErrTimeout, "%s: read timeout after %s", action, c.readTimeout)
	case <-c.connCtx.Done():
		return nil, 0, c.closedReason()
	}
}

func (c *Client) registerPending(onData func(map[string]any)) (uint16, *pendingRequest, error) {
	c.pendingMu.Lock()
	defer c.pendingMu.Unlock()

	if c.isClosed() {
		return 0, nil, ErrClosed
	}

	for range 65535 {
		seq := c.nextSeq
		c.nextSeq++
		if seq == 0 {
			continue
		}
		if _, exists := c.pending[seq]; !exists {
			req := &pendingRequest{done: make(chan struct{}), onData: onData}
			c.pending[seq] = req

			return seq, req, nil
		}
	}

	return 0, nil, errs.Wrapf(ErrConnectionFailed, "exhausted in-flight ubus sequence numbers")
}

func (c *Client) unregisterPending(seq uint16) {
	c.pendingMu.Lock()
	delete(c.pending, seq)
	c.pendingMu.Unlock()
}

func (c *Client) readLoop(connCtx context.Context) {
	defer close(c.readDone)

	for {
		hdr, payload, err := blobmsg.ReadMessage(c.conn)
		if err != nil {
			_ = c.closeWithError(err)

			return
		}

		attrs, parseErr := blobmsg.ParseTopLevelAttributes(payload)
		if parseErr != nil {
			_ = c.closeWithError(errs.Wrapf(ErrInvalidResponse, "parse ubus frame: %v", parseErr))

			return
		}

		c.dispatchFrame(connCtx, hdr, attrs)
	}
}

func (c *Client) dispatchFrame(connCtx context.Context, hdr *blobmsg.UbusMessageHeader, attrs map[string]any) {
	switch hdr.Type {
	case blobmsg.UbusMsgData:
		c.completePendingData(hdr.Seq, attrs)
	case blobmsg.UbusMsgStatus:
		c.completePendingStatus(hdr.Seq, attrs)
	case blobmsg.UbusMsgInvoke:
		c.handleIncomingInvoke(connCtx, hdr, attrs)
	case blobmsg.UbusMsgNotify:
		c.handleIncomingNotify(attrs)
	case blobmsg.UbusMsgUnsubscribe:
		c.handleIncomingUnsubscribe(connCtx, attrs)
	default:
		c.logger.Debug("ignored ubus message", slog.Int("type", int(hdr.Type)), slog.Int("seq", int(hdr.Seq)))
	}
}

func (c *Client) completePendingData(seq uint16, attrs map[string]any) {
	c.logger.Debug("Parsed data attributes", slog.String("data", previewJSON(attrs, logJSONLimit)))

	var hook func(map[string]any)
	c.pendingMu.Lock()
	req, ok := c.pending[seq]
	if ok && len(attrs) > 0 {
		req.dataFrames = append(req.dataFrames, attrs)
		hook = req.onData
	}
	c.pendingMu.Unlock()

	if hook != nil {
		hook(attrs)
	}
}

func (c *Client) completePendingStatus(seq uint16, attrs map[string]any) {
	status, _ := blobmsg.ReadUint(attrs["status"])

	c.pendingMu.Lock()
	req, ok := c.pending[seq]
	if ok {
		delete(c.pending, seq)
		req.status = status
		close(req.done)
	}
	c.pendingMu.Unlock()
}

func (c *Client) setReadTimeoutDeadline() error {
	err := c.conn.SetReadDeadline(deadlineFromTimeout(c.readTimeout))
	if err != nil {
		return errs.Wrapf(ErrConnectionFailed, "set read deadline: %v", err)
	}

	return nil
}

func (c *Client) clearReadTimeoutDeadline() error {
	if c.readTimeout <= 0 {
		return nil
	}

	err := c.conn.SetReadDeadline(time.Time{})
	if err != nil {
		return errs.Wrapf(ErrConnectionFailed, "clear read deadline: %v", err)
	}

	return nil
}

func (c *Client) writeFrame(frame []byte, msgType uint8, seq uint16, body []byte) error {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()

	if c.isClosed() {
		return ErrClosed
	}

	c.logger.Debug("Sending message",
		slog.Int("type", int(msgType)),
		slog.Int("seq", int(seq)),
		slog.Int("body_len", len(body)),
		slog.String("body", hexPreview(body, logHexLimit)))

	err := c.writeLocked(frame)
	if err != nil {
		_ = c.closeWithError(err)

		return err
	}

	return nil
}

func (c *Client) writeLocked(frame []byte) error {
	err := c.conn.SetWriteDeadline(deadlineFromTimeout(c.writeTimeout))
	if err != nil {
		return errs.Wrapf(ErrConnectionFailed, "set write deadline: %v", err)
	}

	_, err = c.conn.Write(frame)
	if err != nil {
		return errs.Wrapf(ErrConnectionFailed, "write message: %v", err)
	}

	if c.writeTimeout > 0 {
		err = c.conn.SetWriteDeadline(time.Time{})
		if err != nil {
			return errs.Wrapf(ErrConnectionFailed, "clear write deadline: %v", err)
		}
	}

	return nil
}

func encodeWireFrame(msgType uint8, seq uint16, peer uint32, body []byte) ([]byte, error) {
	var buf bytes.Buffer
	header := &blobmsg.UbusMessageHeader{
		Version: 0,
		Type:    msgType,
		Seq:     seq,
		Peer:    peer,
	}
	err := blobmsg.EncodeHeader(&buf, header)
	if err != nil {
		return nil, errs.Wrapf(ErrInvalidParameter, "encode header: %v", err)
	}
	if len(body) > 0 {
		_, _ = buf.Write(body)
	}

	return buf.Bytes(), nil
}

func deadlineFromTimeout(timeout time.Duration) time.Time {
	if timeout <= 0 {
		return time.Time{}
	}

	return time.Now().Add(timeout)
}

func normalizeContext(ctx context.Context) context.Context {
	if ctx == nil {
		return context.Background()
	}

	return ctx
}

func wrapCtxErr(ctx context.Context, prefix string) error {
	ctxErr := ctx.Err()
	if ctxErr == nil {
		return nil
	}

	cause := context.Cause(ctx)
	if cause != nil && !errors.Is(cause, ctxErr) {
		return fmt.Errorf("%s: %w: %w", prefix, ctxErr, cause)
	}

	return errs.Wrapf(cause, "%s", prefix)
}

func hexPreview(data []byte, maxLen int) string {
	if len(data) == 0 {
		return ""
	}
	if len(data) > maxLen {
		return hex.EncodeToString(data[:maxLen]) + "..."
	}

	return hex.EncodeToString(data)
}

func previewJSON(v any, maxLen int) string {
	if v == nil {
		return "<nil>"
	}
	raw, err := json.Marshal(v)
	if err != nil {
		return fmt.Sprintf("<error: %v>", err)
	}
	if len(raw) > maxLen {
		return string(raw[:maxLen]) + "..."
	}

	return string(raw)
}
