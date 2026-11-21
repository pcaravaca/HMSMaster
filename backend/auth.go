package main

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func jwtSecret() string { return os.Getenv("HMS_JWT_SECRET") }

// loginHandler issues a 12-hour HS256 token for the single admin account
// configured via HMS_ADMIN_USER / HMS_ADMIN_PASSWORD. No secrets in the repo.
func loginHandler(c *gin.Context) {
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
	user := os.Getenv("HMS_ADMIN_USER")
	pass := os.Getenv("HMS_ADMIN_PASSWORD")
	if user == "" || pass == "" || body.Username != user || body.Password != pass {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}
	claims := jwt.MapClaims{"sub": body.Username, "exp": time.Now().Add(12 * time.Hour).Unix()}
	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"token": signed, "expires_in": 43200})
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
		token, err := jwt.Parse(strings.TrimSpace(header[7:]), func(t *jwt.Token) (interface{}, error) {
			if t.Method.Alg() != jwt.SigningMethodHS256.Alg() {
				return nil, fmt.Errorf("unexpected signing method")
			}
			return []byte(secret), nil
		})
		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
			return
		}
		c.Next()
	}
}
