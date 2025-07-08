package goubus

// Events returns a manager for event-driven operations.
func (c *Client) Events() *EventManager {
	return &EventManager{
		client: c,
	}
}

// EventManager provides methods to publish and subscribe to ubus events.
type EventManager struct {
	client *Client
}

// Publish publishes an event to the ubus system.
func (em *EventManager) Publish(eventType string, data map[string]interface{}) error {
	return em.client.publish(eventType, data)
}

// Subscribe subscribes to events of the specified types.
func (em *EventManager) Subscribe(eventTypes []string, handler EventHandler) error {
	return em.client.subscribe(eventTypes, handler)
}
