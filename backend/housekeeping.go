package main

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pcaravaca/HMSMaster/backend/models"
	"gorm.io/gorm"
)

var taskTransitions = map[string][]string{
	"pending":     {"in_progress", "done"},
	"in_progress": {"done"},
	"done":        {},
}

func validTaskTransition(from, to string) bool {
	for _, allowed := range taskTransitions[from] {
		if allowed == to {
			return true
		}
	}
	return false
}

func currentUsername(c *gin.Context) string {
	if raw, exists := c.Get("claims"); exists {
		if claims, ok := raw.(map[string]any); ok {
			if v, ok := claims["sub"].(string); ok {
				return v
			}
		}
	}
	return ""
}

// ensureCheckoutTask creates one cleaning task per check-out. It is safe to
// call repeatedly: an existing task for the reservation is reused.
func ensureCheckoutTask(db *gorm.DB, r *models.Reservation) {
	var existing models.HousekeepingTask
	if err := db.Where("reservation_id = ? AND type = ?", r.ID, "checkout_clean").First(&existing).Error; err == nil {
		return
	}
	task := models.HousekeepingTask{
		HotelID: r.HotelID, RoomID: r.RoomID, RoomNumber: r.RoomNumber,
		ReservationID: r.ID, Type: "checkout_clean", Status: "pending",
	}
	db.Create(&task)
}

func listHousekeepingTasks(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		q := db.Model(&models.HousekeepingTask{}).Order("id DESC")
		if v := parseID(c.Query("hotel_id")); v != 0 {
			q = q.Where("hotel_id = ?", v)
		}
		if v := c.Query("status"); v != "" {
			q = q.Where("status = ?", v)
		}
		if v := c.Query("assigned_to"); v != "" {
			q = q.Where("assigned_to = ?", v)
		}
		var tasks []models.HousekeepingTask
		if err := q.Find(&tasks).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, tasks)
	}
}

func createHousekeepingTask(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var body struct {
			HotelID    uint   `json:"hotel_id"`
			RoomID     uint   `json:"room_id"`
			Type       string `json:"type"`
			AssignedTo string `json:"assigned_to"`
			Notes      string `json:"notes"`
		}
		if err := c.ShouldBindJSON(&body); err != nil || body.HotelID == 0 || body.RoomID == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "hotel_id and room_id are required"})
			return
		}
		var room models.Room
		if err := db.First(&room, body.RoomID).Error; err != nil || room.HotelID != body.HotelID {
			c.JSON(http.StatusBadRequest, gin.H{"error": "room not found in this hotel"})
			return
		}
		kind := body.Type
		if kind != "stayover" && kind != "maintenance" {
			kind = "checkout_clean"
		}
		task := models.HousekeepingTask{
			HotelID: body.HotelID, RoomID: room.ID, RoomNumber: room.Number,
			Type: kind, Status: "pending", AssignedTo: body.AssignedTo, Notes: body.Notes,
		}
		if err := db.Create(&task).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusCreated, task)
	}
}

func updateHousekeepingTask(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var task models.HousekeepingTask
		if err := db.First(&task, parseID(c.Param("id"))).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "task not found"})
			return
		}
		var body struct {
			Status     string `json:"status"`
			AssignedTo string `json:"assigned_to"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if body.Status != "" {
			if !validTaskTransition(task.Status, body.Status) {
				c.JSON(http.StatusConflict, gin.H{"error": "invalid task transition " + task.Status + " -> " + body.Status})
				return
			}
			task.Status = body.Status
		}
		if body.AssignedTo != "" {
			task.AssignedTo = body.AssignedTo
		} else if task.AssignedTo == "" {
			task.AssignedTo = currentUsername(c)
		}
		if task.Status == "done" {
			now := time.Now().UTC()
			task.CompletedAt = &now
		}
		db.Save(&task)
		if task.Status == "done" {
			db.Model(&models.Room{}).Where("id = ?", task.RoomID).Update("status", "available")
		}
		c.JSON(http.StatusOK, task)
	}
}
