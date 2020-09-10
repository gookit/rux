package rux

import (
	"container/list"
	"sync"
)

// cacheNode struct
type cacheNode struct {
	Key   string
	Value *Route
}

// cachedRoutes struct
type cachedRoutes struct {
	size    int
	list    *list.List
	lock    *sync.Mutex
	hashMap map[string]*list.Element
}

// NewCachedRoutes Get Cache pointer
func NewCachedRoutes(size int) *cachedRoutes {
	return &cachedRoutes{
		size:    size,
		list:    list.New(),
		lock:    new(sync.Mutex),
		hashMap: make(map[string]*list.Element),
	}
}

// Len cache len
func (c *cachedRoutes) Len() int {
	c.lock.Lock()
	defer c.lock.Unlock()

	return c.list.Len()
}

// Set route key and Route
func (c *cachedRoutes) Set(k string, v *Route) bool {
	c.lock.Lock()
	defer c.lock.Unlock()

	// key has been exists, update value
	if element, isFound := c.hashMap[k]; isFound {
		c.list.MoveToFront(element)

		cacheNode := element.Value.(*cacheNode)
		// update value
		cacheNode.Value = v
		return true
	}

	newElement := c.list.PushFront(&cacheNode{k, v})
	c.hashMap[k] = newElement

	if c.list.Len() > c.size {
		lastElement := c.list.Back()
		if lastElement == nil {
			return true
		}

		cacheNode := lastElement.Value.(*cacheNode)

		delete(c.hashMap, cacheNode.Key)
		c.list.Remove(lastElement)
	}

	return true
}

// Get cached Route by key
func (c *cachedRoutes) Get(k string) (*Route, bool) {
	c.lock.Lock()
	defer c.lock.Unlock()

	if element, ok := c.hashMap[k]; ok {
		c.list.MoveToFront(element)

		cacheNode := element.Value.(*cacheNode)
		return cacheNode.Value, true
	}

	return nil, false
}

// Delete Router by key
func (c *cachedRoutes) Delete(k string) bool {
	c.lock.Lock()
	defer c.lock.Unlock()

	if element, ok := c.hashMap[k]; ok {
		cacheNode := element.Value.(*cacheNode)

		delete(c.hashMap, cacheNode.Key)
		c.list.Remove(element)
		return true
	}

	return false
}

// Has returns true if k is exist in the hashmap.
func (c *cachedRoutes) Has(k string) bool {
	_, ok := c.Get(k)
	return ok
}
