package share

import "testing"

func TestStaticObjectKey(t *testing.T) {
	cases := map[string]string{
		"kb/a.png":            "kb/a.png",
		"/kb/a.png":           "kb/a.png",
		"kb/../../etc/passwd": "etc/passwd",
		"kb/%E4%B8%AD.png":    "kb/中.png",
		"":                    "",
		"/":                   "",
	}
	for in, want := range cases {
		if got := staticObjectKey(in); got != want {
			t.Errorf("staticObjectKey(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestStaticContentType(t *testing.T) {
	cases := []struct{ stored, key, want string }{
		{"image/png", "kb/a.png", "image/png"},
		{"application/octet-stream", "kb/a.pdf", "application/pdf"},
		{"", "kb/a.svg", "image/svg+xml"},
		{"text/plain", "kb/a.txt", "text/plain; charset=utf-8"},
		{"", "kb/noext", "application/octet-stream"},
	}
	for _, c := range cases {
		if got := staticContentType(c.stored, c.key); got != c.want {
			t.Errorf("staticContentType(%q, %q) = %q, want %q", c.stored, c.key, got, c.want)
		}
	}
	if inlineStaticTypes["image/svg+xml"] || inlineStaticTypes["text/html"] {
		t.Fatal("svg and html must be served as attachments")
	}
}
