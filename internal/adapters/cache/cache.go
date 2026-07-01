package cache

import (
	"log/slog"
	"sync"
	"time"
)

type Item struct {
	Value any
	TTL   time.Time
}

type cache struct {
	items map[string]Item
	mu    sync.RWMutex
}

var (
	instance *cache
	once     sync.Once
)

func GetInstance() *cache {
	once.Do(func() {
		instance = &cache{
			items: make(map[string]Item),
		}
		go instance.ttlScheduler()
	})
	return instance
}

func (c *cache) Get(key string) (any, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	item, ok := c.items[key]
	if !ok || time.Now().After(item.TTL) {
		return nil, false
	}
	slog.Debug("cache hit", "key", key, "ttl", item.TTL)
	return item.Value, true
}

// Use for debugging scenarios
func (c *cache) GetItem(key string) (*Item, bool) {
	c.mu.RLock()
	item, ok := c.items[key]
	if !ok || time.Now().After(item.TTL) {
		return nil, false
	}
	c.mu.RUnlock()
	return &item, true
}

func (c *cache) Set(key string, val any, duration time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items[key] = Item{
		Value: val,
		TTL:   time.Now().Add(duration),
	}
	slog.Debug("data cached", "key", key, "ttl", c.items[key].TTL)
}

func (c *cache) ClearKey(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.items, key)
	slog.Debug("cache cleared", "key", key)
}

func (c *cache) ttlScheduler() {
	ticker := time.NewTicker(time.Second * 5)
	for tickTime := range ticker.C {
		c.mu.Lock()
		now := time.Now()
		for k, v := range c.items {
			if now.After(v.TTL) {
				slog.Debug("cache ttl expired", "key", k, "ttl", v.TTL, "timestamp", tickTime)
				delete(c.items, k)
			}
		}
		c.mu.Unlock()
	}
}
