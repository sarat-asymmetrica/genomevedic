package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// CacheStore is an interface for caching variant explanations
type CacheStore interface {
	Get(ctx context.Context, key string) (*CacheEntry, error)
	Set(ctx context.Context, key string, entry *CacheEntry, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
	GetHitRate(ctx context.Context) (float64, error)
	Close() error
}

// MemoryCache is an in-memory cache implementation (for development/testing)
type MemoryCache struct {
	store    map[string]*CacheEntry
	hits     int64
	misses   int64
}

// NewMemoryCache creates a new in-memory cache
func NewMemoryCache() *MemoryCache {
	return &MemoryCache{
		store: make(map[string]*CacheEntry),
	}
}

// Get retrieves a cache entry
func (mc *MemoryCache) Get(ctx context.Context, key string) (*CacheEntry, error) {
	entry, exists := mc.store[key]
	if !exists {
		mc.misses++
		return nil, fmt.Errorf("cache miss")
	}

	// Check if expired
	if time.Now().After(entry.ExpiresAt) {
		delete(mc.store, key)
		mc.misses++
		return nil, fmt.Errorf("cache expired")
	}

	mc.hits++
	return entry, nil
}

// Set stores a cache entry
func (mc *MemoryCache) Set(ctx context.Context, key string, entry *CacheEntry, ttl time.Duration) error {
	entry.CachedAt = time.Now()
	entry.ExpiresAt = time.Now().Add(ttl)
	mc.store[key] = entry
	return nil
}

// Delete removes a cache entry
func (mc *MemoryCache) Delete(ctx context.Context, key string) error {
	delete(mc.store, key)
	return nil
}

// GetHitRate returns the cache hit rate
func (mc *MemoryCache) GetHitRate(ctx context.Context) (float64, error) {
	total := mc.hits + mc.misses
	if total == 0 {
		return 0.0, nil
	}
	return float64(mc.hits) / float64(total), nil
}

// Close closes the cache (no-op for memory cache)
func (mc *MemoryCache) Close() error {
	return nil
}

// RedisCache is a Redis-backed cache implementation
type RedisCache struct {
	// Would use github.com/redis/go-redis/v9 in production
	// For now, we'll use memory cache as fallback
	fallback *MemoryCache
	enabled  bool
}

// NewRedisCache creates a new Redis cache
func NewRedisCache(addr, password string, db int) (*RedisCache, error) {
	// In production, this would connect to Redis:
	// client := redis.NewClient(&redis.Options{
	//     Addr:     addr,
	//     Password: password,
	//     DB:       db,
	// })
	//
	// ctx := context.Background()
	// if err := client.Ping(ctx).Err(); err != nil {
	//     return nil, err
	// }

	// For now, fall back to memory cache
	return &RedisCache{
		fallback: NewMemoryCache(),
		enabled:  false, // Set to true when Redis is available
	}, nil
}

// Get retrieves a cache entry
func (rc *RedisCache) Get(ctx context.Context, key string) (*CacheEntry, error) {
	if !rc.enabled {
		return rc.fallback.Get(ctx, key)
	}

	// Redis implementation:
	// val, err := rc.client.Get(ctx, key).Result()
	// if err == redis.Nil {
	//     return nil, fmt.Errorf("cache miss")
	// } else if err != nil {
	//     return nil, err
	// }
	//
	// var entry CacheEntry
	// if err := json.Unmarshal([]byte(val), &entry); err != nil {
	//     return nil, err
	// }
	//
	// return &entry, nil

	return rc.fallback.Get(ctx, key)
}

// Set stores a cache entry
func (rc *RedisCache) Set(ctx context.Context, key string, entry *CacheEntry, ttl time.Duration) error {
	if !rc.enabled {
		return rc.fallback.Set(ctx, key, entry, ttl)
	}

	// Redis implementation:
	// data, err := json.Marshal(entry)
	// if err != nil {
	//     return err
	// }
	//
	// return rc.client.Set(ctx, key, data, ttl).Err()

	return rc.fallback.Set(ctx, key, entry, ttl)
}

// Delete removes a cache entry
func (rc *RedisCache) Delete(ctx context.Context, key string) error {
	if !rc.enabled {
		return rc.fallback.Delete(ctx, key)
	}

	// Redis implementation:
	// return rc.client.Del(ctx, key).Err()

	return rc.fallback.Delete(ctx, key)
}

// GetHitRate returns the cache hit rate
func (rc *RedisCache) GetHitRate(ctx context.Context) (float64, error) {
	if !rc.enabled {
		return rc.fallback.GetHitRate(ctx)
	}

	// Redis implementation:
	// hits, _ := rc.client.Get(ctx, "cache:hits").Int64()
	// misses, _ := rc.client.Get(ctx, "cache:misses").Int64()
	// total := hits + misses
	// if total == 0 {
	//     return 0.0, nil
	// }
	// return float64(hits) / float64(total), nil

	return rc.fallback.GetHitRate(ctx)
}

// Close closes the Redis connection
func (rc *RedisCache) Close() error {
	if !rc.enabled {
		return rc.fallback.Close()
	}

	// Redis implementation:
	// return rc.client.Close()

	return rc.fallback.Close()
}

// GenerateCacheKey creates a cache key for a variant
func GenerateCacheKey(input VariantInput) string {
	return fmt.Sprintf("variant:%s:%s:%s:%d", input.Gene, input.Variant, input.Chromosome, input.Position)
}

// CacheManager manages caching with statistics
type CacheManager struct {
	store CacheStore
	ttl   time.Duration
}

// NewCacheManager creates a new cache manager
func NewCacheManager(store CacheStore, ttlDays int) *CacheManager {
	return &CacheManager{
		store: store,
		ttl:   time.Duration(ttlDays) * 24 * time.Hour,
	}
}

// GetOrCompute retrieves from cache or computes if not found
func (cm *CacheManager) GetOrCompute(ctx context.Context, key string, compute func() (*ExplanationResponse, error)) (*ExplanationResponse, error) {
	// Try to get from cache
	entry, err := cm.store.Get(ctx, key)
	if err == nil && entry != nil {
		// Cache hit!
		return &ExplanationResponse{
			Explanation:  entry.Explanation,
			Context:      entry.Context,
			Cached:       true,
			ResponseTime: 0,
			TokensUsed:   0,
			CostUSD:      0,
		}, nil
	}

	// Cache miss, compute
	result, err := compute()
	if err != nil {
		return nil, err
	}

	// Store in cache
	entry = &CacheEntry{
		Explanation: result.Explanation,
		Context:     result.Context,
		TokensUsed:  result.TokensUsed,
	}

	_ = cm.store.Set(ctx, key, entry, cm.ttl)

	return result, nil
}

// InvalidatePattern invalidates all cache entries matching a pattern
func (cm *CacheManager) InvalidatePattern(ctx context.Context, pattern string) error {
	// For memory cache, this would iterate and delete matching keys
	// For Redis, this would use SCAN + DEL
	return nil
}

// GetStats returns cache statistics
func (cm *CacheManager) GetStats(ctx context.Context) (map[string]interface{}, error) {
	hitRate, err := cm.store.GetHitRate(ctx)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"hit_rate": hitRate,
		"ttl_days": cm.ttl.Hours() / 24,
	}, nil
}

// SerializeCacheEntry serializes a cache entry to JSON
func SerializeCacheEntry(entry *CacheEntry) ([]byte, error) {
	return json.Marshal(entry)
}

// DeserializeCacheEntry deserializes a cache entry from JSON
func DeserializeCacheEntry(data []byte) (*CacheEntry, error) {
	var entry CacheEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		return nil, err
	}
	return &entry, nil
}
