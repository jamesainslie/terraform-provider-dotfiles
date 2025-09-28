// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0.

package services

import (
	"context"
	"sync"
	"sync/atomic"
	"time"
)

// CacheService defines the interface for caching operations.
type CacheService interface {
	// Get retrieves a value from the cache
	Get(ctx context.Context, key string) (interface{}, bool)

	// Set stores a value in the cache with optional TTL
	Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error

	// Delete removes a value from the cache
	Delete(ctx context.Context, key string) error

	// Clear removes all values from the cache
	Clear(ctx context.Context) error

	// Size returns the current number of items in the cache
	Size() int

	// Stats returns cache statistics
	Stats() CacheStats

	// Cleanup removes expired entries
	Cleanup(ctx context.Context) error
}

// CacheEntry represents a cached item.
type CacheEntry struct {
	Value       interface{}
	ExpiresAt   time.Time
	CreatedAt   time.Time
	AccessCount int64
	LastAccess  time.Time
}

// CacheStats contains cache performance statistics.
type CacheStats struct {
	Size        int
	Hits        int64
	Misses      int64
	Evictions   int64
	Cleanups    int64
	HitRatio    float64
	LastCleanup time.Time
}

// InMemoryCache provides an in-memory cache implementation.
type InMemoryCache struct {
	// Configuration
	maxSize    int
	defaultTTL time.Duration

	// Storage
	data map[string]*CacheEntry

	// Statistics
	stats CacheStats

	// Synchronization
	mu sync.RWMutex

	// Cleanup
	cleanupTicker *time.Ticker
	stopCleanup   chan struct{}
}

// CacheConfig contains configuration for cache initialization.
type CacheConfig struct {
	MaxSize         int
	DefaultTTL      time.Duration
	CleanupInterval time.Duration
}

// Cache constants.
const (
	DefaultCacheSize      = 1000
	DefaultCacheTTL       = 300 // 5 minutes in seconds
	CacheCleanupInterval  = 600 // 10 minutes in seconds
	DefaultMaxConcurrency = 10
	MaxConcurrency        = 50
)

// NewInMemoryCache creates a new in-memory cache with the provided configuration.
func NewInMemoryCache(config CacheConfig) *InMemoryCache {
	if config.MaxSize <= 0 {
		config.MaxSize = DefaultCacheSize
	}
	if config.DefaultTTL <= 0 {
		config.DefaultTTL = time.Duration(DefaultCacheTTL) * time.Second
	}
	if config.CleanupInterval <= 0 {
		config.CleanupInterval = time.Duration(CacheCleanupInterval) * time.Second
	}

	cache := &InMemoryCache{
		maxSize:     config.MaxSize,
		defaultTTL:  config.DefaultTTL,
		data:        make(map[string]*CacheEntry),
		stats:       CacheStats{LastCleanup: time.Now()},
		stopCleanup: make(chan struct{}),
	}

	// Start cleanup goroutine
	cache.cleanupTicker = time.NewTicker(config.CleanupInterval)
	go cache.cleanupLoop()

	return cache
}

// Get implements CacheService.Get.
func (c *InMemoryCache) Get(ctx context.Context, key string) (interface{}, bool) {
	c.mu.RLock()
	entry, exists := c.data[key]
	if !exists {
		atomic.AddInt64(&c.stats.Misses, 1)
		c.mu.RUnlock()
		return nil, false
	}

	// Check if expired
	if time.Now().After(entry.ExpiresAt) {
		// Need to upgrade to write lock to delete entry
		c.mu.RUnlock()
		c.mu.Lock()
		// Double-check the entry still exists and is still expired
		entry, exists = c.data[key]
		if exists && time.Now().After(entry.ExpiresAt) {
			delete(c.data, key)
			atomic.AddInt64(&c.stats.Evictions, 1)
			c.stats.Size = len(c.data)
		}
		atomic.AddInt64(&c.stats.Misses, 1)
		c.mu.Unlock()
		return nil, false
	}

	// Update access statistics
	entry.AccessCount++
	entry.LastAccess = time.Now()
	atomic.AddInt64(&c.stats.Hits, 1)
	c.mu.RUnlock()

	return entry.Value, true
}

// Set implements CacheService.Set.
func (c *InMemoryCache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Use default TTL if not specified
	if ttl <= 0 {
		ttl = c.defaultTTL
	}

	// Check if we need to evict entries to make room
	if len(c.data) >= c.maxSize {
		c.evictLRU()
	}

	// Create new entry
	now := time.Now()
	entry := &CacheEntry{
		Value:       value,
		ExpiresAt:   now.Add(ttl),
		CreatedAt:   now,
		AccessCount: 0,
		LastAccess:  now,
	}

	c.data[key] = entry
	c.stats.Size = len(c.data)

	return nil
}

// Delete implements CacheService.Delete.
func (c *InMemoryCache) Delete(ctx context.Context, key string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.data, key)
	c.stats.Size = len(c.data)

	return nil
}

// Clear implements CacheService.Clear.
func (c *InMemoryCache) Clear(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.data = make(map[string]*CacheEntry)
	c.stats.Size = 0

	return nil
}

// Size implements CacheService.Size.
func (c *InMemoryCache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.data)
}

// Stats implements CacheService.Stats.
func (c *InMemoryCache) Stats() CacheStats {
	c.mu.RLock()
	defer c.mu.RUnlock()

	stats := c.stats
	stats.Size = len(c.data)

	// Calculate hit ratio
	total := stats.Hits + stats.Misses
	if total > 0 {
		stats.HitRatio = float64(stats.Hits) / float64(total)
	}

	return stats
}

// Cleanup implements CacheService.Cleanup.
func (c *InMemoryCache) Cleanup(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	removed := 0

	for key, entry := range c.data {
		if now.After(entry.ExpiresAt) {
			delete(c.data, key)
			removed++
		}
	}

	c.stats.Size = len(c.data)
	c.stats.Evictions += int64(removed)
	c.stats.Cleanups++
	c.stats.LastCleanup = now

	return nil
}

// Stop stops the cache cleanup goroutine.
func (c *InMemoryCache) Stop() {
	if c.cleanupTicker != nil {
		c.cleanupTicker.Stop()
	}
	close(c.stopCleanup)
}

// Helper methods

func (c *InMemoryCache) evictLRU() {
	// Find the least recently used entry
	var oldestKey string
	var oldestTime time.Time

	for key, entry := range c.data {
		if oldestKey == "" || entry.LastAccess.Before(oldestTime) {
			oldestKey = key
			oldestTime = entry.LastAccess
		}
	}

	if oldestKey != "" {
		delete(c.data, oldestKey)
		c.stats.Evictions++
	}
}

func (c *InMemoryCache) cleanupLoop() {
	for {
		select {
		case <-c.cleanupTicker.C:
			_ = c.Cleanup(context.Background()) // Ignore cleanup errors in background
		case <-c.stopCleanup:
			return
		}
	}
}

// ConcurrencyManager manages concurrent operations.
type ConcurrencyManager struct {
	maxConcurrency int
	semaphore      chan struct{}
	activeOps      int64
	mu             sync.RWMutex
}

// NewConcurrencyManager creates a new concurrency manager.
func NewConcurrencyManager(maxConcurrency int) *ConcurrencyManager {
	if maxConcurrency <= 0 {
		maxConcurrency = DefaultMaxConcurrency
	}
	if maxConcurrency > MaxConcurrency {
		maxConcurrency = MaxConcurrency
	}

	return &ConcurrencyManager{
		maxConcurrency: maxConcurrency,
		semaphore:      make(chan struct{}, maxConcurrency),
	}
}

// Acquire acquires a slot for concurrent operation.
func (cm *ConcurrencyManager) Acquire(ctx context.Context) error {
	select {
	case cm.semaphore <- struct{}{}:
		cm.mu.Lock()
		cm.activeOps++
		cm.mu.Unlock()
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Release releases a slot after operation completion.
func (cm *ConcurrencyManager) Release() {
	select {
	case <-cm.semaphore:
		cm.mu.Lock()
		cm.activeOps--
		cm.mu.Unlock()
	default:
		// Should not happen, but handle gracefully
	}
}

// ActiveOperations returns the number of currently active operations.
func (cm *ConcurrencyManager) ActiveOperations() int64 {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.activeOps
}

// MaxConcurrency returns the maximum allowed concurrent operations.
func (cm *ConcurrencyManager) MaxConcurrency() int {
	return cm.maxConcurrency
}

// OperationWrapper wraps an operation with concurrency control.
func (cm *ConcurrencyManager) OperationWrapper(ctx context.Context, operation func() error) error {
	if err := cm.Acquire(ctx); err != nil {
		return err
	}
	defer cm.Release()

	return operation()
}
