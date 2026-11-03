package utils

import (
	"net"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
)

func GetClientIPFromRemoteAddr(c echo.Context) string {
	return ExtractHostFromRemoteAddr(c.Request())
}

func ExtractHostFromRemoteAddr(r *http.Request) string {
	addr := r.RemoteAddr
	if addr == "" {
		return ""
	}
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return strings.TrimSpace(addr)
	}
	return host
}

// IsPrivateOrReservedIP checks if the given IP address is private or reserved
func IsPrivateOrReservedIP(ipStr string) bool {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return false // Invalid IP address
	}

	// Private IP ranges:
	// IPv4:
	//   10.0.0.0/8
	//   172.16.0.0/12
	//   192.168.0.0/16
	// IPv6:
	//   fc00::/7 (Unique Local Addresses)
	if ip.IsPrivate() {
		return true
	}

	// Loopback addresses:
	// IPv4: 127.0.0.0/8
	// IPv6: ::1/128
	if ip.IsLoopback() {
		return true
	}

	// Link-local addresses:
	// IPv4: 169.254.0.0/16
	// IPv6: fe80::/10
	if ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
		return true
	}

	// Documentation addresses:
	// IPv4:
	//   192.0.2.0/24 (TEST-NET-1)
	//   198.51.100.0/24 (TEST-NET-2)
	//   203.0.113.0/24 (TEST-NET-3)
	// IPv6:
	//   2001:db8::/32
	if isDocumentationIP(ip) {
		return true
	}

	// Other reserved ranges
	return isOtherReservedIP(ip)
}

// isDocumentationIP checks if the IP is in documentation ranges (RFC 5737 / RFC 3849).
func isDocumentationIP(ip net.IP) bool {
	if ip4 := ip.To4(); ip4 != nil {
		return ip4.Equal(net.IPv4(192, 0, 2, 0)) ||
			ip4.Equal(net.IPv4(198, 51, 100, 0)) ||
			ip4.Equal(net.IPv4(203, 0, 113, 0))
	}
	docV6 := net.IP{0x20, 0x01, 0x0d, 0xb8, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
	return ip.Equal(docV6)
}

// isOtherReservedIP checks for other reserved IP ranges
func isOtherReservedIP(ip net.IP) bool {
	if ip4 := ip.To4(); ip4 != nil {
		// Other reserved IPv4 ranges:
		//   0.0.0.0/8 - Current network (RFC 1122)
		//   100.64.0.0/10 - Shared Address Space (RFC 6598)
		//   192.0.0.0/24 - IETF Protocol Assignments (RFC 6890)
		//   192.88.99.0/24 - IPv6 to IPv4 relay (RFC 3068)
		//   198.18.0.0/15 - Network benchmark tests (RFC 2544)
		//   240.0.0.0/4 - Reserved (RFC 1112)
		return ip4[0] == 0 ||
			(ip4[0] == 100 && (ip4[1]&0xc0) == 64) ||
			(ip4[0] == 192 && ip4[1] == 0 && ip4[2] == 0) ||
			(ip4[0] == 192 && ip4[1] == 88 && ip4[2] == 99) ||
			(ip4[0] == 198 && (ip4[1]&0xfe) == 18) ||
			(ip4[0]&0xf0) == 240
	}

	// Other reserved IPv6 ranges:
	//   ::/128 - Unspecified address
	//   ::1/128 - Loopback address (already covered by IsLoopback())
	//   ::ffff:0:0/96 - IPv4-mapped IPv6 address
	//   64:ff9b::/96 - IPv4-IPv6 translation (RFC 6052)
	//   100::/64 - Discard prefix (RFC 6666)
	//   2001::/23 - IETF Protocol Assignments
	//   2001:2::/48 - Benchmarking (RFC 5180)
	//   2002::/16 - 6to4 (RFC 3056)
	//   fe80::/10 - Link-local (already covered by IsLinkLocalUnicast())
	//   ff00::/8 - Multicast
	return ip.Equal(net.IPv6unspecified) ||
		ip.Equal(net.ParseIP("::ffff:0:0")) ||
		ip.Equal(net.ParseIP("64:ff9b::")) ||
		ip.Equal(net.ParseIP("100::")) ||
		(len(ip) == net.IPv6len && ip[0] == 0x20 && ip[1] == 0x01 && (ip[2]&0xfe) == 0) ||
		(len(ip) == net.IPv6len && ip[0] == 0x20 && ip[1] == 0x01 && ip[2] == 0x00 && ip[3] == 0x02) ||
		(len(ip) == net.IPv6len && ip[0] == 0x20 && ip[1] == 0x02) ||
		(len(ip) == net.IPv6len && ip[0] == 0xff)
}

func IsIPv6(ipStr string) bool {
	ip := net.ParseIP(ipStr)
	return ip != nil && ip.To4() == nil
}
