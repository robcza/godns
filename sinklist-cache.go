package main

import (
	"sync"
)

type SinklistCache interface {
	Get(key string) (sendToSinkhole bool, err error)
	Set(key string, sendToSinkhole bool) error
	Exists(key string) bool
	Remove(key string)
	Length() int
	Replace(cache map[string]bool)
}

type SinklistMemoryCache struct {
	Backend map[string]bool
	mu      sync.RWMutex
}

func (c *SinklistMemoryCache) Get(key string) (bool, error) {
	logger.Debug("SinklistCache Get: called for key: %s", key)
	c.mu.RLock()
	data, ok := c.Backend[key]
	c.mu.RUnlock()
	if !ok {
		logger.Debug("SinklistCache Get: key: %s was not found.", key)
		return false, KeyNotFound{key}
	}

	logger.Debug("SinklistCache Get: key: %s found value: %t", key, data)
	return data, nil
}

func (c *SinklistMemoryCache) Set(key string, sendToSinkhole bool) error {
	logger.Debug("SinklistCache Set: key: %s, value: %t", key, sendToSinkhole)
	c.mu.Lock()
	c.Backend[key] = sendToSinkhole
	c.mu.Unlock()
	return nil
}

// Replace whole content of cache in atomic operation
func (c *SinklistMemoryCache) Replace(cache map[string]bool) {
	logger.Debug("SinklistCache Replace")
	c.mu.Lock()
	c.Backend = cache
	c.mu.Unlock()
}

func (c *SinklistMemoryCache) Remove(key string) {
	logger.Debug("SinklistCache Remove: key: %s was removed.", key)
	c.mu.Lock()
	delete(c.Backend, key)
	c.mu.Unlock()
}

func (c *SinklistMemoryCache) Exists(key string) bool {
	c.mu.RLock()
	_, ok := c.Backend[key]
	c.mu.RUnlock()
	logger.Debug("SinklistCache Exists: key: %s exists: %t", key, ok)
	return ok
}

func (c *SinklistMemoryCache) Length() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.Backend)
}
