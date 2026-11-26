package goubus

import (
	"cmp"
	"context"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
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
	onData      func(map[string]any)
	done        chan struct{}
	streamWake  chan struct{}
	dataFrames  []map[string]any
	streamQueue []map[string]any
	recvFD      int
	status      uint32
	streaming   bool
	wantFD      bool
}

type requestOptions struct {
	onData    func(map[string]any)
	sendFile  *os.File
	streaming bool
	wantFD    bool
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
	frames, status, _, err := c.sendRequestWithOptions(
		ctx, action, msgType, peer, body, requestOptions{onData: onData},
	)

	return frames, status, err
}

func (c *Client) sendRequestWithOptions(
	ctx context.Context,
	action string,
	msgType uint8,
	peer uint32,
	body []byte,
	opts requestOptions,
) ([]map[string]any, uint32, int, error) {
	ctxErr := wrapCtxErr(ctx, action)
	if ctxErr != nil {
		return nil, 0, noFD, ctxErr
	}
	if c.isClosed() {
		return nil, 0, noFD, ErrClosed
	}

	seq, req, err := c.registerPending(opts)
	if err != nil {
		return nil, 0, noFD, err
	}
	defer c.unregisterPending(seq)

	frame := encodeWireFrame(msgType, seq, peer, body)
	writeErr := c.writeFrame(frame, msgType, seq, body, opts.sendFile)
	if writeErr != nil {
		return nil, 0, noFD, writeErr
	}

	return c.awaitPending(ctx, action, req)
}

func (c *Client) sendStreamRequest(
	ctx context.Context,
	action string,
	msgType uint8,
	peer uint32,
	body []byte,
	onFrame func(Result) error,
) (uint32, int, error) {
	ctxErr := wrapCtxErr(ctx, action)
	if ctxErr != nil {
		return 0, 0, ctxErr
	}
	if c.isClosed() {
		return 0, 0, ErrClosed
	}

	seq, req, err := c.registerPending(requestOptions{streaming: true})
	if err != nil {
		return 0, 0, err
	}
	defer c.unregisterPending(seq)

	frame := encodeWireFrame(msgType, seq, peer, body)
	writeErr := c.writeFrame(frame, msgType, seq, body, nil)
	if writeErr != nil {
		return 0, 0, writeErr
	}

	return c.awaitStream(ctx, action, req, onFrame)
}

func (c *Client) awaitPending(
	ctx context.Context,
	action string,
	req *pendingRequest,
) ([]map[string]any, uint32, int, error) {
	var timeoutCh <-chan time.Time
	if c.readTimeout > 0 {
		timer := time.NewTimer(c.readTimeout)
		defer timer.Stop()
		timeoutCh = timer.C
	}

	select {
	case <-req.done:
		return req.dataFrames, req.status, c.claimPendingFD(req), nil
	case <-ctx.Done():
		if c.isClosed() {
			return nil, 0, noFD, c.closedReason()
		}

		return nil, 0, noFD, wrapCtxErr(ctx, action)
	case <-timeoutCh:
		return nil, 0, noFD, errs.Wrapf(ErrTimeout, "%s: read timeout after %s", action, c.readTimeout)
	case <-c.connCtx.Done():
		return nil, 0, noFD, c.closedReason()
	}
}

func (c *Client) awaitStream(
	ctx context.Context,
	action string,
	req *pendingRequest,
	onFrame func(Result) error,
) (uint32, int, error) {
	var timeoutCh <-chan time.Time
	if c.readTimeout > 0 {
		timer := time.NewTimer(c.readTimeout)
		defer timer.Stop()
		timeoutCh = timer.C
	}

	delivered := 0
	for {
		n, cbErr := c.drainStreamQueue(req, onFrame)
		delivered += n
		if cbErr != nil {
			return 0, delivered, cbErr
		}

		select {
		case <-req.streamWake:
		case <-req.done:
			n, cbErr = c.drainStreamQueue(req, onFrame)
			delivered += n
			if cbErr != nil {
				return 0, delivered, cbErr
			}

			return req.status, delivered, nil
		case <-ctx.Done():
			if c.isClosed() {
				return 0, delivered, c.closedReason()
			}

			return 0, delivered, wrapCtxErr(ctx, action)
		case <-timeoutCh:
			return 0, delivered, errs.Wrapf(ErrTimeout, "%s: read timeout after %s", action, c.readTimeout)
		case <-c.connCtx.Done():
			return 0, delivered, c.closedReason()
		}
	}
}

func (c *Client) drainStreamQueue(req *pendingRequest, onFrame func(Result) error) (int, error) {
	delivered := 0
	for {
		c.pendingMu.Lock()
		if len(req.streamQueue) == 0 {
			c.pendingMu.Unlock()

			return delivered, nil
		}
		attrs := req.streamQueue[0]
		req.streamQueue[0] = nil
		req.streamQueue = req.streamQueue[1:]
		c.pendingMu.Unlock()

		delivered++
		err := onFrame(&socketResult{data: blobmsg.ExtractDataSection(attrs)})
		if err != nil {
			return delivered, err
		}
	}
}

func (c *Client) claimPendingFD(req *pendingRequest) int {
	c.pendingMu.Lock()
	fd := req.recvFD
	req.recvFD = noFD
	c.pendingMu.Unlock()

	return fd
}

func (c *Client) registerPending(opts requestOptions) (uint16, *pendingRequest, error) {
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
			req := &pendingRequest{
				done:      make(chan struct{}),
				onData:    opts.onData,
				recvFD:    noFD,
				streaming: opts.streaming,
				wantFD:    opts.wantFD,
			}
			if opts.streaming {
				req.streamWake = make(chan struct{}, 1)
			}
			c.pending[seq] = req

			return seq, req, nil
		}
	}

	return 0, nil, errs.Wrapf(ErrConnectionFailed, "exhausted in-flight ubus sequence numbers")
}

func (c *Client) unregisterPending(seq uint16) {
	c.pendingMu.Lock()
	req := c.pending[seq]
	delete(c.pending, seq)
	unclaimedFD := noFD
	if req != nil {
		unclaimedFD = req.recvFD
		req.recvFD = noFD
	}
	c.pendingMu.Unlock()

	closeFD(unclaimedFD)
}

func (c *Client) readLoop(connCtx context.Context) {
	defer close(c.readDone)

	for {
		hdr, payload, fd, err := readWireMessage(c.conn)
		if err != nil {
			closeFD(fd)
			_ = c.closeWithError(err)

			return
		}

		if hdr.Type == blobmsg.UbusMsgMonitor {
			closeFD(fd)
			c.processMonitorPayload(payload)

			continue
		}

		attrs, parseErr := blobmsg.ParseTopLevelAttributes(payload)
		if parseErr != nil {
			closeFD(fd)
			_ = c.closeWithError(errs.Wrapf(ErrInvalidResponse, "parse ubus frame: %v", parseErr))

			return
		}

		c.dispatchFrame(connCtx, hdr, attrs, fd)
	}
}

func (c *Client) processMonitorPayload(payload []byte) {
	monFrame, err := blobmsg.ParseMonitorAttributes(payload)
	if err != nil {
		c.logger.Debug("ignored malformed ubus monitor frame", slog.Any("error", err))

		return
	}
	c.handleIncomingMonitor(monFrame)
}

func (c *Client) dispatchFrame(connCtx context.Context, hdr *blobmsg.UbusMessageHeader, attrs map[string]any, fd int) {
	switch hdr.Type {
	case blobmsg.UbusMsgData:
		c.completePendingData(hdr.Seq, attrs, fd)
	case blobmsg.UbusMsgStatus:
		c.completePendingStatus(hdr.Seq, attrs, fd)
	case blobmsg.UbusMsgInvoke:
		closeFD(fd)
		c.handleIncomingInvoke(connCtx, hdr, attrs)
	case blobmsg.UbusMsgNotify:
		closeFD(fd)
		c.handleIncomingNotify(attrs)
	case blobmsg.UbusMsgUnsubscribe:
		closeFD(fd)
		c.handleIncomingUnsubscribe(connCtx, attrs)
	default:
		closeFD(fd)
		c.logger.Debug("ignored ubus message", slog.Int("type", int(hdr.Type)), slog.Int("seq", int(hdr.Seq)))
	}
}

func (c *Client) completePendingData(seq uint16, attrs map[string]any, fd int) {
	c.logger.Debug("Parsed data attributes", slog.String("data", previewJSON(attrs, logJSONLimit)))

	var (
		hook       func(map[string]any)
		streamWake chan struct{}
		unusedFD   = fd
	)
	c.pendingMu.Lock()
	req, ok := c.pending[seq]
	if ok {
		if fd >= 0 && req.wantFD && req.recvFD < 0 {
			req.recvFD = fd
			unusedFD = noFD
		}
		if req.streaming {
			req.streamQueue = append(req.streamQueue, attrs)
			streamWake = req.streamWake
		} else if len(attrs) > 0 {
			req.dataFrames = append(req.dataFrames, attrs)
			hook = req.onData
		}
	}
	c.pendingMu.Unlock()

	closeFD(unusedFD)
	if streamWake != nil {
		select {
		case streamWake <- struct{}{}:
		default:
		}
	}
	if hook != nil {
		hook(attrs)
	}
}

func (c *Client) completePendingStatus(seq uint16, attrs map[string]any, fd int) {
	status, _ := blobmsg.ReadUint(attrs["status"])
	unusedFD := fd

	c.pendingMu.Lock()
	req, ok := c.pending[seq]
	if ok {
		if fd >= 0 && req.wantFD && req.recvFD < 0 {
			req.recvFD = fd
			unusedFD = noFD
		}
		delete(c.pending, seq)
		req.status = status
		close(req.done)
	}
	c.pendingMu.Unlock()

	closeFD(unusedFD)
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

func (c *Client) writeFrame(frame []byte, msgType uint8, seq uint16, body []byte, sendFile *os.File) error {
	if len(body) > blobmsg.MaxMessageBytes {
		return errs.Wrapf(
			ErrInvalidParameter,
			"ubus message body %d bytes exceeds max %d",
			len(body),
			blobmsg.MaxMessageBytes,
		)
	}

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

	err := c.writeLocked(frame, sendFile)
	if err != nil {
		if errors.Is(err, ErrInvalidParameter) {
			return err
		}
		if c.isClosed() {
			return c.closedReason()
		}
		_ = c.closeWithError(err)

		return err
	}

	return nil
}

func (c *Client) writeLocked(frame []byte, sendFile *os.File) error {
	err := c.conn.SetWriteDeadline(deadlineFromTimeout(c.writeTimeout))
	if err != nil {
		return errs.Wrapf(ErrConnectionFailed, "set write deadline: %v", err)
	}

	err = writeWireBytes(c.conn, frame, sendFile)
	if err != nil {
		if errors.Is(err, ErrInvalidParameter) {
			return err
		}

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

func encodeWireFrame(msgType uint8, seq uint16, peer uint32, body []byte) []byte {
	frame := make([]byte, blobmsg.HeaderBytes+len(body))
	frame[1] = msgType
	binary.BigEndian.PutUint16(frame[2:4], seq)
	binary.BigEndian.PutUint32(frame[4:8], peer)
	copy(frame[blobmsg.HeaderBytes:], body)

	return frame
}

func deadlineFromTimeout(timeout time.Duration) time.Time {
	if timeout <= 0 {
		return time.Time{}
	}

	return time.Now().Add(timeout)
}

func normalizeContext(ctx context.Context) context.Context {
	return cmp.Or(ctx, context.Background())
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
