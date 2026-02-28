package services

import (
	"sync"
	"time"
)

// ============================================================
// 内存缓存服务 - 用于热点数据缓存
// ============================================================
// 特点:
// 1. 线程安全 (sync.Map)
// 2. 自动过期 (TTL)
// 3. 零外部依赖
// ============================================================

// CacheEntry 缓存条目
type CacheEntry struct {
	Data      interface{}
	ExpiresAt time.Time
}

// CacheService 内存缓存服务
type CacheService struct {
	data       sync.Map
	defaultTTL time.Duration
}

// 全局缓存实例
var Cache = &CacheService{defaultTTL: 5 * time.Minute}

// Get 获取缓存数据
func (c *CacheService) Get(key string) (interface{}, bool) {
	if v, ok := c.data.Load(key); ok {
		entry := v.(*CacheEntry)
		if time.Now().Before(entry.ExpiresAt) {
			return entry.Data, true
		}
		// 已过期，删除
		c.data.Delete(key)
	}
	return nil, false
}

// Set 设置缓存数据
func (c *CacheService) Set(key string, data interface{}, ttl ...time.Duration) {
	duration := c.defaultTTL
	if len(ttl) > 0 {
		duration = ttl[0]
	}
	c.data.Store(key, &CacheEntry{
		Data:      data,
		ExpiresAt: time.Now().Add(duration),
	})
}

// Delete 删除缓存
func (c *CacheService) Delete(key string) {
	c.data.Delete(key)
}

// DeletePrefix 删除指定前缀的所有缓存
func (c *CacheService) DeletePrefix(prefix string) {
	c.data.Range(func(key, value interface{}) bool {
		if k, ok := key.(string); ok {
			if len(k) >= len(prefix) && k[:len(prefix)] == prefix {
				c.data.Delete(key)
			}
		}
		return true
	})
}

// Clear 清空所有缓存
func (c *CacheService) Clear() {
	c.data.Range(func(key, value interface{}) bool {
		c.data.Delete(key)
		return true
	})
}

// Stats 获取缓存统计
func (c *CacheService) Stats() map[string]int {
	total := 0
	expired := 0
	now := time.Now()

	c.data.Range(func(key, value interface{}) bool {
		total++
		if entry, ok := value.(*CacheEntry); ok {
			if now.After(entry.ExpiresAt) {
				expired++
			}
		}
		return true
	})

	return map[string]int{
		"total":   total,
		"active":  total - expired,
		"expired": expired,
	}
}

// CleanExpired 清理过期条目 (可定期调用)
func (c *CacheService) CleanExpired() int {
	cleaned := 0
	now := time.Now()

	c.data.Range(func(key, value interface{}) bool {
		if entry, ok := value.(*CacheEntry); ok {
			if now.After(entry.ExpiresAt) {
				c.data.Delete(key)
				cleaned++
			}
		}
		return true
	})

	return cleaned
}

// ============================================================
// 缓存 Key 常量定义
// ============================================================

const (
	CacheKeyDashboardStats = "dashboard:stats"
	CacheKeyPortsAll       = "ports:all"
	CacheKeyAirportsAll    = "airports:all"
	CacheKeyShippingLines  = "shipping_lines:all"
)

// CacheTTL 预定义的 TTL 时间
var (
	CacheTTLShort  = 30 * time.Second  // 30秒 - Dashboard 等实时性要求高的数据
	CacheTTLMedium = 5 * time.Minute   // 5分钟 - 一般业务数据
	CacheTTLLong   = 1 * time.Hour     // 1小时 - 港口、机场等静态数据
	CacheTTLDay    = 24 * time.Hour    // 24小时 - 极少变化的数据
)
