package share

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestLoginVerifyAndPermission(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		switch r.URL.Path {
		case loginVerifyPath:
			var req struct {
				JSONArg struct {
					Tgt string `json:"tgt"`
				} `json:"jsonArg"`
			}
			if err := json.Unmarshal(body, &req); err != nil || req.JSONArg.Tgt != "token-1" {
				t.Fatalf("unexpected loginVerify body: %s", body)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"code": 0,
				"data": map[string]any{
					"trueName":  "张三",
					"loginName": "test",
					"userId":    144,
				},
			})
		case checkPermissionsPath:
			var req struct {
				JSONArg struct {
					Permissions []string `json:"permissions"`
				} `json:"jsonArg"`
			}
			if err := json.Unmarshal(body, &req); err != nil || len(req.JSONArg.Permissions) != 1 || req.JSONArg.Permissions[0] != kfsUserPermission {
				t.Fatalf("unexpected permission body: %s", body)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"code":    0,
				"data":    []string{kfsUserPermission},
				"message": "",
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := newCloudAuthClient(server.URL)
	user, err := client.loginVerify(context.Background(), "token-1", "192.0.0.1")
	if err != nil {
		t.Fatal(err)
	}
	if user.LoginName != "test" || user.UserID != 144 {
		t.Fatalf("unexpected user: %+v", user)
	}
	allowed, err := client.hasKfsUserPermission(context.Background(), user, "192.0.0.1")
	if err != nil || !allowed {
		t.Fatalf("expected permission, allowed=%v err=%v", allowed, err)
	}
}

func TestPermissionRejectedWhenKfsUserMissing(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"code": 0,
			"data": []string{"100000:function:kfsadmin"},
		})
	}))
	defer server.Close()

	client := newCloudAuthClient(server.URL + "/")
	allowed, err := client.hasKfsUserPermission(context.Background(), cloudSessionUser{
		LoginName: "test",
		UserID:    168,
	}, "192.168.90.10")
	if err != nil {
		t.Fatal(err)
	}
	if allowed {
		t.Fatal("missing kfsuser permission should fail")
	}
}
