package usecase

import (
	"testing"

	"github.com/chaitin/panda-wiki/domain"
)

func TestNormalizeCloudAuthSetting(t *testing.T) {
	setting, err := NormalizeCloudAuthSetting(&domain.CloudAuthSetting{
		Host: " 10.10.10.252 ",
		Port: 8080,
	})
	if err != nil {
		t.Fatal(err)
	}
	if setting.Host != "10.10.10.252" || setting.Port != 8080 {
		t.Fatalf("unexpected setting: %+v", setting)
	}
	if got := CloudAuthBaseURL(setting); got != "http://10.10.10.252:8080" {
		t.Fatalf("unexpected url: %s", got)
	}

	if _, err := NormalizeCloudAuthSetting(&domain.CloudAuthSetting{
		Host: "http://10.10.10.252",
		Port: 8080,
	}); err == nil {
		t.Fatal("scheme should be rejected")
	}
	if _, err := NormalizeCloudAuthSetting(&domain.CloudAuthSetting{
		Host: "10.10.10.252",
	}); err == nil {
		t.Fatal("missing port should be rejected")
	}
	empty, err := NormalizeCloudAuthSetting(&domain.CloudAuthSetting{})
	if err != nil || empty.Host != "" || empty.Port != 0 {
		t.Fatalf("empty setting should clear config: %+v %v", empty, err)
	}
}

func TestCloudAuthBaseURLIPv6(t *testing.T) {
	got := CloudAuthBaseURL(&domain.CloudAuthSetting{Host: "::1", Port: 8080})
	if got != "http://[::1]:8080" {
		t.Fatalf("unexpected ipv6 url: %s", got)
	}
}
