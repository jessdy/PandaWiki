package ratelimit

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/chaitin/panda-wiki/log"
	"github.com/chaitin/panda-wiki/store/cache"
)

type RateLimiter struct {
	logger *log.Logger
	cache  *cache.Cache
}

func NewRateLimiter(logger *log.Logger, cache *cache.Cache) *RateLimiter {
	return &RateLimiter{
		logger: logger,
		cache:  cache,
	}
}

const (
	LockThreshold1    = 5  // 第一次锁定阈值
	LockThreshold2    = 10 // 第二次锁定阈值
	LockThreshold3    = 15 // 第三次锁定阈值
	AttemptsKeyExpiry = 24 * time.Hour
)

func lockSubjects(ip, account string) []string {
	var subjects []string
	if ip = strings.TrimSpace(ip); ip != "" {
		subjects = append(subjects, "ip:"+ip)
	}
	if account = strings.ToLower(strings.TrimSpace(account)); account != "" {
		subjects = append(subjects, "acct:"+account)
	}
	return subjects
}

// CheckLocked 同时检查来源 IP 与账号是否处于锁定（任一命中即拒绝）。
func (r *RateLimiter) CheckLocked(ctx context.Context, ip, account string) (bool, time.Duration) {
	var longest time.Duration
	locked := false
	for _, subject := range lockSubjects(ip, account) {
		ok, ttl := r.locked(ctx, subject)
		if ok {
			locked = true
			if ttl > longest {
				longest = ttl
			}
		}
	}
	return locked, longest
}

// CheckIPLocked 兼容仅按 IP 检查的调用方。
func (r *RateLimiter) CheckIPLocked(ctx context.Context, ip string) (bool, time.Duration) {
	return r.CheckLocked(ctx, ip, "")
}

func (r *RateLimiter) locked(ctx context.Context, subject string) (bool, time.Duration) {
	ttl, err := r.cache.TTL(ctx, "login_lock:"+subject).Result()
	if err != nil {
		r.logger.Error("failed to check lock status", "error", err, "subject", subject)
		return false, 0
	}
	if ttl > 0 {
		return true, ttl
	}
	return false, 0
}

// LockAttempt 失败一次：IP 与账号（若有）各自累计，避免换 IP 绕过爆破。
func (r *RateLimiter) LockAttempt(ctx context.Context, ip, account string) {
	for _, subject := range lockSubjects(ip, account) {
		r.lockSubject(ctx, subject)
	}
}

func (r *RateLimiter) lockSubject(ctx context.Context, subject string) {
	attemptsKey := "login_attempts:" + subject
	lockKey := "login_lock:" + subject

	attempts, err := r.cache.Incr(ctx, attemptsKey).Result()
	if err != nil {
		r.logger.Error("failed to increment attempts", "error", err, "subject", subject)
		return
	}
	if err := r.cache.Expire(ctx, attemptsKey, AttemptsKeyExpiry).Err(); err != nil {
		r.logger.Error("failed to set expiry on attempts key", "error", err, "subject", subject)
	}

	var lockDuration time.Duration
	if attempts%5 == 0 {
		switch {
		case attempts == LockThreshold1:
			lockDuration = time.Minute
		case attempts == LockThreshold2:
			lockDuration = 15 * time.Minute
		case attempts >= LockThreshold3:
			lockDuration = time.Hour
		}
		if lockDuration > 0 {
			if err := r.cache.Set(ctx, lockKey, 1, lockDuration).Err(); err != nil {
				r.logger.Error("failed to set lock key", "error", err, "subject", subject)
				return
			}
			r.logger.Info("login subject locked", "subject", subject, "lockDuration", lockDuration)
		}
	}
}

// ResetLoginAttempts 登录成功后清零 IP 与账号计数。
func (r *RateLimiter) ResetLoginAttempts(ctx context.Context, ip, account string) error {
	pipe := r.cache.Pipeline()
	for _, subject := range lockSubjects(ip, account) {
		pipe.Del(ctx, "login_attempts:"+subject)
		pipe.Del(ctx, "login_lock:"+subject)
	}
	_, err := pipe.Exec(ctx)
	if err != nil && !errors.Is(err, redis.Nil) {
		return fmt.Errorf("failed to reset login attempts: %w", err)
	}
	return nil
}
