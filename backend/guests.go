package main

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/pcaravaca/HMSMaster/backend/models"
	"gorm.io/gorm"
)

func listGuests(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var guests []models.Guest
		q := db.Order("last_name, first_name")
		if hid := c.Query("hotel_id"); hid != "" {
			q = q.Where("hotel_id = ?", parseID(hid))
		}
		if term := strings.TrimSpace(c.Query("q")); term != "" {
			like := "%" + term + "%"
			q = q.Where(`LOWER(first_name) LIKE LOWER(?) OR LOWER(last_name) LIKE LOWER(?)
				OR LOWER(email) LIKE LOWER(?) OR LOWER(document_number) LIKE LOWER(?)`,
				like, like, like, like)
		}
		q.Find(&guests)
		c.JSON(http.StatusOK, guests)
	}
}

func createGuestHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var g models.Guest
		if err := c.ShouldBindJSON(&g); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if g.HotelID == 0 || strings.TrimSpace(g.FirstName) == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "hotel_id and first_name are required"})
			return
		}
		var hotel models.Hotel
		if err := db.First(&hotel, g.HotelID).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "hotel not found"})
			return
		}
		if err := db.Create(&g).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusCreated, g)
	}
}

func getGuest(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var g models.Guest
		if err := db.First(&g, parseID(c.Param("id"))).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "guest not found"})
			return
		}
		var stays []models.Reservation
		db.Where("guest_id = ?", g.ID).Order("check_in DESC").Find(&stays)
		c.JSON(http.StatusOK, gin.H{"guest": g, "reservations": stays})
	}
}

func updateGuest(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var g models.Guest
		if err := db.First(&g, parseID(c.Param("id"))).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "guest not found"})
			return
		}
		if err := c.ShouldBindJSON(&g); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if strings.TrimSpace(g.FirstName) == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "first_name is required"})
			return
		}
		if err := db.Save(&g).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, g)
	}
}
