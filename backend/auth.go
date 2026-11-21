package main

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"
)

func jwtSecret() string { return os.Getenv("HMS_JWT_SECRET") }

// loginHandler issues a 12-hour HS256 token. Users come from the DB (bcrypt)
// with the env admin as backward-compatible fallback. No secrets in the repo.
func loginHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		secret := jwtSecret()
		if secret == "" {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "authentication not configured"})
			return
		}
		var body struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		role, uid, hotelID, ok := authenticateUser(db, body.Username, body.Password)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
			return
		}
		claims := jwt.MapClaims{
			"sub": body.Username, "role": role, "uid": uid, "hotel_id": hotelID,
			"exp": time.Now().Add(12 * time.Hour).Unix(),
		}
		signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"token": signed, "expires_in": 43200, "role": role})
	}
}

// authMeHandler exposes the current identity to the UI for capability checks.
func authMeHandler(c *gin.Context) {
	username, role := "", "admin"
	if raw, exists := c.Get("claims"); exists {
		if claims, ok := raw.(map[string]any); ok {
			if v, ok := claims["sub"].(string); ok {
				username = v
			}
			if v, ok := claims["role"].(string); ok && v != "" {
				role = v
			}
		}
	}
	c.JSON(http.StatusOK, gin.H{"username": username, "role": role})
}

// authMiddleware is a no-op while HMS_JWT_SECRET is unset (local development).
// Once set, every /api route except /api/auth/login requires a Bearer token.
func authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		secret := jwtSecret()
		if secret == "" {
			c.Next()
			return
		}
		header := c.GetHeader("Authorization")
		if !strings.HasPrefix(header, "Bearer ") || len(header) <= 7 {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "bearer token required"})
			return
		}
		claims := jwt.MapClaims{}
		token, err := jwt.ParseWithClaims(strings.TrimSpace(header[7:]), claims, func(t *jwt.Token) (interface{}, error) {
			if t.Method.Alg() != jwt.SigningMethodHS256.Alg() {
				return nil, fmt.Errorf("unexpected signing method")
			}
			return []byte(secret), nil
		})
		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
			return
		}
		c.Set("claims", map[string]any(claims))
		c.Next()
	}
}
