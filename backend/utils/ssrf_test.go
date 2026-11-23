package utils

import "testing"

func TestRejectInternalURL(t *testing.T) {
	blocked := []string{
		"http://127.0.0.1/admin",
		"http://localhost/secret",
		"http://10.1.2.3/a",
		"http://192.168.1.1/",
		"http://169.254.169.254/latest",
		"file:///etc/passwd",
		"http://[::1]/",
		"",
	}
	for _, raw := range blocked {
		if err := RejectInternalURL(raw); err == nil {
			t.Fatalf("expected reject %q", raw)
		}
	}
	if err := RejectInternalURL("https://1.1.1.1/wiki"); err != nil {
		t.Fatal(err)
	}
}
