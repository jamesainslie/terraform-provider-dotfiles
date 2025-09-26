// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0.

package services

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestNewInMemoryCache(t *testing.T) {
	config := CacheConfig{
		MaxSize:         100,
		DefaultTTL:      5 * time.Minute,
		CleanupInterval: 1 * time.Minute,
	}

	cache := NewInMemoryCache(config)
	if cache == nil {
		t.Fatal("Expected cache to be created")
	}

	if cache.maxSize != 100 {
		t.Errorf("Expected max size 100, got %d", cache.maxSize)
	}

	if cache.defaultTTL != 5*time.Minute {
		t.Errorf("Expected default TTL 5m, got %v", cache.defaultTTL)
	}

	// Cleanup
	cache.Stop()
}

func TestNewInMemoryCacheDefaults(t *testing.T) {
	config := CacheConfig{} // Empty config should use defaults

	cache := NewInMemoryCache(config)
	if cache == nil {
		t.Fatal("Expected cache to be created")
	}

	if cache.maxSize != DefaultCacheSize {
		t.Errorf("Expected default max size %d, got %d", DefaultCacheSize, cache.maxSize)
	}

	expectedTTL := time.Duration(DefaultCacheTTL) * time.Second
	if cache.defaultTTL != expectedTTL {
		t.Errorf("Expected default TTL %v, got %v", expectedTTL, cache.defaultTTL)
	}

	cache.Stop()
}

func TestCacheSetAndGet(t *testing.T) {
	cache := NewInMemoryCache(CacheConfig{
		MaxSize:    10,
		DefaultTTL: 1 * time.Minute,
	})
	defer cache.Stop()

	ctx := context.Background()

	// Test setting and getting a value
	err := cache.Set(ctx, "test-key", "test-value", 0)
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	value, found := cache.Get(ctx, "test-key")
	if !found {
		t.Fatal("Expected to find cached value")
	}

	if value != "test-value" {
		t.Errorf("Expected 'test-value', got %v", value)
	}
}

func TestCacheGetMissing(t *testing.T) {
	cache := NewInMemoryCache(CacheConfig{MaxSize: 10})
	defer cache.Stop()

	ctx := context.Background()
	value, found := cache.Get(ctx, "missing-key")

	if found {
		t.Error("Expected not to find missing key")
	}

	if value != nil {
		t.Errorf("Expected nil value, got %v", value)
	}
}

func TestCacheTTLExpiration(t *testing.T) {
	cache := NewInMemoryCache(CacheConfig{MaxSize: 10})
	defer cache.Stop()

	ctx := context.Background()

	// Set value with very short TTL
	err := cache.Set(ctx, "expire-key", "expire-value", 50*time.Millisecond)
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	// Should be available immediately
	value, found := cache.Get(ctx, "expire-key")
	if !found || value != "expire-value" {
		t.Error("Expected to find value immediately after set")
	}

	// Wait for expiration
	time.Sleep(100 * time.Millisecond)

	// Should be expired now
	value, found = cache.Get(ctx, "expire-key")
	if found {
		t.Error("Expected value to be expired")
	}
}

func TestCacheDelete(t *testing.T) {
	cache := NewInMemoryCache(CacheConfig{MaxSize: 10})
	defer cache.Stop()

	ctx := context.Background()

	// Set and verify
	err := cache.Set(ctx, "delete-key", "delete-value", 0)
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	value, found := cache.Get(ctx, "delete-key")
	if !found || value != "delete-value" {
		t.Error("Expected to find value before delete")
	}

	// Delete and verify
	err = cache.Delete(ctx, "delete-key")
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	value, found = cache.Get(ctx, "delete-key")
	if found {
		t.Error("Expected value to be deleted")
	}
}

func TestCacheClear(t *testing.T) {
	cache := NewInMemoryCache(CacheConfig{MaxSize: 10})
	defer cache.Stop()

	ctx := context.Background()

	// Set multiple values
	for i := 0; i < 5; i++ {
		key := fmt.Sprintf("key-%d", i)
		value := fmt.Sprintf("value-%d", i)
		err := cache.Set(ctx, key, value, 0)
		if err != nil {
			t.Fatalf("Set failed for %s: %v", key, err)
		}
	}

	// Verify size
	if cache.Size() != 5 {
		t.Errorf("Expected size 5, got %d", cache.Size())
	}

	// Clear cache
	err := cache.Clear(ctx)
	if err != nil {
		t.Fatalf("Clear failed: %v", err)
	}

	// Verify empty
	if cache.Size() != 0 {
		t.Errorf("Expected size 0 after clear, got %d", cache.Size())
	}

	// Verify values are gone
	for i := 0; i < 5; i++ {
		key := fmt.Sprintf("key-%d", i)
		_, found := cache.Get(ctx, key)
		if found {
			t.Errorf("Expected %s to be cleared", key)
		}
	}
}

func TestCacheLRUEviction(t *testing.T) {
	cache := NewInMemoryCache(CacheConfig{
		MaxSize:    3, // Small size to trigger eviction
		DefaultTTL: 1 * time.Minute,
	})
	defer cache.Stop()

	ctx := context.Background()

	// Fill cache to capacity
	for i := 0; i < 3; i++ {
		key := fmt.Sprintf("key-%d", i)
		value := fmt.Sprintf("value-%d", i)
		err := cache.Set(ctx, key, value, 0)
		if err != nil {
			t.Fatalf("Set failed for %s: %v", key, err)
		}
		time.Sleep(1 * time.Millisecond) // Ensure different access times
	}

	// Access key-1 to make it more recent
	cache.Get(ctx, "key-1")

	// Add one more item, should evict key-0 (least recently used)
	err := cache.Set(ctx, "key-3", "value-3", 0)
	if err != nil {
		t.Fatalf("Set failed for key-3: %v", err)
	}

	// key-0 should be evicted
	_, found := cache.Get(ctx, "key-0")
	if found {
		t.Error("Expected key-0 to be evicted")
	}

	// key-1 should still be there (was accessed recently)
	_, found = cache.Get(ctx, "key-1")
	if !found {
		t.Error("Expected key-1 to still be in cache")
	}
}

func TestCacheStats(t *testing.T) {
	cache := NewInMemoryCache(CacheConfig{MaxSize: 10})
	defer cache.Stop()

	ctx := context.Background()

	// Initial stats
	stats := cache.Stats()
	if stats.Size != 0 {
		t.Errorf("Expected initial size 0, got %d", stats.Size)
	}
	if stats.Hits != 0 {
		t.Errorf("Expected initial hits 0, got %d", stats.Hits)
	}
	if stats.Misses != 0 {
		t.Errorf("Expected initial misses 0, got %d", stats.Misses)
	}

	// Add some data and access patterns
	cache.Set(ctx, "key1", "value1", 0)
	cache.Set(ctx, "key2", "value2", 0)

	// Hit
	cache.Get(ctx, "key1")
	cache.Get(ctx, "key1") // Second hit

	// Miss
	cache.Get(ctx, "missing")

	stats = cache.Stats()
	if stats.Size != 2 {
		t.Errorf("Expected size 2, got %d", stats.Size)
	}
	if stats.Hits != 2 {
		t.Errorf("Expected hits 2, got %d", stats.Hits)
	}
	if stats.Misses != 1 {
		t.Errorf("Expected misses 1, got %d", stats.Misses)
	}

	expectedHitRatio := float64(2) / float64(3) // 2 hits out of 3 total requests
	if stats.HitRatio != expectedHitRatio {
		t.Errorf("Expected hit ratio %f, got %f", expectedHitRatio, stats.HitRatio)
	}
}

func TestCacheCleanup(t *testing.T) {
	cache := NewInMemoryCache(CacheConfig{
		MaxSize:         10,
		DefaultTTL:      50 * time.Millisecond,
		CleanupInterval: 1 * time.Second, // Long interval so we can test manual cleanup
	})
	defer cache.Stop()

	ctx := context.Background()

	// Add items with short TTL
	cache.Set(ctx, "expire1", "value1", 30*time.Millisecond)
	cache.Set(ctx, "expire2", "value2", 30*time.Millisecond)
	cache.Set(ctx, "keep", "value3", 1*time.Minute) // Long TTL

	// Wait for expiration
	time.Sleep(100 * time.Millisecond)

	// Manual cleanup
	err := cache.Cleanup(ctx)
	if err != nil {
		t.Fatalf("Cleanup failed: %v", err)
	}

	// Expired items should be gone
	_, found := cache.Get(ctx, "expire1")
	if found {
		t.Error("Expected expire1 to be cleaned up")
	}

	_, found = cache.Get(ctx, "expire2")
	if found {
		t.Error("Expected expire2 to be cleaned up")
	}

	// Non-expired item should remain
	_, found = cache.Get(ctx, "keep")
	if !found {
		t.Error("Expected keep to remain after cleanup")
	}
}

func TestCacheConcurrency(t *testing.T) {
	cache := NewInMemoryCache(CacheConfig{MaxSize: 100})
	defer cache.Stop()

	ctx := context.Background()
	numGoroutines := 10
	numOperations := 100

	var wg sync.WaitGroup

	// Concurrent writes
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				key := fmt.Sprintf("key-%d-%d", id, j)
				value := fmt.Sprintf("value-%d-%d", id, j)
				err := cache.Set(ctx, key, value, 0)
				if err != nil {
					t.Errorf("Set failed: %v", err)
				}
			}
		}(i)
	}

	// Concurrent reads
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				key := fmt.Sprintf("key-%d-%d", id, j)
				cache.Get(ctx, key)
			}
		}(i)
	}

	wg.Wait()

	// Cache should still be functional
	cache.Set(ctx, "test-after-concurrent", "test-value", 0)
	value, found := cache.Get(ctx, "test-after-concurrent")
	if !found || value != "test-value" {
		t.Error("Cache not functional after concurrent operations")
	}
}

func TestNewConcurrencyManager(t *testing.T) {
	cm := NewConcurrencyManager(5)
	if cm == nil {
		t.Fatal("Expected concurrency manager to be created")
	}

	if cm.MaxConcurrency() != 5 {
		t.Errorf("Expected max concurrency 5, got %d", cm.MaxConcurrency())
	}

	if cm.ActiveOperations() != 0 {
		t.Errorf("Expected 0 active operations, got %d", cm.ActiveOperations())
	}
}

func TestNewConcurrencyManagerDefaults(t *testing.T) {
	// Test with 0 (should use default)
	cm := NewConcurrencyManager(0)
	if cm.MaxConcurrency() != DefaultMaxConcurrency {
		t.Errorf("Expected default max concurrency %d, got %d", DefaultMaxConcurrency, cm.MaxConcurrency())
	}

	// Test with too high value (should cap at max)
	cm = NewConcurrencyManager(1000)
	if cm.MaxConcurrency() != MaxConcurrency {
		t.Errorf("Expected capped max concurrency %d, got %d", MaxConcurrency, cm.MaxConcurrency())
	}
}

func TestConcurrencyManagerAcquireRelease(t *testing.T) {
	cm := NewConcurrencyManager(2)
	ctx := context.Background()

	// First acquire
	err := cm.Acquire(ctx)
	if err != nil {
		t.Fatalf("First acquire failed: %v", err)
	}

	if cm.ActiveOperations() != 1 {
		t.Errorf("Expected 1 active operation, got %d", cm.ActiveOperations())
	}

	// Second acquire
	err = cm.Acquire(ctx)
	if err != nil {
		t.Fatalf("Second acquire failed: %v", err)
	}

	if cm.ActiveOperations() != 2 {
		t.Errorf("Expected 2 active operations, got %d", cm.ActiveOperations())
	}

	// Release one
	cm.Release()
	if cm.ActiveOperations() != 1 {
		t.Errorf("Expected 1 active operation after release, got %d", cm.ActiveOperations())
	}

	// Release second
	cm.Release()
	if cm.ActiveOperations() != 0 {
		t.Errorf("Expected 0 active operations after second release, got %d", cm.ActiveOperations())
	}
}

func TestConcurrencyManagerBlocking(t *testing.T) {
	cm := NewConcurrencyManager(1) // Only allow 1 concurrent operation

	// Acquire the only slot
	ctx := context.Background()
	err := cm.Acquire(ctx)
	if err != nil {
		t.Fatalf("Acquire failed: %v", err)
	}

	// Try to acquire with timeout - should block and timeout
	timeoutCtx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	err = cm.Acquire(timeoutCtx)
	if err == nil {
		t.Error("Expected acquire to timeout")
	}

	if err != context.DeadlineExceeded {
		t.Errorf("Expected deadline exceeded error, got %v", err)
	}

	// Release and try again - should succeed
	cm.Release()

	err = cm.Acquire(ctx)
	if err != nil {
		t.Errorf("Acquire after release failed: %v", err)
	}

	cm.Release()
}

func TestConcurrencyManagerOperationWrapper(t *testing.T) {
	cm := NewConcurrencyManager(2)
	ctx := context.Background()

	executed := false
	operation := func() error {
		executed = true
		time.Sleep(10 * time.Millisecond) // Simulate work
		return nil
	}

	err := cm.OperationWrapper(ctx, operation)
	if err != nil {
		t.Fatalf("OperationWrapper failed: %v", err)
	}

	if !executed {
		t.Error("Expected operation to be executed")
	}

	if cm.ActiveOperations() != 0 {
		t.Errorf("Expected 0 active operations after wrapper completion, got %d", cm.ActiveOperations())
	}
}

func TestConcurrencyManagerOperationWrapperError(t *testing.T) {
	cm := NewConcurrencyManager(2)
	ctx := context.Background()

	expectedError := fmt.Errorf("operation failed")
	operation := func() error {
		return expectedError
	}

	err := cm.OperationWrapper(ctx, operation)
	if err != expectedError {
		t.Errorf("Expected error %v, got %v", expectedError, err)
	}

	// Should still release the slot even on error
	if cm.ActiveOperations() != 0 {
		t.Errorf("Expected 0 active operations after wrapper error, got %d", cm.ActiveOperations())
	}
}
