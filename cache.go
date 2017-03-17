package main

import (
	"sync"
	"time"
)

type KeyNotFound struct {
	key string
}

func (e KeyNotFound) Error() string {
	return e.key + " " + "not found"
}

type KeyExpired struct {
	Key string
}

func (e KeyExpired) Error() string {
	return e.Key + " " + "expired"
}

type CacheIsFull struct {
}

func (e CacheIsFull) Error() string {
	return "Cache is Full"
}

type SerializerError struct {
}

func (e SerializerError) Error() string {
	return "Serializer error"
}

type Data struct {
	OraculumResponse *bool
	Expire           time.Time
}

type Cache interface {
	Get(key string) (OraculumResponse *bool, err error)
	Set(key string, OraculumResponse *bool) error
	Exists(key string) bool
	Remove(key string)
	Length() int
	Clear()
}

type MemoryCache struct {
	Backend  map[string]Data
	Expire   time.Duration
	Maxcount int
	mu       sync.RWMutex
}

func (c *MemoryCache) Get(key string) (*bool, error) {
	logger.Debug("Cache Get: called for key: %s", key)
	c.mu.RLock()
	data, ok := c.Backend[key]
	c.mu.RUnlock()
	if !ok {
		logger.Debug("Cache Get: key: %s was not found.", key)
		return nil, KeyNotFound{key}
	}

	if data.Expire.Before(time.Now()) {
		logger.Debug("Cache Get: key: %s expired at %s. Returning nil.", key, data.Expire.Format(time.RFC3339))
		c.Remove(key)
		return nil, KeyExpired{key}
	}
	logger.Debug("Cache Get: key: %s found value: %t", key, data.OraculumResponse)
	return data.OraculumResponse, nil
}

func (c *MemoryCache) Set(key string, oraculumResponse *bool) error {
	if c.Full() && !c.Exists(key) {
		logger.Debug("Cache Set: key: %s, Cache is full.", key)
		return CacheIsFull{}
	}

	expire := time.Now().Add(c.Expire)
	data := Data{oraculumResponse, expire}
	// logger.Debug("Cache Set: key: %s, value: %t, expires at: %s", key, oraculumResponse, expire.Format(time.RFC3339))
	c.mu.Lock()
	c.Backend[key] = data
	c.mu.Unlock()
	return nil
}

func (c *MemoryCache) Remove(key string) {
	logger.Debug("Cache Remove: key: %s was removed.", key)
	c.mu.Lock()
	delete(c.Backend, key)
	c.mu.Unlock()
}

func (c *MemoryCache) Exists(key string) bool {
	c.mu.RLock()
	_, ok := c.Backend[key]
	c.mu.RUnlock()
	logger.Debug("Cache Exists: key: %s exists: %t", key, ok)
	return ok
}

func (c *MemoryCache) Length() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.Backend)
}

func (c *MemoryCache) Full() bool {
	// if Maxcount is zero. the cache will never be full.
	if c.Maxcount == 0 {
		return false
	}
	return c.Length() >= c.Maxcount
}

func (c *MemoryCache) Clear() {
	c.mu.Lock()
	c.Backend = make(map[string]Data)
	c.mu.Unlock()
}
