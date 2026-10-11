package share

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
)

func TestDecodeUserLink(t *testing.T) {
	payload, err := json.Marshal(map[string]string{
		"username": "zhangsan",
		"region":   "华东",
	})
	if err != nil {
		t.Fatal(err)
	}
	encoded := base64.StdEncoding.EncodeToString(payload)
	username, region, err := decodeUserLink(strings.ReplaceAll(encoded, "+", " "))
	if err != nil {
		t.Fatal(err)
	}
	if username != "zhangsan" || region != "华东" {
		t.Fatalf("unexpected payload: %s %s", username, region)
	}

	chinese, err := json.Marshal(map[string]string{
		"用户名": "lisi",
		"地区":  "华北",
	})
	if err != nil {
		t.Fatal(err)
	}
	username, region, err = decodeUserLink(base64.RawURLEncoding.EncodeToString(chinese))
	if err != nil {
		t.Fatal(err)
	}
	if username != "lisi" || region != "华北" {
		t.Fatalf("unexpected chinese payload: %s %s", username, region)
	}

	if _, _, err := decodeUserLink("not-base64"); err == nil {
		t.Fatal("invalid payload should fail")
	}
}
