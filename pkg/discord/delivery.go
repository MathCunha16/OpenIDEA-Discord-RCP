package discord

// DeliveryCache tracks the last activity accepted by Discord. It decides when
// an activity may be deduplicated, with one hard rule: a cached payload is only
// trusted while the IPC connection is still valid. After any failed delivery
// the cache is invalidated, so a later event with the same payload triggers a
// reconnect instead of being silently skipped.
//
// This is pure state: the caller owns the connection and reports the outcome.
type DeliveryCache struct {
	last      *Activity
	connected bool
}

// NewDeliveryCache returns an empty cache.
func NewDeliveryCache() *DeliveryCache { return &DeliveryCache{} }

// SetConnected records the current IPC connection state. Call it after a dial
// succeeds or a write/read failure closes the socket.
func (c *DeliveryCache) SetConnected(connected bool) { c.connected = connected }

// Connected reports the last known connection state.
func (c *DeliveryCache) Connected() bool { return c.connected }

// ShouldSkip reports whether next can be deduplicated. A nil or stale cache,
// or a dead connection, must always be delivered.
func (c *DeliveryCache) ShouldSkip(next *Activity) bool {
	return c.connected && c.last != nil && c.last.Equal(next)
}

// Delivered records a successfully published activity.
func (c *DeliveryCache) Delivered(activity *Activity) {
	c.last = activity
	c.connected = true
}

// Invalidate forgets the cached activity after a failed delivery. The caller
// is responsible for closing the connection; the cache must not assume it is
// still usable.
func (c *DeliveryCache) Invalidate() {
	c.last = nil
	c.connected = false
}

// Last returns the cached activity (nil when invalidated).
func (c *DeliveryCache) Last() *Activity { return c.last }
