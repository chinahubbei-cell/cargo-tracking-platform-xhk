package middleware

import (
	"sync"
	"time"

	"trackcard-server/utils"

	"github.com/gin-gonic/gin"
)

// LoginAttempt 记录登录尝试
type LoginAttempt struct {
	Count     int
	FirstTime time.Time
	LockUntil time.Time
}

// LoginRateLimiter 登录限流器
type LoginRateLimiter struct {
	attempts       map[string]*LoginAttempt
	mu             sync.RWMutex
	maxAttempts    int
	windowDuration time.Duration
	lockDuration   time.Duration
}

var loginLimiter = &LoginRateLimiter{
	attempts:       make(map[string]*LoginAttempt),
	maxAttempts:    5,                // 最多5次尝试
	windowDuration: 10 * time.Minute, // 10分钟内
	lockDuration:   15 * time.Minute, // 锁定15分钟
}

// LoginRateLimit 登录限流中间件
func LoginRateLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()

		loginLimiter.mu.Lock()
		defer loginLimiter.mu.Unlock()

		attempt, exists := loginLimiter.attempts[ip]
		now := time.Now()

		if exists {
			// 检查是否被锁定
			if now.Before(attempt.LockUntil) {
				remaining := attempt.LockUntil.Sub(now).Minutes()
				utils.TooManyRequests(c, int(remaining)+1)
				c.Abort()
				return
			}

			// 检查是否超出时间窗口，重置计数
			if now.Sub(attempt.FirstTime) > loginLimiter.windowDuration {
				attempt.Count = 0
				attempt.FirstTime = now
			}
		} else {
			loginLimiter.attempts[ip] = &LoginAttempt{
				Count:     0,
				FirstTime: now,
			}
			attempt = loginLimiter.attempts[ip]
		}

		c.Next()

		// 登录失败时增加计数
		if c.Writer.Status() == 401 {
			attempt.Count++
			if attempt.Count >= loginLimiter.maxAttempts {
				attempt.LockUntil = now.Add(loginLimiter.lockDuration)
			}
		} else if c.Writer.Status() == 200 {
			// 登录成功，重置计数
			attempt.Count = 0
		}
	}
}

// RecordLoginSuccess 记录登录成功（重置计数）
func RecordLoginSuccess(ip string) {
	loginLimiter.mu.Lock()
	defer loginLimiter.mu.Unlock()

	if attempt, exists := loginLimiter.attempts[ip]; exists {
		attempt.Count = 0
	}
}
