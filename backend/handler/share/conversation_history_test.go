package share

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/sessions"
	echosession "github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"

	"github.com/chaitin/panda-wiki/domain"
)

func TestShareAuthUserID(t *testing.T) {
	e := echo.New()
	c := e.NewContext(httptest.NewRequest("GET", "/", nil), httptest.NewRecorder())
	if _, ok := shareAuthUserID(c, "kb-1"); ok {
		t.Fatal("anonymous context must not expose history")
	}

	c.Set("user_id", uint(42))
	userID, ok := shareAuthUserID(c, "kb-1")
	if !ok || userID != 42 {
		t.Fatalf("got userID=%d ok=%v", userID, ok)
	}

	c.Set("user_id", "42")
	if _, ok := shareAuthUserID(c, "kb-1"); ok {
		t.Fatal("invalid user id type must not expose history")
	}
}

func TestShareAuthUserIDFromSession(t *testing.T) {
	e := echo.New()
	e.Use(echosession.Middleware(sessions.NewCookieStore([]byte("test-secret"))))
	e.GET("/login", func(c echo.Context) error {
		sess, err := echosession.Get(domain.SessionName, c)
		if err != nil {
			return err
		}
		sess.Values["kb_id"] = "kb-1"
		sess.Values["user_id"] = uint(23)
		if err := sess.Save(c.Request(), c.Response()); err != nil {
			return err
		}
		return c.NoContent(http.StatusNoContent)
	})
	e.GET("/check", func(c echo.Context) error {
		userID, ok := shareAuthUserID(c, c.QueryParam("kb_id"))
		if !ok {
			return c.NoContent(http.StatusUnauthorized)
		}
		return c.String(http.StatusOK, http.StatusText(int(userID)))
	})

	loginRecorder := httptest.NewRecorder()
	e.ServeHTTP(loginRecorder, httptest.NewRequest(http.MethodGet, "/login", nil))
	cookies := loginRecorder.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("login did not create a session cookie")
	}

	check := func(kbID string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, "/check?kb_id="+kbID, nil)
		req.AddCookie(cookies[0])
		recorder := httptest.NewRecorder()
		e.ServeHTTP(recorder, req)
		return recorder
	}
	if recorder := check("kb-1"); recorder.Code != http.StatusOK {
		t.Fatalf("matching session returned status %d", recorder.Code)
	}
	if recorder := check("kb-2"); recorder.Code != http.StatusUnauthorized {
		t.Fatalf("cross-kb session returned status %d", recorder.Code)
	}
}
