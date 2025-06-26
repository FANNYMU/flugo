package cache

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

type Item struct {
	Value       interface{}
	Expiration  int64
	CreatedAt   time.Time
	AccessCount int64
	LastAccess  time.Time
}

func (item *Item) IsExpired() bool {
	if item.Expiration == 0 {
		return false
	}
	return time.Now().UnixNano() > item.Expiration
}

type Stats struct {
	Hits      int64   `json:"hits"`
	Misses    int64   `json:"misses"`
	Sets      int64   `json:"sets"`
	Deletes   int64   `json:"deletes"`
	Evictions int64   `json:"evictions"`
	ItemCount int     `json:"item_count"`
	HitRatio  float64 `json:"hit_ratio"`
}

type Cache struct {
	items         map[string]*Item
	mu            sync.RWMutex
	maxSize       int
	defaultTTL    time.Duration
	stats         Stats
	cleanupTicker *time.Ticker
	stopCleanup   chan bool
}

func New(maxSize int, defaultTTL time.Duration) *Cache {
	c := &Cache{
		items:       make(map[string]*Item),
		maxSize:     maxSize,
		defaultTTL:  defaultTTL,
		stopCleanup: make(chan bool),
	}

	c.startCleanup()
	return c
}

var DefaultCache *Cache

func Init(maxSize int, defaultTTL time.Duration) {
	DefaultCache = New(maxSize, defaultTTL)
}

func (c *Cache) startCleanup() {
	c.cleanupTicker = time.NewTicker(5 * time.Minute)
	go func() {
		for {
			select {
			case <-c.cleanupTicker.C:
				c.deleteExpired()
			case <-c.stopCleanup:
				c.cleanupTicker.Stop()
				return
			}
		}
	}()
}

func (c *Cache) Stop() {
	if c.stopCleanup != nil {
		c.stopCleanup <- true
	}
}

func (c *Cache) Set(key string, value interface{}, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if ttl == 0 {
		ttl = c.defaultTTL
	}

	var expiration int64
	if ttl > 0 {
		expiration = time.Now().Add(ttl).UnixNano()
	}

	if len(c.items) >= c.maxSize && c.items[key] == nil {
		c.evictLRU()
	}

	c.items[key] = &Item{
		Value:       value,
		Expiration:  expiration,
		CreatedAt:   time.Now(),
		AccessCount: 0,
		LastAccess:  time.Now(),
	}

	c.stats.Sets++
	c.stats.ItemCount = len(c.items)
}

func (c *Cache) Get(key string) (interface{}, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	item, found := c.items[key]
	if !found {
		c.stats.Misses++
		c.updateHitRatio()
		return nil, false
	}

	if item.IsExpired() {
		delete(c.items, key)
		c.stats.Misses++
		c.stats.ItemCount = len(c.items)
		c.updateHitRatio()
		return nil, false
	}

	item.AccessCount++
	item.LastAccess = time.Now()
	c.stats.Hits++
	c.updateHitRatio()

	return item.Value, true
}

func (c *Cache) GetString(key string) (string, bool) {
	value, found := c.Get(key)
	if !found {
		return "", false
	}
	if str, ok := value.(string); ok {
		return str, true
	}
	return "", false
}

func (c *Cache) GetInt(key string) (int, bool) {
	value, found := c.Get(key)
	if !found {
		return 0, false
	}
	if num, ok := value.(int); ok {
		return num, true
	}
	return 0, false
}

func (c *Cache) GetJSON(key string, target interface{}) bool {
	value, found := c.Get(key)
	if !found {
		return false
	}

	if jsonStr, ok := value.(string); ok {
		err := json.Unmarshal([]byte(jsonStr), target)
		return err == nil
	}

	return false
}

func (c *Cache) SetJSON(key string, value interface{}, ttl time.Duration) error {
	jsonBytes, err := json.Marshal(value)
	if err != nil {
		return err
	}
	c.Set(key, string(jsonBytes), ttl)
	return nil
}

func (c *Cache) Delete(key string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, found := c.items[key]; found {
		delete(c.items, key)
		c.stats.Deletes++
		c.stats.ItemCount = len(c.items)
		return true
	}
	return false
}

func (c *Cache) Exists(key string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	item, found := c.items[key]
	if !found {
		return false
	}

	return !item.IsExpired()
}

func (c *Cache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items = make(map[string]*Item)
	c.stats.ItemCount = 0
}

func (c *Cache) Keys() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	keys := make([]string, 0, len(c.items))
	for key, item := range c.items {
		if !item.IsExpired() {
			keys = append(keys, key)
		}
	}
	return keys
}

func (c *Cache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.items)
}

func (c *Cache) Stats() Stats {
	c.mu.RLock()
	defer c.mu.RUnlock()

	stats := c.stats
	stats.ItemCount = len(c.items)
	return stats
}

func (c *Cache) deleteExpired() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now().UnixNano()
	for key, item := range c.items {
		if item.Expiration > 0 && now > item.Expiration {
			delete(c.items, key)
			c.stats.Evictions++
		}
	}
	c.stats.ItemCount = len(c.items)
}

func (c *Cache) evictLRU() {
	var oldestKey string
	var oldestTime time.Time

	for key, item := range c.items {
		if oldestKey == "" || item.LastAccess.Before(oldestTime) {
			oldestKey = key
			oldestTime = item.LastAccess
		}
	}

	if oldestKey != "" {
		delete(c.items, oldestKey)
		c.stats.Evictions++
	}
}

func (c *Cache) updateHitRatio() {
	total := c.stats.Hits + c.stats.Misses
	if total > 0 {
		c.stats.HitRatio = float64(c.stats.Hits) / float64(total)
	}
}

func (c *Cache) GetOrSet(key string, valueFunc func() interface{}, ttl time.Duration) interface{} {
	if value, found := c.Get(key); found {
		return value
	}

	value := valueFunc()
	c.Set(key, value, ttl)
	return value
}

func (c *Cache) Increment(key string, delta int64) (int64, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	item, found := c.items[key]
	if !found {
		c.items[key] = &Item{
			Value:      delta,
			CreatedAt:  time.Now(),
			LastAccess: time.Now(),
		}
		return delta, nil
	}

	if item.IsExpired() {
		delete(c.items, key)
		c.items[key] = &Item{
			Value:      delta,
			CreatedAt:  time.Now(),
			LastAccess: time.Now(),
		}
		return delta, nil
	}

	if currentValue, ok := item.Value.(int64); ok {
		newValue := currentValue + delta
		item.Value = newValue
		item.LastAccess = time.Now()
		item.AccessCount++
		return newValue, nil
	}

	return 0, fmt.Errorf("value is not an integer")
}

func Set(key string, value interface{}, ttl time.Duration) {
	if DefaultCache != nil {
		DefaultCache.Set(key, value, ttl)
	}
}

func Get(key string) (interface{}, bool) {
	if DefaultCache != nil {
		return DefaultCache.Get(key)
	}
	return nil, false
}

func GetString(key string) (string, bool) {
	if DefaultCache != nil {
		return DefaultCache.GetString(key)
	}
	return "", false
}

func GetInt(key string) (int, bool) {
	if DefaultCache != nil {
		return DefaultCache.GetInt(key)
	}
	return 0, false
}

func GetJSON(key string, target interface{}) bool {
	if DefaultCache != nil {
		return DefaultCache.GetJSON(key, target)
	}
	return false
}

func SetJSON(key string, value interface{}, ttl time.Duration) error {
	if DefaultCache != nil {
		return DefaultCache.SetJSON(key, value, ttl)
	}
	return fmt.Errorf("cache not initialized")
}

func Delete(key string) bool {
	if DefaultCache != nil {
		return DefaultCache.Delete(key)
	}
	return false
}

func Exists(key string) bool {
	if DefaultCache != nil {
		return DefaultCache.Exists(key)
	}
	return false
}

func Clear() {
	if DefaultCache != nil {
		DefaultCache.Clear()
	}
}

func GetOrSet(key string, valueFunc func() interface{}, ttl time.Duration) interface{} {
	if DefaultCache != nil {
		return DefaultCache.GetOrSet(key, valueFunc, ttl)
	}
	return valueFunc()
}
