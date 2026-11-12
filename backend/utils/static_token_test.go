package utils

import (
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestStaticAccessToken(t *testing.T) {
	now := time.Now()
	token := SignStaticAccessToken("secret", "user-1", now.Add(time.Hour))
	if uid, ok := VerifyStaticAccessToken("secret", token, now); !ok || uid != "user-1" {
		t.Fatalf("verify = %q, %v", uid, ok)
	}
	if _, ok := VerifyStaticAccessToken("other", token, now); ok {
		t.Fatal("token signed with another secret must fail")
	}
	if _, ok := VerifyStaticAccessToken("secret", token, now.Add(2*time.Hour)); ok {
		t.Fatal("expired token must fail")
	}
	forged := SignStaticAccessToken("secret", "user-1", now.Add(time.Hour))
	forged = strings.Replace(forged, ".", "x.", 1)
	if _, ok := VerifyStaticAccessToken("secret", forged, now); ok {
		t.Fatal("tampered token must fail")
	}
}

func TestSignedStaticFileURL(t *testing.T) {
	raw := SignedStaticFileURL("secret", "/kb-1/中文 文件.docx", time.Hour)
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	if u.RawQuery != "" || !strings.HasSuffix(u.Path, ".docx") {
		t.Fatalf("signature must live in the path and keep the extension: %s", raw)
	}
	parts := strings.SplitN(strings.TrimPrefix(u.Path, "/static-file-signed/"), "/", 3)
	if len(parts) != 3 || parts[2] != "kb-1/中文 文件.docx" {
		t.Fatalf("unexpected path %q", u.Path)
	}
	if !VerifyStaticFileSignature("secret", parts[2], parts[0], parts[1], time.Now()) {
		t.Fatal("valid signature rejected")
	}
	if VerifyStaticFileSignature("secret", "kb-2/other.docx", parts[0], parts[1], time.Now()) {
		t.Fatal("signature must be bound to the object key")
	}
	if VerifyStaticFileSignature("secret", parts[2], parts[0], parts[1], time.Now().Add(2*time.Hour)) {
		t.Fatal("expired signature accepted")
	}
}
