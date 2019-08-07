package rux

import (
	"sync"
)

// CachedRoutes struct
type CachedRoutes struct {
	m    map[string]*Route
	lock *sync.RWMutex
}

// NewCachedRoutes get CachedRoutes pointer
func NewCachedRoutes(size int) *CachedRoutes {
	return &CachedRoutes{
		lock: new(sync.RWMutex),
		m:    make(map[string]*Route, size),
	}
}

// Get Router pointer
func (c *CachedRoutes) Get(k string) *Route {
	c.lock.RLock()
	defer c.lock.RUnlock()

	if val, ok := c.m[k]; ok {
		return val
	}

	return nil
}

// Set Maps the given key and value. Returns false
// if the key is already in the map and changes nothing.
func (c *CachedRoutes) Set(k string, v *Route) bool {
	c.lock.Lock()
	defer c.lock.Unlock()

	if val, ok := c.m[k]; !ok {
		c.m[k] = v
	} else if val != v {
		c.m[k] = v
	} else {
		return false
	}

	return true
}

// Has Returns true if k is exist in the map.
func (c *CachedRoutes) Has(k string) (*Route, bool) {
	c.lock.RLock()
	defer c.lock.RUnlock()

	if _, ok := c.m[k]; ok {
		return c.m[k], true
	}

	return nil, false
}

// Len the given m total.
func (c *CachedRoutes) Len() int {
	c.lock.RLock()
	defer c.lock.RUnlock()

	return len(c.m)
}

// Items the given m.
func (c *CachedRoutes) Items() map[string]*Route {
	c.lock.RLock()
	defer c.lock.RUnlock()

	r := make(map[string]*Route)

	for k, v := range c.m {
		r[k] = v
	}

	return r
}

// Delete the given key and value.
func (c *CachedRoutes) Delete(k string) {
	c.lock.Lock()
	defer c.lock.Unlock()

	c.m[k] = nil
	delete(c.m, k)
}
