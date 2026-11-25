package goubus

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"
	"time"

	"github.com/honeybbq/goubus/v2/internal/blobmsg"
	"github.com/honeybbq/goubus/v2/internal/errs"
)

const (
	defaultUnsubscribeTimeout = 2 * time.Second
	maxSubscriptionQueue      = 1024
)

// Event represents a broadcast message delivered on the ubus system event bus.
type Event struct {
	// Type is the event identifier (for example "network.interface").
	Type string
	// Data holds the raw JSON payload of the event.
	Data json.RawMessage
}

// EventHandler handles an incoming ubus broadcast event.
type EventHandler func(event Event)

// Notification represents a message emitted by a subscribed ubus object.
type Notification struct {
	// Object is the subscribed ubus object path.
	Object string
	// Method is the notification type name emitted by the object.
	Method string
	// Data holds the raw JSON payload of the notification.
	Data json.RawMessage
}

// NotificationHandler handles an incoming ubus object notification.
type NotificationHandler func(notification Notification)

// Subscription manages an active event listener or object subscription on a Client.
type Subscription struct {
	ctx        context.Context
	client     *Client
	cancel     context.CancelFunc
	deliver    func(method string, data json.RawMessage)
	wake       chan struct{}
	queue      []inboundPush
	mu         sync.Mutex
	objID      uint32
	targetID   uint32
	subscribed bool
	closed     bool
}

type inboundPush struct {
	method string
	data   json.RawMessage
}

// Unmarshal decodes the event payload into target.
func (e Event) Unmarshal(target any) error {
	return NewJSONResult(e.Data).Unmarshal(target)
}

// Unmarshal decodes the notification payload into target.
func (n Notification) Unmarshal(target any) error {
	return NewJSONResult(n.Data).Unmarshal(target)
}

func newSubscription(
	subCtx context.Context,
	subCancel context.CancelFunc,
	client *Client,
	deliver func(method string, data json.RawMessage),
) *Subscription {
	return &Subscription{
		ctx:     subCtx,
		cancel:  subCancel,
		client:  client,
		deliver: deliver,
		wake:    make(chan struct{}, 1),
	}
}

// Done returns a channel that closes when the subscription ends, either via
// Close, client shutdown, or removal of the subscribed target object.
func (s *Subscription) Done() <-chan struct{} {
	return s.ctx.Done()
}

// Close unregisters the subscription and stops invoking its handler.
func (s *Subscription) Close() error {
	cleanCtx, cancel := context.WithTimeout(s.client.connCtx, defaultUnsubscribeTimeout)
	defer cancel()

	return s.closeWithContext(cleanCtx)
}

func (s *Subscription) setSubscribed(targetID uint32) {
	s.mu.Lock()
	if !s.closed {
		s.targetID = targetID
		s.subscribed = true
	}
	s.mu.Unlock()
}

func (s *Subscription) handleTargetRemoved(connCtx context.Context) {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()

		return
	}
	s.closed = true
	s.subscribed = false
	s.queue = nil
	objID := s.objID
	s.mu.Unlock()

	s.cancel()
	s.client.removeLocalObject(objID)
	if !s.client.isClosed() {
		go s.client.cleanupDetachedSubscriber(connCtx, objID)
	}
}

func (s *Subscription) closeWithContext(ctx context.Context) error {
	wasOpen, wasSubscribed, targetID := s.markClosedState()
	if !wasOpen {
		return nil
	}

	s.client.removeLocalObject(s.objID)
	if s.client.isClosed() {
		return nil
	}

	var unsubscribeErr error
	if wasSubscribed {
		unsubscribeErr = s.client.sendUnsubscribe(ctx, s.objID, targetID)
	}
	removeErr := s.client.sendRemoveObject(ctx, s.objID)
	if unsubscribeErr != nil {
		return unsubscribeErr
	}

	return removeErr
}

func (s *Subscription) enqueue(method string, data json.RawMessage) {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()

		return
	}
	if len(s.queue) >= maxSubscriptionQueue {
		s.mu.Unlock()
		s.client.logger.Debug("dropped ubus notification due to full subscription queue", slog.String("method", method))

		return
	}
	s.queue = append(s.queue, inboundPush{method: method, data: data})
	s.mu.Unlock()

	select {
	case s.wake <- struct{}{}:
	default:
	}
}

func (s *Subscription) run(ctx context.Context) {
	for {
		for {
			item, ok := s.pop()
			if !ok {
				break
			}
			s.deliver(item.method, item.data)
		}

		select {
		case <-s.wake:
		case <-ctx.Done():
			return
		}
	}
}

func (s *Subscription) pop() (inboundPush, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed || len(s.queue) == 0 {
		return inboundPush{}, false
	}
	item := s.queue[0]
	s.queue[0] = inboundPush{}
	s.queue = s.queue[1:]

	return item, true
}

func (s *Subscription) markClosed() bool {
	wasOpen, _, _ := s.markClosedState()

	return wasOpen
}

func (s *Subscription) markClosedState() (bool, bool, uint32) {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()

		return false, false, 0
	}
	wasSubscribed := s.subscribed
	targetID := s.targetID
	s.closed = true
	s.subscribed = false
	s.queue = nil
	s.mu.Unlock()

	s.cancel()

	return true, wasSubscribed, targetID
}

// SendEvent broadcasts an event on the ubus system event bus (equivalent to `ubus send`).
func (c *Client) SendEvent(ctx context.Context, id string, data any) error {
	ctx = normalizeContext(ctx)
	if id == "" {
		return errs.Wrapf(ErrInvalidParameter, "event id is required")
	}

	payload, err := blobmsg.NormalizeArgs(data)
	if err != nil {
		return errs.Wrapf(ErrInvalidParameter, "normalize event payload: %v", err)
	}

	args := map[string]any{
		"id":   id,
		"data": payload,
	}
	_, err = c.invokeByID(ctx, "event", ubusSystemObjectEvent, "send", args)

	return err
}

// Listen registers an event listener matching pattern (equivalent to `ubus listen`).
// An empty pattern listens to all events ("*").
func (c *Client) Listen(ctx context.Context, pattern string, handler EventHandler) (*Subscription, error) {
	ctx = normalizeContext(ctx)
	if handler == nil {
		return nil, errs.Wrapf(ErrInvalidParameter, "event handler is required")
	}
	if pattern == "" {
		pattern = "*"
	}

	subCtx, subCancel := context.WithCancel(context.WithoutCancel(ctx))
	sub := newSubscription(subCtx, subCancel, c, func(method string, data json.RawMessage) {
		handler(Event{Type: method, Data: data})
	})

	objID, err := c.registerAnonymousObject(ctx, sub)
	if err != nil {
		sub.markClosed()

		return nil, err
	}
	sub.objID = objID

	regArgs := map[string]any{
		"object":  objID,
		"pattern": pattern,
	}
	_, err = c.invokeByID(ctx, "event", ubusSystemObjectEvent, "register", regArgs)
	if err != nil {
		_ = sub.closeWithContext(ctx)

		return nil, err
	}

	go sub.run(subCtx)

	return sub, nil
}

// Subscribe subscribes to notifications emitted by a named ubus object (equivalent to `ubus subscribe`).
func (c *Client) Subscribe(
	ctx context.Context,
	object string,
	handler NotificationHandler,
) (*Subscription, error) {
	ctx = normalizeContext(ctx)
	if object == "" {
		return nil, errs.Wrapf(ErrInvalidParameter, "target object name is required")
	}
	if handler == nil {
		return nil, errs.Wrapf(ErrInvalidParameter, "notification handler is required")
	}

	targetID, fromCache, err := c.resolveSubscribeTarget(ctx, object)
	if err != nil {
		return nil, err
	}

	subCtx, subCancel := context.WithCancel(context.WithoutCancel(ctx))
	sub := newSubscription(subCtx, subCancel, c, func(method string, data json.RawMessage) {
		handler(Notification{Object: object, Method: method, Data: data})
	})

	objID, err := c.registerAnonymousObject(ctx, sub)
	if err != nil {
		sub.markClosed()

		return nil, err
	}
	sub.objID = objID

	targetID, err = c.subscribeWithRefresh(ctx, object, objID, targetID, fromCache)
	if err != nil {
		_ = sub.closeWithContext(ctx)

		return nil, err
	}
	sub.setSubscribed(targetID)

	go sub.run(subCtx)

	return sub, nil
}

func (c *Client) resolveSubscribeTarget(ctx context.Context, object string) (uint32, bool, error) {
	targetID, fromCache, err := c.getObjectID(ctx, object)
	if err != nil {
		code, ok, statusErr := lookupStatusError(err)
		if ok {
			return 0, false, &CallError{Err: statusErr, Service: object, Method: "subscribe", Code: code}
		}

		return 0, false, err
	}

	return targetID, fromCache, nil
}

func (c *Client) subscribeWithRefresh(
	ctx context.Context,
	object string,
	subscriberID, targetID uint32,
	fromCache bool,
) (uint32, error) {
	err := c.sendSubscribe(ctx, object, subscriberID, targetID)
	if fromCache && isNotFoundCallError(err) {
		c.evictObjectID(object, targetID)
		freshID, _, lookupErr := c.resolveSubscribeTarget(ctx, object)
		if lookupErr != nil {
			return 0, lookupErr
		}

		return freshID, c.sendSubscribe(ctx, object, subscriberID, freshID)
	}

	return targetID, err
}

func (c *Client) registerAnonymousObject(ctx context.Context, sub *Subscription) (uint32, error) {
	body := []byte{0, 0, 0, 4}
	var registeredID uint32

	frames, status, err := c.sendRequestWithHook(
		ctx,
		"ubus add_object",
		blobmsg.UbusMsgAddObject,
		0,
		body,
		func(attrs map[string]any) {
			if id, ok := blobmsg.ReadUint(attrs["objid"]); ok && registeredID == 0 {
				registeredID = id
				c.localMu.Lock()
				c.localObjects[id] = &localObject{id: id, sub: sub}
				c.localMu.Unlock()
				if c.isClosed() {
					sub.markClosed()
				}
			}
		},
	)
	if err != nil {
		if registeredID != 0 {
			c.removeLocalObject(registeredID)
		}

		return 0, err
	}
	if status != StatusOK {
		if registeredID != 0 {
			c.removeLocalObject(registeredID)
		}

		return 0, &CallError{Err: StatusError(int(status)), Method: "add_object", Code: int(status)}
	}

	if registeredID == 0 {
		objID, ok := extractRegisteredObjID(frames)
		if !ok {
			return 0, errs.Wrapf(ErrInvalidResponse, "add_object response missing objid")
		}
		registeredID = objID
		c.localMu.Lock()
		c.localObjects[objID] = &localObject{id: objID, sub: sub}
		c.localMu.Unlock()
	}

	return registeredID, nil
}

func (c *Client) sendSubscribe(ctx context.Context, object string, subscriberID, targetID uint32) error {
	attrs := map[uint32]any{
		blobmsg.UbusAttrObjID:  subscriberID,
		blobmsg.UbusAttrTarget: targetID,
	}
	body, err := blobmsg.CreateBlobMessage(attrs, []uint32{blobmsg.UbusAttrObjID, blobmsg.UbusAttrTarget})
	if err != nil {
		return err
	}

	_, status, err := c.sendRequest(ctx, "ubus subscribe", blobmsg.UbusMsgSubscribe, 0, body)
	if err != nil {
		return err
	}
	if status != StatusOK {
		return &CallError{Err: StatusError(int(status)), Service: object, Method: "subscribe", Code: int(status)}
	}

	return nil
}

func (c *Client) sendUnsubscribe(ctx context.Context, subscriberID, targetID uint32) error {
	attrs := map[uint32]any{
		blobmsg.UbusAttrObjID:  subscriberID,
		blobmsg.UbusAttrTarget: targetID,
	}
	body, err := blobmsg.CreateBlobMessage(attrs, []uint32{blobmsg.UbusAttrObjID, blobmsg.UbusAttrTarget})
	if err != nil {
		return err
	}

	_, status, err := c.sendRequest(ctx, "ubus unsubscribe", blobmsg.UbusMsgUnsubscribe, 0, body)
	if err != nil {
		return err
	}
	if status != StatusOK {
		return &CallError{Err: StatusError(int(status)), Method: "unsubscribe", Code: int(status)}
	}

	return nil
}

func (c *Client) cleanupDetachedSubscriber(connCtx context.Context, objID uint32) {
	cleanCtx, cancel := context.WithTimeout(connCtx, defaultUnsubscribeTimeout)
	defer cancel()

	_ = c.sendRemoveObject(cleanCtx, objID)
}
