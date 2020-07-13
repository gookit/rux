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
	hashMap map[string]*list.Element
	lock    *sync.Mutex
}

// NewCachedRoutes Get Cache pointer
func NewCachedRoutes(size int) *cachedRoutes {
	return &cachedRoutes{
		size:    size,
		list:    list.New(),
		hashMap: make(map[string]*list.Element),
		lock:    new(sync.Mutex),
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

	if c.list == nil {
		return false
	}

	if element, ok := c.hashMap[k]; ok {
		c.list.MoveToFront(element)
		element.Value.(*cacheNode).Value = v

		return true
	}

	var newElement = c.list.PushFront(&cacheNode{k, v})

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

// Get Router by key
func (c *cachedRoutes) Get(k string) *Route {
	c.lock.Lock()
	defer c.lock.Unlock()

	if c.hashMap == nil {
		return nil
	}

	if element, ok := c.hashMap[k]; ok {
		c.list.MoveToFront(element)

		return element.Value.(*cacheNode).Value
	}

	return nil
}

// Delete Router by key
func (c *cachedRoutes) Delete(k string) bool {
	c.lock.Lock()
	defer c.lock.Unlock()

	if c.hashMap == nil {
		return false
	}

	if element, ok := c.hashMap[k]; ok {
		cacheNode := element.Value.(*cacheNode)

		delete(c.hashMap, cacheNode.Key)

		c.list.Remove(element)

		return true
	}

	return false
}

// Has Returns true if k is exist in the hashmap.
func (c *cachedRoutes) Has(k string) (*Route, bool) {
	var r = c.Get(k)

	if r != nil {
		return r, true
	}

	return nil, false
}
