package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func authTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/api/auth/login", loginHandler(nil))
	api := r.Group("/api")
	api.Use(authMiddleware())
	api.GET("/stats", func(c *gin.Context) { c.JSON(200, gin.H{"ok": true}) })
	return r
}

func TestAuthDisabledByDefault(t *testing.T) {
	t.Setenv("HMS_JWT_SECRET", "")
	r := authTestRouter()
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/stats", nil))
	if w.Code != 200 { t.Fatalf("expected open dev mode, got %d", w.Code) }
}

func TestLoginAndBearer(t *testing.T) {
	t.Setenv("HMS_JWT_SECRET", "test-secret-only")
	t.Setenv("HMS_ADMIN_USER", "admin")
	t.Setenv("HMS_ADMIN_PASSWORD", "test-pass")
	r := authTestRouter()

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/stats", nil))
	if w.Code != 401 { t.Fatalf("expected 401 without token, got %d", w.Code) }

	w = httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(`{"username":"admin","password":"wrong"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != 401 { t.Fatalf("expected 401 on bad password, got %d", w.Code) }

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(`{"username":"admin","password":"test-pass"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != 200 { t.Fatalf("login failed: %d %s", w.Code, w.Body.String()) }
	var out map[string]any
	mustJSON(t, w.Body.String(), &out)
	tok, _ := out["token"].(string)
	if tok == "" { t.Fatal("no token in login response") }

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/stats", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	r.ServeHTTP(w, req)
	if w.Code != 200 { t.Fatalf("expected 200 with token, got %d: %s", w.Code, w.Body.String()) }

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/stats", nil)
	req.Header.Set("Authorization", "Bearer "+tok+"tampered")
	r.ServeHTTP(w, req)
	if w.Code != 401 { t.Fatalf("expected 401 on tampered token, got %d", w.Code) }
}

func mustJSON(t *testing.T, body string, out any) {
	t.Helper()
	if err := json.Unmarshal([]byte(body), out); err != nil {
		t.Fatalf("bad JSON %q: %v", body, err)
	}
}
