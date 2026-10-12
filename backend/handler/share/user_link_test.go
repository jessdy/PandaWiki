package share

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"

	"github.com/chaitin/panda-wiki/consts"
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
	link, err := decodeUserLink(strings.ReplaceAll(encoded, "+", " "))
	if err != nil {
		t.Fatal(err)
	}
	if link.Username != "zhangsan" || link.Region != "华东" || link.Role != consts.UserRoleGuest {
		t.Fatalf("unexpected payload: %+v", link)
	}

	chinese, err := json.Marshal(map[string]string{
		"用户名": "lisi",
		"地区":  "华北",
		"角色":  "管理员",
	})
	if err != nil {
		t.Fatal(err)
	}
	link, err = decodeUserLink(base64.RawURLEncoding.EncodeToString(chinese))
	if err != nil {
		t.Fatal(err)
	}
	if link.Username != "lisi" || link.Region != "华北" || link.Role != consts.UserRoleUser {
		t.Fatalf("unexpected chinese payload: %+v", link)
	}

	admin, err := json.Marshal(map[string]string{
		"username": "wangwu",
		"role":     "admin",
	})
	if err != nil {
		t.Fatal(err)
	}
	link, err = decodeUserLink(base64.StdEncoding.EncodeToString(admin))
	if err != nil {
		t.Fatal(err)
	}
	if link.Role != consts.UserRoleUser {
		t.Fatalf("admin role should map to manager: %s", link.Role)
	}

	invalidRole, err := json.Marshal(map[string]string{
		"username": "zhaoliu",
		"role":     "super",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := decodeUserLink(base64.StdEncoding.EncodeToString(invalidRole)); err == nil {
		t.Fatal("invalid role should fail")
	}

	if _, err := decodeUserLink("not-base64"); err == nil {
		t.Fatal("invalid payload should fail")
	}
}
