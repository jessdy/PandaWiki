package utils

import (
	"fmt"
	"net"
	"net/url"
	"strings"
)

// RejectInternalURL 拒绝把爬虫指到本机、链路本地、私网或未解析地址。
// 只允许 http/https。DNS 解析结果里只要有一个被禁地址就拒绝。
func RejectInternalURL(raw string) error {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Host == "" {
		return fmt.Errorf("无效的抓取地址")
	}
	switch strings.ToLower(u.Scheme) {
	case "http", "https":
	default:
		return fmt.Errorf("抓取地址只允许 http 或 https")
	}
	host := u.Hostname()
	if host == "" {
		return fmt.Errorf("无效的抓取地址")
	}
	if ip := net.ParseIP(host); ip != nil {
		if blockedIP(ip) {
			return fmt.Errorf("不允许抓取内网或本机地址")
		}
		return nil
	}
	ips, err := net.LookupIP(host)
	if err != nil || len(ips) == 0 {
		return fmt.Errorf("无法解析抓取地址")
	}
	for _, ip := range ips {
		if blockedIP(ip) {
			return fmt.Errorf("不允许抓取内网或本机地址")
		}
	}
	return nil
}

func blockedIP(ip net.IP) bool {
	if ip == nil {
		return true
	}
	return ip.IsLoopback() ||
		ip.IsPrivate() ||
		ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast() ||
		ip.IsUnspecified() ||
		ip.IsMulticast()
}
