package utils

import (
	"crypto/hmac"
	"encoding/hex"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/tjfoc/gmsm/sm3"
)

// 容器网络内访问 API 的地址；MinIO 桶是私有的，其他服务都经 API 读取文件。
// 可通过 INTERNAL_API_BASE_URL 覆盖为 https://...（服务间 mTLS/网关场景）。
func internalAPIBaseURL() string {
	if v := strings.TrimSpace(os.Getenv("INTERNAL_API_BASE_URL")); v != "" {
		return strings.TrimRight(v, "/")
	}
	return (&url.URL{Scheme: "http", Host: "panda-wiki-api:8000"}).String()
}

func escapeObjectKey(key string) string {
	segments := strings.Split(strings.TrimLeft(key, "/"), "/")
	for i, s := range segments {
		segments[i] = url.PathEscape(s)
	}
	return strings.Join(segments, "/")
}

// SignedStaticFileURL 供 anydoc 等容器内服务下载文件。签名放在路径里而不是查询串，
// 链接以原始对象 key 结尾，对方仍能按扩展名识别文件类型。
func SignedStaticFileURL(secret, objectKey string, ttl time.Duration) string {
	key := strings.TrimLeft(objectKey, "/")
	exp := strconv.FormatInt(time.Now().Add(ttl).Unix(), 10)
	return internalAPIBaseURL() + "/static-file-signed/" + exp + "/" + staticFileSignature(secret, key, exp) + "/" + escapeObjectKey(key)
}

func VerifyStaticFileSignature(secret, objectKey, exp, sig string, now time.Time) bool {
	if secret == "" {
		return false
	}
	expires, err := strconv.ParseInt(exp, 10, 64)
	if err != nil || now.Unix() > expires {
		return false
	}
	return hmac.Equal([]byte(sig), []byte(staticFileSignature(secret, objectKey, exp)))
}

func staticFileSignature(secret, key, exp string) string {
	mac := hmac.New(sm3.New, []byte(secret))
	mac.Write([]byte("panda-wiki/static-file-signed|" + key + "|" + exp))
	return hex.EncodeToString(mac.Sum(nil))
}

// ProductDocsURL returns the product documentation link used in bot footers.
func ProductDocsURL() string {
	if v := strings.TrimSpace(os.Getenv("PRODUCT_DOCS_URL")); v != "" {
		return v
	}
	return ""
}

// ModelAutoBaseURL returns the auto model gateway base URL.
func ModelAutoBaseURL() string {
	if v := strings.TrimSpace(os.Getenv("MODEL_AUTO_BASE_URL")); v != "" {
		return strings.TrimRight(v, "/")
	}
	return ""
}

// IsHTTPURL reports whether s is an absolute http or https URL.
func IsHTTPURL(s string) bool {
	u, err := url.Parse(strings.TrimSpace(s))
	if err != nil || u.Host == "" {
		return false
	}
	return u.Scheme == "http" || u.Scheme == "https"
}

// ResolveObjectURL turns a relative static-file path into an absolute URL served by the API.
func ResolveObjectURL(pathOrURL string) string {
	pathOrURL = strings.TrimSpace(pathOrURL)
	if pathOrURL == "" {
		return ""
	}
	if IsHTTPURL(pathOrURL) {
		return pathOrURL
	}
	key := pathOrURL
	if idx := strings.Index(pathOrURL, "/static-file/"); idx >= 0 {
		key = pathOrURL[idx+len("/static-file/"):]
	}
	key = strings.TrimLeft(key, "/")
	if key == "" {
		return ""
	}
	return internalAPIBaseURL() + "/static-file/" + escapeObjectKey(key)
}
