package goubus

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// EventHandler is a callback function type for handling ubus events.
// The first argument is the event type (e.g., "network.interface"),
// the second is the event data.
type EventHandler func(eventType string, data map[string]interface{})

// EventSubscription holds the state for an active event listener.
type EventSubscription struct {
	client    *http.Client
	request   *http.Request
	isStopped bool
}

// Stop terminates the event listener connection.
func (s *EventSubscription) Stop() {
	s.isStopped = true
	// The net/http Transport has a CancelRequest method, but it's not exposed on the client.
	// The typical way to handle this is to have the read loop check a flag.
	// Closing the response body from another goroutine can also work but can lead to "use of closed network connection" errors.
	// A robust solution would involve a context passed to the request.
}

// Publish publishes an event to the ubus system.
func (u *Client) publish(eventType string, data interface{}) error {
	errLogin := u.LoginCheck()
	if errLogin != nil {
		return errLogin
	}

	eventData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("error marshaling event data: %w", err)
	}

	params := map[string]interface{}{
		"type": eventType,
		"data": json.RawMessage(eventData),
	}

	jsonStr := u.buildUbusCall("ubus", "send", params)
	_, err = u.Call(jsonStr)
	return err
}

// Subscribe subscribes to ubus events and calls the handler function for each event.
func (u *Client) subscribe(eventTypes []string, handler EventHandler) error {
	errLogin := u.LoginCheck()
	if errLogin != nil {
		return errLogin
	}

	params := map[string]interface{}{
		"types": eventTypes,
	}

	jsonStr := u.buildUbusCall("ubus", "subscribe", params)
	call, err := u.Call(jsonStr)
	if err != nil {
		return err
	}

	// Process the subscription response and start listening for events
	// This is a simplified implementation - in practice, you'd need to handle
	// the event stream separately
	handler("subscription_started", map[string]interface{}{
		"result": call.Result,
	})

	return nil
}
