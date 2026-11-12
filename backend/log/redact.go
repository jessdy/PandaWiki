package log

import (
	"encoding/json"
	"log/slog"
	"reflect"
	"regexp"
	"strings"
	"time"
	"unicode"
)

const masked = "******"

type keyClass int

const (
	keyPlain keyClass = iota
	keySecret
	keyPII
)

// 按字段名归一化（小写、去掉分隔符）后做后缀匹配，max_tokens 这类计数字段不会命中 token。
var (
	secretKeySuffixes = []string{
		"password", "passwd", "pwd", "secret", "token", "apikey", "authorization",
		"cookie", "accesskey", "secretkey", "privatekey", "sessionid", "ticket", "tgt",
	}
	piiKeySuffixes = []string{"mobile", "phone", "telephone", "email", "idcard"}

	sensitiveQueryParam = regexp.MustCompile(`(?i)([?&](?:[\w-]*(?:token|ticket|password|api_key|apikey|secret|signature|credential)|tgt|code|sign)=)[^&#\s"]+`)
	sensitiveJSONField  = regexp.MustCompile(`(?i)("[\w-]*(?:password|passwd|secret|token|api_key|apikey|authorization|ticket|access_key)"\s*:\s*")[^"]*(")|("tgt"\s*:\s*")[^"]*(")`)
	mobilePattern       = regexp.MustCompile(`(^|[^0-9])(1[3-9][0-9])[0-9]{4}([0-9]{4})($|[^0-9])`)
	emailPattern        = regexp.MustCompile(`([A-Za-z0-9._%+-])[A-Za-z0-9._%+-]*(@[A-Za-z0-9.-]+\.[A-Za-z]{2,})`)
)

func classifyKey(key string) keyClass {
	var b strings.Builder
	for _, r := range strings.ToLower(key) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		}
	}
	n := b.String()
	for _, s := range secretKeySuffixes {
		if strings.HasSuffix(n, s) {
			return keySecret
		}
	}
	for _, s := range piiKeySuffixes {
		if strings.HasSuffix(n, s) {
			return keyPII
		}
	}
	return keyPlain
}

func redactString(s string) string {
	s = sensitiveQueryParam.ReplaceAllString(s, "${1}"+masked)
	s = sensitiveJSONField.ReplaceAllString(s, "${1}${3}"+masked+"${2}${4}")
	s = mobilePattern.ReplaceAllString(s, "${1}${2}****${3}${4}")
	s = emailPattern.ReplaceAllString(s, "${1}***${2}")
	return s
}

func maskPII(s string) string {
	if s == "" {
		return s
	}
	return redactString(s)
}

func redactJSON(v any) any {
	switch t := v.(type) {
	case map[string]any:
		for k, val := range t {
			switch classifyKey(k) {
			case keySecret:
				t[k] = masked
			case keyPII:
				if s, ok := val.(string); ok {
					t[k] = maskPII(s)
				} else {
					t[k] = masked
				}
			default:
				t[k] = redactJSON(val)
			}
		}
		return t
	case []any:
		for i := range t {
			t[i] = redactJSON(t[i])
		}
		return t
	case string:
		return redactString(t)
	default:
		return v
	}
}

// redactComposite 把结构体 / map / 切片转成 JSON 后按字段名脱敏，其余类型原样返回。
func redactComposite(v any) (string, bool) {
	if v == nil {
		return "", false
	}
	switch v.(type) {
	case error, time.Time, time.Duration, []byte, json.RawMessage:
		return "", false
	}
	rv := reflect.ValueOf(v)
	for rv.Kind() == reflect.Pointer || rv.Kind() == reflect.Interface {
		if rv.IsNil() {
			return "", false
		}
		rv = rv.Elem()
	}
	switch rv.Kind() {
	case reflect.Struct, reflect.Map, reflect.Slice, reflect.Array:
	default:
		return "", false
	}
	raw, err := json.Marshal(v)
	if err != nil {
		return "", false
	}
	var generic any
	if err := json.Unmarshal(raw, &generic); err != nil {
		return "", false
	}
	out, err := json.Marshal(redactJSON(generic))
	if err != nil {
		return "", false
	}
	return string(out), true
}

func redactAttr(_ []string, a slog.Attr) slog.Attr {
	switch classifyKey(a.Key) {
	case keySecret:
		if a.Value.Kind() == slog.KindString && a.Value.String() == "" {
			return a
		}
		return slog.String(a.Key, masked)
	case keyPII:
		return slog.String(a.Key, maskPII(a.Value.Resolve().String()))
	}
	v := a.Value.Resolve()
	switch v.Kind() {
	case slog.KindString:
		return slog.String(a.Key, redactString(v.String()))
	case slog.KindAny:
		if s, ok := redactComposite(v.Any()); ok {
			return slog.String(a.Key, s)
		}
		if err, ok := v.Any().(error); ok && err != nil {
			return slog.String(a.Key, redactString(err.Error()))
		}
	}
	return a
}
