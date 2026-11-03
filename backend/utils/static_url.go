package utils

import (
	"net/url"
	"os"
	"strings"
)

// StaticFilePublicURL builds an internal object URL for MinIO/static-file access.
// Endpoint comes from S3_ENDPOINT (or config-provided host); scheme from S3_USE_SSL.
func StaticFilePublicURL(endpoint, objectKey string) string {
	host := strings.TrimSpace(endpoint)
	if host == "" {
		host = strings.TrimSpace(os.Getenv("S3_ENDPOINT"))
	}
	host = strings.TrimPrefix(strings.TrimPrefix(host, "https://"), "http://")
	host = strings.TrimRight(host, "/")
	if host == "" || strings.TrimSpace(objectKey) == "" {
		return ""
	}
	scheme := "http"
	if strings.EqualFold(os.Getenv("S3_USE_SSL"), "true") {
		scheme = "https"
	}
	u := url.URL{
		Scheme: scheme,
		Host:   host,
		Path:   "/static-file/" + strings.TrimLeft(objectKey, "/"),
	}
	return u.String()
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

// ResolveObjectURL turns a relative static-file path into an absolute object URL.
func ResolveObjectURL(pathOrURL string) string {
	pathOrURL = strings.TrimSpace(pathOrURL)
	if pathOrURL == "" {
		return ""
	}
	lower := strings.ToLower(pathOrURL)
	if strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://") {
		return pathOrURL
	}
	key := pathOrURL
	if idx := strings.Index(pathOrURL, "/static-file/"); idx >= 0 {
		key = pathOrURL[idx+len("/static-file/"):]
	}
	return StaticFilePublicURL("", strings.TrimLeft(key, "/"))
}
