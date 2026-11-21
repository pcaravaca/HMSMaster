package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestNormalizeRole(t *testing.T) {
	for _, in := range []string{"ADMIN", " Manager ", "reception", "housekeeping", "accounting"} {
		if got := normalizeRole(in); got == "reception" && in != "reception" {
			t.Fatalf("role %q normalized to reception", in)
		}
	}
	if got := normalizeRole("wizard"); got != "reception" {
		t.Fatalf("unknown role = %q", got)
	}
}

func TestRequireRole(t *testing.T) {
	t.Setenv("HMS_JWT_SECRET", "roles-test-secret")
	gin.SetMode(gin.TestMode)
	cases := []struct {
		name   string
		claims map[string]any
		want   int
	}{
		{"admin", map[string]any{"role": "admin"}, 200},
		{"reception blocked", map[string]any{"role": "reception"}, 403},
		{"legacy token", map[string]any{}, 200},
		{"no auth claims", nil, 200},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := gin.New()
			claims := tc.claims
			r.Use(func(c *gin.Context) {
				if claims != nil {
					c.Set("claims", claims)
				}
				c.Next()
			})
			r.GET("/admin-only", requireRole("admin"), func(c *gin.Context) { c.JSON(200, gin.H{"ok": true}) })
			w := httptest.NewRecorder()
			r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/admin-only", nil))
			if w.Code != tc.want {
				t.Fatalf("got %d want %d: %s", w.Code, tc.want, w.Body.String())
			}
		})
	}
}

func TestAuthenticateEnvFallback(t *testing.T) {
	t.Setenv("HMS_ADMIN_USER", "boss")
	t.Setenv("HMS_ADMIN_PASSWORD", "secret-pass")
	role, _, _, ok := authenticateUser(nil, "boss", "secret-pass")
	if !ok || role != "admin" {
		t.Fatalf("env fallback failed: role=%q ok=%v", role, ok)
	}
	if _, _, _, ok := authenticateUser(nil, "boss", "wrong"); ok {
		t.Fatal("bad password accepted")
	}
}
