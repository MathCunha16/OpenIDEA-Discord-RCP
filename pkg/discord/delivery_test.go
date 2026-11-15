package discord

import "testing"

// DeliveryCache decides which activities may be deduplicated. Regression for
// the P2 finding: a failed IPC write must invalidate the cache, otherwise
// returning to the previous file is deduplicated while the connection is dead
// and presence never recovers.
func TestDeliveryCacheInvalidationOnFailure(t *testing.T) {
	cache := NewDeliveryCache()
	old := &Activity{Details: "old.go", State: "project"}
	other := &Activity{Details: "new.go", State: "project"}

	// 1. Successfully display old.go over a live connection.
	cache.SetConnected(true)
	if cache.ShouldSkip(old) {
		t.Fatal("empty cache must not dedup")
	}
	cache.Delivered(old)

	// 2. Switch to new.go while the socket returns a write error. The client
	// closes the connection and the caller invalidates the cache.
	if cache.ShouldSkip(other) {
		t.Fatal("different payload must not dedup")
	}
	cache.SetConnected(false)
	cache.Invalidate()

	// 3. Return to old.go. Its payload matches the previously cached activity,
	// but the connection is dead: it must NOT be deduplicated, so the caller
	// reaches EnsureConnection, dials again and republishes.
	if cache.ShouldSkip(old) {
		t.Fatal("stale cache on a dead connection must not dedup (P2 regression)")
	}
	if cache.Last() != nil {
		t.Fatal("cache should be empty after invalidation")
	}

	// Simulate the recovery: reconnect + successful delivery.
	cache.SetConnected(true)
	cache.Delivered(old)
	if !cache.ShouldSkip(old) {
		t.Fatal("expected dedup once reconnected and republished")
	}
}

func TestDeliveryCacheShouldSkipRequiresLiveConnection(t *testing.T) {
	last := &Activity{Details: "old.go", State: "project"}
	same := &Activity{Details: "old.go", State: "project"}

	cache := NewDeliveryCache()
	cache.SetConnected(true)
	cache.Delivered(last)
	if !cache.ShouldSkip(same) {
		t.Error("expected dedup while connected")
	}

	cache.SetConnected(false)
	if cache.ShouldSkip(same) {
		t.Error("must not dedup a matching payload on a dead connection")
	}

	cache.Invalidate()
	if cache.ShouldSkip(same) {
		t.Error("must not dedup after invalidation")
	}
	cache.SetConnected(true)
	if cache.ShouldSkip(same) {
		t.Error("must not dedup without a cached activity")
	}
}
