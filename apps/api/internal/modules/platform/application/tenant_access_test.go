package application

import (
	"testing"
	"time"
)

func TestAccessCacheDropsStaleWriteAfterInvalidate(t *testing.T) {
	c := newAccessCache(time.Minute)
	g := c.generation(7) // a lookup starts...
	c.invalidate(7)      // ...the tenant is frozen meanwhile...
	c.put(7, accessEntry{found: true, at: time.Now()}, g)
	if _, ok := c.get(7, time.Now()); ok {
		t.Fatal("a result read before the invalidation must not be cached")
	}
	g = c.generation(7)
	c.put(7, accessEntry{found: true, at: time.Now()}, g)
	if _, ok := c.get(7, time.Now()); !ok {
		t.Fatal("a fresh result must be cached")
	}
}
