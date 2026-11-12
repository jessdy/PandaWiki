package middleware

import (
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"golang.org/x/time/rate"

	"github.com/chaitin/panda-wiki/config"
	"github.com/chaitin/panda-wiki/domain"
)

// ClientIPExtractor 只采信回环地址、容器网段和 TRUSTED_PROXIES 中代理追加的 X-Forwarded-For，
// 客户端自带的 X-Forwarded-For 不会被当作来源 IP（登录锁定、限流、审计都依赖它）。
func ClientIPExtractor(cfg *config.Config) echo.IPExtractor {
	opts := []echo.TrustOption{
		echo.TrustLoopback(true),
		echo.TrustLinkLocal(false),
		echo.TrustPrivateNet(false),
	}
	for _, n := range TrustedProxyNets(cfg) {
		opts = append(opts, echo.TrustIPRange(n))
	}
	return echo.ExtractIPFromXFFHeader(opts...)
}

func TrustedProxyNets(cfg *config.Config) []*net.IPNet {
	cidrs := append([]string{cfg.SubnetPrefix + ".0/24"}, cfg.HTTP.TrustedProxies...)
	nets := make([]*net.IPNet, 0, len(cidrs))
	for _, s := range cidrs {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		if !strings.Contains(s, "/") {
			ip := net.ParseIP(s)
			if ip == nil {
				continue
			}
			bits := 128
			if ip.To4() != nil {
				bits = 32
			}
			s = fmt.Sprintf("%s/%d", s, bits)
		}
		if _, n, err := net.ParseCIDR(s); err == nil {
			nets = append(nets, n)
		}
	}
	return nets
}

// RateLimit 按来源 IP 做全局限流；容器网段内的调用（前台 SSR、爬虫回传）不计入。
func RateLimit(cfg *config.Config) echo.MiddlewareFunc {
	if cfg.HTTP.RateLimitRPS <= 0 {
		return func(next echo.HandlerFunc) echo.HandlerFunc { return next }
	}
	internal := TrustedProxyNets(cfg)
	store := middleware.NewRateLimiterMemoryStoreWithConfig(middleware.RateLimiterMemoryStoreConfig{
		Rate:      rate.Limit(cfg.HTTP.RateLimitRPS),
		Burst:     cfg.HTTP.RateLimitBurst,
		ExpiresIn: 3 * time.Minute,
	})
	return middleware.RateLimiterWithConfig(middleware.RateLimiterConfig{
		Skipper: func(c echo.Context) bool {
			// 一篇文档可能同时加载几十张图片，静态文件读取不计入限流
			if strings.HasPrefix(c.Request().URL.Path, "/static-file/") {
				return true
			}
			ip := net.ParseIP(c.RealIP())
			if ip == nil {
				return false
			}
			if ip.IsLoopback() {
				return true
			}
			for _, n := range internal {
				if n.Contains(ip) {
					return true
				}
			}
			return false
		},
		Store: store,
		IdentifierExtractor: func(c echo.Context) (string, error) {
			return c.RealIP(), nil
		},
		ErrorHandler: func(c echo.Context, err error) error {
			return c.JSON(http.StatusForbidden, domain.PWResponse{Message: "无法识别请求来源"})
		},
		DenyHandler: func(c echo.Context, identifier string, err error) error {
			return c.JSON(http.StatusTooManyRequests, domain.PWResponse{Message: "请求过于频繁，请稍后再试"})
		},
	})
}
