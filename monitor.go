package goubus

import (
	"context"
	"encoding/json"

	"github.com/honeybbq/goubus/v2/internal/blobmsg"
	"github.com/honeybbq/goubus/v2/internal/errs"
)

// MonitorMessage represents a ubus frame observed through Client.Monitor (UBUS_MSG_MONITOR).
type MonitorMessage struct {
	// Data holds the JSON-encoded top-level attributes of the monitored frame, when present.
	Data json.RawMessage
	// Client is the peer ID of the ubus client sending or receiving the frame.
	Client uint32
	// Peer is the peer field from the monitored message header.
	Peer uint32
	// Seq is the sequence number from the monitored message header.
	Seq uint16
	// Type is the ubus message type (for example 5 for UBUS_MSG_INVOKE).
	Type uint8
	// Send reports whether ubusd sent the frame to Client (true) or received it from Client (false).
	Send bool
}

// MonitorHandler handles a ubus bus frame captured by Client.Monitor.
type MonitorHandler func(msg MonitorMessage)

// Unmarshal decodes the monitored frame attributes into target.
func (m MonitorMessage) Unmarshal(target any) error {
	return NewJSONResult(m.Data).Unmarshal(target)
}

// Monitor attaches a root-privileged UBUS_MSG_MONITOR stream on ubusd (equivalent to `ubus monitor`).
// Only one monitor can be active per Client; calling Monitor again replaces the previous monitor.
// Close the returned Subscription to remove the monitor from ubusd.
func (c *Client) Monitor(ctx context.Context, handler MonitorHandler) (*Subscription, error) {
	ctx = normalizeContext(ctx)
	if handler == nil {
		return nil, errs.Wrapf(ErrInvalidParameter, "monitor handler is required")
	}
	if c.isClosed() {
		return nil, ErrClosed
	}

	subCtx, subCancel := context.WithCancel(context.WithoutCancel(ctx))
	sub := &Subscription{
		ctx:    subCtx,
		cancel: subCancel,
		client: c,
		deliver: func(item inboundPush) {
			handler(item.monitor)
		},
		wake: make(chan struct{}, 1),
	}
	sub.closeFn = func(closeCtx context.Context) error {
		c.clearMonitor(sub)
		if c.isClosed() {
			return nil
		}
		_, removeErr := c.invokeByID(closeCtx, "monitor", ubusSystemObjectMonitor, "remove", nil)

		return removeErr
	}

	prev := c.replaceMonitor(sub)
	if prev != nil {
		prev.markClosed()
	}

	_, err := c.invokeByID(ctx, "monitor", ubusSystemObjectMonitor, "add", nil)
	if err != nil {
		c.clearMonitor(sub)
		sub.markClosed()

		return nil, err
	}

	go sub.run(subCtx)

	return sub, nil
}

func (c *Client) replaceMonitor(sub *Subscription) *Subscription {
	c.localMu.Lock()
	prev := c.monitor
	c.monitor = sub
	c.localMu.Unlock()

	return prev
}

func (c *Client) clearMonitor(sub *Subscription) {
	c.localMu.Lock()
	if c.monitor == sub {
		c.monitor = nil
	}
	c.localMu.Unlock()
}

func (c *Client) handleIncomingMonitor(frame blobmsg.MonitorFrame) {
	c.localMu.RLock()
	mon := c.monitor
	c.localMu.RUnlock()
	if mon == nil {
		return
	}

	mon.enqueuePush(inboundPush{
		method: "monitor",
		monitor: MonitorMessage{
			Data:   marshalAttrData(frame.Data),
			Client: frame.Client,
			Peer:   frame.Peer,
			Seq:    frame.Seq,
			Type:   frame.Type,
			Send:   frame.Send,
		},
	})
}
