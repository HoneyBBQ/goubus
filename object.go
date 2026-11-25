package goubus

import (
	"context"
	"encoding/json"
	"sync"
	"sync/atomic"

	"github.com/honeybbq/goubus/v2/internal/blobmsg"
	"github.com/honeybbq/goubus/v2/internal/errs"
)

// CallRequest describes an incoming ubus method invocation on a locally registered object.
type CallRequest struct {
	// Object is the registered ubus object path.
	Object string
	// Method is the invoked method name.
	Method string
	// User is the caller's Unix username when reported by ubusd.
	User string
	// Group is the caller's Unix group when reported by ubusd.
	Group string
	// Data holds the raw JSON arguments supplied by the caller.
	Data json.RawMessage
}

// MethodHandler handles an incoming ubus method invocation.
// Returning a non-nil value sends a UBUS_MSG_DATA reply frame (must marshal to a JSON object).
// Returning a *CallError or status sentinel (such as ErrInvalidParameter) reports that status code.
type MethodHandler func(ctx context.Context, req CallRequest) (any, error)

// Method defines one invokable method and its optional argument policy on an ObjectDef.
type Method struct {
	// Handler executes when ubusd invokes the method.
	Handler MethodHandler
	// Policy declares input argument types advertised in ubus list signatures.
	Policy MethodSignature
}

// ObjectDef defines a named ubus object to publish on ubusd.
type ObjectDef struct {
	// Methods maps method names to their handlers and argument policies.
	Methods map[string]Method
	// OnSubscribe is an optional callback invoked in order when the object transitions
	// between having active subscribers (true) and having none (false).
	OnSubscribe func(active bool)
	// Name is the ubus object path (for example "myapp.status").
	Name string
}

// RegisteredObject represents a named ubus object published by a Client.
type RegisteredObject struct {
	ctx         context.Context
	client      *Client
	cancel      context.CancelFunc
	subSignal   chan struct{}
	activeCalls map[*context.CancelFunc]struct{}
	name        string
	mu          sync.Mutex
	id          atomic.Uint32
	subscribers atomic.Bool
	closed      bool
}

type localObject struct {
	sub     *Subscription
	pub     *RegisteredObject
	methods map[string]MethodHandler
	name    string
	id      uint32
}

// Unmarshal decodes the incoming call arguments into target.
func (r CallRequest) Unmarshal(target any) error {
	return NewJSONResult(r.Data).Unmarshal(target)
}

func newRegisteredObject(
	objCtx context.Context,
	objCancel context.CancelFunc,
	client *Client,
	name string,
	onSubscribe func(bool),
) *RegisteredObject {
	pub := &RegisteredObject{
		ctx:         objCtx,
		cancel:      objCancel,
		client:      client,
		activeCalls: make(map[*context.CancelFunc]struct{}),
		name:        name,
	}
	if onSubscribe != nil {
		pub.subSignal = make(chan struct{}, 1)
	}

	return pub
}

// ID returns the numeric object identifier assigned by ubusd.
func (o *RegisteredObject) ID() uint32 {
	return o.id.Load()
}

// Name returns the published ubus object path.
func (o *RegisteredObject) Name() string {
	return o.name
}

// Done returns a channel that closes when the object is unregistered via Close
// or when the parent Client closes.
func (o *RegisteredObject) Done() <-chan struct{} {
	return o.ctx.Done()
}

// HasSubscribers reports whether at least one subscriber is currently attached to the object.
func (o *RegisteredObject) HasSubscribers() bool {
	return o.subscribers.Load()
}

// Notify emits a notification from the published object to all active subscribers.
func (o *RegisteredObject) Notify(ctx context.Context, method string, data any) error {
	ctx = normalizeContext(ctx)
	if method == "" {
		return errs.Wrapf(ErrInvalidParameter, "notification method is required")
	}
	if o.isClosed() || o.client.isClosed() {
		return ErrClosed
	}

	args, err := blobmsg.NormalizeArgs(data)
	if err != nil {
		return errs.Wrapf(ErrInvalidParameter, "normalize notification payload: %v", err)
	}

	objID := o.ID()
	body, err := createNotifyBody(objID, method, args)
	if err != nil {
		return err
	}

	_, status, err := o.client.sendRequest(ctx, "ubus notify", blobmsg.UbusMsgNotify, objID, body)
	if err != nil {
		return err
	}
	if status != StatusOK {
		return &CallError{Err: StatusError(int(status)), Service: o.name, Method: method, Code: int(status)}
	}

	return nil
}

// Close unregisters the published object from ubusd.
func (o *RegisteredObject) Close() error {
	if !o.markClosed() {
		return nil
	}

	objID := o.ID()
	o.client.removeLocalObject(objID)
	if o.client.isClosed() {
		return nil
	}

	cleanCtx, cancel := context.WithTimeout(o.client.connCtx, defaultUnsubscribeTimeout)
	defer cancel()

	return o.client.sendRemoveObject(cleanCtx, objID)
}

func (o *RegisteredObject) abortRegistration(client *Client) {
	if id := o.ID(); id != 0 {
		client.removeLocalObject(id)
	}
	o.markClosed()
}

func (o *RegisteredObject) notifySubscriberState(active bool) {
	o.subscribers.Store(active)
	if o.subSignal != nil {
		select {
		case o.subSignal <- struct{}{}:
		default:
		}
	}
}

func (o *RegisteredObject) watchSubscribers(ctx context.Context, onSubscribe func(bool)) {
	var last bool
	for {
		select {
		case <-o.subSignal:
			current := o.subscribers.Load()
			if current != last {
				last = current
				onSubscribe(current)
			}
		case <-ctx.Done():
			return
		}
	}
}

func (o *RegisteredObject) isClosed() bool {
	o.mu.Lock()
	defer o.mu.Unlock()

	return o.closed
}

func (o *RegisteredObject) trackCall(cancel *context.CancelFunc) bool {
	o.mu.Lock()
	defer o.mu.Unlock()

	if o.closed {
		return false
	}
	o.activeCalls[cancel] = struct{}{}

	return true
}

func (o *RegisteredObject) untrackCall(cancel *context.CancelFunc) {
	o.mu.Lock()
	delete(o.activeCalls, cancel)
	o.mu.Unlock()
}

func (o *RegisteredObject) markClosed() bool {
	o.mu.Lock()
	if o.closed {
		o.mu.Unlock()

		return false
	}
	o.closed = true
	cancels := make([]context.CancelFunc, 0, len(o.activeCalls))
	for fn := range o.activeCalls {
		cancels = append(cancels, *fn)
	}
	clear(o.activeCalls)
	o.mu.Unlock()

	o.cancel()
	for _, fn := range cancels {
		fn()
	}

	return true
}

// AddObject publishes a named ubus object and its methods on ubusd.
func (c *Client) AddObject(ctx context.Context, def ObjectDef) (*RegisteredObject, error) {
	ctx = normalizeContext(ctx)
	if def.Name == "" {
		return nil, errs.Wrapf(ErrInvalidParameter, "object name is required")
	}

	handlers, sigTable, err := buildObjectDefinition(def)
	if err != nil {
		return nil, err
	}

	body, err := createAddObjectBody(def.Name, sigTable)
	if err != nil {
		return nil, err
	}

	objCtx, objCancel := context.WithCancel(context.WithoutCancel(ctx))
	pub := newRegisteredObject(objCtx, objCancel, c, def.Name, def.OnSubscribe)
	frames, status, err := c.registerNamedObject(ctx, body, def.Name, handlers, pub)
	if err != nil {
		pub.abortRegistration(c)

		return nil, err
	}
	if status != StatusOK {
		pub.abortRegistration(c)

		return nil, &CallError{Err: StatusError(int(status)), Service: def.Name, Method: "add_object", Code: int(status)}
	}

	if pub.ID() == 0 {
		objID, ok := extractRegisteredObjID(frames)
		if !ok {
			pub.abortRegistration(c)

			return nil, errs.Wrapf(ErrInvalidResponse, "add_object response missing objid")
		}
		c.installNamedObject(objID, def.Name, handlers, pub)
	}
	if def.OnSubscribe != nil {
		go pub.watchSubscribers(objCtx, def.OnSubscribe)
	}

	return pub, nil
}

func (c *Client) registerNamedObject(
	ctx context.Context,
	body []byte,
	name string,
	handlers map[string]MethodHandler,
	pub *RegisteredObject,
) ([]map[string]any, uint32, error) {
	return c.sendRequestWithHook(
		ctx,
		"ubus add_object",
		blobmsg.UbusMsgAddObject,
		0,
		body,
		func(attrs map[string]any) {
			if id, ok := blobmsg.ReadUint(attrs["objid"]); ok && pub.ID() == 0 {
				c.installNamedObject(id, name, handlers, pub)
			}
		},
	)
}

func (c *Client) installNamedObject(
	objID uint32,
	name string,
	handlers map[string]MethodHandler,
	pub *RegisteredObject,
) {
	pub.id.Store(objID)
	c.localMu.Lock()
	c.localObjects[objID] = &localObject{
		id:      objID,
		name:    name,
		methods: handlers,
		pub:     pub,
	}
	c.localMu.Unlock()
	if c.isClosed() {
		pub.markClosed()
	}
}

func (c *Client) sendRemoveObject(ctx context.Context, objID uint32) error {
	body, err := blobmsg.CreateBlobMessage(map[uint32]any{
		blobmsg.UbusAttrObjID: objID,
	}, []uint32{blobmsg.UbusAttrObjID})
	if err != nil {
		return err
	}

	_, status, err := c.sendRequest(ctx, "ubus remove_object", blobmsg.UbusMsgRemoveObject, 0, body)
	if err != nil {
		return err
	}
	if status != StatusOK {
		return &CallError{Err: StatusError(int(status)), Method: "remove_object", Code: int(status)}
	}

	return nil
}

func buildObjectDefinition(def ObjectDef) (map[string]MethodHandler, map[string]any, error) {
	handlers := make(map[string]MethodHandler, len(def.Methods))
	sigTable := make(map[string]any, len(def.Methods))

	for methodName, methodDef := range def.Methods {
		if methodName == "" || methodDef.Handler == nil {
			return nil, nil, errs.Wrapf(ErrInvalidParameter, "method %q requires a non-nil handler", methodName)
		}
		handlers[methodName] = methodDef.Handler

		policyMap := make(map[string]any, len(methodDef.Policy))
		for argName, argType := range methodDef.Policy {
			if argName == "" {
				return nil, nil, errs.Wrapf(ErrInvalidParameter, "method %q has empty argument name", methodName)
			}
			code, ok := argTypeToBlobmsg(argType)
			if !ok {
				return nil, nil, errs.Wrapf(ErrInvalidParameter, "method %q argument %q has invalid ArgType %q",
					methodName, argName, argType)
			}
			policyMap[argName] = code
		}
		sigTable[methodName] = policyMap
	}

	return handlers, sigTable, nil
}

func createAddObjectBody(name string, sigTable map[string]any) ([]byte, error) {
	encodedSig, err := blobmsg.CreateBlobmsgTable(sigTable)
	if err != nil {
		return nil, err
	}

	attrs := map[uint32]any{
		blobmsg.UbusAttrObjPath:   name,
		blobmsg.UbusAttrSignature: encodedSig[blobmsg.BlobHeaderBytes:],
	}

	return blobmsg.CreateBlobMessage(attrs, []uint32{blobmsg.UbusAttrObjPath, blobmsg.UbusAttrSignature})
}

func createNotifyBody(objID uint32, method string, args map[string]any) ([]byte, error) {
	argData, err := blobmsg.CreateBlobmsgData(args)
	if err != nil {
		return nil, err
	}

	attrs := map[uint32]any{
		blobmsg.UbusAttrObjID:  objID,
		blobmsg.UbusAttrMethod: method,
	}
	if len(argData) > 0 {
		attrs[blobmsg.UbusAttrData] = argData
	}

	return blobmsg.CreateBlobMessage(attrs, []uint32{
		blobmsg.UbusAttrObjID,
		blobmsg.UbusAttrMethod,
		blobmsg.UbusAttrData,
	})
}

func extractRegisteredObjID(frames []map[string]any) (uint32, bool) {
	for _, frame := range frames {
		if id, ok := blobmsg.ReadUint(frame["objid"]); ok {
			return id, true
		}
	}

	return 0, false
}
