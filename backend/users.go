package main

import (
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/pcaravaca/HMSMaster/backend/models"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

var validRoles = map[string]bool{
	"admin": true, "manager": true, "reception": true, "housekeeping": true, "accounting": true,
}

func normalizeRole(role string) string {
	r := strings.ToLower(strings.TrimSpace(role))
	if validRoles[r] {
		return r
	}
	return "reception"
}

// seedAdminUser keeps the env-configured admin usable after adding DB users.
func seedAdminUser(db *gorm.DB) {
	user := os.Getenv("HMS_ADMIN_USER")
	pass := os.Getenv("HMS_ADMIN_PASSWORD")
	if user == "" || pass == "" {
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(pass), bcrypt.DefaultCost)
	if err != nil {
		return
	}
	var u models.User
	if err := db.Where("username = ?", user).First(&u).Error; err != nil {
		u = models.User{Username: user, Role: "admin"}
	}
	u.PasswordHash = string(hash)
	u.Role = "admin"
	u.Active = true
	if u.ID == 0 {
		db.Create(&u)
	} else {
		db.Save(&u)
	}
}

// requireRole guards mutations. Tokens issued before roles existed are treated
// as admin because only the admin account could log in back then.
func requireRole(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if jwtSecret() == "" {
			c.Next()
			return
		}
		claimValue, exists := c.Get("claims")
		role := "admin"
		if exists {
			if claims, ok := claimValue.(map[string]any); ok {
				if r, ok := claims["role"].(string); ok && r != "" {
					role = r
				}
			}
		}
		for _, allowed := range roles {
			if role == allowed {
				c.Next()
				return
			}
		}
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "role " + role + " cannot perform this action"})
	}
}

// guardRoles chains a role check with a regular handler for route registration.
func guardRoles(handler gin.HandlerFunc, roles ...string) gin.HandlerFunc {
	guard := requireRole(roles...)
	return func(c *gin.Context) {
		guard(c)
		if c.IsAborted() {
			return
		}
		handler(c)
	}
}

func listUsers(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var users []models.User
		if err := db.Order("id").Find(&users).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, users)
	}
}

func createUser(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var body struct {
			Username string `json:"username"`
			Password string `json:"password"`
			Role     string `json:"role"`
			HotelID  uint   `json:"hotel_id"`
		}
		if err := c.ShouldBindJSON(&body); err != nil || body.Username == "" || len(body.Password) < 8 || !validRoles[normalizeRole(body.Role)] {
			c.JSON(http.StatusBadRequest, gin.H{"error": "username, password (min 8 chars) and valid role are required"})
			return
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(body.Password), bcrypt.DefaultCost)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		u := models.User{Username: body.Username, PasswordHash: string(hash), Role: normalizeRole(body.Role), HotelID: body.HotelID, Active: true}
		if err := db.Create(&u).Error; err != nil {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusCreated, u)
	}
}

func updateUser(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var u models.User
		if err := db.First(&u, parseID(c.Param("id"))).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
			return
		}
		var body struct {
			Role   *string `json:"role"`
			Active *bool   `json:"active"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if body.Role != nil {
			u.Role = normalizeRole(*body.Role)
		}
		if body.Active != nil {
			u.Active = *body.Active
		}
		db.Save(&u)
		c.JSON(http.StatusOK, u)
	}
}

func deleteUser(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := parseID(c.Param("id"))
		var u models.User
		if err := db.First(&u, id).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
			return
		}
		if u.Username == os.Getenv("HMS_ADMIN_USER") {
			c.JSON(http.StatusConflict, gin.H{"error": "the env admin account cannot be deleted"})
			return
		}
		db.Delete(&u)
		c.JSON(http.StatusOK, gin.H{"deleted": true})
	}
}

func authenticateUser(db *gorm.DB, username, password string) (role string, uid, hotelID uint, ok bool) {
	var u models.User
	if db != nil {
		if err := db.Where("username = ? AND active = ?", username, true).First(&u).Error; err == nil {
			if bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)) == nil {
				return u.Role, u.ID, u.HotelID, true
			}
			return "", 0, 0, false
		}
	}
	if username == os.Getenv("HMS_ADMIN_USER") && password == os.Getenv("HMS_ADMIN_PASSWORD") && username != "" {
		return "admin", 0, 0, true
	}
	return "", 0, 0, false
}
